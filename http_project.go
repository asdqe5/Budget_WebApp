// 프로젝트 결산 프로그램
//
// Description : http 프로젝트 관련 스크립트

package main

import (
	"context"
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

// handleProjectsFunc 함수는 프로젝트 관리 페이지를 띄우는 함수이다.
func handleProjectsFunc(w http.ResponseWriter, r *http.Request) {
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
		Token             Token
		User              User
		Date              string   // yyyy-MM
		Status            []Status // 프로젝트 상태 리스트
		SelectedStatus    string   // 선택한 프로젝트 상태
		SearchWord        string   // 검색어
		ExcludeRNDProject string   // true: RND, ETC 프로젝트 제외, false: RND, ETC 프로젝트 포함

		Projects []Project // 프로젝트 리스트
	}
	rcp := Recipe{}
	rcp.Token = token
	rcp.User, err = getUserFunc(client, token.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	q := r.URL.Query()
	date := q.Get("date")
	if date == "" { // date 값이 없으면 올해로 검색
		y, m, _ := time.Now().Date()
		date = fmt.Sprintf("%04d-%02d", y, m)
	}
	rcp.Date = date
	rcp.Status = adminSetting.ProjectStatus
	rcp.SelectedStatus = q.Get("status")
	rcp.SearchWord = q.Get("searchword")
	rcp.ExcludeRNDProject = q.Get("excluderndproject")
	if rcp.ExcludeRNDProject == "" {
		rcp.ExcludeRNDProject = "true"
	}

	// 프로젝트 검색
	searchword := "date:" + rcp.Date
	if rcp.SearchWord != "" {
		searchword = searchword + " " + rcp.SearchWord
	}
	searchedProjects, err := searchProjectFunc(client, searchword, "id") // DB에서 searchword로 프로젝트 검색
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var projects []Project
	if rcp.SelectedStatus != "" { // 선택한 status가 있다면 DB에서 검색한 프로젝트들의 status 확인하여 Projects에 추가
		statusList := stringToListFunc(rcp.SelectedStatus, ",")
		for _, status := range statusList {
			for _, p := range searchedProjects {
				if p.SMStatus[date] == status {
					projects = append(projects, p)
				}
			}
		}
	} else {
		projects = searchedProjects
	}

	for _, project := range projects {
		// RND, ETC 프로젝트를 제외해야 한다면 프로젝트가 RND, ETC 프로젝트에 속하는지 확인한다.
		if rcp.ExcludeRNDProject == "true" {
			if strings.Contains(project.ID, "ETC") || strings.Contains(project.ID, "RND") {
				continue
			}
		}

		rcp.Projects = append(rcp.Projects, project)
	}

	err = genProjectExcelFunc(date, rcp.Projects, token.ID) // 엑셀 파일 미리 생성
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "projects", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleSearchProjectsFunc 함수는 프로젝트 페이지에서 Search를 눌렀을 때 실행되는 함수이다.
func handleSearchProjectsFunc(w http.ResponseWriter, r *http.Request) {
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

	date := r.FormValue("date")
	status := r.FormValue("status")
	searchword := r.FormValue("searchword")
	excludeRNDProject := "false"
	if r.FormValue("excluderndproject") == "on" {
		excludeRNDProject = "true"
	}

	http.Redirect(w, r, fmt.Sprintf("/projects?date=%s&status=%s&searchword=%s&excluderndproject=%s", date, status, searchword, excludeRNDProject), http.StatusSeeOther)
}

// handleAddProjectFunc 함수는 프로젝트 페이지에서 +를 눌렀을 때 실행되는 함수이다.
func handleAddProjectFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// manager 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < ManagerLevel {
		http.Redirect(w, r, "/invalidaccess", http.StatusSeeOther)
		return
	}

	type Recipe struct {
		Token Token
	}

	rcp := Recipe{
		Token: token,
	}

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "add-project", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleAddProjectSubmitFunc 함수는 add-project에서 ADD 버튼을 눌렀을 때 실행되는 함수이다.
func handleAddProjectSubmitFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// manager 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < ManagerLevel {
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

	p := Project{}
	p.ID = strings.TrimSpace(strings.ToUpper(r.FormValue("id")))
	p.Name = strings.TrimSpace(r.FormValue("name"))
	p.StartDate = r.FormValue("startdate")
	p.SMEndDate = r.FormValue("enddate")
	p.DirectorName = r.FormValue("directorname")
	p.ProducerName = r.FormValue("producername")

	// 프로젝트 부가정보 컷수
	if r.FormValue("contractcuts") != "" {
		contractCuts := r.FormValue("contractcuts")
		if strings.Contains(contractCuts, ",") {
			contractCuts = strings.ReplaceAll(contractCuts, ",", "")
		}
		p.ContractCuts, err = strconv.Atoi(contractCuts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if r.FormValue("workingcuts") != "" {
		workingCuts := r.FormValue("workingcuts")
		if strings.Contains(workingCuts, ",") {
			workingCuts = strings.ReplaceAll(workingCuts, ",", "")
		}
		p.WorkingCuts, err = strconv.Atoi(workingCuts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	var payment Payment
	expenses := r.FormValue("payment0")
	if strings.Contains(expenses, ",") {
		expenses = strings.ReplaceAll(expenses, ",", "")
	}
	payment.Expenses, err = encryptAES256Func(expenses)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	payment.Date = r.FormValue("paymentdate0")
	p.Payment = append(p.Payment, payment)
	err = p.CheckErrorFunc()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if r.FormValue("isfinished") == "on" { // 체크박스가 체크되어 있으면 on, 체크되어 있지않으면 빈 문자열이 들어온다.
		p.IsFinished = true
		if r.FormValue("totalamount") != "" {
			totalamount := r.FormValue("totalamount")
			if strings.Contains(totalamount, ",") {
				totalamount = strings.ReplaceAll(totalamount, ",", "")
			}
			p.TotalAmount, err = encryptAES256Func(totalamount)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if r.FormValue("laborcost") != "" { // 내부인건비의 총 금액은 VFX 내부인건비로 저장한다.
			laborcost := r.FormValue("laborcost")
			if strings.Contains(laborcost, ",") {
				laborcost = strings.ReplaceAll(laborcost, ",", "")
			}
			p.FinishedCost.LaborCost.VFX, err = encryptAES256Func(laborcost)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if r.FormValue("progresscost") != "" {
			progresscost := r.FormValue("progresscost")
			if strings.Contains(progresscost, ",") {
				progresscost = strings.ReplaceAll(progresscost, ",", "")
			}
			p.FinishedCost.ProgressCost, err = encryptAES256Func(progresscost)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if r.FormValue("purchasecost") != "" {
			purchasecost := r.FormValue("purchasecost")
			if strings.Contains(purchasecost, ",") {
				purchasecost = strings.ReplaceAll(purchasecost, ",", "")
			}
			p.FinishedCost.PurchaseCost, err = encryptAES256Func(purchasecost)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if r.FormValue("difference") != "" {
			difference := r.FormValue("difference")
			if strings.Contains(difference, ",") {
				difference = strings.ReplaceAll(difference, ",", "")
			}
			p.SMDifference, err = encryptAES256Func(difference)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	err = addProjectFunc(client, p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log := Log{
		UserID:    token.ID,
		CreatedAt: time.Now(),
		Content:   fmt.Sprintf("프로젝트 %s %s가 추가되었습니다.", p.ID, p.Name),
	}

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/addproject-success", http.StatusSeeOther)
}

// handleAddProjectSuccessFunc 함수는 프로젝트 추가를 성공했다는 페이지를 연다.
func handleAddProjectSuccessFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// manager 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < ManagerLevel {
		http.Redirect(w, r, "/invalidaccess", http.StatusSeeOther)
		return
	}

	type Recipe struct {
		Token
	}
	rcp := Recipe{}
	rcp.Token = token

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "addproject-success", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleEditProjectSMFunc 함수는 project 페이지에서 edit 버튼을 눌렀을 때 실행되는 함수이다.
func handleEditProjectSMFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// manager 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < ManagerLevel {
		http.Redirect(w, r, "/invalidaccess", http.StatusSeeOther)
		return
	}

	q := r.URL.Query()
	id := q.Get("id")
	if id == "" {
		http.Error(w, "URL에 id를 입력해주세요", http.StatusBadRequest)
		return
	}
	date := q.Get("date")
	if date == "" { // date 값이 없으면 올해로 검색
		y, m, _ := time.Now().Date()
		date = fmt.Sprintf("%04d-%02d", y, m)
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
		Token        Token
		User         User
		Project      Project  // 프로젝트 정보
		Date         []string // 테이블에 보여줄 프로젝트 작업기간
		Status       []Status // 프로젝트(결산) 상태 리스트
		FinishedType bool     // 정산 타입(true: 월별 합산 값으로 저장, false: 최종 입력 값으로 저장)

		SearchedDate string // 이전에 검색된 날짜
	}

	rcp := Recipe{}
	rcp.Token = token
	rcp.SearchedDate = date
	rcp.User, err = getUserFunc(client, token.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rcp.Project, err = getProjectFunc(client, id) // 프로젝트 ID를 통해 프로젝트를 가져온다.
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rcp.Date, err = getDatesFunc(rcp.Project.StartDate, rcp.Project.SMEndDate) // 기존의 작업시작과 작업마감 사이의 Date를 가져온다.
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if rcp.Project.SMStatus == nil && rcp.Project.SMMonthlyPayment == nil && rcp.Project.SMMonthlyProgressCost == nil && rcp.Project.SMMonthlyPurchaseCost == nil && rcp.Project.SMMonthlyLaborCost == nil {
		rcp.FinishedType = false
	} else {
		rcp.FinishedType = true
	}

	rcp.Status = adminSetting.ProjectStatus // 테이블에 들어갈 Status 설정

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "edit-projectsm", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleEditProjectSMSubmitFunc 함수는 프로젝트(결산) Edit 페이지에서 Update 버튼을 눌렀을 때 실행되는 함수이다.
func handleEditProjectSMSubmitFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// manager 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < ManagerLevel {
		http.Redirect(w, r, "/invalidaccess", http.StatusSeeOther)
		return
	}
	id := r.FormValue("id")
	searchedDate := r.FormValue("searcheddate")

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

	project, err := getProjectFunc(client, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dateList, err := getDatesFunc(project.StartDate, project.SMEndDate) // 기존의 작업시작과 작업마감 사이의 Date를 가져온다.
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 프로젝트의 이름이 바뀐 경우 해당 프로젝트의 벤더들의 프로젝트명도 변경해준다.
	orginalName := r.FormValue("originalname")
	project.Name = strings.TrimSpace(r.FormValue("name"))
	if orginalName != project.Name {
		vendors, err := searchVendorFunc(client, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, v := range vendors {
			v.ProjectName = project.Name
			err = setVendorFunc(client, v)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
	project.StartDate = r.FormValue("startdate")
	project.SMEndDate = r.FormValue("enddate")
	project.DirectorName = r.FormValue("directorname")
	project.ProducerName = r.FormValue("producername")

	// 프로젝트 부가정보 컷수
	if r.FormValue("contractcuts") != "" {
		contractCuts := r.FormValue("contractcuts")
		if strings.Contains(contractCuts, ",") {
			contractCuts = strings.ReplaceAll(contractCuts, ",", "")
		}
		project.ContractCuts, err = strconv.Atoi(contractCuts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		project.ContractCuts = 0
	}

	if r.FormValue("workingcuts") != "" {
		workingCuts := r.FormValue("workingcuts")
		if strings.Contains(workingCuts, ",") {
			workingCuts = strings.ReplaceAll(workingCuts, ",", "")
		}
		project.WorkingCuts, err = strconv.Atoi(workingCuts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		project.WorkingCuts = 0
	}

	// 총 매출
	paymentNum, err := strconv.Atoi(r.FormValue("paymentNum"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var paymentList []Payment
	for i := 0; i < paymentNum; i++ {
		payment := r.FormValue(fmt.Sprintf("payment%d", i))
		paymentDate := r.FormValue(fmt.Sprintf("paymentdate%d", i))
		if payment == "" && paymentDate == "" {
			continue
		}

		// 금액에 ","이 포함되어 있으면 ""으로 바꿔준다.
		if strings.Contains(payment, ",") {
			payment = strings.ReplaceAll(payment, ",", "")
		}

		encryptedPayment, err := encryptAES256Func(payment)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		pay := Payment{
			Date:     paymentDate,
			Expenses: encryptedPayment,
		}
		paymentList = append(paymentList, pay)
	}
	project.Payment = paymentList

	err = project.CheckErrorFunc()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if r.FormValue("isfinished") == "on" { // 체크박스가 체크되어 있으면 on, 체크되어 있지않으면 빈 문자열이 들어온다.
		project.IsFinished = true
		if r.FormValue("difference") != "" {
			difference := r.FormValue("difference")
			if strings.Contains(difference, ",") {
				difference = strings.ReplaceAll(difference, ",", "")
			}
			project.SMDifference, err = encryptAES256Func(difference)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if r.FormValue("typeCheckbox1") == "false" { // 최종 입력 값으로 저장 radio 버튼이 클릭되어있는 경우
			if r.FormValue("totalamount") != "" {
				totalamount := r.FormValue("totalamount")
				if strings.Contains(totalamount, ",") {
					totalamount = strings.ReplaceAll(totalamount, ",", "")
				}
				project.TotalAmount, err = encryptAES256Func(totalamount)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			if r.FormValue("laborcost") != "" {
				laborcost := r.FormValue("laborcost")
				if strings.Contains(laborcost, ",") {
					laborcost = strings.ReplaceAll(laborcost, ",", "")
				}
				project.FinishedCost.LaborCost.VFX, err = encryptAES256Func(laborcost)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			project.FinishedCost.LaborCost.CM = ""
			if r.FormValue("progresscost") != "" {
				progresscost := r.FormValue("progresscost")
				if strings.Contains(progresscost, ",") {
					progresscost = strings.ReplaceAll(progresscost, ",", "")
				}
				project.FinishedCost.ProgressCost, err = encryptAES256Func(progresscost)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			if r.FormValue("purchasecost") != "" {
				purchasecost := r.FormValue("purchasecost")
				if strings.Contains(purchasecost, ",") {
					purchasecost = strings.ReplaceAll(purchasecost, ",", "")
				}
				project.FinishedCost.PurchaseCost, err = encryptAES256Func(purchasecost)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
		}
	} else { // 정산 완료된 프로젝트에 체크가 되어있지않는 경우 FinishedCost를 초기화해준다.
		project.IsFinished = false
		project.TotalAmount = ""
		project.FinishedCost = Cost{}
		project.SMDifference = ""
	}

	statusmap := make(map[string]string)
	monthlyProgressCost := make(map[string]string)
	monthlyLaborCost := make(map[string]LaborCost)
	if project.SMMonthlyLaborCost != nil {
		monthlyLaborCost = project.SMMonthlyLaborCost
	}
	supervisorIDs := adminSetting.SMSupervisorIDs

	for _, d := range dateList {
		// 각 날짜의 status를 가져온다
		ds := fmt.Sprintf("%sstatus", d)
		if r.FormValue(ds) != "" {
			statusmap[d] = r.FormValue(ds)
		}

		// 월별 결산 진행비 정보를 가져온다.
		dproc := fmt.Sprintf("%ssmprogresscost", d)
		smprogresscost := r.FormValue(dproc)
		if smprogresscost != "" {
			if strings.Contains(smprogresscost, ",") {
				smprogresscost = strings.ReplaceAll(smprogresscost, ",", "")
			}
			monthlyProgressCost[d], err = encryptAES256Func(smprogresscost)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	project.SMStatus = statusmap
	project.SMMonthlyProgressCost = monthlyProgressCost
	project.SMMonthlyLaborCost = monthlyLaborCost

	// 최종 입력 값으로 정산하는 경우 월별 금액을 모두 초기화한다.
	if r.FormValue("isfinished") == "on" && r.FormValue("typeCheckbox1") == "false" {
		project.SMStatus = nil
		project.SMMonthlyPayment = nil
		project.SMMonthlyProgressCost = nil
		project.SMMonthlyPurchaseCost = nil
		project.SMMonthlyLaborCost = nil

		// 슈퍼바이저들의 월별 타임로그를 모두 삭제한다.
		for _, d := range dateList {
			for _, id := range supervisorIDs {
				var st Timelog
				st.UserID = id
				st.Year, _ = strconv.Atoi(strings.Split(d, "-")[0])
				st.Month, _ = strconv.Atoi(strings.Split(d, "-")[1])
				st.Project = project.ID

				timelog, err := getTimelogFunc(client, st.UserID, st.Year, st.Month, st.Project)
				if err != nil {
					if err == mongo.ErrNoDocuments { // 타임로그가 없으면 continue
						continue
					}
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				err = rmTimelogFunc(client, timelog) // 타임로그가 있으면 삭제한다.
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
		}
	}

	err = setProjectFunc(client, project)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 월별 합산 값으로 정산할 경우 FinishedCost를 계산한다.
	if r.FormValue("isfinished") == "on" && r.FormValue("typeCheckbox1") == "true" {
		err = calFinishedProjectCostFunc(project)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	log := Log{
		UserID:    token.ID,
		CreatedAt: time.Now(),
		Content:   fmt.Sprintf("프로젝트 %s의 정보가 수정되었습니다.", id),
	}

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/editprojectsm-success?id=%s&date=%s", id, searchedDate), http.StatusSeeOther)
}

// handleEditProjectSMSuccessFunc 함수는 프로젝트 결산 정보 수정을 성공했다는 페이지를 연다.
func handleEditProjectSMSuccessFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// manager 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < ManagerLevel {
		http.Redirect(w, r, "/invalidaccess", http.StatusSeeOther)
		return
	}

	q := r.URL.Query()
	id := q.Get("id")
	if id == "" {
		http.Error(w, "URL에 id를 입력해주세요", http.StatusBadRequest)
		return
	}
	date := q.Get("date")
	if date == "" { // date 값이 없으면 올해로 검색
		y, m, _ := time.Now().Date()
		date = fmt.Sprintf("%04d-%02d", y, m)
	}

	type Recipe struct {
		Token
		ID           string // 프로젝트 ID
		SearchedDate string // 검색한 달
	}
	rcp := Recipe{
		Token:        token,
		ID:           id,
		SearchedDate: date,
	}

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "editprojectsm-success", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// genProjectExcelFunc 함수는 프로젝트 데이터를 엑셀 파일로 생성하는 함수이다.
func genProjectExcelFunc(date string, projects []Project, userID string) error {
	path := os.TempDir() + "/budget/" + userID + "/project/"
	excelFileName := fmt.Sprintf("project_%s.xlsx", strings.ReplaceAll(date, "-", "_"))

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

	// 제목 입력
	f.SetCellValue(sheet, "A1", "Status")
	f.MergeCell(sheet, "A1", "A2")
	f.SetCellValue(sheet, "B1", "ID")
	f.MergeCell(sheet, "B1", "B2")
	f.SetCellValue(sheet, "C1", "이름")
	f.MergeCell(sheet, "C1", "C2")
	f.SetCellValue(sheet, "D1", "작업 기간")
	f.MergeCell(sheet, "D1", "D2")
	f.SetCellValue(sheet, "E1", "컷수 정보")
	f.MergeCell(sheet, "E1", "F1")
	f.SetCellValue(sheet, "E2", "계약 컷수")
	f.SetCellValue(sheet, "F2", "작업 컷수")
	f.SetCellValue(sheet, "G1", "총 매출")
	f.MergeCell(sheet, "G1", "G2")
	f.SetCellValue(sheet, "H1", "진행비")
	f.MergeCell(sheet, "H1", "H2")
	f.SetCellValue(sheet, "I1", "구매비")
	f.MergeCell(sheet, "I1", "I2")

	f.SetColWidth(sheet, "A", "I", 20)
	f.SetColWidth(sheet, "A", "B", 10)
	f.SetColWidth(sheet, "C", "D", 30)
	f.SetColWidth(sheet, "E", "F", 15)

	f.SetRowHeight(sheet, 1, 25)
	f.SetRowHeight(sheet, 2, 25)

	// 데이터 입력
	pos := ""
	for i, project := range projects {
		// 프로젝트 Status
		pos, err = excelize.CoordinatesToCellName(1, i+3) // ex) pos = "A3"
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, project.SMStatus[date])

		// 프로젝트 ID
		pos, err = excelize.CoordinatesToCellName(2, i+3) // ex) pos = "B3"
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, project.ID)

		// 프로젝트 이름
		pos, err = excelize.CoordinatesToCellName(3, i+3) // ex) pos = "C3"
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, project.Name)

		// 프로젝트 작업 기간
		pos, err = excelize.CoordinatesToCellName(4, i+3) // ex) pos = "D3"
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, fmt.Sprintf("%s ~ %s", stringToDateFunc(project.StartDate), stringToDateFunc(project.SMEndDate)))

		// 프로젝트 컷수 정보
		pos, err = excelize.CoordinatesToCellName(5, i+3) // ex) pos = "E3"
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, project.ContractCuts) // 계약 컷수

		pos, err = excelize.CoordinatesToCellName(6, i+3) // ex) pos = "F3"
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, project.WorkingCuts) // 작업 컷수

		// 총 매출
		pos, err = excelize.CoordinatesToCellName(7, i+3) // ex) pos = "G3"
		if err != nil {
			return err
		}
		intPayment := 0
		for _, payment := range project.Payment {
			// 매출 내역
			expenses, err := decryptAES256Func(payment.Expenses)
			if err != nil {
				return err
			}
			if expenses != "" {
				expensesInt, err := strconv.Atoi(expenses)
				if err != nil {
					return err
				}
				intPayment += expensesInt
			}
		}
		f.SetCellValue(sheet, pos, intPayment)

		// 프로젝트 진행비
		pos, err = excelize.CoordinatesToCellName(8, i+3) // ex) pos = "H3"
		if err != nil {
			return err
		}
		progressCost, err := decryptAES256Func(project.SMMonthlyProgressCost[date]) // 복호화
		if err != nil {
			return err
		}
		intProgressCost := 0
		if progressCost != "" {
			intProgressCost, err = strconv.Atoi(progressCost)
			if err != nil {
				return err
			}
		}
		f.SetCellValue(sheet, pos, intProgressCost)

		// 프로젝트 구매비
		pos, err = excelize.CoordinatesToCellName(9, i+3) // ex) pos = "I3"
		if err != nil {
			return err
		}
		totalPurchaseCost := 0
		for _, value := range project.SMMonthlyPurchaseCost[date] {
			purchaseCost, err := decryptAES256Func(value.Expenses)
			if err != nil {
				return err
			}
			intPurchaseCost, err := strconv.Atoi(purchaseCost)
			if err != nil {
				return err
			}
			totalPurchaseCost += intPurchaseCost
		}
		f.SetCellValue(sheet, pos, totalPurchaseCost)

		f.SetRowHeight(sheet, i+3, 20)
	}

	f.SetCellStyle(sheet, "A1", pos, style)
	f.SetCellStyle(sheet, "G3", pos, numberStyle)

	// 엑셀 파일 저장
	err = f.SaveAs(path + "/" + excelFileName)
	if err != nil {
		return err
	}

	return nil
}

// handleExportProjectsFunc 함수는 임시 폴더에 저장된 엑셀 파일을 다운로드하는 함수이다.
func handleExportProjectsFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// admin 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < AdminLevel {
		http.Redirect(w, r, "/invalidaccess", http.StatusSeeOther)
		return
	}

	// Post 메소드가 아니면 에러
	if r.Method != http.MethodPost {
		http.Error(w, "Post Method Only", http.StatusMethodNotAllowed)
		return
	}

	path := os.TempDir() + "/budget/" + token.ID + "/project"

	// path에 있는 파일들을 가져온다.
	fileInfo, err := ioutil.ReadDir(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// path에 파일의 개수가 하나가 아니면 엑셀 파일이 다시 생성되도록 리다이렉트
	if len(fileInfo) != 1 {
		http.Redirect(w, r, "/projects", http.StatusSeeOther)
		return
	}

	// 파일의 확장자가 xlsx가 아니면 엑셀 파일이 다시 생성되도록 리다이렉트
	ext := filepath.Ext(fileInfo[0].Name())
	if ext != ".xlsx" {
		http.Redirect(w, r, "/projects", http.StatusSeeOther)
		return
	}

	filename := strings.Split(strings.Split(fileInfo[0].Name(), ".")[0], "_")

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
		Content:   fmt.Sprintf("프로젝트 관리 페이지에서 %s년 %s월의 데이터를 다운로드하였습니다.", filename[1], filename[2]),
	}

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Disposition", fmt.Sprintf("Attachment; filename=%s", fileInfo[0].Name()))
	http.ServeFile(w, r, path+"/"+fileInfo[0].Name())
}
