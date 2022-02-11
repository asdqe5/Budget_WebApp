// 프로젝트 결산 프로그램
//
// Description : http admin setting 관련 스크립트

package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// handleAdminSettingFunc 함수는 AdminSetting 페이지를 여는 함수이다.
func handleAdminSettingFunc(w http.ResponseWriter, r *http.Request) {
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
		User                    User
		AdminSetting            AdminSetting
		BeforeLastMonthlyStatus MonthlyStatus           // 지지난달의 결산 상태
		LastMonthlyStatus       MonthlyStatus           // 지난달의 결산 상태
		CurMonthlyStatus        MonthlyStatus           // 이번달의 결산 상태
		LastFTStatus            []FinishedTimelogStatus // 전달의 끝난 프로젝트의 타임로그 처리 상태
		CurFTStatus             []FinishedTimelogStatus // 이번달의 끝난 프로젝트의 타임로그 처리 상태
	}
	rcp := Recipe{}
	rcp.Token = token
	rcp.User, err = getUserFunc(client, token.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rcp.AdminSetting, err = getAdminSettingFunc(client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 지지난 달의 결산 상태
	beforeLastYear, beforeLastMonth, _ := time.Now().AddDate(0, -2, -time.Now().Day()+1).Date()
	rcp.BeforeLastMonthlyStatus, err = getMonthlyStatusFunc(client, fmt.Sprintf("%04d-%02d", beforeLastYear, beforeLastMonth))
	if err != nil {
		if err == mongo.ErrNoDocuments {
			rcp.BeforeLastMonthlyStatus = MonthlyStatus{
				Date:   fmt.Sprintf("%04d-%02d", beforeLastYear, beforeLastMonth),
				Status: false,
			}
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// 지난 달의 결산 상태
	lastYear, lastMonth, _ := time.Now().AddDate(0, -1, -time.Now().Day()+1).Date()
	rcp.LastMonthlyStatus, err = getMonthlyStatusFunc(client, fmt.Sprintf("%04d-%02d", lastYear, lastMonth))
	if err != nil {
		if err == mongo.ErrNoDocuments {
			rcp.LastMonthlyStatus = MonthlyStatus{
				Date:   fmt.Sprintf("%04d-%02d", lastYear, lastMonth),
				Status: false,
			}
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// 이번 달의 결산 상태
	year, month, _ := time.Now().Date()
	rcp.CurMonthlyStatus, err = getMonthlyStatusFunc(client, fmt.Sprintf("%04d-%02d", year, month))
	if err != nil {
		if err == mongo.ErrNoDocuments {
			rcp.CurMonthlyStatus = MonthlyStatus{
				Date:   fmt.Sprintf("%04d-%02d", year, month),
				Status: false,
			}
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// 지난 달의 끝난 프로젝트 타임로그 처리 상태
	rcp.LastFTStatus, err = getFTStatusByMonth(client, lastYear, int(lastMonth))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 이번 달의 끝난 프로젝트 타임로그 처리 상태
	rcp.CurFTStatus, err = getFTStatusByMonth(client, year, int(month))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "adminsetting", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleAdminSettingSubmitFunc 함수는 AdminSetting 페이지에서 Update 버튼을 눌렀을 때 실행되는 함수이다.
func handleAdminSettingSubmitFunc(w http.ResponseWriter, r *http.Request) {
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

	a, err := getAdminSettingFunc(client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	a.VFXDepts = stringToListFunc(r.FormValue("vfxdepts"), " ")
	vfxTeams, err := sgGetTeamMapFunc(a.VFXDepts) // 팀 태그 리스트를 통해서 map 형식의 팀리스트를 얻는다.
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	a.VFXTeams = vfxTeams
	a.CMTeams = stringToListFunc(r.FormValue("cmteams"), " ")
	a.SGExcludeID = stringToListFunc(r.FormValue("sgexcludeid"), " ")
	a.SGExcludeProjects = stringToListFunc(r.FormValue("sgexcludeprojects"), " ")
	projectStatusNum, err := strconv.Atoi(r.FormValue("projectStatusNum"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.ProjectStatus = nil
	for i := 0; i < projectStatusNum; i++ {
		var status Status
		statusID := r.FormValue(fmt.Sprintf("statusid%d", i))
		if statusID == "" {
			continue
		}
		status.ID = statusID
		status.TextColor = "#" + r.FormValue(fmt.Sprintf("textcolor%d", i))
		status.BGColor = "#" + r.FormValue(fmt.Sprintf("bgcolor%d", i))
		err = status.CheckErrorFunc()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		a.ProjectStatus = append(a.ProjectStatus, status)
	}
	a.RNDProjects = stringToListFunc(r.FormValue("rndprojects"), " ")
	a.ETCProjects = stringToListFunc(r.FormValue("etcprojects"), " ")
	a.TaskProjects = stringToListFunc(r.FormValue("taskprojects"), " ")
	a.SMSupervisorIDs = stringToListFunc(r.FormValue("smsupervisorids"), " ")
	a.GWIDsForProject = stringToListFunc(r.FormValue("gwidsforproject"), " ")
	a.GWIDs = stringToListFunc(r.FormValue("gwids"), " ")

	// 예산 관련 수퍼바이저 / 프로덕션 / 매니지먼트 팀 설정
	a.BGSupervisorTeams = r.Form["bgsupervisorteams"] // 예산 관련 슈퍼바이저 팀
	a.BGProductionTeams = r.Form["bgproductionteams"] // 예산 관련 프로덕션 팀
	a.BGManagementTeams = r.Form["bgmanagementteams"] // 예산 관련 매니지먼트 팀

	err = a.CheckErrorFunc()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = updateAdminSettingFunc(client, a)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 지지난달의 결산 상태 저장
	beforeLastYear, beforeLastMonth, _ := time.Now().AddDate(0, -2, -time.Now().Day()+1).Date()
	beforeLastStatus, err := strconv.ParseBool(r.FormValue("beforeLastMonthlyStatus1"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	beforeLastMonthlyStatus := MonthlyStatus{
		Date:   fmt.Sprintf("%04d-%02d", beforeLastYear, beforeLastMonth),
		Status: beforeLastStatus,
	}
	err = beforeLastMonthlyStatus.CheckErrorFunc()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = setMonthlyStatusFunc(client, beforeLastMonthlyStatus)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 지난달의 결산 상태 저장
	lastYear, lastMonth, _ := time.Now().AddDate(0, -1, -time.Now().Day()+1).Date()
	lastStatus, err := strconv.ParseBool(r.FormValue("lastMonthlyStatus1"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	lastMonthlyStatus := MonthlyStatus{
		Date:   fmt.Sprintf("%04d-%02d", lastYear, lastMonth),
		Status: lastStatus,
	}
	err = lastMonthlyStatus.CheckErrorFunc()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = setMonthlyStatusFunc(client, lastMonthlyStatus)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 이번달의 결산 상태 저장
	year, month, _ := time.Now().Date()
	curStatus, err := strconv.ParseBool(r.FormValue("curMonthlyStatus1"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	curMonthlyStatus := MonthlyStatus{
		Date:   fmt.Sprintf("%04d-%02d", year, month),
		Status: curStatus,
	}
	err = curMonthlyStatus.CheckErrorFunc()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = setMonthlyStatusFunc(client, curMonthlyStatus)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log := Log{
		UserID:    token.ID,
		CreatedAt: time.Now(),
		Content:   "AdminSetting이 수정되었습니다.",
	}

	// 끝난 프로젝트의 결산 처리 상태 확인 - 이번달
	recalculate := false
	statusMap := make(map[string]bool)
	etcTimelogInfo := make(map[string]map[string]float64)
	curFTStatusNum, err := strconv.Atoi(r.FormValue("curFTStatusNum"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for i := 0; i < curFTStatusNum; i++ {
		// 상태가 바뀐지 안바뀐지 체크한다.
		curFTStatus := strings.Split(r.FormValue(fmt.Sprintf("curFTStatus%d", i)), "-")
		if curFTStatus[1] == curFTStatus[2] {
			continue
		}
		recalculate = true
		projectName := curFTStatus[0]

		// 프로젝트로 처리 -> ETC로 처리인 경우
		if curFTStatus[2] == "true" {
			// 상태 맵의 해당 프로젝트를 true로 저장한다.
			statusMap[projectName] = true

			// 해당 프로젝트의 타임로그를 가져온다.
			timelogs, err := getTimelogOfTheProjectVFXFunc(client, year, int(month), projectName)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// 프로젝트로 저장된 타임로그를 가져와 ETC 프로젝트에 duration을 합쳐주고, 해당 타임로그는 삭제한다.
			etcProjectName := fmt.Sprintf("ETC%04d", year)
			for _, t := range timelogs {
				etcTimelog, err := getTimelogFunc(client, t.UserID, year, int(month), etcProjectName)
				if err != nil {
					if err == mongo.ErrNoDocuments {
						etcTimelog = Timelog{
							UserID:   t.UserID,
							Year:     t.Year,
							Month:    t.Month,
							Project:  etcProjectName,
							Duration: 0,
						}
					} else {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}
				etcTimelog.Duration = etcTimelog.Duration + t.Duration
				err = addTimelogFunc(client, etcTimelog)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				err = rmTimelogFunc(client, t)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if _, exists := etcTimelogInfo[projectName]; !exists {
					etcTimelogInfo[projectName] = make(map[string]float64)
				}
				etcTimelogInfo[projectName][t.UserID] = t.Duration
			}

			// 프로젝트의 CM을 제외한 이번달 인건비를 초기화한다.
			project, err := getProjectFunc(client, projectName) // DB에서 해당 프로젝트를 가져온다.
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			laborCost := project.SMMonthlyLaborCost[curMonthlyStatus.Date]
			laborCost.VFX = ""
			laborCost.RND = ""
			project.SMMonthlyLaborCost[curMonthlyStatus.Date] = laborCost
			err = setProjectFunc(client, project)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else { // ETC로 처리 -> 프로젝트 처리인 경우
			// 상태 맵의 해당 프로젝트를 false로 저장한다.
			statusMap[projectName] = false

			// 정산 완료된 프로젝트의 처리 상태의 ETC로 저장된 타임로그 정보를 가져온다.
			fts, err := getFinishedTimelogStatusFunc(client, year, int(month), projectName)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// ETC로 저장된 타임로그를 빼고 해당 프로젝트로 타임로그를 저장한다.
			etcProjectName := fmt.Sprintf("ETC%04d", year)
			for userID, duration := range fts.TimelogInfo {
				etcTimelog, err := getTimelogFunc(client, userID, year, int(month), etcProjectName)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				etcTimelog.Duration = etcTimelog.Duration - duration
				err = addTimelogFunc(client, etcTimelog)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				timelog := Timelog{
					UserID:   userID,
					Year:     year,
					Month:    int(month),
					Project:  projectName,
					Duration: duration,
				}
				err = addTimelogFunc(client, timelog)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}

			// ETC로 저장된 타임로그의 정보를 비워준다.
			etcTimelogInfo[projectName] = nil
		}
	}

	if recalculate {
		// 이 달에 진행한 프로젝트 리스트를 가져온다.
		var nowProjectList []string
		timelogs, err := getTimelogOfTheMonthVFXFunc(client, year, int(month)) // 이번 달의 타임로그 데이터를 가져온다.
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, timelog := range timelogs {
			if !checkStringInListFunc(timelog.Project, nowProjectList) {
				nowProjectList = append(nowProjectList, timelog.Project)
			}
		}

		for _, np := range nowProjectList { // 이번달 인건비 계산
			project, err := getProjectFunc(client, np) // DB에서 해당 프로젝트를 가져온다.
			if err != nil {
				if err == mongo.ErrNoDocuments { // DB에 해당 프로젝트가 존재하지 않는 경우
					continue
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			monthlyLaborCost := make(map[string]LaborCost)
			if project.SMMonthlyLaborCost != nil {
				monthlyLaborCost = project.SMMonthlyLaborCost
			}
			laborCost := LaborCost{}

			// VFX 인건비 계산
			vfxLaborCost, err := calMonthlyVFXLaborCostFunc(np, year, int(month))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			laborCost.VFX, err = encryptAES256Func(strconv.Itoa(vfxLaborCost))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// CM 인건비 계산 -> VFX 타임로그와 상관없이 변하면 안되기 때문에 저장된 값을 가져온다.
			laborCost.CM = project.SMMonthlyLaborCost[curMonthlyStatus.Date].CM

			monthlyLaborCost[curMonthlyStatus.Date] = laborCost
			project.SMMonthlyLaborCost = monthlyLaborCost
			err = setProjectFunc(client, project)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		log.Content = log.Content + "이번 달의 정산완료된 프로젝트의 타임로그 처리가 변경되었습니다."
	}

	// DB에 프로젝트의 ETC 처리 설정값을 저장한다.
	for projectName := range statusMap {
		fts := FinishedTimelogStatus{
			Year:        year,
			Month:       int(month),
			Project:     projectName,
			Status:      statusMap[projectName],
			TimelogInfo: etcTimelogInfo[projectName],
		}
		err = updateFinishedTimelogStatusFunc(client, fts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// 끝난 프로젝트의 결산 처리 상태 확인 - 지난달
	recalculate = false
	statusMap = make(map[string]bool)
	etcTimelogInfo = make(map[string]map[string]float64)
	lastFTStatusNum, err := strconv.Atoi(r.FormValue("lastFTStatusNum"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for i := 0; i < lastFTStatusNum; i++ {
		// 상태가 바뀐지 안바뀐지 체크한다.
		lastFTStatus := strings.Split(r.FormValue(fmt.Sprintf("lastFTStatus%d", i)), "-")
		if lastFTStatus[1] == lastFTStatus[2] {
			continue
		}
		recalculate = true
		projectName := lastFTStatus[0]

		// 프로젝트로 처리 -> ETC로 처리인 경우
		if lastFTStatus[2] == "true" {
			// 상태 맵의 해당 프로젝트를 true로 저장한다.
			statusMap[projectName] = true

			// 해당 프로젝트의 타임로그를 가져온다.
			timelogs, err := getTimelogOfTheProjectVFXFunc(client, lastYear, int(lastMonth), projectName)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// 프로젝트로 저장된 타임로그를 가져와 ETC 프로젝트에 duration을 합쳐주고, 해당 타임로그는 삭제한다.
			etcProjectName := fmt.Sprintf("ETC%04d", lastYear)
			for _, t := range timelogs {
				etcTimelog, err := getTimelogFunc(client, t.UserID, lastYear, int(lastMonth), etcProjectName)
				if err != nil {
					if err == mongo.ErrNoDocuments {
						etcTimelog = Timelog{
							UserID:   t.UserID,
							Year:     t.Year,
							Month:    t.Month,
							Project:  etcProjectName,
							Duration: 0,
						}
					} else {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}
				etcTimelog.Duration = etcTimelog.Duration + t.Duration
				err = addTimelogFunc(client, etcTimelog)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				err = rmTimelogFunc(client, t)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if _, exists := etcTimelogInfo[projectName]; !exists {
					etcTimelogInfo[projectName] = make(map[string]float64)
				}
				etcTimelogInfo[projectName][t.UserID] = t.Duration
			}

			// 프로젝트의 CM을 제외한 이번달 인건비를 초기화한다.
			project, err := getProjectFunc(client, projectName) // DB에서 해당 프로젝트를 가져온다.
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			laborCost := project.SMMonthlyLaborCost[lastMonthlyStatus.Date]
			laborCost.VFX = ""
			laborCost.RND = ""
			project.SMMonthlyLaborCost[lastMonthlyStatus.Date] = laborCost
			err = setProjectFunc(client, project)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else { // ETC로 처리 -> 프로젝트 처리인 경우
			// 상태 맵의 해당 프로젝트를 false로 저장한다.
			statusMap[projectName] = false

			// 정산 완료된 프로젝트의 처리 상태의 ETC로 저장된 타임로그 정보를 가져온다.
			fts, err := getFinishedTimelogStatusFunc(client, lastYear, int(lastMonth), projectName)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// ETC로 저장된 타임로그를 빼고 해당 프로젝트로 타임로그를 저장한다.
			etcProjectName := fmt.Sprintf("ETC%04d", lastYear)
			for userID, duration := range fts.TimelogInfo {

				etcTimelog, err := getTimelogFunc(client, userID, lastYear, int(lastMonth), etcProjectName)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				etcTimelog.Duration = etcTimelog.Duration - duration
				err = addTimelogFunc(client, etcTimelog)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				timelog := Timelog{
					UserID:   userID,
					Year:     lastYear,
					Month:    int(lastMonth),
					Project:  projectName,
					Duration: duration,
				}
				err = addTimelogFunc(client, timelog)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}

			// ETC로 저장된 타임로그의 정보를 비워준다.
			etcTimelogInfo[projectName] = nil
		}
	}

	if recalculate {
		// 지난달에 진행한 프로젝트 리스트를 가져온다.
		var lastProjectList []string
		timelogs, err := getTimelogOfTheMonthVFXFunc(client, lastYear, int(lastMonth)) // 지난 달의 타임로그 데이터를 가져온다.
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, timelog := range timelogs {
			if !checkStringInListFunc(timelog.Project, lastProjectList) {
				lastProjectList = append(lastProjectList, timelog.Project)
			}
		}

		for _, lp := range lastProjectList { // 지난달 인건비 계산
			project, err := getProjectFunc(client, lp) // DB에서 해당 프로젝트를 가져온다.
			if err != nil {
				if err == mongo.ErrNoDocuments { // DB에 해당 프로젝트가 존재하지 않는 경우
					continue
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			monthlyLaborCost := make(map[string]LaborCost)
			if project.SMMonthlyLaborCost != nil {
				monthlyLaborCost = project.SMMonthlyLaborCost
			}
			laborCost := LaborCost{}

			// VFX 인건비 계산
			vfxLaborCost, err := calMonthlyVFXLaborCostFunc(lp, lastYear, int(lastMonth))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			laborCost.VFX, err = encryptAES256Func(strconv.Itoa(vfxLaborCost))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// CM 인건비 계산 -> VFX 타임로그와 상관없이 변하면 안되기 때문에 저장된 값을 가져온다.
			laborCost.CM = project.SMMonthlyLaborCost[lastMonthlyStatus.Date].CM

			monthlyLaborCost[lastMonthlyStatus.Date] = laborCost
			project.SMMonthlyLaborCost = monthlyLaborCost
			err = setProjectFunc(client, project)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		log.Content = log.Content + "\n지난 달의 정산완료된 프로젝트의 타임로그 처리가 변경되었습니다."
	}

	// DB에 프로젝트의 ETC 처리 설정값을 저장한다.
	for projectName := range statusMap {
		fts := FinishedTimelogStatus{
			Year:        lastYear,
			Month:       int(lastMonth),
			Project:     projectName,
			Status:      statusMap[projectName],
			TimelogInfo: etcTimelogInfo[projectName],
		}
		err = updateFinishedTimelogStatusFunc(client, fts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/adminsetting-success", http.StatusSeeOther)
}

// handleAdminSettingSuccessFunc 함수는 AdminSetting Update가 성공했다는 페이지를 연다.
func handleAdminSettingSuccessFunc(w http.ResponseWriter, r *http.Request) {
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

	type Recipe struct {
		Token
	}
	rcp := Recipe{}
	rcp.Token = token

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "adminsetting-success", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
