// 프로젝트 결산 프로그램
//
// Description : http 예산 프로젝트 관련 스크립트

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/360EntSecGroup-Skylar/excelize/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// handleBGProjectsFunc 함수는 예산 프로젝트 관리 페이지를 띄우는 함수이다.
func handleBGProjectsFunc(w http.ResponseWriter, r *http.Request) {
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
		Token      Token
		User       User
		Date       string      // yyyy-MM
		SearchWord string      // 검색어
		BGProjects []BGProject // 예산 프로젝트 리스트
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
	rcp.SearchWord = q.Get("searchword")

	// 프로젝트 검색
	searchword := "date:" + rcp.Date
	if rcp.SearchWord != "" {
		searchword = searchword + " " + rcp.SearchWord
	}
	rcp.BGProjects, err = searchBGProjectFunc(client, searchword, "id") // DB 에서 searchword로 프로젝트 검색
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = genBGProjectsExcelFunc(date, rcp.BGProjects, token.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "bgprojects", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleSearchBGProjectsFunc 함수는 예산 프로젝트 페이지에서 Search를 눌렀을 때 실행되는 함수이다.
func handleSearchBGProjectsFunc(w http.ResponseWriter, r *http.Request) {
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
	searchword := r.FormValue("searchword")

	http.Redirect(w, r, fmt.Sprintf("/bgprojects?date=%s&searchword=%s", date, searchword), http.StatusSeeOther)
}

// handleAddBGProjectFunc 함수는 예산 프로젝트를 추가하는 페이지를 띄우는 함수이다.
func handleAddBGProjectFunc(w http.ResponseWriter, r *http.Request) {
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

	adminSetting, err := getAdminSettingFunc(client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type Recipe struct {
		Token      Token
		Supervisor []Artist // AdminSetting에서 설정한 수퍼바이저 팀에 해당하는 아티스트
		Production []Artist // AdminSetting에서 설정한 프로덕션 팀에 해당하는 아티스트
		Management []Artist // AdminSetting에서 설정한 매니지먼트 팀에 해당하는 아티스트
	}

	rcp := Recipe{}
	rcp.Token = token
	rcp.Supervisor, err = getArtistByTeamsFunc(client, adminSetting.BGSupervisorTeams)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rcp.Production, err = getArtistByTeamsFunc(client, adminSetting.BGProductionTeams)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rcp.Management, err = getArtistByTeamsFunc(client, adminSetting.BGManagementTeams)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "add-bgproject", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleAddBGProjectSubmitFunc 함수는 예산 프로젝트 추가 페이지에서 ADD 버튼을 눌렀을 때 실행하는 함수이다.
func handleAddBGProjectSubmitFunc(w http.ResponseWriter, r *http.Request) {
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

	// 팀세팅 정보 가져오기
	bgts, err := getBGTeamSettingFunc(client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	bgts.UpdatedTime = time.Now().Format(time.RFC3339) // 팀세팅의 마지막 업데이트된 시간을 현재 시간으로 설정

	// 예산 프로젝트 기본 정보
	bgp := BGProject{}
	bgp.ID = strings.TrimSpace(strings.ToUpper(r.FormValue("id"))) // 프로젝트 ID
	bgp.Name = strings.TrimSpace(r.FormValue("name"))              // 프로젝트 한글명
	bgp.StartDate = r.FormValue("startdate")                       // 프로젝트 예상 시작일
	bgp.EndDate = r.FormValue("enddate")                           // 프로젝트 예상 마감일
	bgp.DirectorName = r.FormValue("directorname")                 // 프로젝트 감독이름
	bgp.ProducerName = r.FormValue("producername")                 // 프로젝트 제작사
	bgp.Status, err = strconv.ParseBool(r.FormValue("status"))     // 프로젝트 상태 (계약 완료 or 사전 검토)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	bgp.Type = r.FormValue("type")

	bgtype := strings.TrimSpace(r.FormValue("bgtype")) // 예산안 타입
	bgp.TypeList = append(bgp.TypeList, bgtype)        // 예산안 타입 리스트
	bgp.TypeData = make(map[string]BGTypeData)         // 예산안 데이터

	// 예산안 정보
	bgtd := BGTypeData{}                              // 예산안 타입 데이터
	bgtd.ID = primitive.NewObjectID()                 // 예산안 Object ID 생성 후 저장
	bgtd.TeamSetting = bgts                           // 예산안 팀세팅 저장
	bgtd.ContractDate = r.FormValue("bgcontractdate") // 예산안 계약일
	if r.FormValue("bgmaintypestatus") != "" {
		status, err := strconv.ParseBool(r.FormValue("bgmaintypestatus"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if status {
			bgp.MainType = bgtype // 예산 프로젝트 메인 타입
		}
	}
	if r.FormValue("bgproposal") != "" { // 예산안 제안 견적
		proposal := r.FormValue("bgproposal")
		if strings.Contains(proposal, ",") {
			proposal = strings.ReplaceAll(proposal, ",", "")
		}
		bgtd.Proposal, err = encryptAES256Func(proposal)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if r.FormValue("bgdecision") != "" { // 예산안 계약 결정액
		decision := r.FormValue("bgdecision")
		if strings.Contains(decision, ",") {
			decision = strings.ReplaceAll(decision, ",", "")
		}
		bgtd.Decision, err = encryptAES256Func(decision)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if r.FormValue("bgcontractcuts") != "" { // 예산안 프로젝트 계약 컷수
		contractCuts := r.FormValue("bgcontractcuts")
		if strings.Contains(contractCuts, ",") {
			contractCuts = strings.ReplaceAll(contractCuts, ",", "")
		}
		bgtd.ContractCuts, err = strconv.Atoi(contractCuts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if r.FormValue("bgworkingcuts") != "" { // 예산안 프로젝트 작업 컷수
		workingCuts := r.FormValue("bgworkingcuts")
		if strings.Contains(workingCuts, ",") {
			workingCuts = strings.ReplaceAll(workingCuts, ",", "")
		}
		bgtd.WorkingCuts, err = strconv.Atoi(workingCuts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// 실예산안 계산 비율 정보
	if r.FormValue("bgretakeratio") != "" {
		bgtd.RetakeRatio, err = strconv.ParseFloat(r.FormValue("bgretakeratio"), 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if r.FormValue("bgprogressratio") != "" {
		bgtd.ProgressRatio, err = strconv.ParseFloat(r.FormValue("bgprogressratio"), 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if r.FormValue("bgvendorratio") != "" {
		bgtd.VendorRatio, err = strconv.ParseFloat(r.FormValue("bgvendorratio"), 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// 수퍼바이저, 프로덕션, 매니지먼트 정보 기입
	var controlUserIDList []string
	supNum, err := strconv.Atoi(r.FormValue("bgsupervisornum"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for i := 0; i < supNum; i++ { // 수퍼바이저 정보 기입
		if r.FormValue(fmt.Sprintf("supervisor%d", i)) == "" {
			continue
		}
		bgmng := BGManagement{}
		bgmng.UserID = r.FormValue(fmt.Sprintf("supervisor%d", i))
		if !checkStringInListFunc(bgmng.UserID, controlUserIDList) {
			controlUserIDList = append(controlUserIDList, bgmng.UserID)
		}
		bgmng.Work = r.FormValue(fmt.Sprintf("supervisor%d-bgmanagementwork", i))
		if r.FormValue(fmt.Sprintf("supervisor%d-bgmanagementperiod", i)) != "" {
			bgmng.Period, err = strconv.Atoi(r.FormValue(fmt.Sprintf("supervisor%d-bgmanagementperiod", i)))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if r.FormValue(fmt.Sprintf("supervisor%d-bgmanagementratio", i)) != "" {
			bgmng.Ratio, err = strconv.ParseFloat(r.FormValue(fmt.Sprintf("supervisor%d-bgmanagementratio", i)), 64)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		bgtd.Supervisors = append(bgtd.Supervisors, bgmng)
	}

	// 프로덕션 정보 기입
	prodNum, err := strconv.Atoi(r.FormValue("bgproductionnum"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for i := 0; i < prodNum; i++ {
		if r.FormValue(fmt.Sprintf("production%d", i)) == "" {
			continue
		}
		bgmng := BGManagement{}
		bgmng.UserID = r.FormValue(fmt.Sprintf("production%d", i))
		if !checkStringInListFunc(bgmng.UserID, controlUserIDList) {
			controlUserIDList = append(controlUserIDList, bgmng.UserID)
		}
		bgmng.Work = r.FormValue(fmt.Sprintf("production%d-bgmanagementwork", i))
		if r.FormValue(fmt.Sprintf("production%d-bgmanagementperiod", i)) != "" {
			bgmng.Period, err = strconv.Atoi(r.FormValue(fmt.Sprintf("production%d-bgmanagementperiod", i)))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if r.FormValue(fmt.Sprintf("production%d-bgmanagementratio", i)) != "" {
			bgmng.Ratio, err = strconv.ParseFloat(r.FormValue(fmt.Sprintf("production%d-bgmanagementratio", i)), 64)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		bgtd.Production = append(bgtd.Production, bgmng)
	}

	// 매니지먼트 정보 기입
	mngNum, err := strconv.Atoi(r.FormValue("bgmanagementnum"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for i := 0; i < mngNum; i++ {
		if r.FormValue(fmt.Sprintf("management%d", i)) == "" {
			continue
		}
		bgmng := BGManagement{}
		bgmng.UserID = r.FormValue(fmt.Sprintf("management%d", i))
		if !checkStringInListFunc(bgmng.UserID, controlUserIDList) {
			controlUserIDList = append(controlUserIDList, bgmng.UserID)
		}
		bgmng.Work = r.FormValue(fmt.Sprintf("management%d-bgmanagementwork", i))
		if r.FormValue(fmt.Sprintf("management%d-bgmanagementperiod", i)) != "" {
			bgmng.Period, err = strconv.Atoi(r.FormValue(fmt.Sprintf("management%d-bgmanagementperiod", i)))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if r.FormValue(fmt.Sprintf("management%d-bgmanagementratio", i)) != "" {
			bgmng.Ratio, err = strconv.ParseFloat(r.FormValue(fmt.Sprintf("management%d-bgmanagementratio", i)), 64)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		bgtd.Management = append(bgtd.Management, bgmng)
	}

	// 예산 팀세팅에서 Controls의 Key에 따른 userID 정리하기
	userIDsByHead := make(map[string][]string)
	for key, value := range bgts.Controls {
		for _, control := range value {
			for _, part := range control.Parts {
				artists, err := getArtistByTeamsFunc(client, part.Teams)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				for _, artist := range artists {
					if !checkStringInListFunc(artist.ID, userIDsByHead[key]) {
						userIDsByHead[key] = append(userIDsByHead[key], artist.ID)
					}
				}
			}
		}
	}

	// 매니지먼트에 적힌 UserID를 head에 따라 정리한 ID와 비교하여 분류하기
	manageUserIDs := make(map[string][]string)
	for _, id := range controlUserIDList {
		for head, idList := range userIDsByHead {
			if checkStringInListFunc(id, idList) {
				if !checkStringInListFunc(id, manageUserIDs[head]) {
					manageUserIDs[head] = append(manageUserIDs[head], id)
				}
			}
		}
	}

	// 정리된 Head 별 매니지먼트 ID에 따른 비용 계산
	for head, idList := range manageUserIDs {
		bgls := BGLaborCost{}
		bgls.Headquarter = head

		managementCost := 0
		for _, sup := range bgtd.Supervisors { // 수퍼바이저 비용 계산
			if !checkStringInListFunc(sup.UserID, idList) {
				continue
			}
			artist, err := getArtistFunc(client, sup.UserID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			salary := artist.Salary[strconv.Itoa(time.Now().Year())] // 아티스트의 올해 연봉 정보

			// 아티스트의 동일 연도 연봉 변경 정보가 있는지 체크
			if artist.Changed {
				// 오늘 날짜와 동일 연도 연봉 변경일 비교
				thisDate := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC)
				for key, value := range artist.ChangedSalary {
					changedDate, err := time.Parse("2006-01-02", key)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					if thisDate.Before(changedDate) { // 변경 전 연봉으로 계산
						salary = value
					}
				}
			}

			// 연봉 정보가 있다면 계산
			if salary != "" {
				decryptSalary, err := decryptAES256Func(salary)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if decryptSalary != "" { // 복호화된 금액 정보가 있다면
					intSalary, err := strconv.Atoi(decryptSalary)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					supCost := int(math.Round((float64(intSalary) * 10000 / 12) * (sup.Ratio / 100) * float64(sup.Period))) // 30일 기준 월급 * 비율 * 기간
					managementCost += supCost
				}
			}
		}

		for _, prod := range bgtd.Production { // 프로덕션 비용 계산
			if !checkStringInListFunc(prod.UserID, idList) {
				continue
			}
			artist, err := getArtistFunc(client, prod.UserID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			salary := artist.Salary[strconv.Itoa(time.Now().Year())] // 아티스트의 올해 연봉 정보

			// 아티스트의 동일 연도 연봉 변경 정보가 있는지 체크
			if artist.Changed {
				// 오늘 날짜와 동일 연도 연봉 변경일 비교
				thisDate := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC)
				for key, value := range artist.ChangedSalary {
					changedDate, err := time.Parse("2006-01-02", key)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					if thisDate.Before(changedDate) { // 변경 전 연봉으로 계산
						salary = value
					}
				}
			}

			// 연봉 정보가 있다면 계산
			if salary != "" {
				decryptSalary, err := decryptAES256Func(salary)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if decryptSalary != "" { // 복호화된 금액 정보가 있다면
					intSalary, err := strconv.Atoi(decryptSalary)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					prodCost := int(math.Round((float64(intSalary) * 10000 / 12) * (prod.Ratio / 100) * float64(prod.Period))) // 30일 기준 월급 * 비율 * 기간
					managementCost += prodCost
				}
			}
		}

		for _, mng := range bgtd.Management { // 매니지먼트 비용 계산
			if !checkStringInListFunc(mng.UserID, idList) {
				continue
			}
			artist, err := getArtistFunc(client, mng.UserID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			salary := artist.Salary[strconv.Itoa(time.Now().Year())] // 아티스트의 올해 연봉 정보

			// 아티스트의 동일 연도 연봉 변경 정보가 있는지 체크
			if artist.Changed {
				// 오늘 날짜와 동일 연도 연봉 변경일 비교
				thisDate := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC)
				for key, value := range artist.ChangedSalary {
					changedDate, err := time.Parse("2006-01-02", key)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					if thisDate.Before(changedDate) { // 변경 전 연봉으로 계산
						salary = value
					}
				}
			}

			// 연봉 정보가 있다면 계산
			if salary != "" {
				decryptSalary, err := decryptAES256Func(salary)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if decryptSalary != "" { // 복호화된 금액 정보가 있다면
					intSalary, err := strconv.Atoi(decryptSalary)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					mngCost := int(math.Round((float64(intSalary) * 10000 / 12) * (mng.Ratio / 100) * float64(mng.Period))) // 30일 기준 월급 * 비율 * 기간
					managementCost += mngCost
				}
			}
		}

		// 계산된 매니지먼트 비용 암호화하여 저장
		encryptedCost, err := encryptAES256Func(strconv.Itoa(managementCost))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		bgls.Management = encryptedCost
		bgtd.LaborCosts = append(bgtd.LaborCosts, bgls)
	}

	// 예산 프로젝트 정보에 예산안 데이터 저장
	bgp.TypeData[bgtype] = bgtd

	err = addBGProjectFunc(client, bgp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log := Log{
		UserID:    token.ID,
		CreatedAt: time.Now(),
		Content:   fmt.Sprintf("예산 프로젝트 %s %s가 추가되었습니다.", bgp.ID, bgp.Name),
	}

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/addbgproject-success", http.StatusSeeOther)
}

// handleAddBGProjectSuccessFunc 함수는 예산 프로젝트 추가를 성공했다는 페이지를 연다.
func handleAddBGProjectSuccessFunc(w http.ResponseWriter, r *http.Request) {
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
	err = TEMPLATES.ExecuteTemplate(w, "addbgproject-success", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleEditBGProjectFunc 함수는 예산 프로젝트 관리 페이지에서 edit 버튼을 눌렀을 때 실행되는 함수이다.
func handleEditBGProjectFunc(w http.ResponseWriter, r *http.Request) {
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
		SearchedDate string    // 이전에 검색된 날짜
		BGProject    BGProject // 예산 프로젝트 정보
		Supervisor   []Artist  // AdminSetting에서 설정한 수퍼바이저 팀에 해당하는 아티스트
		Production   []Artist  // AdminSetting에서 설정한 프로덕션 팀에 해당하는 아티스트
		Management   []Artist  // AdminSetting에서 설정한 매니지먼트 팀에 해당하는 아티스트
	}

	rcp := Recipe{}
	rcp.Token = token
	rcp.SearchedDate = date
	rcp.BGProject, err = getBGProjectFunc(client, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rcp.Supervisor, err = getArtistByTeamsFunc(client, adminSetting.BGSupervisorTeams)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rcp.Production, err = getArtistByTeamsFunc(client, adminSetting.BGProductionTeams)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rcp.Management, err = getArtistByTeamsFunc(client, adminSetting.BGManagementTeams)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "edit-bgproject", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleEditBGProjectSubmitFunc 함수는 예산 프로젝트 수정 페이지에서 Update 버튼을 눌렀을 때 실행되는 함수이다.
func handleEditBGProjectSubmitFunc(w http.ResponseWriter, r *http.Request) {
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
	originalID := r.FormValue("originalid")
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

	// 현재 팀세팅 정보 가져오기
	bgts, err := getBGTeamSettingFunc(client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 예산 프로젝트 정보
	bgp, err := getBGProjectFunc(client, originalID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	bgp.ID = strings.TrimSpace(strings.ToUpper(r.FormValue("id"))) // 프로젝트 ID
	bgp.Name = strings.TrimSpace(r.FormValue("name"))              // 프로젝트 한글명
	bgp.StartDate = r.FormValue("startdate")                       // 프로젝트 예상 시작일
	bgp.EndDate = r.FormValue("enddate")                           // 프로젝트 예상 마감일
	bgp.DirectorName = r.FormValue("directorname")                 // 프로젝트 감독이름
	bgp.ProducerName = r.FormValue("producername")                 // 프로젝트 제작사
	bgp.Status, err = strconv.ParseBool(r.FormValue("status"))     // 프로젝트 상태 (계약 완료 or 사전 검토)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	bgp.Type = r.FormValue("type") // 프로젝트 유형 (영화 or 드라마)

	origTypeData := make(map[string]BGTypeData) // 기존의 예산안 데이터
	origTypeData = bgp.TypeData
	bgp.TypeList = nil                         // 예산안 리스트 초기화
	bgp.TypeData = make(map[string]BGTypeData) // 예산안 데이터 초기화

	tabNum, err := strconv.Atoi(r.FormValue("tabnum"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for i := 0; i < tabNum; i++ {
		if r.FormValue(fmt.Sprintf("type%d-bgtype", i)) == "" {
			continue
		}
		bgtype := strings.TrimSpace(r.FormValue(fmt.Sprintf("type%d-bgtype", i))) // 예산안 타입
		if !checkStringInListFunc(bgtype, bgp.TypeList) {
			bgp.TypeList = append(bgp.TypeList, bgtype) // 예산안 타입 리스트
		} else { // 이미 존재하는 이름의 예산안이라면 저장하지 않는다.
			continue
		}

		// 예산안이 기존의 예산안인지 고유 ID로 판단하기
		bgtd := BGTypeData{}                                      // 예산안 타입 데이터
		if r.FormValue(fmt.Sprintf("type%d-bgtypeid", i)) == "" { // 예산안 ID가 존재하지 않는다면 -> 새로 생긴 탭
			bgtd.ID = primitive.NewObjectID()
			bgtd.TeamSetting = bgts
		} else { // 예산안 ID 정보가 있다면 기존의 예산안 정보를 가져오기
			for _, value := range origTypeData {
				if r.FormValue(fmt.Sprintf("type%d-bgtypeid", i)) == value.ID.Hex() {
					bgtd = value
					break
				}
			}
		}

		// 예산안 정보 입력
		bgtd.ContractDate = r.FormValue(fmt.Sprintf("type%d-bgcontractdate", i)) // 예산안 계약일
		if r.FormValue(fmt.Sprintf("type%d-bgmaintypestatus", i)) != "" {
			status, err := strconv.ParseBool(r.FormValue(fmt.Sprintf("type%d-bgmaintypestatus", i)))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if status {
				bgp.MainType = bgtype // 예산 프로젝트 메인 타입
			}
		}
		if r.FormValue(fmt.Sprintf("type%d-bgproposal", i)) != "" { // 예산안 제안 견적
			proposal := r.FormValue(fmt.Sprintf("type%d-bgproposal", i))
			if strings.Contains(proposal, ",") {
				proposal = strings.ReplaceAll(proposal, ",", "")
			}
			bgtd.Proposal, err = encryptAES256Func(proposal)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if r.FormValue(fmt.Sprintf("type%d-bgdecision", i)) != "" { // 예산안 계약 결정액
			decision := r.FormValue(fmt.Sprintf("type%d-bgdecision", i))
			if strings.Contains(decision, ",") {
				decision = strings.ReplaceAll(decision, ",", "")
			}
			bgtd.Decision, err = encryptAES256Func(decision)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if r.FormValue(fmt.Sprintf("type%d-bgcontractcuts", i)) != "" { // 예산안 프로젝트 계약 컷수
			contractCuts := r.FormValue(fmt.Sprintf("type%d-bgcontractcuts", i))
			if strings.Contains(contractCuts, ",") {
				contractCuts = strings.ReplaceAll(contractCuts, ",", "")
			}
			bgtd.ContractCuts, err = strconv.Atoi(contractCuts)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if r.FormValue(fmt.Sprintf("type%d-bgworkingcuts", i)) != "" { // 예산안 프로젝트 작업 컷수
			workingCuts := r.FormValue(fmt.Sprintf("type%d-bgworkingcuts", i))
			if strings.Contains(workingCuts, ",") {
				workingCuts = strings.ReplaceAll(workingCuts, ",", "")
			}
			bgtd.WorkingCuts, err = strconv.Atoi(workingCuts)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		// 실예산안 계산 비율 정보
		if r.FormValue(fmt.Sprintf("type%d-bgretakeratio", i)) != "" {
			bgtd.RetakeRatio, err = strconv.ParseFloat(r.FormValue(fmt.Sprintf("type%d-bgretakeratio", i)), 64)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if r.FormValue(fmt.Sprintf("type%d-bgprogressratio", i)) != "" {
			bgtd.ProgressRatio, err = strconv.ParseFloat(r.FormValue(fmt.Sprintf("type%d-bgprogressratio", i)), 64)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if r.FormValue(fmt.Sprintf("type%d-bgvendorratio", i)) != "" {
			bgtd.VendorRatio, err = strconv.ParseFloat(r.FormValue(fmt.Sprintf("type%d-bgvendorratio", i)), 64)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		// 수퍼바이저, 프로덕션, 매니지먼트 정보 기입
		var controlUserIDList []string
		bgtd.Supervisors = nil
		supNum, err := strconv.Atoi(r.FormValue(fmt.Sprintf("type%d-bgsupervisornum", i)))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for j := 0; j < supNum; j++ { // 수퍼바이저 정보 기입
			if r.FormValue(fmt.Sprintf("type%d-supervisor%d", i, j)) == "" {
				continue
			}
			bgmng := BGManagement{}
			bgmng.UserID = r.FormValue(fmt.Sprintf("type%d-supervisor%d", i, j))
			if !checkStringInListFunc(bgmng.UserID, controlUserIDList) {
				controlUserIDList = append(controlUserIDList, bgmng.UserID)
			}
			bgmng.Work = r.FormValue(fmt.Sprintf("type%d-supervisor%d-bgmanagementwork", i, j))
			if r.FormValue(fmt.Sprintf("type%d-supervisor%d-bgmanagementperiod", i, j)) != "" {
				bgmng.Period, err = strconv.Atoi(r.FormValue(fmt.Sprintf("type%d-supervisor%d-bgmanagementperiod", i, j)))
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			if r.FormValue(fmt.Sprintf("type%d-supervisor%d-bgmanagementratio", i, j)) != "" {
				bgmng.Ratio, err = strconv.ParseFloat(r.FormValue(fmt.Sprintf("type%d-supervisor%d-bgmanagementratio", i, j)), 64)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			bgtd.Supervisors = append(bgtd.Supervisors, bgmng)
		}

		// 프로덕션 정보 기입
		bgtd.Production = nil
		prodNum, err := strconv.Atoi(r.FormValue(fmt.Sprintf("type%d-bgproductionnum", i)))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for j := 0; j < prodNum; j++ {
			if r.FormValue(fmt.Sprintf("type%d-production%d", i, j)) == "" {
				continue
			}
			bgmng := BGManagement{}
			bgmng.UserID = r.FormValue(fmt.Sprintf("type%d-production%d", i, j))
			if !checkStringInListFunc(bgmng.UserID, controlUserIDList) {
				controlUserIDList = append(controlUserIDList, bgmng.UserID)
			}
			bgmng.Work = r.FormValue(fmt.Sprintf("type%d-production%d-bgmanagementwork", i, j))
			if r.FormValue(fmt.Sprintf("type%d-production%d-bgmanagementperiod", i, j)) != "" {
				bgmng.Period, err = strconv.Atoi(r.FormValue(fmt.Sprintf("type%d-production%d-bgmanagementperiod", i, j)))
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			if r.FormValue(fmt.Sprintf("type%d-production%d-bgmanagementratio", i, j)) != "" {
				bgmng.Ratio, err = strconv.ParseFloat(r.FormValue(fmt.Sprintf("type%d-production%d-bgmanagementratio", i, j)), 64)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			bgtd.Production = append(bgtd.Production, bgmng)
		}

		// 매니지먼트 정보 기입
		bgtd.Management = nil
		mngNum, err := strconv.Atoi(r.FormValue(fmt.Sprintf("type%d-bgmanagementnum", i)))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for j := 0; j < mngNum; j++ {
			if r.FormValue(fmt.Sprintf("type%d-management%d", i, j)) == "" {
				continue
			}
			bgmng := BGManagement{}
			bgmng.UserID = r.FormValue(fmt.Sprintf("type%d-management%d", i, j))
			if !checkStringInListFunc(bgmng.UserID, controlUserIDList) {
				controlUserIDList = append(controlUserIDList, bgmng.UserID)
			}
			bgmng.Work = r.FormValue(fmt.Sprintf("type%d-management%d-bgmanagementwork", i, j))
			if r.FormValue(fmt.Sprintf("type%d-management%d-bgmanagementperiod", i, j)) != "" {
				bgmng.Period, err = strconv.Atoi(r.FormValue(fmt.Sprintf("type%d-management%d-bgmanagementperiod", i, j)))
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			if r.FormValue(fmt.Sprintf("type%d-management%d-bgmanagementratio", i, j)) != "" {
				bgmng.Ratio, err = strconv.ParseFloat(r.FormValue(fmt.Sprintf("type%d-management%d-bgmanagementratio", i, j)), 64)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			bgtd.Management = append(bgtd.Management, bgmng)
		}

		// 예산 팀세팅에서 Control의 Key에 따른 userID 정리하기
		userIDsByHead := make(map[string][]string)
		for key, value := range bgtd.TeamSetting.Controls {
			for _, control := range value {
				for _, part := range control.Parts {
					artists, err := getArtistByTeamsFunc(client, part.Teams)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					for _, artist := range artists {
						if !checkStringInListFunc(artist.ID, userIDsByHead[key]) {
							userIDsByHead[key] = append(userIDsByHead[key], artist.ID)
						}
					}
				}
			}
		}

		// 매니지먼트에 적힌 UserID를 head에 따라 정리한 ID와 비교하여 분류하기
		manageUserIDs := make(map[string][]string)
		for _, id := range controlUserIDList {
			for head, idList := range userIDsByHead {
				if checkStringInListFunc(id, idList) {
					if !checkStringInListFunc(id, manageUserIDs[head]) {
						manageUserIDs[head] = append(manageUserIDs[head], id)
					}
				}
			}
		}

		// 정리된 Head 별 매니지먼트 ID에 따른 비용 계산
		bglsList := []BGLaborCost{}
		for head, idList := range manageUserIDs {
			managementCost := 0
			for _, sup := range bgtd.Supervisors { // 수퍼바이저 비용 계산
				if !checkStringInListFunc(sup.UserID, idList) {
					continue
				}
				artist, err := getArtistFunc(client, sup.UserID)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				salary := artist.Salary[strconv.Itoa(time.Now().Year())] // 아티스트의 올해 연봉 정보

				// 아티스트의 동일 연도 연봉 변경 정보가 있는지 체크
				if artist.Changed {
					// 오늘 날짜와 동일 연도 연봉 변경일 비교
					thisDate := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC)
					for key, value := range artist.ChangedSalary {
						changedDate, err := time.Parse("2006-01-02", key)
						if err != nil {
							http.Error(w, err.Error(), http.StatusInternalServerError)
							return
						}
						if thisDate.Before(changedDate) { // 변경 전 연봉으로 계산
							salary = value
						}
					}
				}

				// 연봉 정보가 있다면 계산
				if salary != "" {
					decryptSalary, err := decryptAES256Func(salary)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					if decryptSalary != "" { // 복호화된 금액 정보가 있다면
						intSalary, err := strconv.Atoi(decryptSalary)
						if err != nil {
							http.Error(w, err.Error(), http.StatusInternalServerError)
							return
						}
						supCost := int(math.Round((float64(intSalary) * 10000 / 12) * (sup.Ratio / 100) * float64(sup.Period))) // 30일 기준 월급 * 비율 * 기간
						managementCost += supCost
					}
				}
			}

			for _, prod := range bgtd.Production { // 프로덕션 비용 계산
				if !checkStringInListFunc(prod.UserID, idList) {
					continue
				}
				artist, err := getArtistFunc(client, prod.UserID)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				salary := artist.Salary[strconv.Itoa(time.Now().Year())] // 아티스트의 올해 연봉 정보

				// 아티스트의 동일 연도 연봉 변경 정보가 있는지 체크
				if artist.Changed {
					// 오늘 날짜와 동일 연도 연봉 변경일 비교
					thisDate := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC)
					for key, value := range artist.ChangedSalary {
						changedDate, err := time.Parse("2006-01-02", key)
						if err != nil {
							http.Error(w, err.Error(), http.StatusInternalServerError)
							return
						}
						if thisDate.Before(changedDate) { // 변경 전 연봉으로 계산
							salary = value
						}
					}
				}

				// 연봉 정보가 있다면 계산
				if salary != "" {
					decryptSalary, err := decryptAES256Func(salary)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					if decryptSalary != "" { // 복호화된 금액 정보가 있다면
						intSalary, err := strconv.Atoi(decryptSalary)
						if err != nil {
							http.Error(w, err.Error(), http.StatusInternalServerError)
							return
						}
						prodCost := int(math.Round((float64(intSalary) * 10000 / 12) * (prod.Ratio / 100) * float64(prod.Period))) // 30일 기준 월급 * 비율 * 기간
						managementCost += prodCost
					}
				}
			}

			for _, mng := range bgtd.Management { // 매니지먼트 비용 계산
				if !checkStringInListFunc(mng.UserID, idList) {
					continue
				}
				artist, err := getArtistFunc(client, mng.UserID)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				salary := artist.Salary[strconv.Itoa(time.Now().Year())] // 아티스트의 올해 연봉 정보

				// 아티스트의 동일 연도 연봉 변경 정보가 있는지 체크
				if artist.Changed {
					// 오늘 날짜와 동일 연도 연봉 변경일 비교
					thisDate := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC)
					for key, value := range artist.ChangedSalary {
						changedDate, err := time.Parse("2006-01-02", key)
						if err != nil {
							http.Error(w, err.Error(), http.StatusInternalServerError)
							return
						}
						if thisDate.Before(changedDate) { // 변경 전 연봉으로 계산
							salary = value
						}
					}
				}

				// 연봉 정보가 있다면 계산
				if salary != "" {
					decryptSalary, err := decryptAES256Func(salary)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					if decryptSalary != "" { // 복호화된 금액 정보가 있다면
						intSalary, err := strconv.Atoi(decryptSalary)
						if err != nil {
							http.Error(w, err.Error(), http.StatusInternalServerError)
							return
						}
						mngCost := int(math.Round((float64(intSalary) * 10000 / 12) * (mng.Ratio / 100) * float64(mng.Period))) // 30일 기준 월급 * 비율 * 기간
						managementCost += mngCost
					}
				}
			}

			// 계산된 매니지먼트 비용 암호화하여 저장
			encryptedCost, err := encryptAES256Func(strconv.Itoa(managementCost))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// 기존의 있던 예산안인지 아닌지 비교 후에 매니지먼트 비용 계산 필요
			bgls := BGLaborCost{}
			bgls.Headquarter = head
			if bgtd.LaborCosts != nil { // 기존 예산안이 있는 경우
				for _, ls := range bgtd.LaborCosts {
					if ls.Headquarter == head { // 기존에 저장된 본부의 비용의 경우
						bgls = ls
						break
					}
				}
			}
			bgls.Management = encryptedCost
			bglsList = append(bglsList, bgls)
		}
		bgtd.LaborCosts = bglsList

		// 예산 프로젝트 정보에 예산안 데이터 저장
		bgp.TypeData[bgtype] = bgtd
	}

	bgp.UpdatedTime = time.Now().Format(time.RFC3339) // 프로젝트의 마지막 업데이트된 시간을 현재 시간으로 설정

	err = setBGProjectFunc(client, bgp, originalID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log := Log{
		UserID:    token.ID,
		CreatedAt: time.Now(),
		Content:   fmt.Sprintf("예산 프로젝트 %s의 정보가 수정되었습니다.", originalID),
	}

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/editbgproject-success?id=%s&date=%s", bgp.ID, searchedDate), http.StatusSeeOther)
}

// handleEditBGProjectSuccessFunc 함수는 예산 프로젝트 정보 수정을 성공했다는 페이지를 연다.
func handleEditBGProjectSuccessFunc(w http.ResponseWriter, r *http.Request) {
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
	err = TEMPLATES.ExecuteTemplate(w, "editbgproject-success", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// genBGProjectsExcelFunc 함수는 예산 프로젝트 데이터를 엑셀 파일로 생성하는 함수이다.
func genBGProjectsExcelFunc(date string, bgprojects []BGProject, userID string) error {
	path := os.TempDir() + "/budget/" + userID + "/bgproject/"
	excelFileName := fmt.Sprintf("bgproject_%s.xlsx", strings.ReplaceAll(date, "-", "_"))

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
	mainStyle, err := f.NewStyle(
		`
		{"alignment":{"horizontal":"center","vertical":"center","wrap_text":true},
		"font":{"bold":true},
		"fill":{"type":"pattern","color":["#959595"],"pattern":1}}
		`)
	if err != nil {
		return err
	}
	mainNumStyle, err := f.NewStyle(
		`
		{"alignment":{"horizontal":"right","vertical":"center","wrap_text":true},
		"number_format": 3,
		"font":{"bold":true},
		"fill":{"type":"pattern","color":["#959595"],"pattern":1}}
		`)

	// 제목 입력
	f.SetCellValue(sheet, "A1", "Status")
	f.MergeCell(sheet, "A1", "A2")
	f.SetCellValue(sheet, "B1", "ID")
	f.MergeCell(sheet, "B1", "B2")
	f.SetCellValue(sheet, "C1", "이름")
	f.MergeCell(sheet, "C1", "C2")
	f.SetCellValue(sheet, "D1", "작업 예상 기간")
	f.MergeCell(sheet, "D1", "D2")
	f.SetCellValue(sheet, "E1", "총 매출")
	f.MergeCell(sheet, "E1", "E2")
	f.SetCellValue(sheet, "F1", "계약일")
	f.MergeCell(sheet, "F1", "F2")
	f.SetCellValue(sheet, "G1", "컷수 정보")
	f.MergeCell(sheet, "G1", "H1")
	f.SetCellValue(sheet, "G2", "계약 컷수")
	f.SetCellValue(sheet, "H2", "작업 컷수")
	f.SetCellValue(sheet, "I1", "예산안 타입")
	f.MergeCell(sheet, "I1", "I2")

	f.SetColWidth(sheet, "A", "I", 20)
	f.SetColWidth(sheet, "A", "B", 12)
	f.SetColWidth(sheet, "C", "D", 30)
	f.SetColWidth(sheet, "G", "H", 15)

	f.SetRowHeight(sheet, 1, 25)
	f.SetRowHeight(sheet, 2, 25)

	// 데이터 입력
	pos := ""
	mpos := ""
	ppos := ""                          // 총 매출 숫자 스타일을 위한 Position
	mainType := make(map[string]string) // MainType 스타일 지정을 위한 Map
	i := 0
	for _, bgp := range bgprojects {
		typLen := len(bgp.TypeList)
		// 프로젝트 Status
		pos, err = excelize.CoordinatesToCellName(1, i+3) // ex) pos = "A3"
		if err != nil {
			return err
		}
		if bgp.Status {
			f.SetCellValue(sheet, pos, "계약 완료")
		} else {
			f.SetCellValue(sheet, pos, "사전 검토")
		}
		mpos, err = excelize.CoordinatesToCellName(1, i+3+typLen-1)
		if err != nil {
			return err
		}
		f.MergeCell(sheet, pos, mpos)

		// 프로젝트 ID
		pos, err = excelize.CoordinatesToCellName(2, i+3) // ex) pos = "B3"
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, bgp.ID)
		mpos, err = excelize.CoordinatesToCellName(2, i+3+typLen-1)
		if err != nil {
			return err
		}
		f.MergeCell(sheet, pos, mpos)

		// 프로젝트 이름
		pos, err = excelize.CoordinatesToCellName(3, i+3) // ex) pos = "C3"
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, bgp.Name)
		mpos, err = excelize.CoordinatesToCellName(3, i+3+typLen-1)
		if err != nil {
			return err
		}
		f.MergeCell(sheet, pos, mpos)

		// 프로젝트 작업 예상 기간
		pos, err = excelize.CoordinatesToCellName(4, i+3) // ex) pos = "D3"
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, fmt.Sprintf("%s ~ %s", stringToDateFunc(bgp.StartDate), stringToDateFunc(bgp.EndDate)))
		mpos, err = excelize.CoordinatesToCellName(4, i+3+typLen-1)
		if err != nil {
			return err
		}
		f.MergeCell(sheet, pos, mpos)

		// 프로젝트 예산안 정보
		for j, typeName := range bgp.TypeList {
			// 프로젝트 예산안 총 매출
			paymentInt := 0
			ppos, err = excelize.CoordinatesToCellName(5, i+3+j) // ex) ppos = "E3"
			if err != nil {
				return err
			}
			// 계약 결정액이 있는지 확인
			if bgp.TypeData[typeName].Decision == "" {
				proposal, err := decryptAES256Func(bgp.TypeData[typeName].Proposal)
				if err != nil {
					return err
				}
				if proposal != "" {
					paymentInt, err = strconv.Atoi(proposal)
					if err != nil {
						return err
					}
				}
			} else {
				decision, err := decryptAES256Func(bgp.TypeData[typeName].Decision)
				if err != nil {
					return err
				}
				if decision != "" {
					paymentInt, err = strconv.Atoi(decision)
					if err != nil {
						return err
					}
				}
			}
			f.SetCellValue(sheet, ppos, paymentInt)

			// 프로젝트 계약일
			pos, err = excelize.CoordinatesToCellName(6, i+3+j) // ex) pos = "F3"
			if err != nil {
				return err
			}
			f.SetCellValue(sheet, pos, stringToDateFunc(bgp.TypeData[typeName].ContractDate))

			// 프로젝트 컷수 정보
			pos, err = excelize.CoordinatesToCellName(7, i+3+j) // ex) pos = "G3"
			if err != nil {
				return err
			}
			f.SetCellValue(sheet, pos, bgp.TypeData[typeName].ContractCuts)

			pos, err = excelize.CoordinatesToCellName(8, i+3+j) // ex) pos = "H3"
			if err != nil {
				return err
			}
			f.SetCellValue(sheet, pos, bgp.TypeData[typeName].WorkingCuts)

			// 프로젝트 예산안 타입
			pos, err = excelize.CoordinatesToCellName(9, i+3+j)
			if err != nil {
				return err
			}
			f.SetCellValue(sheet, pos, typeName)

			// 현재 예산안이 MainType인지 확인
			if typeName == bgp.MainType {
				mainType[ppos] = pos
			}

			f.SetRowHeight(sheet, i+3+j, 20)
		}
		i += typLen
	}
	f.SetCellStyle(sheet, "A1", pos, style)
	f.SetCellStyle(sheet, "E3", ppos, numberStyle)

	// Main Type 스타일 지정
	for start, end := range mainType {
		f.SetCellStyle(sheet, start, end, mainStyle)
		f.SetCellStyle(sheet, start, start, mainNumStyle)
	}

	// 엑셀 파일 저장
	err = f.SaveAs(path + "/" + excelFileName)
	if err != nil {
		return err
	}

	return nil
}

// handleExportBGProjectsFunc 함수는 함수는 임시 폴더에 저장된 엑셀 파일을 다운로드하는 함수이다.
func handleExportBGProjectsFunc(w http.ResponseWriter, r *http.Request) {
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

	path := os.TempDir() + "/budget/" + token.ID + "/bgproject"

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
		Content:   fmt.Sprintf("예산 프로젝트 관리 페이지에서 %s년 %s월의 데이터를 다운로드하였습니다.", filename[1], filename[2]),
	}

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Disposition", fmt.Sprintf("Attachment; filename=%s", fileInfo[0].Name()))
	http.ServeFile(w, r, path+"/"+fileInfo[0].Name())
}

// handleBGProjectTSFunc 함수는 예산 프로젝트 관리 페이지에서 예산안 별 팀세팅을 설정하는 페이지로 이동한다.
func handleBGProjectTSFunc(w http.ResponseWriter, r *http.Request) {
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

	q := r.URL.Query()
	id := q.Get("id")
	bgtype := q.Get("bgtype")
	date := q.Get("date")

	bgp, err := getBGProjectFunc(client, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	bgTypeData := bgp.TypeData[bgtype]

	type Recipe struct {
		Token        Token         // 토큰
		User         User          // 유저 정보
		TeamSetting  BGTeamSetting // 예산 팀 세팅
		ControlTeams []string      // admin setting에 설정한 수퍼바이저, 프로덕션, 매니지먼트 팀 리스트
		Tasks        []string      // 태스크 목록
		VFXTeams     []string      // admmin setting에 설정한 VFX 팀 목록
		CMTeams      []string      // admin setting에 설정한 CM 팀 목록

		ID     string // 예산 프로젝트 ID
		BGType string // 예산 프로젝트 예산안 타입
		Date   string // 예산 프로젝트 관리 페이지 날짜
	}
	rcp := Recipe{}
	rcp.Token = token
	rcp.User, err = getUserFunc(client, token.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rcp.TeamSetting = bgTypeData.TeamSetting
	adminSetting, err := getAdminSettingFunc(client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rcp.ControlTeams = adminSetting.BGSupervisorTeams
	rcp.ControlTeams = append(rcp.ControlTeams, adminSetting.BGProductionTeams...)
	rcp.ControlTeams = append(rcp.ControlTeams, adminSetting.BGManagementTeams...)

	// 태스크별 팀 설정을 위한 태스크 목록
	for _, dept := range rcp.TeamSetting.Departments {
		for _, d := range dept {
			for _, p := range d.Parts {
				for _, t := range p.Tasks {
					if !checkStringInListFunc(t, rcp.Tasks) {
						rcp.Tasks = append(rcp.Tasks, t)
					}
				}
			}
		}
	}

	// 태스크별 팀 설정을 위한 팀 목록
	for _, teams := range adminSetting.VFXTeams {
		rcp.VFXTeams = append(rcp.VFXTeams, teams...)
	}
	rcp.CMTeams = adminSetting.CMTeams

	// 예산 프로젝트 관련 정보
	rcp.ID = id
	rcp.BGType = bgtype
	rcp.Date = date

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "bgproject-teamsetting", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleBGProjectTSSubmitFunc 함수는 예산 프로젝트의 예산안별 팀세팅 페이지에서 Update 버튼을 클릭했을 때 실행되는 함수이다.
func handleBGProjectTSSubmitFunc(w http.ResponseWriter, r *http.Request) {
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

	q := r.URL.Query()
	id := q.Get("id")
	bgtype := q.Get("bgtype")
	date := q.Get("date")

	bgp, err := getBGProjectFunc(client, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	bgTypeData := bgp.TypeData[bgtype]

	ts := BGTeamSetting{}
	ts.Departments = make(map[string][]BGDept)
	ts.Controls = make(map[string][]BGControl)
	ts.Teams = make(map[string][]string)

	headNum, err := strconv.Atoi(r.FormValue("headnum"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 본부별 부서 및 태스크 설정
	for hIndex := 0; hIndex < headNum; hIndex++ {
		headName := r.FormValue(fmt.Sprintf("head%d", hIndex))
		if headName == "" {
			continue
		}

		ts.Headquarters = append(ts.Headquarters, headName)

		// 본부별 부서
		deptNum, err := strconv.Atoi(r.FormValue(fmt.Sprintf("head%d-deptnum", hIndex)))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for dIndex := 0; dIndex < deptNum; dIndex++ {
			deptName := r.FormValue(fmt.Sprintf("head%d-dept%d", hIndex, dIndex))
			if deptName == "" {
				continue
			}

			typ := false
			if r.FormValue(fmt.Sprintf("head%d-type%d", hIndex, dIndex)) == "on" {
				typ = true
			}

			var parts []BGPart
			partNum, err := strconv.Atoi(r.FormValue(fmt.Sprintf("head%d-dept%d-partnum", hIndex, dIndex)))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			for pIndex := 0; pIndex < partNum; pIndex++ {
				partName := r.FormValue(fmt.Sprintf("head%d-dept%d-part%d", hIndex, dIndex, pIndex))
				if partName == "" {
					continue
				}

				task := BGPart{
					Name:  partName,
					Tasks: stringToListFunc(r.FormValue(fmt.Sprintf("head%d-dept%d-task%d", hIndex, dIndex, pIndex)), " "),
				}
				parts = append(parts, task)
			}

			dept := BGDept{
				Name:  deptName,
				Parts: parts,
				Type:  typ,
			}

			ts.Departments[headName] = append(ts.Departments[headName], dept)
		}

		// 본부별 Control 부서
		controlNumStr := r.FormValue(fmt.Sprintf("head%d-controlnum", hIndex))
		if controlNumStr == "" {
			continue
		}

		controlNum, err := strconv.Atoi(controlNumStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for cIndex := 0; cIndex < controlNum; cIndex++ {
			controlName := r.FormValue(fmt.Sprintf("head%d-control%d", hIndex, cIndex))
			if controlName == "" {
				continue
			}

			var parts []BGControlPart
			teamNum, err := strconv.Atoi(r.FormValue(fmt.Sprintf("head%d-control%d-teamnum", hIndex, cIndex)))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			for tIndex := 0; tIndex < teamNum; tIndex++ {
				partName := r.FormValue(fmt.Sprintf("head%d-control%d-part%d", hIndex, cIndex, tIndex))
				if partName == "" {
					continue
				}

				controlPart := BGControlPart{
					Name:  partName,
					Teams: r.Form[fmt.Sprintf("head%d-control%d-team%d", hIndex, cIndex, tIndex)],
				}
				parts = append(parts, controlPart)
			}

			control := BGControl{
				Name:  controlName,
				Parts: parts,
			}

			ts.Controls[headName] = append(ts.Controls[headName], control)
		}
	}

	// 태스크별 팀 설정
	taskNum, err := strconv.Atoi(r.FormValue("tasknum"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for tIndex := 0; tIndex < taskNum; tIndex++ {
		taskName := r.FormValue(fmt.Sprintf("task%d", tIndex))
		teamList := r.Form[fmt.Sprintf("team%d", tIndex)]
		ts.Teams[taskName] = teamList
	}

	bgTypeData.TeamSetting = ts
	bgTypeData.TeamSetting.UpdatedTime = time.Now().Format(time.RFC3339) // 팀세팅의 마지막 업데이트된 시간을 현재 시간으로 설정
	bgp.TypeData[bgtype] = bgTypeData

	err = setBGProjectFunc(client, bgp, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/bgproject-teamsetting-success?id=%s&bgtype=%s&date=%s", id, bgtype, date), http.StatusSeeOther)
}

// handleBGProjectTSSuccessFunc 함수는 예산 프로젝트의 예산안별 팀세팅 Update가 성공했다는 페이지를 연다.
func handleBGProjectTSSuccessFunc(w http.ResponseWriter, r *http.Request) {
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
		Token  Token
		ID     string
		BGType string
		Date   string
	}
	rcp := Recipe{}
	rcp.Token = token

	q := r.URL.Query()
	rcp.ID = q.Get("id")
	rcp.BGType = q.Get("bgtype")
	rcp.Date = q.Get("date")

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "bgproject-teamsetting-success", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
