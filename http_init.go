// 프로젝트 결산 프로그램
//
// Description : http 메인 페이지 관련 스크립트

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

// handleIconFunc 함수는 무시 -> "/"" 두번 로드 오류 제거
func handleIconFunc(w http.ResponseWriter, r *http.Request) {}

// handleInitFunc 함수는 메인 페이지를 띄우는 함수이다.
func handleInitFunc(w http.ResponseWriter, r *http.Request) {
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

	type ProjectInfo struct {
		Project          Project // 프로젝트 정보
		Revenue          string  // 프로젝트 수익
		Vendor           string  // 프로젝트 외주비
		TotalExpenditure string  // 프로젝트 총 지출(내부 비용 + 외주비 + 경영관리실)
	}

	type Recipe struct {
		Token
		User User

		AllProject        []Project // 검색바에 들어갈 DB에 저장된 모든 프로젝트 리스트
		SelectedProjectID string    // 선택한 프로젝트 ID
		Status            []Status  // 프로젝트 상태 리스트
		SelectedStatus    string    // 선택한 프로젝트 상태
		SearchWord        string    // 검색어
		FinishedStatus    string    // ing: 진행중인 프로젝트만, end: 정산 완료된 프로젝트만, all: 모든 프로젝트
		RevenueStatus     string    // profit: 이익이 난 프로젝트, loss: 손해 난 프로젝트
		ExcludeRNDProject string    // true: RND, ETC 프로젝트 제외, false: RND, ETC 프로젝트 포함
		UpdatedTime       string

		Projects        []ProjectInfo // 프로젝트 정보
		ProjectsByToday []Project     // 세금계산서 발행일이 오늘인 프로젝트 리스트
		VendorsByToday  []Vendor      // 계약금, 중도금, 잔금 세금계산서 발행일이 오늘인 벤더 리스트
	}
	rcp := Recipe{}
	rcp.Token = token
	rcp.User, err = getUserFunc(client, token.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	q := r.URL.Query()
	rcp.AllProject, err = getAllProjectsFunc(client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rcp.SelectedProjectID = q.Get("project")
	rcp.Status = adminSetting.ProjectStatus
	rcp.SelectedStatus = q.Get("status")
	rcp.SearchWord = q.Get("searchword")
	fs := q.Get("finishedstatus")
	if fs == "" {
		fs = "ing"
	}
	rcp.FinishedStatus = fs
	rcp.RevenueStatus = q.Get("revenuestatus")
	rcp.ExcludeRNDProject = q.Get("excluderndproject")
	if rcp.ExcludeRNDProject == "" {
		rcp.ExcludeRNDProject = "true"
	}
	rcp.UpdatedTime = adminSetting.SGUpdatedTime

	// 프로젝트 검색
	searchword := "id:" + rcp.SelectedProjectID + " finishedstatus:" + rcp.FinishedStatus
	if rcp.SearchWord != "" {
		searchword = searchword + " " + rcp.SearchWord
	}
	searchedProjects, err := searchProjectFunc(client, searchword, "") // DB에서 searchword로 프로젝트 검색
	var projects []Project
	if rcp.SelectedStatus != "" { // 선택한 status가 있다면 DB에서 검색한 프로젝트들의 status 확인하여 Projects에 추가
		statusList := stringToListFunc(rcp.SelectedStatus, ",")
		for _, status := range statusList {
			for _, p := range searchedProjects {
				lastStatus := ""
				if p.IsFinished == true { // 정산 완료된 프로젝트인 경우 프로젝트의 제일 마지막 status를 가져온다.
					lastStatus, err = getLastStatusOfProjectFunc(p)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				} else { // 정산 완료되지 않은 프로젝트인 경우 이번 달의 status를 가져온다.
					lastStatus = getThisMonthStatusOfProjectFunc(p)
				}

				if lastStatus == status {
					projects = append(projects, p)
				}
			}
		}
	} else {
		projects = searchedProjects
	}

	// 내부 비용 계산
	for _, project := range projects {
		// RND, ETC 프로젝트를 제외해야 한다면 프로젝트가 RND, ETC 프로젝트에 속하는지 확인한다.
		if rcp.ExcludeRNDProject == "true" {
			if strings.Contains(project.ID, "ETC") || strings.Contains(project.ID, "RND") {
				continue
			}
		}

		// 진행중인 프로젝트라면 임시로 FinishedCost에 총 금액을 계산하여 넣어준다.
		if project.IsFinished == false {
			// 총 진행비 계산
			totalProgressCost, err := getTotalProgressCostFunc(project)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			project.FinishedCost.ProgressCost, err = encryptAES256Func(strconv.Itoa(totalProgressCost))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// 총 구매비 계산
			totalPurchaseCost, err := getTotalPurchaseCostFunc(project)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			project.FinishedCost.PurchaseCost, err = encryptAES256Func(strconv.Itoa(totalPurchaseCost))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// 총 인건비 계산
			totalLaborCostVFX, err := getLaborCostVFXFunc(project) // VFX
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			project.FinishedCost.LaborCost.VFX, err = encryptAES256Func(strconv.Itoa(totalLaborCostVFX))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			totalLaborCostCM, err := getLaborCostCMFunc(project) // CM
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			project.FinishedCost.LaborCost.CM, err = encryptAES256Func(strconv.Itoa(totalLaborCostCM))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// 총 내부비용 계산
			totalAmount, err := calTotalAmountOfFPFunc(project)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			project.TotalAmount, err = encryptAES256Func(strconv.Itoa(totalAmount))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		// 외주비 계산
		vendorSearchWord := "project:" + project.ID
		vendors, err := searchVendorFunc(client, vendorSearchWord)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		totalExpensesInt := 0
		for _, v := range vendors {
			expenses, err := decryptAES256Func(v.Expenses)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			expensesInt := 0
			if expenses != "" {
				expensesInt, err = strconv.Atoi(expenses)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			totalExpensesInt += expensesInt
		}

		// 총 지출 계산(내부 비용 + 외주비 + 경영관리실)
		totalAmount, err := decryptAES256Func(project.TotalAmount)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		totalAmountInt := 0
		if totalAmount != "" {
			totalAmountInt, err = strconv.Atoi(totalAmount)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		smDifference, err := decryptAES256Func(project.SMDifference)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		smDifferenceInt := 0
		if smDifference != "" {
			smDifferenceInt, err = strconv.Atoi(smDifference)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		totalExpenditure := totalAmountInt + totalExpensesInt + smDifferenceInt
		encryptedTotalAmount, err := encryptAES256Func(strconv.Itoa(totalExpenditure))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 수익 계산
		revenue, err := getRevenueOfFPFunc(project)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		revenue -= totalExpensesInt

		// 수익 상태 옵션에 맞게 rcp.ProjectInfo에 데이터를 넣는다.
		if rcp.RevenueStatus == "profit" && revenue < 0 { // 옵션이 수익이 난 프로젝트인데 손해가 났을 때 continue
			continue
		} else if rcp.RevenueStatus == "loss" && revenue > 0 { // 옵션이 손해가 난 프로젝트인데 수익이 났을 때 continue
			continue
		} else {
			encryptedRevenue, err := encryptAES256Func(strconv.Itoa(revenue))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			encryptedVendor, err := encryptAES256Func(strconv.Itoa(totalExpensesInt))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			pi := ProjectInfo{
				Project:          project,
				Revenue:          encryptedRevenue,
				Vendor:           encryptedVendor,
				TotalExpenditure: encryptedTotalAmount,
			}
			rcp.Projects = append(rcp.Projects, pi)
		}
	}

	// ProjectInfo 자료구조를 인자로 넘길 수 없어서 json 파일을 생성한다.
	path := os.TempDir() + "/budget/" + token.ID + "/init/" // json으로 바꾼 프로젝트 데이터를 저장할 임시 폴더 경로
	err = createFolderFunc(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonData, _ := json.Marshal(rcp.Projects)
	_ = ioutil.WriteFile(path+"/projects.json", jsonData, 0644)

	err = genInitExcelFunc(token.ID) // 엑셀 파일 미리 생성
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Admin 권한일 경우 세금 계산서 발행일이 오늘 날짜인 프로젝트와 벤더가 있는지 확인한다.
	if token.AccessLevel == AdminLevel {
		// 프로젝트의 월별 매출 발행일이 오늘인 프로젝트가 있는지 확인한다.
		projects, err := getProjectsByTodayFunc(client)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// 쿠키에서 해당 프로젝트를 오늘하루 보지 않겠다고 설정했는지 확인한다.
		cookie := r.Header["Cookie"]
		for _, project := range projects {
			if !strings.Contains(cookie[0], fmt.Sprintf("popup-project-%s=no", project.ID)) {
				rcp.ProjectsByToday = append(rcp.ProjectsByToday, project)
			}
		}

		// 벤더의 계약금, 중도금, 잔금 날짜가 오늘인 벤더들이 있는지 확인한다.
		vendors, err := getVendorsByTodayFunc(client)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// 쿠키에서 해당 벤더를 오늘하루 보지 않겠다고 설정했는지 확인한다.
		for _, vendor := range vendors {
			if !strings.Contains(cookie[0], fmt.Sprintf("popup-vendor-%s=no", vendor.ID.Hex())) {
				rcp.VendorsByToday = append(rcp.VendorsByToday, vendor)
			}
		}
	}

	err = TEMPLATES.ExecuteTemplate(w, "init", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleSearchFunc 함수는 메인 페이지에서 Search를 눌렀을 때 실행되는 함수이다.
func handleSearchFunc(w http.ResponseWriter, r *http.Request) {
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

	project := r.FormValue("project")
	status := r.FormValue("status")
	searchword := r.FormValue("searchword")
	revenueStatus := r.FormValue("revenueStatus")
	finishedStatus := ""
	if r.FormValue("finishedCheckbox1") == "on" && r.FormValue("finishedCheckbox2") == "on" {
		finishedStatus = "all"
	} else if r.FormValue("finishedCheckbox1") == "on" {
		finishedStatus = "ing"
	} else if r.FormValue("finishedCheckbox2") == "on" {
		finishedStatus = "end"
	} else {
		finishedStatus = "none"
	}
	excludeRNDProject := "false"
	if r.FormValue("excluderndproject") == "on" {
		excludeRNDProject = "true"
	}

	http.Redirect(w, r, fmt.Sprintf("/?finishedstatus=%s&project=%s&status=%s&searchword=%s&revenuestatus=%s&excluderndproject=%s", finishedStatus, project, status, searchword, revenueStatus, excludeRNDProject), http.StatusSeeOther)
}

// genInitExcelFunc 함수는 메인 페이지의 엑셀 파일을 만드는 함수이다.
func genInitExcelFunc(userID string) error {
	path := os.TempDir() + "/budget/" + userID + "/init/"

	// json 파일에서 프로젝트 데이터를 가져온다.
	jsonData, err := ioutil.ReadFile(path + "projects.json")
	if err != nil {
		return err
	}

	type ProjectInfo struct {
		Project          Project // 프로젝트 정보
		Revenue          string  // 프로젝트 수익
		Vendor           string  // 프로젝트 외주비
		TotalExpenditure string  // 프로젝트 총 지출(내부 비용 + 외주비 + 경영관리실)
	}

	var projects []ProjectInfo
	json.Unmarshal(jsonData, &projects)

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

	// 제목 입력
	f.SetCellValue(sheet, "A1", "Status")
	f.MergeCell(sheet, "A1", "A2")
	f.SetCellValue(sheet, "B1", "프로젝트")
	f.MergeCell(sheet, "B1", "B2")
	f.SetCellValue(sheet, "C1", "작업 기간")
	f.MergeCell(sheet, "C1", "C2")
	f.SetCellValue(sheet, "D1", "총 매출")
	f.MergeCell(sheet, "D1", "D2")
	f.SetCellValue(sheet, "E1", "총 지출")
	f.MergeCell(sheet, "E1", "J1")
	f.SetCellValue(sheet, "E2", "내부 인건비")
	f.SetCellValue(sheet, "F2", "진행비")
	f.SetCellValue(sheet, "G2", "구매비")
	f.SetCellValue(sheet, "H2", "외주비")
	f.SetCellValue(sheet, "I2", "공통노무비 외")
	f.SetCellValue(sheet, "J2", "합계")
	f.SetCellValue(sheet, "K1", "수익")
	f.MergeCell(sheet, "K1", "K2")
	f.SetCellValue(sheet, "L1", "내부 비용")
	f.MergeCell(sheet, "L1", "L2")
	f.SetCellValue(sheet, "M1", "외주비")
	f.MergeCell(sheet, "M1", "M2")
	f.SetCellValue(sheet, "N1", "공통노무비 외")
	f.MergeCell(sheet, "N1", "N2")
	f.SetCellValue(sheet, "O1", "수익")
	f.MergeCell(sheet, "O1", "O2")

	f.SetColWidth(sheet, "A", "K", 18)
	f.SetColWidth(sheet, "L", "O", 12)
	f.SetColWidth(sheet, "A", "A", 10)
	f.SetColWidth(sheet, "C", "C", 25)
	f.SetRowHeight(sheet, 1, 30)
	f.SetRowHeight(sheet, 2, 30)

	// 데이터 입력
	pos := ""
	for i, projectInfo := range projects {
		project := projectInfo.Project

		// status
		pos, err = excelize.CoordinatesToCellName(1, i+3) // ex) pos = "A3"
		if err != nil {
			return err
		}
		status := ""
		if project.IsFinished == true { // 정산 완료된 프로젝트인 경우 제일 마지막 Status
			status, err = getLastStatusOfProjectFunc(project)
			if err != nil {
				return err
			}
		} else { // 정산 완료되지 않은 프로젝트인 경우 이번달의 Status
			status = getThisMonthStatusOfProjectFunc(project)
		}
		f.SetCellValue(sheet, pos, status)

		// 이름
		pos, err = excelize.CoordinatesToCellName(2, i+3) // ex) pos = "B3"
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, project.Name)

		// 작업 기간
		pos, err = excelize.CoordinatesToCellName(3, i+3) // ex) pos = "C3"
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, fmt.Sprintf("%s ~ %s", project.StartDate, project.SMEndDate))

		// 총 매출
		pos, err = excelize.CoordinatesToCellName(4, i+3) // ex) pos = "D3"
		if err != nil {
			return err
		}
		paymentInt := 0
		for _, payment := range project.Payment {
			decrypted, err := decryptAES256Func(payment.Expenses)
			if err != nil {
				return err
			}
			if decrypted != "" {
				decryptedInt, err := strconv.Atoi(decrypted)
				if err != nil {
					return err
				}
				paymentInt += decryptedInt
			}
		}
		f.SetCellValue(sheet, pos, paymentInt)

		// 내부 인건비
		pos, err = excelize.CoordinatesToCellName(5, i+3) // ex) pos = "E3"
		if err != nil {
			return err
		}
		laborCost, err := getTotalLaborCostOfFPFunc(project)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, laborCost)

		// 진행비
		pos, err = excelize.CoordinatesToCellName(6, i+3) // ex) pos = "F3"
		if err != nil {
			return err
		}
		progressCost, err := decryptAES256Func(project.FinishedCost.ProgressCost)
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

		// 구매비
		pos, err = excelize.CoordinatesToCellName(7, i+3) // ex) pos = "F3"
		if err != nil {
			return err
		}
		purchaseCost, err := decryptAES256Func(project.FinishedCost.PurchaseCost)
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

		// 외주비
		pos, err = excelize.CoordinatesToCellName(8, i+3) // ex) pos = "F3"
		if err != nil {
			return err
		}
		vendor, err := decryptAES256Func(projectInfo.Vendor)
		if err != nil {
			return err
		}
		vendorInt, err := strconv.Atoi(vendor)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, vendorInt)

		// 경영관리실
		pos, err = excelize.CoordinatesToCellName(9, i+3)
		if err != nil {
			return err
		}
		difference, err := decryptAES256Func(project.SMDifference)
		if err != nil {
			return err
		}
		differenceInt := 0
		if difference != "" {
			differenceInt, err = strconv.Atoi(difference)
			if err != nil {
				return err
			}
		}
		f.SetCellValue(sheet, pos, differenceInt)

		// 총 지출(내부 비용 + 외주비 + 경영관리실)
		pos, err = excelize.CoordinatesToCellName(10, i+3)
		if err != nil {
			return err
		}
		totalAmount, err := decryptAES256Func(projectInfo.TotalExpenditure)
		if err != nil {
			return err
		}
		totalAmountInt := 0
		if totalAmount != "" {
			totalAmountInt, err = strconv.Atoi(totalAmount)
			if err != nil {
				return err
			}
		}
		f.SetCellValue(sheet, pos, totalAmountInt)

		// 수익
		pos, err = excelize.CoordinatesToCellName(11, i+3)
		if err != nil {
			return err
		}
		r, err := decryptAES256Func(projectInfo.Revenue)
		if err != nil {
			return err
		}
		rInt := 0
		if r != "" {
			rInt, err = strconv.Atoi(r)
			if err != nil {
				return err
			}
		}
		f.SetCellValue(sheet, pos, rInt)

		// 내부 비용 비율
		pos, err = excelize.CoordinatesToCellName(12, i+3)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, fmt.Sprintf("%s", calRatioFunc(project.TotalAmount, project.Payment))+" %")

		// 외주비 비율
		pos, err = excelize.CoordinatesToCellName(13, i+3)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, fmt.Sprintf("%s", calRatioFunc(projectInfo.Vendor, project.Payment))+" %")

		// 경영관리실 비율
		pos, err = excelize.CoordinatesToCellName(14, i+3)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, fmt.Sprintf("%s", calRatioFunc(project.SMDifference, project.Payment))+" %")

		// 수익 비율
		pos, err = excelize.CoordinatesToCellName(15, i+3)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, fmt.Sprintf("%s", calRatioFunc(projectInfo.Revenue, project.Payment))+" %")

		f.SetRowHeight(sheet, i+3, 20)
	}

	f.SetCellStyle(sheet, "A1", pos, style)
	f.SetCellStyle(sheet, "D3", strings.ReplaceAll(pos, "O", "K"), numberStyle)

	// 엑셀 파일 저장
	excelFileName := "budget.xlsx"
	err = f.SaveAs(path + "/" + excelFileName)
	if err != nil {
		return err
	}

	return nil
}

// handleInitExcelDownloadFunc 함수는 임시 폴더에 저장된 엑셀 파일을 다운로드하는 함수이다.
func handleExportInitFunc(w http.ResponseWriter, r *http.Request) {
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

	path := os.TempDir() + "/budget/" + token.ID + "/init"

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

	log := Log{
		UserID:    token.ID,
		CreatedAt: time.Now(),
		Content:   "메인 페이지에서 프로젝트 데이터를 다운로드하였습니다.",
	}

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Disposition", fmt.Sprintf("Attachment; filename=%s", fileInfo[0].Name()))
	http.ServeFile(w, r, path+"/"+fileInfo[0].Name())
}
