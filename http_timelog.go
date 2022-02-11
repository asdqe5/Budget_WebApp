package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// handleFinishedTimelogFunc 함수는 finishedtimelog 페이지를 띄우는 함수이다.
func handleFinishedTimelogFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// Admin 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < AdminLevel {
		http.Redirect(w, r, "/invalidaccess", http.StatusSeeOther)
		return
	}

	q := r.URL.Query()
	status := q.Get("status")

	type Recipe struct {
		Token    Token
		Status   string
		NowDate  string
		LastDate string
	}
	rcp := Recipe{
		Token:  token,
		Status: status,
	}

	ny, nm, _ := time.Now().Date()
	rcp.NowDate = fmt.Sprintf("%04d-%02d", ny, nm)
	ld := time.Now().AddDate(0, -1, time.Now().Day()+1)
	rcp.LastDate = fmt.Sprintf("%04d-%02d", ld.Year(), ld.Month())

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "finishedtimelog", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleFinishedTimelogSubmitFunc 함수는 finishedtimelog 페이지에서 Confirm 버튼을 클릭했을 때 실행되는 함수이다.
func handleFinishedTimelogSubmitFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// Admin 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < AdminLevel {
		http.Redirect(w, r, "/invalidaccess", http.StatusSeeOther)
		return
	}

	q := r.URL.Query()
	status := q.Get("status")

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

	// 이번달의 타임로그를 처리한다.
	ny, nm, _ := time.Now().Date()
	nowDate := fmt.Sprintf("%04d-%02d", ny, nm)

	timelogNum, err := strconv.Atoi(r.FormValue("timelognum"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	recalculate := false
	statusMap := make(map[string]bool)
	etcTimelogInfo := make(map[string]map[string]float64)
	for i := 0; i < timelogNum; i++ {
		projectName := r.FormValue(fmt.Sprintf("project%d", i))
		userid := r.FormValue(fmt.Sprintf("userid%d", i))

		if _, exists := statusMap[projectName]; !exists { // statusMap에 프로젝트 키값이 없으면 프로젝트의 ETC로 처리 여부를 추가한다.
			etcStatus := r.FormValue(fmt.Sprintf("etcstatus%d", i))
			if etcStatus == "on" {
				statusMap[projectName] = true
			} else {
				statusMap[projectName] = false
			}
		}
		if statusMap[projectName] == false { // 프로젝트로 처리한다면 타임로그를 수정할 필요없다.
			continue
		}

		recalculate = true
		// 프로젝트로 저장된 타임로그를 가져와 ETC 프로젝트에 duration을 합쳐주고, 해당 타임로그는 삭제한다.
		timelog, err := getTimelogFunc(client, userid, ny, int(nm), projectName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		etcTimelog, err := getTimelogFunc(client, userid, ny, int(nm), fmt.Sprintf("ETC%04d", ny))
		if err != nil {
			if err == mongo.ErrNoDocuments {
				etcTimelog = Timelog{
					UserID:   timelog.UserID,
					Year:     timelog.Year,
					Month:    timelog.Month,
					Project:  fmt.Sprintf("ETC%04d", ny),
					Duration: 0,
				}
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		etcTimelog.Duration = etcTimelog.Duration + timelog.Duration
		err = addTimelogFunc(client, etcTimelog)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = rmTimelogFunc(client, timelog)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if _, exists := etcTimelogInfo[projectName]; !exists {
			etcTimelogInfo[projectName] = make(map[string]float64)
		}
		etcTimelogInfo[projectName][userid] = timelog.Duration

		// 프로젝트의 CM을 제외한 이번달 인건비를 초기화한다.
		project, err := getProjectFunc(client, projectName) // DB에서 해당 프로젝트를 가져온다.
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		laborCost := project.SMMonthlyLaborCost[nowDate]
		laborCost.VFX = ""
		laborCost.RND = ""
		project.SMMonthlyLaborCost[nowDate] = laborCost
		err = setProjectFunc(client, project)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// ETC로 처리되어야 할 프로젝트가 있을 경우 인건비를 다시 계산한다.
	if recalculate {
		// 이 달에 진행한 프로젝트 리스트를 가져온다.
		var nowProjectList []string
		timelogs, err := getTimelogOfTheMonthVFXFunc(client, ny, int(nm)) // 검색한 달의 타임로그 데이터를 가져온다.
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
			vfxLaborCost, err := calMonthlyVFXLaborCostFunc(np, ny, int(nm))
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
			laborCost.CM = project.SMMonthlyLaborCost[nowDate].CM

			monthlyLaborCost[nowDate] = laborCost
			project.SMMonthlyLaborCost = monthlyLaborCost
			err = setProjectFunc(client, project)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	// DB에 프로젝트의 ETC 처리 설정값을 저장한다.
	for projectName := range statusMap {
		fts := FinishedTimelogStatus{
			Year:        ny,
			Month:       int(nm),
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

	if status == "false" { // 지난달의 타임로그도 업데이트했다면 ETC로 처리할 타임로그가 있는지 확인한다.
		// 지난달의 타임로그를 처리한다.
		ld := time.Now().AddDate(0, -1, -time.Now().Day()+1)
		lastDate := fmt.Sprintf("%04d-%02d", ld.Year(), ld.Month())

		timelogNum, err = strconv.Atoi(r.FormValue("lasttimelognum"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		recalculate = false
		statusMap = make(map[string]bool)
		etcTimelogInfo = make(map[string]map[string]float64)
		for i := 0; i < timelogNum; i++ {
			projectName := r.FormValue(fmt.Sprintf("lastproject%d", i))
			userid := r.FormValue(fmt.Sprintf("lastuserid%d", i))

			if _, exists := statusMap[projectName]; !exists { // statusMap에 프로젝트 키값이 없으면 프로젝트의 ETC로 처리 여부를 추가한다.
				etcStatus := r.FormValue(fmt.Sprintf("lastetcstatus%d", i))
				if etcStatus == "on" {
					statusMap[projectName] = true
				} else {
					statusMap[projectName] = false
				}
			}
			if statusMap[projectName] == false { // 프로젝트로 처리한다면 타임로그를 수정할 필요없다.
				continue
			}

			recalculate = true
			// 프로젝트로 저장된 타임로그를 가져와 ETC 프로젝트에 duration을 합쳐주고, 해당 타임로그는 삭제한다.
			timelog, err := getTimelogFunc(client, userid, ld.Year(), int(ld.Month()), projectName)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			etcTimelog, err := getTimelogFunc(client, userid, ld.Year(), int(ld.Month()), fmt.Sprintf("ETC%04d", ld.Year()))
			if err != nil {
				if err == mongo.ErrNoDocuments {
					etcTimelog = Timelog{
						UserID:   timelog.UserID,
						Year:     timelog.Year,
						Month:    timelog.Month,
						Project:  fmt.Sprintf("ETC%04d", ld.Year()),
						Duration: 0,
					}
				} else {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			etcTimelog.Duration = etcTimelog.Duration + timelog.Duration
			err = addTimelogFunc(client, etcTimelog)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			err = rmTimelogFunc(client, timelog)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if _, exists := etcTimelogInfo[projectName]; !exists {
				etcTimelogInfo[projectName] = make(map[string]float64)
			}
			etcTimelogInfo[projectName][userid] = timelog.Duration

			// 프로젝트의 CM을 제외한 지난달 인건비를 초기화한다.
			project, err := getProjectFunc(client, projectName) // DB에서 해당 프로젝트를 가져온다.
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			laborCost := project.SMMonthlyLaborCost[lastDate]
			laborCost.VFX = ""
			laborCost.RND = ""
			project.SMMonthlyLaborCost[lastDate] = laborCost
			err = setProjectFunc(client, project)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		// ETC로 처리되어야 할 프로젝트가 있을 경우 인건비를 다시 계산한다.
		if recalculate {
			// 지난달에 진행한 프로젝트 리스트를 가져온다.
			var lastProjectList []string
			timelogs, err := getTimelogOfTheMonthVFXFunc(client, ld.Year(), int(ld.Month())) // 검색한 달의 타임로그 데이터를 가져온다.
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
				vfxLaborCost, err := calMonthlyVFXLaborCostFunc(lp, ld.Year(), int(ld.Month()))
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
				laborCost.CM = project.SMMonthlyLaborCost[lastDate].CM

				monthlyLaborCost[lastDate] = laborCost
				project.SMMonthlyLaborCost = monthlyLaborCost
				err = setProjectFunc(client, project)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
		}

		// DB에 프로젝트의 ETC 처리 설정값을 저장한다.
		for projectName := range statusMap {
			fts := FinishedTimelogStatus{
				Year:        ld.Year(),
				Month:       int(ld.Month()),
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
	}

	type Recipe struct {
		Token Token
	}
	rcp := Recipe{
		Token: token,
	}

	log := Log{}
	log.UserID = token.ID
	log.CreatedAt = time.Now()
	log.Content = "정산완료된 프로젝트의 타임로그 처리가 완료되었습니다."

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "finishedtimelog-success", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
