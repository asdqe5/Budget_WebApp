// 프로젝트 결산 프로그램
//
// Description : http 디테일 페이지 관련 스크립트

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize/v2"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// handleDetailSMFunc 메인 페이지에서 각 프로젝트의 Detail 버튼을 눌렀을 때 실행하는 함수이다.
func handleDetailSMFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// default 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < DefaultLevel {
		http.Redirect(w, r, "/invalidaccess", http.StatusSeeOther)
		return
	}

	// mongoDB client 연결
	credential := options.Credential{
		Username: *flagDBID,
		Password: *flagDBPW,
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(*flagMongoDBURI).SetAuth(credential))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	adminSetting, err := getAdminSettingFunc(client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	projectID := r.FormValue("id")

	type Info struct {
		Date      string // 날짜
		Payment   string // 월별 매출
		LaborCost string // 월별 총 인건비
		Vendor    string // 월별 외주비
		Revenue   string // 월별 수익
	}

	type Recipe struct {
		Token       Token
		UpdatedTime string
		Status      []Status // 프로젝트 Status

		Project     Project           // Detail 프로젝트
		Vendors     map[string]string // 프로젝트에 해당하는 벤더 업체별 금액 정보
		MonthlyInfo []Info            // 프로젝트의 월별 데이터
		CostSum     map[string]string // 프로젝트 월별 각 지출의 합
	}
	rcp := Recipe{}
	rcp.Token = token
	rcp.UpdatedTime = adminSetting.SGUpdatedTime
	rcp.Status = adminSetting.ProjectStatus

	rcp.Project, err = getProjectFunc(client, projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	vendorSearchWord := "project:" + projectID
	vendors, err := searchVendorFunc(client, vendorSearchWord)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	dates, err := getDatesFunc(rcp.Project.StartDate, rcp.Project.SMEndDate) // 기존의 작업시작과 작업마감 사이의 Date를 가져온다.
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 월별 금액 합계를 위한 변수
	costSum := make(map[string]string)
	paymentSum := 0
	vfxLaborCostSum := 0
	cmLaborCostSum := 0
	progressCostSum := 0
	purchaseCostSum := 0
	vendorSum := 0
	revenueSum := 0

	for _, date := range dates {
		// 월별 매출
		totalPayment := 0
		for _, monthlyPayment := range rcp.Project.SMMonthlyPayment[date] {
			payment, err := decryptAES256Func(monthlyPayment.Expenses)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			paymentInt := 0
			if payment != "" {
				paymentInt, err = strconv.Atoi(payment)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				totalPayment += paymentInt
			}
		}
		paymentSum += totalPayment

		// 월별 지출 - 인건비
		vfxLaborCost, err := decryptAES256Func(rcp.Project.SMMonthlyLaborCost[date].VFX)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		vfxLaborCostInt := 0
		if vfxLaborCost != "" {
			vfxLaborCostInt, err = strconv.Atoi(vfxLaborCost)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		vfxLaborCostSum += vfxLaborCostInt

		cmLaborCost, err := decryptAES256Func(rcp.Project.SMMonthlyLaborCost[date].CM)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cmLaborCostInt := 0
		if cmLaborCost != "" {
			cmLaborCostInt, err = strconv.Atoi(cmLaborCost)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		cmLaborCostSum += cmLaborCostInt
		totalLaborCost := vfxLaborCostInt + cmLaborCostInt

		// 월별 지출 - 진행비
		progressCost, err := decryptAES256Func(rcp.Project.SMMonthlyProgressCost[date])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		progressCostInt := 0
		if progressCost != "" {
			progressCostInt, err = strconv.Atoi(progressCost)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		progressCostSum += progressCostInt

		// 월별 지출 - 구매비
		for _, pur := range rcp.Project.SMMonthlyPurchaseCost[date] {
			purchaseCost, err := decryptAES256Func(pur.Expenses)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			purchaseCostInt, err := strconv.Atoi(purchaseCost)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			purchaseCostSum += purchaseCostInt
		}

		// 월별 지출 - 외주비
		intVendor := 0
		for _, v := range vendors {
			// 계약금, 중도금, 잔금 지출일에 지출완료된 비용만 가져온다.
			if v.Downpayment.Expenses != "" { // 계약금
				downpayment, err := decryptAES256Func(v.Downpayment.Expenses)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				downpaymentInt, err := strconv.Atoi(downpayment)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				downpaymentMonth := dateToMonthFunc(v.Downpayment.Date)
				if date == downpaymentMonth {
					intVendor = intVendor + downpaymentInt
				}
			}
			for _, mp := range v.MediumPlating { // 중도금
				mediumplating, err := decryptAES256Func(mp.Expenses)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				mediumplatingInt, err := strconv.Atoi(mediumplating)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				mediumplatingMonth := dateToMonthFunc(mp.Date)
				if date == mediumplatingMonth {
					intVendor = intVendor + mediumplatingInt
				}
			}
			if v.Balance.Expenses != "" { // 잔금
				balance, err := decryptAES256Func(v.Balance.Expenses)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				balanceInt, err := strconv.Atoi(balance)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				balanceMonth := dateToMonthFunc(v.Balance.Date)
				if date == balanceMonth {
					intVendor = intVendor + balanceInt
				}
			}
		}
		vendorSum += intVendor

		// 월별 수익
		intRevenue, err := getMonthlyRevenueFunc(rcp.Project, date)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		intRevenue -= intVendor
		revenueSum += intRevenue

		encryptedPayment, err := encryptAES256Func(strconv.Itoa(totalPayment))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		encryptedLaborCost, err := encryptAES256Func(strconv.Itoa(totalLaborCost))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		encryptedVendor, err := encryptAES256Func(strconv.Itoa(intVendor))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		encryptedRevenue, err := encryptAES256Func(strconv.Itoa(intRevenue))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		info := Info{
			Date:      date,
			Payment:   encryptedPayment,
			LaborCost: encryptedLaborCost,
			Vendor:    encryptedVendor,
			Revenue:   encryptedRevenue,
		}
		rcp.MonthlyInfo = append(rcp.MonthlyInfo, info)
	}

	// 월별 매출 및 지출 합계 정리
	costSum["Payment"], err = encryptAES256Func(strconv.Itoa(paymentSum))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	costSum["VFX"], err = encryptAES256Func(strconv.Itoa(vfxLaborCostSum))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	costSum["CM"], err = encryptAES256Func(strconv.Itoa(cmLaborCostSum))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	costSum["Progress"], err = encryptAES256Func(strconv.Itoa(progressCostSum))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	costSum["Purchase"], err = encryptAES256Func(strconv.Itoa(purchaseCostSum))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	costSum["ProPur"], err = encryptAES256Func(strconv.Itoa(progressCostSum + purchaseCostSum)) // 진행비와 구매비의 총합
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	costSum["Vendor"], err = encryptAES256Func(strconv.Itoa(vendorSum))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	costSum["Total"], err = encryptAES256Func(strconv.Itoa(vfxLaborCostSum + cmLaborCostSum + progressCostSum + purchaseCostSum + vendorSum))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	costSum["Revenue"], err = encryptAES256Func(strconv.Itoa(revenueSum))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 정산 완료된 프로젝트는 FinishedCost에서 합계를 가져온다.
	if rcp.Project.IsFinished == true {
		paymentInt := 0
		for _, payment := range rcp.Project.Payment {
			decrypted, err := decryptAES256Func(payment.Expenses)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			expenses, err := strconv.Atoi(decrypted)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if payment.Status {
				paymentInt += expenses
			}
		}
		costSum["Payment"], err = encryptAES256Func(strconv.Itoa(paymentInt))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		costSum["VFX"] = rcp.Project.FinishedCost.LaborCost.VFX
		costSum["CM"] = rcp.Project.FinishedCost.LaborCost.CM
		costSum["Progress"] = rcp.Project.FinishedCost.ProgressCost
		costSum["Purchase"] = rcp.Project.FinishedCost.PurchaseCost

		revenueOfFP, err := getRevenueOfFPFunc(rcp.Project)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		revenueOfFP -= vendorSum
		costSum["Revenue"], err = encryptAES256Func(strconv.Itoa(revenueOfFP))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	rcp.CostSum = costSum

	// 프로젝트에 해당하는 외주비를 업체별로 보여주기 위해 정리한다.
	vendorsMap := make(map[string]int)
	for _, v := range vendors {
		if checkStringInListFunc(dateToMonthFunc(v.Downpayment.Date), dates) { // 계약금이 프로젝트의 현재 작업기간에 해당하는지 확인한다
			downpayment, err := decryptAES256Func(v.Downpayment.Expenses)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if downpayment != "" {
				downpaymentInt, err := strconv.Atoi(downpayment)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				vendorsMap[v.Name] += downpaymentInt
			}
		}
		for _, mp := range v.MediumPlating {
			if checkStringInListFunc(dateToMonthFunc(mp.Date), dates) { // 중도금이 프로젝트의 현재 작업기간에 해당하는지 확인한다
				mediumplating, err := decryptAES256Func(mp.Expenses)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if mediumplating != "" {
					mediumplatingInt, err := strconv.Atoi(mediumplating)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					vendorsMap[v.Name] += mediumplatingInt
				}
			}
		}
		if checkStringInListFunc(dateToMonthFunc(v.Balance.Date), dates) { // 잔금이 프로젝트의 현재 작업기간에 해당하는지 확인한다
			balance, err := decryptAES256Func(v.Balance.Expenses)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if balance != "" {
				balanceInt, err := strconv.Atoi(balance)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				vendorsMap[v.Name] += balanceInt
			}
		}
	}

	// 프로젝트에 해당하는 업체별로 정리된 외주비 정보를 암호화한다.
	rcp.Vendors = make(map[string]string)
	for key, value := range vendorsMap {
		rcp.Vendors[key], err = encryptAES256Func(strconv.Itoa(value))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Info 자료구조를 인자로 넘길 수 없어서 json 파일을 생성한다.
	path := os.TempDir() + "/budget/" + token.ID + "/detailSM/" // json으로 바꾼 프로젝트 월별 데이터를 저장할 임시 폴더 경로
	err = createFolderFunc(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonData, _ := json.Marshal(rcp.MonthlyInfo)
	_ = ioutil.WriteFile(path+"/monthlyinfo.json", jsonData, 0644)

	err = genDetailSMExcelFunc(rcp.Project, costSum, token.ID) // 엑셀 파일 미리 생성
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = TEMPLATES.ExecuteTemplate(w, "detail-sm", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// genDetailSMExcelFunc 함수는 엑셀파일을 생성하는 함수이다.
func genDetailSMExcelFunc(project Project, costSum map[string]string, userID string) error {
	path := os.TempDir() + "/budget/" + userID + "/detailSM/"
	excelFileName := fmt.Sprintf("detailSM_%s.xlsx", project.ID)

	// json 파일에서 프로젝트 월별 데이터를 가져온다.
	jsonData, err := ioutil.ReadFile(path + "monthlyinfo.json")
	if err != nil {
		return err
	}

	type Info struct {
		Date      string // 날짜
		Payment   string // 월별 매출
		LaborCost string // 월별 총 인건비
		Vendor    string // 월별 외주비
		Revenue   string // 월별 수익
	}

	var monthlyInfo []Info
	json.Unmarshal(jsonData, &monthlyInfo)

	err = delAllFilesFunc(path)
	if err != nil {
		return err
	}

	// 엑셀 파일 생성
	f := excelize.NewFile()
	sheet := "Sheet1"
	index := f.NewSheet(sheet)
	f.SetActiveSheet(index)

	// 스타일
	style, err := f.NewStyle(`{"alignment":{"horizontal":"center","vertical":"center","wrap_text":true}}`)
	if err != nil {
		return err
	}
	numberStyle, err := f.NewStyle(`{"alignment":{"horizontal":"right","vertical":"center","wrap_text":true}, "number_format": 3}`)
	if err != nil {
		return err
	}
	purStyle, err := f.NewStyle(
		`
		{"alignment":{"horizontal":"center","vertical":"center","wrap_text":true},
		"font":{"bold":true}, 
		"fill":{"type":"pattern","color":["#F3E5B8"],"pattern":1}}
		`)
	if err != nil {
		return err
	}
	purNumStyle, err := f.NewStyle(
		`
		{"alignment":{"horizontal":"right","vertical":"center","wrap_text":true},
		"font":{"bold":true}, 
		"fill":{"type":"pattern","color":["#F3E5B8"],"pattern":1},
		"number_format": 3}
		`)
	if err != nil {
		return err
	}
	totalStyle, err := f.NewStyle(
		`
		{"alignment":{"horizontal":"center","vertical":"center","wrap_text":true},
		"font":{"bold":true}, 
		"fill":{"type":"pattern","color":["#FFC000"],"pattern":1}}
		`)
	if err != nil {
		return err
	}
	totalNumStyle, err := f.NewStyle(
		`
		{"alignment":{"horizontal":"right","vertical":"center","wrap_text":true},
		"font":{"bold":true}, 
		"fill":{"type":"pattern","color":["#FFC000"],"pattern":1},
		"number_format": 3}
		`)
	if err != nil {
		return err
	}

	// 제목 입력
	f.SetCellValue(sheet, "A1", "Status")
	f.MergeCell(sheet, "A1", "A2")
	f.SetCellValue(sheet, "B1", "날짜")
	f.MergeCell(sheet, "B1", "B2")
	f.SetCellValue(sheet, "C1", "월별 매출")
	f.MergeCell(sheet, "C1", "C2")
	f.SetCellValue(sheet, "D1", "내부 인건비")
	f.MergeCell(sheet, "D1", "D2")
	f.SetCellValue(sheet, "E1", "진행비")
	f.MergeCell(sheet, "E1", "E2")
	f.SetCellValue(sheet, "F1", "구매비")
	f.MergeCell(sheet, "F1", "H1")
	f.SetCellValue(sheet, "F2", "업체명")
	f.SetCellValue(sheet, "G2", "내역")
	f.SetCellValue(sheet, "H2", "금액")
	f.SetCellValue(sheet, "I1", "외주비")
	f.MergeCell(sheet, "I1", "I2")
	f.SetCellValue(sheet, "J1", "수익")
	f.MergeCell(sheet, "J1", "J2")

	f.SetColWidth(sheet, "A", "J", 18)
	f.SetColWidth(sheet, "A", "A", 10)
	f.SetRowHeight(sheet, 1, 30)
	f.SetRowHeight(sheet, 2, 30)

	// 데이터 입력
	totalRevenue := 0

	pos := ""  // Data를 적기 위한 위치
	mpos := "" // Data Cell을 합치기 위한 위치
	i := 0     // Data를 적기 위한 기준
	purPosMap := make(map[string]string)
	for _, info := range monthlyInfo {
		date := info.Date

		// 그달에 저장된 구매비 개수를 계산한다.
		purNum := len(project.SMMonthlyPurchaseCost[date])

		// Status
		pos, err = excelize.CoordinatesToCellName(1, i+3) // ex) pos = "A3"
		if err != nil {
			return err
		}
		mpos, err = excelize.CoordinatesToCellName(1, i+3+purNum) // ex) mpos = "A6"
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, project.SMStatus[date])
		f.MergeCell(sheet, pos, mpos)

		// 날짜
		pos, err = excelize.CoordinatesToCellName(2, i+3) // ex) pos = "B3"
		if err != nil {
			return err
		}
		mpos, err = excelize.CoordinatesToCellName(2, i+3+purNum) // ex) mpos = "B6"
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, date)
		f.MergeCell(sheet, pos, mpos)

		// 월별 매출
		pos, err = excelize.CoordinatesToCellName(3, i+3) // ex) pos = "C3"
		if err != nil {
			return err
		}
		mpos, err = excelize.CoordinatesToCellName(3, i+3+purNum) // ex) mpos = "C6"
		if err != nil {
			return err
		}
		payment, err := decryptAES256Func(info.Payment)
		if err != nil {
			return err
		}
		paymentInt := 0
		if payment != "" {
			paymentInt, err = strconv.Atoi(payment)
			if err != nil {
				return err
			}
		}
		f.SetCellValue(sheet, pos, paymentInt)
		f.MergeCell(sheet, pos, mpos)

		// 내부 인건비
		pos, err = excelize.CoordinatesToCellName(4, i+3) // ex) pos = "D3"
		if err != nil {
			return err
		}
		mpos, err = excelize.CoordinatesToCellName(4, i+3+purNum) // ex) pos = "D6"
		if err != nil {
			return err
		}
		laborCost, err := decryptAES256Func(info.LaborCost)
		if err != nil {
			return err
		}
		laborCostInt := 0
		if laborCost != "" {
			laborCostInt, err = strconv.Atoi(laborCost)
			if err != nil {
				return err
			}
		}

		f.SetCellValue(sheet, pos, laborCostInt)
		f.MergeCell(sheet, pos, mpos)

		// 진행비
		pos, err = excelize.CoordinatesToCellName(5, i+3) // ex) pos = "E3"
		if err != nil {
			return err
		}
		mpos, err = excelize.CoordinatesToCellName(5, i+3+purNum) // ex) pos = "E6"
		if err != nil {
			return err
		}
		progressCost, err := decryptAES256Func(project.SMMonthlyProgressCost[date])
		if err != nil {
			return err
		}
		progressCostInt := 0
		if progressCost != "" {
			progressCostInt, err = strconv.Atoi(progressCost)
			if err != nil {
				return err
			}
		}
		f.SetCellValue(sheet, pos, progressCostInt)
		f.MergeCell(sheet, pos, mpos)

		// 구매비
		totalExpenseInt := 0
		for n, pur := range project.SMMonthlyPurchaseCost[date] {
			// 업체명
			pos, err = excelize.CoordinatesToCellName(6, i+3+n) // ex) pos = "F3"
			if err != nil {
				return err
			}
			f.SetCellValue(sheet, pos, pur.CompanyName)
			// 내역
			pos, err = excelize.CoordinatesToCellName(7, i+3+n) // ex) pos = "G3"
			if err != nil {
				return err
			}
			f.SetCellValue(sheet, pos, pur.Detail)
			// 금액
			pos, err = excelize.CoordinatesToCellName(8, i+3+n) // ex) pos = "H3"
			if err != nil {
				return err
			}
			expense, err := decryptAES256Func(pur.Expenses)
			if err != nil {
				return err
			}
			expenseInt, err := strconv.Atoi(expense)
			if err != nil {
				return err
			}
			f.SetCellValue(sheet, pos, expenseInt)
			totalExpenseInt += expenseInt
		}

		// 구매 내역이 없으면 합계가 안보이도록 한다.
		if len(project.SMMonthlyPurchaseCost[date]) != 0 {
			// 구매비 합계
			purSumPos, err := excelize.CoordinatesToCellName(6, i+3+purNum) // ex) pos = "F6"
			if err != nil {
				return err
			}
			mpos, err = excelize.CoordinatesToCellName(7, i+3+purNum) // ex) mpos = "G6"
			if err != nil {
				return err
			}
			f.SetCellValue(sheet, purSumPos, "합계")
			f.MergeCell(sheet, purSumPos, mpos)

			purPos, err := excelize.CoordinatesToCellName(8, i+3+purNum) // ex) pos = "H6"
			if err != nil {
				return err
			}
			f.SetCellValue(sheet, purPos, totalExpenseInt)
			purPosMap[purSumPos] = purPos
		}

		// 외주비
		pos, err = excelize.CoordinatesToCellName(9, i+3) // ex) pos = "I3"
		if err != nil {
			return err
		}
		mpos, err = excelize.CoordinatesToCellName(9, i+3+purNum) // ex) mpos = "I6"
		if err != nil {
			return err
		}
		expenses, err := decryptAES256Func(info.Vendor)
		if err != nil {
			return err
		}
		expensesInt := 0
		if expenses != "" {
			expensesInt, err = strconv.Atoi(expenses)
			if err != nil {
				return err
			}
		}
		f.SetCellValue(sheet, pos, expensesInt)
		f.MergeCell(sheet, pos, mpos)

		// 수익
		pos, err = excelize.CoordinatesToCellName(10, i+3) // ex) pos = "J3"
		if err != nil {
			return err
		}
		mpos, err = excelize.CoordinatesToCellName(10, i+3+purNum) // ex) mpos  = "J6"
		if err != nil {
			return err
		}
		rev, err := decryptAES256Func(info.Revenue)
		if err != nil {
			return err
		}
		revInt := 0
		if rev != "" {
			revInt, err = strconv.Atoi(rev)
			if err != nil {
				return err
			}
		}
		totalRevenue += revInt
		f.SetCellValue(sheet, pos, revInt)
		f.MergeCell(sheet, pos, mpos)

		// 셀 높이 설정
		for pn := 0; pn <= purNum; pn++ {
			f.SetRowHeight(sheet, i+3+pn, 20)
		}

		// i 기준 변경
		i = i + 1 + purNum
	}

	// 월별 합계 입력
	tpos, err := excelize.CoordinatesToCellName(1, i+3)
	if err != nil {
		return err
	}
	f.SetCellValue(sheet, tpos, "합계")
	tmpos, err := excelize.CoordinatesToCellName(2, i+3)
	if err != nil {
		return err
	}
	f.MergeCell(sheet, tpos, tmpos)

	tnpos, err := excelize.CoordinatesToCellName(3, i+3)
	if err != nil {
		return err
	}
	payment, err := decryptAES256Func(costSum["Payment"])
	if err != nil {
		return err
	}
	paymentInt := 0
	if payment != "" {
		paymentInt, err = strconv.Atoi(payment)
		if err != nil {
			return err
		}
	}
	f.SetCellValue(sheet, tnpos, paymentInt)

	pos, err = excelize.CoordinatesToCellName(4, i+3)
	if err != nil {
		return err
	}
	vfxLaborCost, err := decryptAES256Func(costSum["VFX"])
	if err != nil {
		return err
	}
	vfxLaborCostInt := 0
	if vfxLaborCost != "" {
		vfxLaborCostInt, err = strconv.Atoi(vfxLaborCost)
		if err != nil {
			return err
		}
	}
	cmLaborCost, err := decryptAES256Func(costSum["CM"])
	if err != nil {
		return err
	}
	cmLaborCostInt := 0
	if cmLaborCost != "" {
		cmLaborCostInt, err = strconv.Atoi(cmLaborCost)
		if err != nil {
			return err
		}
	}
	f.SetCellValue(sheet, pos, vfxLaborCostInt+cmLaborCostInt)

	pos, err = excelize.CoordinatesToCellName(5, i+3)
	if err != nil {
		return err
	}
	progressCost, err := decryptAES256Func(costSum["Progress"])
	if err != nil {
		return err
	}
	progressCostInt := 0
	if progressCost != "" {
		progressCostInt, err = strconv.Atoi(progressCost)
		if err != nil {
			return err
		}
	}
	f.SetCellValue(sheet, pos, progressCostInt)

	pos, err = excelize.CoordinatesToCellName(8, i+3)
	if err != nil {
		return err
	}
	purchaseCost, err := decryptAES256Func(costSum["Purchase"])
	if err != nil {
		return err
	}
	purchaseCostInt := 0
	if purchaseCost != "" {
		purchaseCostInt, err = strconv.Atoi(purchaseCost)
		if err != nil {
			return err
		}
	}
	f.SetCellValue(sheet, pos, purchaseCostInt)

	pos, err = excelize.CoordinatesToCellName(9, i+3)
	if err != nil {
		return err
	}
	expenses, err := decryptAES256Func(costSum["Vendor"])
	if err != nil {
		return err
	}
	expensesInt := 0
	if expenses != "" {
		expensesInt, err = strconv.Atoi(expenses)
		if err != nil {
			return err
		}
	}
	f.SetCellValue(sheet, pos, expensesInt)

	pos, err = excelize.CoordinatesToCellName(10, i+3)
	if err != nil {
		return err
	}
	f.SetCellValue(sheet, pos, totalRevenue)
	f.SetRowHeight(sheet, i+3, 20)

	f.SetCellStyle(sheet, "A1", pos, style)
	f.SetCellStyle(sheet, "C3", strings.ReplaceAll(pos, "J", "E"), numberStyle)
	f.SetCellStyle(sheet, "H3", pos, numberStyle)
	for key, value := range purPosMap {
		f.SetCellStyle(sheet, key, key, purStyle)
		f.SetCellStyle(sheet, value, value, purNumStyle)
	}
	f.SetCellStyle(sheet, tpos, tmpos, totalStyle)
	f.SetCellStyle(sheet, tnpos, pos, totalNumStyle)

	// 엑셀 파일 저장
	err = f.SaveAs(path + "/" + excelFileName)
	if err != nil {
		return err
	}

	return nil
}

// handleExportDetailSMFunc 함수는 임시 폴더에 저장된 엑셀 파일을 다운로드하는 함수이다.
func handleExportDetailSMFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// admin 레벨 미만이면 invalideaccess 페이지로 리다이렉트
	if token.AccessLevel < AdminLevel {
		http.Redirect(w, r, "/invalidaccess", http.StatusSeeOther)
		return
	}

	// Post 메소드가 아니면 에러
	if r.Method != http.MethodPost {
		http.Error(w, "Post Method Only", http.StatusMethodNotAllowed)
		return
	}

	path := os.TempDir() + "/budget/" + token.ID + "/detailSM"

	// path에 있는 파일들을 가져온다.
	fileInfo, err := ioutil.ReadDir(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 파일의 확장자가 xlsx가 아니면 엑셀 파일이 다시 생성되도록 리다이렉트
	ext := filepath.Ext(fileInfo[0].Name())
	if ext != ".xlsx" {
		http.Redirect(w, r, "/detail-sm", http.StatusSeeOther)
		return
	}

	// mongoDB client 연결
	credential := options.Credential{
		Username: *flagDBID,
		Password: *flagDBPW,
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(*flagMongoDBURI).SetAuth(credential))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	project := strings.Split(strings.Split(fileInfo[0].Name(), "_")[1], ".")[0]
	log := Log{
		UserID:    token.ID,
		CreatedAt: time.Now(),
		Content:   fmt.Sprintf("%s 디테일 페이지에서 프로젝트 데이터를 다운로드하였습니다.", project),
	}

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Disposition", fmt.Sprintf("Attachment; filename=%s", fileInfo[0].Name()))
	http.ServeFile(w, r, path+"/"+fileInfo[0].Name())
}
