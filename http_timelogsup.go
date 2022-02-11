// 프로젝트 결산 프로그램
//
// Description : http SUP 타임로그 관련 스크립트

package main

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// handleTimelogSUPFunc 함수는 슈퍼바이저의 타임로그를 관리하는 페이지를 띄우는 함수이다.
func handleTimelogSUPFunc(w http.ResponseWriter, r *http.Request) {
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

	// Admin Setting 값들을 가져온다.
	adminSetting, err := getAdminSettingFunc(client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type Recipe struct {
		Token                  Token
		User                   User
		Date                   string               // yyyy-MM
		Projects               []Project            // 타임로그 정보가 있는 프로젝트 리스트
		Supervisors            []Artist             // 슈퍼바이저 목록
		SupervisorTimelog      map[string][]Timelog // 수퍼바이저 타임로그
		TotalSupervisorTimelog map[string]float64   // 수퍼바이저 Total 타임로그
		NoneArtists            []string             // DB에 존재하지 않는 아티스트들의 ID
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
	year, err := strconv.Atoi(strings.Split(date, "-")[0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	month, err := strconv.Atoi(strings.Split(date, "-")[1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	thisDate, err := time.Parse("2006-01", date)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 입력받은 달에 진행중인 프로젝트 정보를 가져온다.
	searchword := "date:" + rcp.Date
	rcp.Projects, err = searchProjectFunc(client, searchword, "id")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 입력받은 달의 슈퍼바이저 타임로그 정보를 가져온다.
	supIDs := adminSetting.SMSupervisorIDs
	supTimelogs := make(map[string][]Timelog)
	totalSupTimelog := make(map[string]float64)
	for _, id := range supIDs {
		supervisor, err := getArtistFunc(client, id)
		if err != nil {
			if err == mongo.ErrNoDocuments { // DB에 없을 경우 errArtistID에 추가한다.
				if !checkStringInListFunc(id, rcp.NoneArtists) {
					rcp.NoneArtists = append(rcp.NoneArtists, id)
					continue
				}
				continue
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 슈퍼바이저의 입사일을 비교한다.
		if supervisor.StartDay != "" { // 입사일이 있는 경우
			startDay := strings.Split(supervisor.StartDay, "-")
			startDate, err := time.Parse("2006-01", fmt.Sprintf("%s-%s", startDay[0], startDay[1]))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if startDate.After(thisDate) {
				continue
			}
		} else {
			continue
		}

		// 슈퍼바이저의 퇴사여부를 확인한다.
		if supervisor.Resination == true { // 퇴사했다면 퇴사날짜를 비교해서 이번달 전인지를 비교한다.
			endDay := strings.Split(supervisor.EndDay, "-")
			endDate, err := time.Parse("2006-01", fmt.Sprintf("%s-%s", endDay[0], endDay[1]))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if endDate.Before(thisDate) {
				continue
			} else {
				rcp.Supervisors = append(rcp.Supervisors, supervisor)
			}
		} else {
			rcp.Supervisors = append(rcp.Supervisors, supervisor)
		}

		for _, p := range rcp.Projects {
			timelog, err := getTimelogFunc(client, id, year, month, p.ID)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					continue
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			supTimelogs[id] = append(supTimelogs[id], timelog)
			totalSupTimelog[id] += timelog.Duration
		}
	}
	rcp.SupervisorTimelog = supTimelogs
	rcp.TotalSupervisorTimelog = totalSupTimelog
	sort.Slice(rcp.Supervisors, func(i, j int) bool { // 수퍼바이저 이름을 기준으로 오름차순
		return rcp.Supervisors[i].Name < rcp.Supervisors[j].Name
	})

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "timelogs-sup", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleEditSUPTimelogFunc 함수는 슈퍼바이저 타임로그 페이지에서 Update 버튼을 눌렀을 때 실행하는 함수이다.
func handleEditSUPTimelogFunc(w http.ResponseWriter, r *http.Request) {
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

	// Admin Setting 값들을 가져온다.
	adminSetting, err := getAdminSettingFunc(client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	q := r.URL.Query()
	date := q.Get("date")
	if date == "" {
		http.Error(w, "날짜를 입력해주세요", http.StatusInternalServerError)
		return
	}
	year, err := strconv.Atoi(strings.Split(date, "-")[0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	month, err := strconv.Atoi(strings.Split(date, "-")[1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	thisDate, err := time.Parse("2006-01", date)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 입력받은 달에 진행중인 프로젝트 정보를 가져온다.
	searchword := "date:" + date
	projects, err := searchProjectFunc(client, searchword, "id")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 입력받은 달의 슈퍼바이저 타임로그 정보를 가져온다.
	supIDs := adminSetting.SMSupervisorIDs
	for _, id := range supIDs {
		supervisor, err := getArtistFunc(client, id)
		if err != nil {
			if err == mongo.ErrNoDocuments { // DB에 없을 경우 errArtistID에 추가한다.
				continue
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 슈퍼바이저의 입사일을 비교한다.
		if supervisor.StartDay != "" { // 입사일이 있는 경우
			startDay := strings.Split(supervisor.StartDay, "-")
			startDate, err := time.Parse("2006-01", fmt.Sprintf("%s-%s", startDay[0], startDay[1]))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if startDate.After(thisDate) {
				continue
			}
		} else {
			continue
		}

		// 슈퍼바이저의 퇴사여부를 확인한다.
		if supervisor.Resination == true { // 퇴사했다면 퇴사날짜를 비교해서 이번달 전인지를 비교한다.
			endDay := strings.Split(supervisor.EndDay, "-")
			endDate, err := time.Parse("2006-01", fmt.Sprintf("%s-%s", endDay[0], endDay[1]))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if endDate.Before(thisDate) {
				continue
			}
		}

		for _, p := range projects {
			monthlyLaborCost := make(map[string]LaborCost)
			if p.SMMonthlyLaborCost != nil {
				monthlyLaborCost = p.SMMonthlyLaborCost
			}
			suptimelog := r.FormValue(fmt.Sprintf("%s-%s-timelog", id, p.ID))
			var st Timelog
			st.UserID = id
			st.Year = year
			st.Month = month
			st.Project = p.ID

			if suptimelog != "" {
				supduration, err := strconv.ParseFloat(suptimelog, 64)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				st.Duration = supduration * 60

				searchWord := fmt.Sprintf("userid:%s year:%d month:%d project:%s duration:%.f", st.UserID, st.Year, st.Month, st.Project, st.Duration)
				timelog, err := searchTimelogFunc(client, searchWord) // DB에 일치하는 타임로그가 존재하는지 확인한다.
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if timelog == nil { // DB에 일치하는 타임로그가 존재하지 않으면 타임로그를 업데이트하고 인건비를 다시 계산한다.
					err = addTimelogFunc(client, st)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					laborCost := LaborCost{}
					// VFX 본부의 순수 프로젝트 인건비
					vfxLaborCost, err := calMonthlyVFXLaborCostFunc(st.Project, st.Year, st.Month)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					laborCost.VFX, err = encryptAES256Func(strconv.Itoa(vfxLaborCost))
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					laborCost.CM = p.SMMonthlyLaborCost[date].CM // CM 인건비는 다시 계산할 필요 없음
					monthlyLaborCost[date] = laborCost
				}
			} else {
				timelog, err := getTimelogFunc(client, st.UserID, st.Year, st.Month, st.Project)
				if err != nil {
					if err == mongo.ErrNoDocuments { // 수정되지 않았다면 continue
						continue
					}
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				err = rmTimelogFunc(client, timelog) // 타임로그를 수정한 경우이기 때문에 타임로그를 삭제한다.
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				laborCost := LaborCost{}
				// VFX 본부의 순수 프로젝트 인건비
				vfxLaborCost, err := calMonthlyVFXLaborCostFunc(st.Project, st.Year, st.Month)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				laborCost.VFX, err = encryptAES256Func(strconv.Itoa(vfxLaborCost))
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				laborCost.CM = p.SMMonthlyLaborCost[date].CM // CM 인건비는 다시 계산할 필요 없음
				monthlyLaborCost[date] = laborCost
			}
			p.SMMonthlyLaborCost = monthlyLaborCost

			err = setProjectFunc(client, p)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	log := Log{
		UserID:    token.ID,
		CreatedAt: time.Now(),
		Content:   fmt.Sprintf("%d년 %d월의 슈퍼바이저 타임로그 정보가 수정되었습니다.", year, month),
	}

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/editsuptimelogs-success?date=%s", date), http.StatusSeeOther)
}

// handleEditSUPTimelogSuccessFunc 함수는 슈퍼바이저 타임로그 수정이 성공했다는 페이지를 띄운다.
func handleEditSUPTimelogSuccessFunc(w http.ResponseWriter, r *http.Request) {
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

	date := q.Get("date")
	if date == "" { // date 값이 없으면 올해로 검색
		y, m, _ := time.Now().Date()
		date = fmt.Sprintf("%04d-%02d", y, m)
	}

	type Recipe struct {
		Token
		Date string // 검색한 달
	}
	rcp := Recipe{
		Token: token,
		Date:  date,
	}

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "editsuptimelogs-success", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
