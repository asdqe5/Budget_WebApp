// 프로젝트 결산 프로그램
//
// Description : http 결산 현황 관련 스크립트

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize/v2"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// handleSMPaymentStatusFunc 함수는 결산 월별 매출 현황 페이지를 보여주는 함수이다.
func handleSMPaymentStatusFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// member 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < MemberLevel {
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

	type Recipe struct {
		Token                  Token
		Year                   string
		Projects               []Project
		Dates                  []string
		Status                 []Status
		TotalMonthlyPaymentMap map[string]string
		TotalProjectPaymentMap map[string]string
		TotalPayment           string
		SumPayment             string
	}

	rcp := Recipe{}
	rcp.Token = token
	year := r.FormValue("year")
	if year == "" { // year 값이 없으면 올해로 검색
		y, _, _ := time.Now().Date()
		year = strconv.Itoa(y)
	}
	rcp.Year = year
	projects, err := getProjectsByYearFunc(client, rcp.Year)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sort.Slice(projects, func(i, j int) bool { // 이름으로 오름차순 정렬
		return projects[i].Name < projects[j].Name
	})
	// RND, ETC 프로젝트는 제외한다.
	for _, p := range projects {
		if !strings.Contains(p.ID, "ETC") && !strings.Contains(p.ID, "RND") {
			rcp.Projects = append(rcp.Projects, p)
		}
	}
	dateList := []string{}
	for i := 1; i <= 12; i++ {
		dateList = append(dateList, fmt.Sprintf("%s-%02d", rcp.Year, i))
	}
	rcp.Dates = dateList
	intTotalMonthlyPaymentMap := make(map[string]int)
	intTotalProjectPaymentMap := make(map[string]int)
	intTotalPayment := 0
	intSumPayment := 0
	for _, p := range rcp.Projects {
		// 총 매출 합산 계산
		for _, pay := range p.Payment {
			decrypted, err := decryptAES256Func(pay.Expenses)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			decryptedInt := 0
			if decrypted != "" {
				decryptedInt, err = strconv.Atoi(decrypted)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				intTotalPayment += decryptedInt
			}
		}

		// 월별 매출 합산 계산
		for _, d := range rcp.Dates {
			for _, monthlyPayment := range p.SMMonthlyPayment[d] {
				decrypted, err := decryptAES256Func(monthlyPayment.Expenses)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				decryptedInt := 0
				if decrypted != "" {
					decryptedInt, err = strconv.Atoi(decrypted)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					intTotalMonthlyPaymentMap[d] += decryptedInt
					intTotalProjectPaymentMap[p.ID] += decryptedInt
					intSumPayment += decryptedInt
				}
			}
		}
	}
	rcp.Status = adminSetting.ProjectStatus

	// 월별 매출 합산 값과 총 매출 합산의 암호화
	totalMonthlyPaymentMap := make(map[string]string)
	for _, d := range rcp.Dates {
		if intTotalMonthlyPaymentMap[d] != 0 {
			totalMonthlyPaymentMap[d], err = encryptAES256Func(strconv.Itoa(intTotalMonthlyPaymentMap[d]))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
	rcp.TotalMonthlyPaymentMap = totalMonthlyPaymentMap
	rcp.TotalPayment, err = encryptAES256Func(strconv.Itoa(intTotalPayment))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 프로젝트별 월별 매출 합산 값과 그 합을 암호화
	rcp.TotalProjectPaymentMap = make(map[string]string)
	for key, value := range intTotalProjectPaymentMap {
		rcp.TotalProjectPaymentMap[key], err = encryptAES256Func(strconv.Itoa(value))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	rcp.SumPayment, err = encryptAES256Func(strconv.Itoa(intSumPayment))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 월별 매출 현황의 엑셀 파일을 만든다.
	err = genSMPaymentStatusExcelFunc(rcp.Projects, rcp.Dates, rcp.TotalMonthlyPaymentMap, rcp.TotalPayment, token.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "smpaymentstatus", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// genSMPaymentStatusExcelFunc 함수는 월별 매출 현황표의 엑셀파일을 만드는 함수이다.
func genSMPaymentStatusExcelFunc(projects []Project, dates []string, totalMonthlyPaymentMap map[string]string, totalPayment string, userID string) error {
	path := os.TempDir() + "/budget/" + userID + "/smpaymentstatus/"
	excelFileName := fmt.Sprintf("smpaymentstatus_%s.xlsx", strings.Split(dates[0], "-")[0])

	err := createFolderFunc(path)
	if err != nil {
		return err
	}
	err = delAllFilesFunc(path) // 경로에 있는 모든 파일 삭제
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
	f.SetCellValue(sheet, "B1", "프로젝트")
	f.MergeCell(sheet, "B1", "B2")
	f.SetCellValue(sheet, "C1", "제작사")
	f.MergeCell(sheet, "C1", "C2")
	f.SetCellValue(sheet, "D1", "감독")
	f.MergeCell(sheet, "D1", "D2")
	f.SetCellValue(sheet, "E1", "계약일")
	f.MergeCell(sheet, "E1", "E2")
	f.SetCellValue(sheet, "F1", "계약 금액")
	f.MergeCell(sheet, "F1", "F2")
	f.SetCellValue(sheet, "G1", strings.Split(dates[0], "-")[0]+"년")
	f.MergeCell(sheet, "G1", "R1")
	for i := 1; i <= 12; i++ {
		pos, err := excelize.CoordinatesToCellName(i+6, 2)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, strconv.Itoa(i)+"월")
	}
	f.SetCellValue(sheet, "S1", "Total")
	f.MergeCell(sheet, "S1", "S2")
	f.SetColWidth(sheet, "A", "S", 18)
	f.SetColWidth(sheet, "A", "A", 10)
	f.SetColWidth(sheet, "B", "B", 25)
	f.SetRowHeight(sheet, 1, 25)
	f.SetRowHeight(sheet, 2, 25)

	// 데이터 입력
	totalSum := 0
	rowLen := 0
	i := 0
	for _, project := range projects {
		// 프로젝트 Status
		pos, err := excelize.CoordinatesToCellName(1, i+3)
		if err != nil {
			return err
		}
		mpos, err := excelize.CoordinatesToCellName(1, i+3+len(project.Payment)-1)
		if err != nil {
			return err
		}
		status := ""
		if project.IsFinished == true {
			status, err = getLastStatusOfProjectFunc(project)
			if err != nil {
				return err
			}
		} else {
			status = getThisMonthStatusOfProjectFunc(project)
		}
		f.SetCellValue(sheet, pos, status)
		f.MergeCell(sheet, pos, mpos)

		// 프로젝트
		pos, err = excelize.CoordinatesToCellName(2, i+3)
		if err != nil {
			return err
		}
		mpos, err = excelize.CoordinatesToCellName(2, i+3+len(project.Payment)-1)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, project.Name)
		f.MergeCell(sheet, pos, mpos)

		// 제작사
		pos, err = excelize.CoordinatesToCellName(3, i+3)
		if err != nil {
			return err
		}
		mpos, err = excelize.CoordinatesToCellName(3, i+3+len(project.Payment)-1)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, project.ProducerName)
		f.MergeCell(sheet, pos, mpos)

		// 감독
		pos, err = excelize.CoordinatesToCellName(4, i+3)
		if err != nil {
			return err
		}
		mpos, err = excelize.CoordinatesToCellName(4, i+3+len(project.Payment)-1)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, project.DirectorName)
		f.MergeCell(sheet, pos, mpos)

		// 계약일 및 계약 금액
		for n, payment := range project.Payment {
			pos, err = excelize.CoordinatesToCellName(5, i+3+n) // 계약일
			if err != nil {
				return err
			}
			f.SetCellValue(sheet, pos, payment.Date)

			pos, err = excelize.CoordinatesToCellName(6, i+3+n) // 계약 금액
			if err != nil {
				return err
			}
			paymentInt := 0
			decrypted, err := decryptAES256Func(payment.Expenses)
			if err != nil {
				return err
			}
			if decrypted != "" {
				paymentInt, err = strconv.Atoi(decrypted)
				if err != nil {
					return err
				}
			}
			f.SetCellValue(sheet, pos, paymentInt)
			f.SetRowHeight(sheet, i+3+n, 20)
			rowLen++
		}

		// 프로젝트 월별 매출
		monthlyPaymentSum := 0
		for n, d := range dates {
			pos, err = excelize.CoordinatesToCellName(n+7, i+3)
			if err != nil {
				return err
			}
			mpos, err = excelize.CoordinatesToCellName(n+7, i+3+len(project.Payment)-1)
			if err != nil {
				return err
			}
			monthlyPaymentInt := 0
			for _, monthlyPayment := range project.SMMonthlyPayment[d] {
				decrypted, err := decryptAES256Func(monthlyPayment.Expenses)
				if err != nil {
					return err
				}
				if decrypted != "" {
					decryptedInt, err := strconv.Atoi(decrypted)
					if err != nil {
						return err
					}
					monthlyPaymentInt += decryptedInt
				}
			}
			if monthlyPaymentInt == 0 {
				f.SetCellValue(sheet, pos, "")
			} else {
				f.SetCellValue(sheet, pos, monthlyPaymentInt)
			}
			f.MergeCell(sheet, pos, mpos)

			monthlyPaymentSum += monthlyPaymentInt
		}

		// 프로젝트 월별 매출 합계
		pos, err = excelize.CoordinatesToCellName(len(dates)+7, i+3)
		if err != nil {
			return err
		}
		mpos, err = excelize.CoordinatesToCellName(len(dates)+7, i+3+len(project.Payment)-1)
		if err != nil {
			return err
		}
		if monthlyPaymentSum == 0 {
			f.SetCellValue(sheet, pos, "")
		} else {
			f.SetCellValue(sheet, pos, monthlyPaymentSum)
		}
		f.MergeCell(sheet, pos, mpos)

		totalSum += monthlyPaymentSum
		i += len(project.Payment)
	}

	tpos, err := excelize.CoordinatesToCellName(1, rowLen+3)
	if err != nil {
		return err
	}
	mpos, err := excelize.CoordinatesToCellName(5, rowLen+3)
	if err != nil {
		return err
	}
	f.SetCellValue(sheet, tpos, "Total")
	f.MergeCell(sheet, tpos, mpos)

	tnpos, err := excelize.CoordinatesToCellName(6, rowLen+3)
	if err != nil {
		return err
	}
	payment, err := decryptAES256Func(totalPayment)
	if err != nil {
		return err
	}
	paymentInt, err := strconv.Atoi(payment)
	if err != nil {
		return err
	}
	f.SetCellValue(sheet, tnpos, paymentInt)

	pos := ""
	for n, d := range dates {
		pos, err = excelize.CoordinatesToCellName(n+7, rowLen+3)
		if err != nil {
			return err
		}
		monthlyPayment, err := decryptAES256Func(totalMonthlyPaymentMap[d])
		if err != nil {
			return err
		}
		monthlyPaymentInt := 0
		if monthlyPayment != "" {
			monthlyPaymentInt, err = strconv.Atoi(monthlyPayment)
			if err != nil {
				return err
			}
		}
		if monthlyPaymentInt == 0 {
			f.SetCellValue(sheet, pos, "")
		} else {
			f.SetCellValue(sheet, pos, monthlyPaymentInt)
		}
	}

	// 월별 Total의 합계 입력
	pos, err = excelize.CoordinatesToCellName(len(dates)+7, rowLen+3)
	if err != nil {
		return err
	}
	if totalSum == 0 {
		f.SetCellValue(sheet, pos, "")
	} else {
		f.SetCellValue(sheet, pos, totalSum)
	}

	f.SetRowHeight(sheet, rowLen+3, 20)

	f.SetCellStyle(sheet, "A1", pos, style)
	f.SetCellStyle(sheet, "F3", pos, numberStyle)
	f.SetCellStyle(sheet, "S1", pos, totalStyle)
	f.SetCellStyle(sheet, tpos, pos, totalStyle)
	f.SetCellStyle(sheet, "S3", pos, totalNumStyle)
	f.SetCellStyle(sheet, tnpos, pos, totalNumStyle)

	// 엑셀 파일 저장
	err = f.SaveAs(path + "/" + excelFileName)
	if err != nil {
		return err
	}

	return nil
}

