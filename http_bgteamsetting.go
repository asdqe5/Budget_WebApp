// 프로젝트 결산 프로그램
//
// Description : http 예산 Team Setting 관련 스크립트

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

// handleBGTeamSettingFunc 함수는 예산 TeamSetting 페이지를 여는 함수이다.
func handleBGTeamSettingFunc(w http.ResponseWriter, r *http.Request) {
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
		Token        Token         // 토큰
		User         User          // 유저 정보
		TeamSetting  BGTeamSetting // 예산 팀 세팅
		ControlTeams []string      // admin setting에 설정한 수퍼바이저, 프로덕션, 매니지먼트 팀 리스트
		Tasks        []string      // 태스크 목록
		VFXTeams     []string      // admmin setting에 설정한 VFX 팀 목록
		CMTeams      []string      // admin setting에 설정한 CM 팀 목록
	}
	rcp := Recipe{}
	rcp.Token = token
	rcp.User, err = getUserFunc(client, token.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rcp.TeamSetting, err = getBGTeamSettingFunc(client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "bgteamsetting", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleBGTeamSettingSubmitFunc 함수는 예산 TeamSetting 페이지에서 Update 버튼을 클릭했을 때 실행되는 함수이다.
func handleBGTeamSettingSubmitFunc(w http.ResponseWriter, r *http.Request) {
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

	taskNum, err := strconv.Atoi(r.FormValue("tasknum"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 태스크별 팀 설정
	for tIndex := 0; tIndex < taskNum; tIndex++ {
		taskName := r.FormValue(fmt.Sprintf("task%d", tIndex))
		teamList := r.Form[fmt.Sprintf("team%d", tIndex)]
		ts.Teams[taskName] = teamList
	}

	err = setBGTeamSettingFunc(client, ts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/bgteamsetting-success", http.StatusSeeOther)
}

// handleBGTeamSEttingSuccessFunc 함수는 BG Team Setting Update가 성공했다는 페이지를 연다.
func handleBGTeamSEttingSuccessFunc(w http.ResponseWriter, r *http.Request) {
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
	err = TEMPLATES.ExecuteTemplate(w, "bgteamsetting-success", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
