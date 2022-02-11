// 프로젝트 결산 프로그램
//
// Description : 타임로그 관련 rest API를 작성한 스크립트

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// handleAPICheckMonthlyStatusFunc 함수는 월별 결산 상태를 확인하여 restapi로 보내주는 함수이다.
func handleAPICheckMonthlyStatusFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Post method only", http.StatusMethodNotAllowed)
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

	// AccessLevel 확인
	accesslevel, err := getAccessLevelFromHeaderFunc(r, client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if accesslevel < DefaultLevel {
		http.Error(w, "권한이 없는 계정입니다", http.StatusUnauthorized)
		return
	}

	y, m, _ := time.Now().Date()
	ld := time.Now().AddDate(0, -1, 0)
	thisdate := fmt.Sprintf("%04d-%02d", y, m)
	thisMonsthStatus, err := getMonthlyStatusFunc(client, thisdate)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			thisMonsthStatus.Date = thisdate
			thisMonsthStatus.Status = false
			err = setMonthlyStatusFunc(client, thisMonsthStatus)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	lastdate := fmt.Sprintf("%04d-%02d", ld.Year(), ld.Month())
	lastMonthStatus, err := getMonthlyStatusFunc(client, lastdate)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			lastMonthStatus.Date = lastdate
			lastMonthStatus.Status = false
			err = setMonthlyStatusFunc(client, lastMonthStatus)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	result := map[string]bool{
		"thismonth": thisMonsthStatus.Status,
		"lastmonth": lastMonthStatus.Status,
	}

	//json으로 결과 전송
	data, err := json.Marshal(result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// handleAPIUpdateTimelogFunc 함수는 restapi로 타임로그를 업데이트하는 함수이다.
func handleAPIUpdateTimelogFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Post method only", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	status := q.Get("status")
	if status == "" {
		http.Error(w, "URL에 status를 입력해주세요", http.StatusBadRequest)
		return
	}
	checkStatus, err := strconv.ParseBool(status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

	// AccessLevel 확인
	accesslevel, err := getAccessLevelFromHeaderFunc(r, client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if accesslevel < DefaultLevel {
		http.Error(w, "업데이트 권한이 없는 계정입니다", http.StatusUnauthorized)
		return
	}

	// Token 가져오기
	token, _ := getTokenFromHeaderFunc(w, r)
	log := Log{}
	log.UserID = token.ID

	// DB에서 Admin setting 데이터를 가져온다.
	adminSetting, err := getAdminSettingFunc(client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type Result struct {
		Status        bool      // true: 이번달만 업데이트, false: 지난달, 이번달 모두 업데이트
		Project       string    // DB에 없는 프로젝트 리스트
		Timelog       []Timelog // 정산 완료된 프로젝트에 작성한 타임로그 리스트
		InvalidAccess bool      // true: admin 권한이 아닐 경우, false: admin 권한
	}
	result := Result{}
	result.Status = checkStatus

	excludeID := adminSetting.SGExcludeID
	excludeProjects := adminSetting.SGExcludeProjects
	taskProjects := adminSetting.TaskProjects
	updateErr := ""
	lasttimelogID := "0" // 마지막 타임로그 아이디 값이 들어갈 변수
	var timelogList []Timelog

	// 1. 샷건에서 타임로그 가져오기
	// 샷건에서 타임로그를 가져와 아티스트가 작성한 타임로그를 프로젝트별로 합산하여 정리한다.
	for { // 업데이트할 타임로그가 없을때까지 반복
		timelogs, timelogID, err := sgGetTimelogsFunc(lasttimelogID, excludeID, excludeProjects, taskProjects, checkStatus)
		lasttimelogID = timelogID
		if err != nil {
			updateErr = fmt.Sprintf("%s", err)
			break
		}
		if len(timelogs) == 0 {
			break
		}

		for _, t := range timelogs {
			err = t.CheckErrorFunc()
			if err != nil {
				updateErr = fmt.Sprintf("%d년 %d월 %s(shotgun ID) : %s", t.Year, t.Month, t.UserID, err)
				break
			}
			if timelogList == nil { // 처음에 timelogList가 비어있을 경우
				timelogList = append(timelogList, t)
				continue
			}
			for i, l := range timelogList {
				if t.UserID == l.UserID && t.Project == l.Project && t.Year == l.Year && t.Month == l.Month { // user id와 project name이 같은 경우
					timelogList[i].Duration = l.Duration + t.Duration
					break
				} else if i == len(timelogList)-1 { // 마지막까지 user id와 project name이 같지 않은 경우
					timelogList = append(timelogList, t)
					break
				}
			}
		}
		if updateErr != "" {
			break
		}
	}

	// 월별 인건비를 저장할 월 지정
	ny, nm, _ := time.Now().Date()
	nowDate := fmt.Sprintf("%04d-%02d", ny, nm)
	ld := time.Now().AddDate(0, -1, 0)
	lastDate := fmt.Sprintf("%04d-%02d", ld.Year(), ld.Month())

	// 2. 정산 완료된 프로젝트에 작성한 타임로그가 있는지 확인
	var finishedTimelog []Timelog   // 정산 완료된 프로젝트에 작성한 타임로그 리스트
	var nowEtcProjectList []string  // 이번달에 작성한 타임로그 중 ETC로 처리해야할 프로젝트 리스트
	var lastEtcProjectList []string // 지난달에 작성한 타임로그 중 ETC로 처리해야할 프로젝트 리스트
	for _, t := range timelogList {
		project, err := getProjectFunc(client, t.Project)
		if err != nil {
			if err == mongo.ErrNoDocuments { // 인건비 계산할 때 DB에 없는 프로젝트를 처리하고 있으므로 여기서는 continue한다.
				continue
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if project.IsFinished == true {
			fts, err := getFinishedTimelogStatusFunc(client, t.Year, t.Month, t.Project) // DB에서 정산 완료된 프로젝트에 작성한 타임로그의 ETC 처리 여부를 가져온다.
			if err != nil {
				if err == mongo.ErrNoDocuments { // DB에 없으면 finishedTimelog에 타임로그를 추가한다.
					finishedTimelog = append(finishedTimelog, t)
					continue
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if fts.Status { // ETC로 처리해야 할 경우 월별로 EtcProjectList에 추가한다.
				if t.Year == ny && t.Month == int(nm) {
					if !checkStringInListFunc(t.Project, nowEtcProjectList) {
						nowEtcProjectList = append(nowEtcProjectList, t.Project)
					}
				} else {
					if !checkStringInListFunc(t.Project, lastEtcProjectList) {
						lastEtcProjectList = append(lastEtcProjectList, t.Project)
					}
				}
			}
		}
	}

	// 정산 완료된 프로젝트에 타임로그를 작성했을 경우 admin 권한이 아니면 리턴한다.
	if finishedTimelog != nil && accesslevel < AdminLevel {
		result.Timelog = finishedTimelog
		result.InvalidAccess = true

		log.CreatedAt = time.Now()
		log.Content = "Admin 권한이 아닌 유저가 타임로그를 업데이트하는 중에 정산 완료된 프로젝트에 타임로그가 존재하였습니다."

		err = addLogsFunc(client, log)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// json으로 결과 전송
		data, err := json.Marshal(result)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return
	}

	// 3. 내부 인건비와 타임로그 삭제
	// 이미 존재하는 타임로그의 프로젝트 내부 인건비(VFX, RND)를 비워주고, DB에 저장하기 전에 VFX 타임로그 정보를 삭제한다.
	// 이번달
	projects, err := getProjectsByTimelogFunc(ny, int(nm), "vfx")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, p := range projects {
		laborCost := p.SMMonthlyLaborCost[nowDate]
		if laborCost != (LaborCost{}) { // 인건비가 비어있는지 확인한다.
			laborCost.VFX = ""
			laborCost.RND = ""
			p.SMMonthlyLaborCost[nowDate] = laborCost
			err = setProjectFunc(client, p)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
	err = rmVFXTimelogFunc(client, ny, int(nm), adminSetting.SMSupervisorIDs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 지난달
	if !checkStatus {
		projects, err = getProjectsByTimelogFunc(ld.Year(), int(ld.Month()), "vfx")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, p := range projects {
			laborCost := p.SMMonthlyLaborCost[lastDate]
			if laborCost != (LaborCost{}) { // 인건비가 비어있는지 확인한다.
				laborCost.VFX = ""
				laborCost.RND = ""
				p.SMMonthlyLaborCost[lastDate] = laborCost
				err = setProjectFunc(client, p)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
		}
		err = rmVFXTimelogFunc(client, ld.Year(), int(ld.Month()), adminSetting.SMSupervisorIDs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// 4. DB에 타임로그 저장
	var nowProjectList []string  // 월별 인건비를 계산할 프로젝트 리스트 - 이번달
	var lastProjectList []string // 월별 인건비를 계산할 프로젝트 리스트 - 지난달
	nowEtcTimelogInfo := make(map[string]map[string]float64)
	lastEtcTimelogInfo := make(map[string]map[string]float64)
	rndProjects := adminSetting.RNDProjects // RND 프로젝트 리스트
	etcProjects := adminSetting.ETCProjects // ETC 프로젝트 리스트

	for _, t := range timelogList {
		// ETC로 처리해야하는 프로젝트 리스트에 포함되어 있다면 타임로그의 프로젝트를 ETC로 수정한다.(정산 완료된 프로젝트)
		if t.Year == ny && t.Month == int(nm) {
			if checkStringInListFunc(t.Project, nowEtcProjectList) {
				if _, exists := nowEtcTimelogInfo[t.Project]; !exists {
					nowEtcTimelogInfo[t.Project] = make(map[string]float64)
				}
				nowEtcTimelogInfo[t.Project][t.UserID] = t.Duration
				t.Project = fmt.Sprintf("ETC%04d", t.Year)

				etcTimelog, err := getTimelogFunc(client, t.UserID, t.Year, t.Month, t.Project)
				if err != nil {
					if err != mongo.ErrNoDocuments {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}
				t.Duration = t.Duration + etcTimelog.Duration
			}
		} else {
			if checkStringInListFunc(t.Project, lastEtcProjectList) {
				if _, exists := lastEtcTimelogInfo[t.Project]; !exists {
					lastEtcTimelogInfo[t.Project] = make(map[string]float64)
				}
				lastEtcTimelogInfo[t.Project][t.UserID] = t.Duration
				t.Project = fmt.Sprintf("ETC%04d", t.Year)

				etcTimelog, err := getTimelogFunc(client, t.UserID, t.Year, t.Month, t.Project)
				if err != nil {
					if err != mongo.ErrNoDocuments {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}
				t.Duration = t.Duration + etcTimelog.Duration
			}
		}

		// rnd 프로젝트 리스트에 포함되어 있다면 타임로그의 프로젝트를 RND2021 형태로 수정하고, rnd 프로젝트의 duration과 합쳐춘다.
		if checkStringInListFunc(t.Project, rndProjects) {
			t.Project = fmt.Sprintf("RND%04d", t.Year)
			rndTimelog, err := getTimelogFunc(client, t.UserID, t.Year, t.Month, t.Project)
			if err != nil {
				if err != mongo.ErrNoDocuments {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			t.Duration = t.Duration + rndTimelog.Duration
		}
		// etc 프로젝트 리스트에 포함되어 있다면 타임로그의 프로젝트를 ETC2021 형태로 수정하고, etc 프로젝트의 duration과 합쳐춘다.
		if checkStringInListFunc(t.Project, etcProjects) {
			t.Project = fmt.Sprintf("ETC%04d", t.Year)
			etcTimelog, err := getTimelogFunc(client, t.UserID, t.Year, t.Month, t.Project)
			if err != nil {
				if err != mongo.ErrNoDocuments {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			t.Duration = t.Duration + etcTimelog.Duration
		}

		if t.Year == ny && t.Month == int(nm) {
			if !checkStringInListFunc(t.Project, nowProjectList) {
				nowProjectList = append(nowProjectList, t.Project)
			}
		} else {
			if !checkStringInListFunc(t.Project, lastProjectList) {
				lastProjectList = append(lastProjectList, t.Project)
			}
		}

		err = addTimelogFunc(client, t)
		if err != nil {
			updateErr = fmt.Sprintf("%s", err)
			break
		}
	}

	// 5. finishedtimelogstatus의 타임로그 정보 업데이트
	if nowEtcTimelogInfo != nil {
		for projectID := range nowEtcTimelogInfo {
			fts, err := getFinishedTimelogStatusFunc(client, ny, int(nm), projectID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			fts.TimelogInfo = nowEtcTimelogInfo[projectID]
			err = updateFinishedTimelogStatusFunc(client, fts)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
	if lastEtcTimelogInfo != nil {
		for projectID := range lastEtcTimelogInfo {
			fts, err := getFinishedTimelogStatusFunc(client, ld.Year(), int(ld.Month()), projectID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			fts.TimelogInfo = lastEtcTimelogInfo[projectID]
			err = updateFinishedTimelogStatusFunc(client, fts)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	// 6. 인건비 계산
	// 이번달
	var errProject []string // 프로젝트가 존재하지 않을 때의 에러 처리
	for _, np := range nowProjectList {
		project, err := getProjectFunc(client, np)
		if err != nil {
			if err == mongo.ErrNoDocuments { // DB에 해당 프로젝트가 존재하지 않는 경우 errProject에 추가한다.
				if !checkStringInListFunc(np, errProject) {
					errProject = append(errProject, np)
					continue
				}
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

	// 지난달
	for _, lp := range lastProjectList {
		project, err := getProjectFunc(client, lp) // DB에서 해당 프로젝트를 가져온다.
		if err != nil {
			if err == mongo.ErrNoDocuments {
				// DB에 해당 프로젝트가 존재하지 않는 경우
				errProject = append(errProject, lp)
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

	// 7. admin setting의 업데이트 시간을 현재 시간으로 설정
	adminSetting.SGUpdatedTime = time.Now().Format(time.RFC3339)
	err = updateAdminSettingFunc(client, adminSetting)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if updateErr != "" {
		http.Error(w, updateErr, http.StatusInternalServerError)
		return
	}

	result.Project = strings.Join(errProject, ",")
	result.Timelog = finishedTimelog

	// 로그 추가
	log.CreatedAt = time.Now()
	if checkStatus {
		if result.Project != "" {
			log.Content = "이번달의 VFX 타임로그 업데이트 중 존재하지 않는 프로젝트가 있습니다."
			if result.Timelog != nil {
				log.Content = log.Content + "\n이번달의 VFX 타임로그 업데이트 중 정산완료된 프로젝트에 타임로그가 존재합니다."
			}
		} else if result.Timelog != nil {
			log.Content = "이번달의 VFX 타임로그 업데이트 중 정산완료된 프로젝트에 타임로그가 존재합니다."
		} else {
			log.Content = "이번달의 VFX 타임로그를 업데이트하였습니다."
		}
	} else {
		if result.Project != "" {
			log.Content = "지난달과 이번달의 VFX 타임로그 업데이트 중 존재하지 않는 프로젝트가 있습니다."
			if result.Timelog != nil {
				log.Content = log.Content + "\n지난달과 이번달의 VFX 타임로그 업데이트 중 정산완료된 프로젝트에 타임로그가 존재합니다."
			}
		} else if result.Timelog != nil {
			log.Content = "지난달과 이번달의 VFX 타임로그 업데이트 중 정산완료된 프로젝트에 타임로그가 존재합니다."
		} else {
			log.Content = "지난달과 이번달의 VFX 타임로그를 업데이트하였습니다."
		}
	}
	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// json으로 결과 전송
	data, err := json.Marshal(result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// handleAPIRmTimelogByIDFunc 함수는 입력받은 id가 작성한 타임로그를 모두 삭제하는 함수이다.
func handleAPIRmTimelogByIDFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Delete method only", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	id := q.Get("id")
	if id == "" {
		http.Error(w, "URL에 id를 입력해주세요", http.StatusBadRequest)
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

	// Access Level 확인
	accesslevel, err := getAccessLevelFromHeaderFunc(r, client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if accesslevel < AdminLevel {
		http.Error(w, "삭제 권한이 없는 계정입니다", http.StatusUnauthorized)
		return
	}

	idList := stringToListFunc(id, " ")
	err = rmTimelogByIDFunc(client, idList)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Token 가져오기
	token, _ := getTokenFromHeaderFunc(w, r)
	log := Log{}
	log.UserID = token.ID
	log.CreatedAt = time.Now()
	log.Content = "AdminSetting에서 제외할 ID의 타임로그를 삭제했습니다."

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// json으로 결과 전송
	data, err := json.Marshal(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// handleAPIRmTimelogByProjectFunc 함수는 입력받은 프로젝트에 작성한 타임로그를 모두 삭제하는 함수이다.
func handleAPIRmTimelogByProjectFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Delete method only", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	project := q.Get("project")
	if project == "" {
		http.Error(w, "URL에 project를 입력해주세요", http.StatusBadRequest)
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

	// Access Level 확인
	accesslevel, err := getAccessLevelFromHeaderFunc(r, client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if accesslevel < AdminLevel {
		http.Error(w, "삭제 권한이 없는 계정입니다", http.StatusUnauthorized)
		return
	}

	projectList := stringToListFunc(project, " ")
	err = rmTimelogByProjectFunc(client, projectList)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Token 가져오기
	token, _ := getTokenFromHeaderFunc(w, r)
	log := Log{}
	log.UserID = token.ID
	log.CreatedAt = time.Now()
	log.Content = "AdminSetting에서 제외할 프로젝트의 타임로그를 삭제했습니다."

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// json으로 결과 전송
	data, err := json.Marshal(project)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// handleAPIResetTimelogFunc 함수는 타임로그를 리셋하는 함수이다.
func handleAPIResetTimelogFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Post method only", http.StatusMethodNotAllowed)
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

	// AccessLevel 확인
	accesslevel, err := getAccessLevelFromHeaderFunc(r, client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if accesslevel < AdminLevel {
		http.Error(w, "리셋 권한이 없는 계정입니다", http.StatusUnauthorized)
		return
	}

	// DB에서 Admin setting 데이터를 가져온다.
	adminSetting, err := getAdminSettingFunc(client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	excludeID := adminSetting.SGExcludeID
	excludeProjects := adminSetting.SGExcludeProjects
	taskProjects := adminSetting.TaskProjects
	updateErr := ""
	lasttimelogID := "0"      // 마지막 타임로그 아이디 값이 들어갈 변수
	var timelogList []Timelog // 타임로그 정리
	for {                     // 업데이트할 타임로그가 없을때까지 반복
		timelogs, timelogID, err := sgResetTimelogsFunc(lasttimelogID, excludeID, excludeProjects, taskProjects)
		lasttimelogID = timelogID
		if err != nil {
			updateErr = fmt.Sprintf("%s", err)
			break
		}
		if len(timelogs) == 0 {
			break
		}

		for _, t := range timelogs {
			err = t.CheckErrorFunc()
			if err != nil {
				updateErr = fmt.Sprintf("%s", err)
				break
			}
			if timelogList == nil { // 처음에 timelogList가 비어있을 경우
				timelogList = append(timelogList, t)
				continue
			}
			for i, l := range timelogList {
				if t.UserID == l.UserID && t.Project == l.Project && t.Year == l.Year && t.Month == l.Month { // id, project, year, month가 같은 경우
					timelogList[i].Duration = l.Duration + t.Duration
					break
				} else if i == len(timelogList)-1 {
					timelogList = append(timelogList, t) // 마지막까지 user id와 project name이 같지 않은 경우
					break
				}
			}
		}
		if updateErr != "" {
			break
		}
	}

	// DB에 저장하기 전에 VFX 타임로그 정보를 삭제한다.
	err = rmVFXAllTimelogFunc(client, adminSetting.SMSupervisorIDs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, t := range timelogList { // timelogList(모든)를 db에 저장
		err = addTimelogFunc(client, t)
		if err != nil {
			updateErr = fmt.Sprintf("%s", err)
			break
		}
		if updateErr != "" {
			break
		}
	}

	adminSetting.SGUpdatedTime = time.Now().Format(time.RFC3339)
	err = updateAdminSettingFunc(client, adminSetting)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if updateErr != "" {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Token 가져오기
	token, _ := getTokenFromHeaderFunc(w, r)
	log := Log{}
	log.UserID = token.ID
	log.CreatedAt = time.Now()
	log.Content = "AdminSetting에서 타임로그를 리셋했습니다."

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// json으로 결과 전송
	data, err := json.Marshal("done")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