// handleExportSMPaymentStatusFunc 함수는 임시 폴더에 저장된 엑셀 파일을 다운로드하는 함수이다.
func handleExportSMPaymentStatusFunc(w http.ResponseWriter, r *http.Request) {
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

	path := os.TempDir() + "/budget/" + token.ID + "/smpaymentstatus"

	// path에 있는 파일들을 가져온다.
	fileInfo, err := ioutil.ReadDir(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 파일의 확장자가 xlsx가 아니면 엑셀 파일이 다시 생성되도록 리다이렉트
	ext := filepath.Ext(fileInfo[0].Name())
	if ext != ".xlsx" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
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

	filename := strings.Split(strings.Split(fileInfo[0].Name(), ".")[0], "_")[1]

	log := Log{
		UserID:    token.ID,
		CreatedAt: time.Now(),
		Content:   fmt.Sprintf("매출 현황 페이지에서 %s년의 데이터를 다운로드하였습니다.", filename),
	}

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Disposition", fmt.Sprintf("Attachment; filename=%s", fileInfo[0].Name()))
	http.ServeFile(w, r, path+"/"+fileInfo[0].Name())
}

// handleSMVendorStatusFunc 함수는 결산 월별 외주 현황 페이지를 보여주는 함수이다.
func handleSMVendorStatusFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// member 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < MemberLevel {
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

	type Recipe struct {
		Token                   Token
		Year                    string
		Dates                   []string
		Vendors                 map[string]map[string][]Vendor
		TotalMonthlyExpensesMap map[string]string // 월별 벤더 비용 합계
		TotalDetailExpensesMap  map[string]string // 해당 연도 벤더 계약별 합계 비용
		TotalExpenses           string            // 벤더 총 계약 금액 합계
		SumTotalExpenses        string            // 해당 연도 안에 지급된 비용의 합계
	}

	rcp := Recipe{}
	rcp.Token = token
	year := r.FormValue("year")
	if year == "" { // year 값이 없으면 올해로 검색
		y, _, _ := time.Now().Date()
		year = strconv.Itoa(y)
	}
	rcp.Year = year
	dateList := []string{}
	for i := 1; i <= 12; i++ {
		dateList = append(dateList, fmt.Sprintf("%s-%02d", rcp.Year, i))
	}
	rcp.Dates = dateList

	// Vendor 정보 가져오기
	vendors, err := getVendorsByYearFunc(client, rcp.Year)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 프로젝트별로 정리하기
	vendorsMap := make(map[string]map[string][]Vendor)
	var projectIDList []string
	intTotalMonthlyExpensesMap := make(map[string]int) // 월별 벤더 비용 합계
	intTotalExpenses := 0                              // 벤더 계약 금액

	intTotalDetailExpensesMap := make(map[string]int) // 해당 연도 벤더 계약별 합계 비용
	intSumTotalExpenses := 0                          // 해당 연도 벤더 비용 합계
	for _, v := range vendors {
		pid := fmt.Sprintf("%s-%s", v.ProjectName, v.Project) // 프로젝트 이름 순으로 정렬을 하려는데 같은 이름의 프로젝트도 있을 수가 있다.
		if !checkStringInListFunc(pid, projectIDList) {
			projectIDList = append(projectIDList, pid)
		}

		// 외주비 합계 계산
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
			intTotalMonthlyExpensesMap[downpaymentMonth] += downpaymentInt
			intTotalExpenses += downpaymentInt

			// 연도 합계 계산
			if checkStringInListFunc(downpaymentMonth, rcp.Dates) { // 해당 월이 입력한 연도이면 더해준다.
				intTotalDetailExpensesMap[v.ID.Hex()] += downpaymentInt
				intSumTotalExpenses += downpaymentInt
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
			intTotalMonthlyExpensesMap[mediumplatingMonth] += mediumplatingInt
			intTotalExpenses += mediumplatingInt

			// 연도 합계 계산
			if checkStringInListFunc(mediumplatingMonth, rcp.Dates) { // 해당 월이 입력한 연도이면 더해준다.
				intTotalDetailExpensesMap[v.ID.Hex()] += mediumplatingInt
				intSumTotalExpenses += mediumplatingInt
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
			intTotalMonthlyExpensesMap[balanceMonth] += balanceInt
			intTotalExpenses += balanceInt

			// 연도 합계 계산
			if checkStringInListFunc(balanceMonth, rcp.Dates) { // 해당 월이 입력한 연도이면 더해준다.
				intTotalDetailExpensesMap[v.ID.Hex()] += balanceInt
				intSumTotalExpenses += balanceInt
			}
		}
	}

	// 프로젝트 명으로 정렬되도록 정리
	for _, pid := range projectIDList {
		vendorsMap[pid] = make(map[string][]Vendor)
	}
	for _, v := range vendors {
		pid := fmt.Sprintf("%s-%s", v.ProjectName, v.Project)
		vendorsMap[pid][v.Name] = append(vendorsMap[pid][v.Name], v)
	}
	rcp.Vendors = vendorsMap

	// 벤더 계약금액 합계 암호화
	rcp.TotalExpenses, err = encryptAES256Func(strconv.Itoa(intTotalExpenses))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 월별 벤더 비용 합계 암호화
	totalMonthlyExpensesMap := make(map[string]string)
	for _, d := range rcp.Dates {
		if intTotalMonthlyExpensesMap[d] != 0 {
			totalMonthlyExpensesMap[d], err = encryptAES256Func(strconv.Itoa(intTotalMonthlyExpensesMap[d]))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
	rcp.TotalMonthlyExpensesMap = totalMonthlyExpensesMap

	// 해당 연도 벤더 계약별 비용 합계 암호화
	rcp.TotalDetailExpensesMap = make(map[string]string)
	for id, expenses := range intTotalDetailExpensesMap {
		rcp.TotalDetailExpensesMap[id], err = encryptAES256Func(strconv.Itoa(expenses))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// 해당 연도 벤더 비용 합계 암호화
	rcp.SumTotalExpenses, err = encryptAES256Func(strconv.Itoa(intSumTotalExpenses))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 월별 외주 현황의 엑셀 파일을 만든다
	err = genSMVendorStatusExcelFunc(rcp.Vendors, rcp.Dates, rcp.TotalMonthlyExpensesMap, rcp.TotalExpenses, token.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "smvendorstatus", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// genSMVendorStatusExcelFunc 함수는 외주 현황 엑셀 파일을 만드는 함수이다.
func genSMVendorStatusExcelFunc(vendors map[string]map[string][]Vendor, dates []string, totalMonthlyExpensesMap map[string]string, totalExpenses string, userID string) error {
	path := os.TempDir() + "/budget/" + userID + "/smvendorstatus/"
	excelFileName := fmt.Sprintf("smvendorstatus_%s.xlsx", strings.Split(dates[0], "-")[0])

	err := createFolderFunc(path)
	if err != nil {
		return err
	}
	err = delAllFilesFunc(path) // 경로에 있는 모든 파일 삭제
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
	f.SetCellValue(sheet, "A1", "프로젝트")
	f.MergeCell(sheet, "A1", "A2")
	f.SetCellValue(sheet, "B1", "벤더명")
	f.MergeCell(sheet, "B1", "B2")
	f.SetCellValue(sheet, "C1", "계약일")
	f.MergeCell(sheet, "C1", "C2")
	f.SetCellValue(sheet, "D1", "계약 금액")
	f.MergeCell(sheet, "D1", "D2")
	f.SetCellValue(sheet, "E1", strings.Split(dates[0], "-")[0]+"년")
	f.MergeCell(sheet, "E1", "P1")
	for i := 1; i <= 12; i++ {
		pos, err := excelize.CoordinatesToCellName(i+4, 2)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, strconv.Itoa(i)+"월")
	}
	f.SetCellValue(sheet, "Q1", "Total")
	f.MergeCell(sheet, "Q1", "Q2")
	f.SetColWidth(sheet, "A", "Q", 15)
	f.SetColWidth(sheet, "A", "A", 20)
	f.SetColWidth(sheet, "D", "D", 20)
	f.SetRowHeight(sheet, 1, 25)
	f.SetRowHeight(sheet, 2, 25)

	// 데이터 입력
	pos := ""
	i := 0
	for _, v := range vendors {
		plen := lenOfVendorsMapFunc(v, false)
		pnum := 0
		for _, vendor := range v {
			vlen := len(vendor)
			vnum := 0
			for _, data := range vendor {
				if pnum == 0 {
					// 프로젝트
					pos, err = excelize.CoordinatesToCellName(1, i+3)
					if err != nil {
						return err
					}
					mpos, err := excelize.CoordinatesToCellName(1, i+3+plen-1)
					if err != nil {
						return err
					}
					f.SetCellValue(sheet, pos, data.ProjectName)
					f.MergeCell(sheet, pos, mpos)
				}
				if vnum == 0 {
					// 벤더명
					pos, err = excelize.CoordinatesToCellName(2, i+3)
					if err != nil {
						return err
					}
					mpos, err := excelize.CoordinatesToCellName(2, i+3+vlen-1)
					if err != nil {
						return err
					}
					f.SetCellValue(sheet, pos, data.Name)
					f.MergeCell(sheet, pos, mpos)
				}

				// 계약일
				pos, err = excelize.CoordinatesToCellName(3, i+3)
				if err != nil {
					return err
				}
				f.SetCellValue(sheet, pos, data.Date)

				// 계약 금액
				expenses, err := decryptAES256Func(data.Expenses)
				if err != nil {
					return err
				}
				expensesInt, err := strconv.Atoi(expenses)
				if err != nil {
					return err
				}
				pos, err = excelize.CoordinatesToCellName(4, i+3)
				if err != nil {
					return err
				}
				f.SetCellValue(sheet, pos, expensesInt)

				// 월별 지출액
				monthlyVEMap := make(map[string]int)
				expensesSum := 0
				for _, d := range dates {
					expenses := 0
					if d == dateToMonthFunc(data.Downpayment.Date) { // 계약금 확인
						downpayment, err := decryptAES256Func(data.Downpayment.Expenses)
						if err != nil {
							return err
						}
						downpaymentInt, err := strconv.Atoi(downpayment)
						if err != nil {
							return err
						}
						expenses += downpaymentInt
						expensesSum += downpaymentInt
					}
					for _, mp := range data.MediumPlating { // 중도금 확인
						if d == dateToMonthFunc(mp.Date) {
							mediumplating, err := decryptAES256Func(mp.Expenses)
							if err != nil {
								return err
							}
							mediumplatingInt, err := strconv.Atoi(mediumplating)
							if err != nil {
								return err
							}
							expenses += mediumplatingInt
							expensesSum += mediumplatingInt
						}
					}
					if d == dateToMonthFunc(data.Balance.Date) { // 잔금 확인
						balance, err := decryptAES256Func(data.Balance.Expenses)
						if err != nil {
							return err
						}
						balanceInt, err := strconv.Atoi(balance)
						if err != nil {
							return err
						}
						expenses += balanceInt
						expensesSum += balanceInt
					}
					monthlyVEMap[d] = expenses
				}
				for n, d := range dates {
					pos, err = excelize.CoordinatesToCellName(5+n, i+3)
					if err != nil {
						return err
					}
					if monthlyVEMap[d] != 0 {
						f.SetCellValue(sheet, pos, monthlyVEMap[d])
					}
				}

				// 해당 연도 벤더 계약별 합계
				pos, err = excelize.CoordinatesToCellName(6+len(dates)-1, i+3)
				if err != nil {
					return err
				}
				if expensesSum == 0 {
					f.SetCellValue(sheet, pos, "")
				} else {
					f.SetCellValue(sheet, pos, expensesSum)
				}

				// 셀 높이 설정
				f.SetRowHeight(sheet, i+3, 20)

				pnum++
				vnum++
				i++
			}
		}
	}

	tpos, err := excelize.CoordinatesToCellName(1, i+3)
	if err != nil {
		return err
	}
	f.SetCellValue(sheet, tpos, "Total")
	mpos, err := excelize.CoordinatesToCellName(3, i+3)
	if err != nil {
		return err
	}
	f.MergeCell(sheet, tpos, mpos)

	tnpos, err := excelize.CoordinatesToCellName(4, i+3)
	if err != nil {
		return err
	}
	expenses, err := decryptAES256Func(totalExpenses)
	if err != nil {
		return err
	}
	expensesInt, err := strconv.Atoi(expenses)
	if err != nil {
		return err
	}
	f.SetCellValue(sheet, tnpos, expensesInt)

	totalSum := 0
	for n, d := range dates {
		pos, err = excelize.CoordinatesToCellName(5+n, i+3)
		if err != nil {
			return err
		}
		monthlyExpenses, err := decryptAES256Func(totalMonthlyExpensesMap[d])
		if err != nil {
			return err
		}
		if monthlyExpenses != "" {
			monthlyExpensesInt, err := strconv.Atoi(monthlyExpenses)
			if err != nil {
				return err
			}
			f.SetCellValue(sheet, pos, monthlyExpensesInt)
			totalSum += monthlyExpensesInt
		}
	}

	// 해당 연도 벤더 비용 합계 입력
	pos, err = excelize.CoordinatesToCellName(6+len(dates)-1, i+3)
	if err != nil {
		return err
	}
	if totalSum == 0 {
		f.SetCellValue(sheet, pos, "")
	} else {
		f.SetCellValue(sheet, pos, totalSum)
	}

	f.SetRowHeight(sheet, i+3, 20)
	f.SetCellStyle(sheet, "A1", pos, style)
	f.SetCellStyle(sheet, "D3", pos, numberStyle)
	f.SetCellStyle(sheet, tpos, pos, totalStyle)
	f.SetCellStyle(sheet, "Q1", pos, totalStyle)
	f.SetCellStyle(sheet, tnpos, pos, totalNumStyle)
	f.SetCellStyle(sheet, "Q3", pos, totalNumStyle)

	// 엑셀 파일 저장
	err = f.SaveAs(path + "/" + excelFileName)
	if err != nil {
		return err
	}

	return nil
}

// handleExportSMVendorStatusFunc 함수는 임시 폴더에 저장된 엑셀 파일을 다운로드하는 함수이다.
func handleExportSMVendorStatusFunc(w http.ResponseWriter, r *http.Request) {
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

	path := os.TempDir() + "/budget/" + token.ID + "/smvendorstatus"

	// path에 있는 파일들을 가져온다.
	fileInfo, err := ioutil.ReadDir(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 파일의 확장자가 xlsx가 아니면 엑셀 파일이 다시 생성되도록 리다이렉트
	ext := filepath.Ext(fileInfo[0].Name())
	if ext != ".xlsx" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
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

	filename := strings.Split(strings.Split(fileInfo[0].Name(), ".")[0], "_")[1]

	log := Log{
		UserID:    token.ID,
		CreatedAt: time.Now(),
		Content:   fmt.Sprintf("외주 현황 페이지에서 %s년의 데이터를 다운로드하였습니다.", filename),
	}

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Disposition", fmt.Sprintf("Attachment; filename=%s", fileInfo[0].Name()))
	http.ServeFile(w, r, path+"/"+fileInfo[0].Name())
}

// handleSMTotalStatusFunc 함수는 매출 및 외주비의 전체 현황을 확인할 수 있는 페이지를 보여준다.
func handleSMTotalStatusFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// member 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < MemberLevel {
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

	type Recipe struct {
		Token    Token
		Year     string
		Dates    []string
		Payment  map[string]string // 월별 매출 합계
		Expenses map[string]string // 월별 외주 합계
		Total    map[string]string // 매출과 외주비가 월별로 계산된 값
		CostSum  map[string]string // 매출, 외주비, Total의 합계
	}
	rcp := Recipe{}
	rcp.Token = token
	year := r.FormValue("year")
	if year == "" { // year 값이 없으면 올해로 검색
		y, _, _ := time.Now().Date()
		year = strconv.Itoa(y)
	}
	rcp.Year = year
	dateList := []string{}
	for i := 1; i <= 12; i++ {
		dateList = append(dateList, fmt.Sprintf("%s-%02d", rcp.Year, i))
	}
	rcp.Dates = dateList

	projects, err := getProjectsByYearFunc(client, rcp.Year)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	vendors, err := getVendorsByYearFunc(client, rcp.Year)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 월별 매출 계산
	totalMap := make(map[string]int)
	paymentMap := make(map[string]int)
	for _, p := range projects {
		for key, value := range p.SMMonthlyPayment {
			for _, monthlyPayment := range value {
				decrypted, err := decryptAES256Func(monthlyPayment.Expenses)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if decrypted != "" {
					decryptedInt, err := strconv.Atoi(decrypted)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					paymentMap[key] += decryptedInt
					totalMap[key] += decryptedInt
				}
			}
		}
	}

	// 월별 외주비 계산
	expensesMap := make(map[string]int)
	for _, v := range vendors {
		// 계약금
		dpMonth := dateToMonthFunc(v.Downpayment.Date)
		if dpMonth != "" {
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
			expensesMap[dpMonth] = expensesMap[dpMonth] + downpaymentInt
			totalMap[dpMonth] = totalMap[dpMonth] - downpaymentInt
		}
		// 중도금
		for _, mp := range v.MediumPlating {
			mpMonth := dateToMonthFunc(mp.Date)
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
			expensesMap[mpMonth] = expensesMap[mpMonth] + mediumplatingInt
			totalMap[mpMonth] = totalMap[mpMonth] - mediumplatingInt
		}
		// 잔금
		blMonth := dateToMonthFunc(v.Balance.Date)
		if blMonth != "" {
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
			expensesMap[blMonth] = expensesMap[blMonth] + balanceInt
			totalMap[blMonth] = totalMap[blMonth] - balanceInt
		}
	}

	// 월별 매출 합산 암호화
	rcp.Payment = make(map[string]string)
	for key, value := range paymentMap {
		rcp.Payment[key], err = encryptAES256Func(strconv.Itoa(value))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// 월별 외주비 합산 암호화
	rcp.Expenses = make(map[string]string)
	for key, value := range expensesMap {
		rcp.Expenses[key], err = encryptAES256Func(strconv.Itoa(value))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// 월별 Total 금액 암호화
	rcp.Total = make(map[string]string)
	for key, value := range totalMap {
		rcp.Total[key], err = encryptAES256Func(strconv.Itoa(value))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// 월별 매출, 외주비, Total 금액 합계
	costSum := make(map[string]int)
	for _, date := range rcp.Dates {
		costSum["Payment"] += paymentMap[date]
		costSum["Expenses"] += expensesMap[date]
		costSum["Total"] += totalMap[date]
	}

	// 월별 매출, 외주비, Total 금액 합계 암호화
	rcp.CostSum = make(map[string]string)
	for key, value := range costSum {
		rcp.CostSum[key], err = encryptAES256Func(strconv.Itoa(value))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Total 현황 엑셀 파일을 만든다
	err = genSMTotalStatusExcelFunc(rcp.Dates, rcp.Payment, rcp.Expenses, rcp.Total, token.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "smtotalstatus", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// genSMTotalStatusExcelFunc 함수는 전체 현황 엑셀 파일을 생성하는 함수이다.
func genSMTotalStatusExcelFunc(dates []string, payment map[string]string, expenses map[string]string, total map[string]string, userID string) error {
	path := os.TempDir() + "/budget/" + userID + "/smtotalstatus/"
	excelFileName := fmt.Sprintf("smtotalstatus_%s.xlsx", strings.Split(dates[0], "-")[0])

	err := createFolderFunc(path)
	if err != nil {
		return err
	}
	err = delAllFilesFunc(path) // 경로에 있는 모든 파일 삭제
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
	f.SetCellValue(sheet, "A1", "")
	f.MergeCell(sheet, "A1", "A2")
	f.SetCellValue(sheet, "B1", strings.Split(dates[0], "-")[0]+"년")
	f.MergeCell(sheet, "B1", "M1")
	for i := 1; i <= 12; i++ {
		pos, err := excelize.CoordinatesToCellName(i+1, 2)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, strconv.Itoa(i)+"월")
	}
	f.SetCellValue(sheet, "N1", "Total")
	f.MergeCell(sheet, "N1", "N2")
	f.SetCellValue(sheet, "A3", "매출")
	f.SetCellValue(sheet, "A4", "외주비")
	f.SetCellValue(sheet, "A5", "Total")
	f.SetColWidth(sheet, "A", "N", 18)
	f.SetRowHeight(sheet, 1, 25)
	f.SetRowHeight(sheet, 2, 25)

	// 데이터 입력
	pos := ""
	paymentSum := 0
	expensesSum := 0
	totalSum := 0
	for i, d := range dates {
		// 월별 매출
		p, err := decryptAES256Func(payment[d])
		if err != nil {
			return err
		}
		pInt := 0
		if p != "" {
			pInt, err = strconv.Atoi(p)
			if err != nil {
				return err
			}
		}
		pos, err = excelize.CoordinatesToCellName(i+2, 3)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, pInt)
		paymentSum += pInt

		// 월별 외주비
		v, err := decryptAES256Func(expenses[d])
		if err != nil {
			return err
		}
		vInt := 0
		if v != "" {
			vInt, err = strconv.Atoi(v)
			if err != nil {
				return err
			}
		}
		pos, err = excelize.CoordinatesToCellName(i+2, 4)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, vInt)
		expensesSum += vInt

		// Total
		t, err := decryptAES256Func(total[d])
		if err != nil {
			return err
		}
		tInt := 0
		if t != "" {
			tInt, err = strconv.Atoi(t)
			if err != nil {
				return err
			}
		}
		pos, err = excelize.CoordinatesToCellName(i+2, 5)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, tInt)
		totalSum += tInt

		// 셀 높이 설정
		f.SetRowHeight(sheet, i+3, 20)
	}

	// 매출, 외주비, Total 합계 입력
	pos, err = excelize.CoordinatesToCellName(len(dates)+2, 3)
	if err != nil {
		return err
	}
	f.SetCellValue(sheet, pos, paymentSum)
	pos, err = excelize.CoordinatesToCellName(len(dates)+2, 4)
	if err != nil {
		return err
	}
	f.SetCellValue(sheet, pos, expensesSum)
	pos, err = excelize.CoordinatesToCellName(len(dates)+2, 5)
	if err != nil {
		return err
	}
	f.SetCellValue(sheet, pos, totalSum)

	// 셀스타일 설정
	f.SetCellStyle(sheet, "A1", pos, style)
	f.SetCellStyle(sheet, "B3", pos, numberStyle)
	f.SetCellStyle(sheet, "A5", "A5", totalStyle)
	f.SetCellStyle(sheet, "N1", "N1", totalStyle)
	f.SetCellStyle(sheet, "B5", pos, totalNumStyle)
	f.SetCellStyle(sheet, "N3", pos, totalNumStyle)

	// 엑셀 파일 저장
	err = f.SaveAs(path + "/" + excelFileName)
	if err != nil {
		return err
	}

	return nil
}

// handleExportSMTotalStatusFunc 함수는 임시 폴더에 저장된 엑셀 파일을 다운로드하는 함수이다.
func handleExportSMTotalStatusFunc(w http.ResponseWriter, r *http.Request) {
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

	path := os.TempDir() + "/budget/" + token.ID + "/smtotalstatus"

	// path에 있는 파일들을 가져온다.
	fileInfo, err := ioutil.ReadDir(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 파일의 확장자가 xlsx가 아니면 엑셀 파일이 다시 생성되도록 리다이렉트
	ext := filepath.Ext(fileInfo[0].Name())
	if ext != ".xlsx" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
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

	filename := strings.Split(strings.Split(fileInfo[0].Name(), ".")[0], "_")[1]

	log := Log{
		UserID:    token.ID,
		CreatedAt: time.Now(),
		Content:   fmt.Sprintf("전체 현황 페이지에서 %s년의 데이터를 다운로드하였습니다.", filename),
	}

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Disposition", fmt.Sprintf("Attachment; filename=%s", fileInfo[0].Name()))
	http.ServeFile(w, r, path+"/"+fileInfo[0].Name())
}
