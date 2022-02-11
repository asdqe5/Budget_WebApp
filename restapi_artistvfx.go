// 프로젝트 결산 프로그램
//
// Description : 아티스트 관련 rest API를 작성한 스크립트

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func handleAPIAddArtistVFXFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Post method only", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	id := q.Get("id")
	if id == "" {
		http.Error(w, "URL에 id를 입력해주세요", http.StatusBadRequest)
		return
	}
	salary := q.Get("salary")
	if salary != "" {
		if !regexSalary.MatchString(salary) {
			http.Error(w, "salary가 2019:2400,2020:2400 형식이 아닙니다", http.StatusBadRequest)
			return
		}
	}
	startday := q.Get("startday")
	if startday != "" {
		if !regexDate2.MatchString(startday) {
			http.Error(w, "startday가 2020-09-01 형식이 아닙니다", http.StatusBadRequest)
			return
		}
	}
	endday := q.Get("endday")
	if endday != "" {
		if !regexDate2.MatchString(endday) {
			http.Error(w, "endday가 2020-09-01 형식이 아닙니다", http.StatusBadRequest)
			return
		}
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
		http.Error(w, "추가 권한이 없는 계정입니다", http.StatusUnauthorized)
		return
	}

	a := Artist{}
	a.ID = id
	a.Salary, _ = stringToMapFunc(salary)
	a.StartDay = startday
	a.EndDay = endday

	// 연봉 암호화
	for key, value := range a.Salary {
		encrypted, err := encryptAES256Func(value)
		a.Salary[key] = encrypted
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Shotgun에서 아티스트 정보를 가져온다.
	artist, err := sgGetArtistFunc(a.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.Name = artist.Name
	a.Dept = artist.Dept
	a.Team = artist.Team

	// 동일 연도 연봉 변경 확인
	change := q.Get("change")
	if change == "true" { // 동일 연도에 연봉이 변경되었다면
		a.ChangedSalary = make(map[string]string)
		changedate := q.Get("changedate")
		changesalary := q.Get("changesalary")
		encrypted, err := encryptAES256Func(changesalary)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		a.Changed = true
		a.ChangedSalary[changedate] = encrypted
	}

	// 입사일, 퇴사일, 동일 연도 연봉 변경일 체크
	startDate, err := time.Parse("2006-01-02", a.StartDay) // 입사일 Date
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if a.EndDay != "" { // 아티스트 퇴사일이 설정된 경우
		endDate, err := time.Parse("2006-01-02", a.EndDay) // 퇴사일 Date
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if endDate.Before(startDate) { // 퇴사일이 입사일 전인 경우 -> 에러 처리
			http.Error(w, "퇴사일이 잘못 입력되었습니다.", http.StatusInternalServerError)
			return
		}
		if a.Changed { // 아티스트 동일 연도 연봉이 변경된 경우
			changeDate, err := time.Parse("2006-01-02", q.Get("changedate")) // 연봉 변경 날짜
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if !startDate.Before(changeDate) || changeDate.After(endDate) { // 연봉 변경일이 잘못 입력된 경우
				http.Error(w, "동일 연도 연봉 변경일이 잘못 입력되었습니다.", http.StatusInternalServerError)
				return
			}
		}
	}
	if a.Changed { // 아티스트 동일 연도 연봉이 변경된 경우
		changeDate, err := time.Parse("2006-01-02", q.Get("changedate")) // 연봉 변경 날짜
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !startDate.Before(changeDate) { // 연봉 변경일이 잘못 입력된 경우
			http.Error(w, "동일 연도 연봉 변경일이 잘못 입력되었습니다.", http.StatusInternalServerError)
			return
		}
	}

	err = a.CheckErrorFunc()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = addArtistFunc(client, a) // 아티스트 추가
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Token 가져오기
	token, _ := getTokenFromHeaderFunc(w, r)
	log := Log{}
	log.UserID = token.ID
	log.CreatedAt = time.Now()
	log.Content = fmt.Sprintf("VFX 아티스트 ID %s가 추가되었습니다.", id)

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// json으로 결과 전송
	data, err := json.Marshal(a)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func handleEventSGAPIAddArtistVFXFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Post method only", http.StatusMethodNotAllowed)
		return
	}

	// URL에서 id, name, dept, team을 가져온다.
	q := r.URL.Query()
	id := q.Get("id")
	if id == "" {
		http.Error(w, "URL에 id를 입력해주세요", http.StatusBadRequest)
		return
	}
	name := q.Get("name")
	if name == "" {
		http.Error(w, "URL에 name을 입력해주세요", http.StatusBadRequest)
		return
	}
	department := q.Get("department")
	if department == "" {
		http.Error(w, "URL에 department를 입력해주세요", http.StatusBadRequest)
		return
	}
	team := q.Get("team")
	if team == "" {
		http.Error(w, "URL에 team을 입려해주세요", http.StatusBadRequest)
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

	a := Artist{
		ID:   id,
		Name: name,
		Dept: department,
		Team: team,
	}

	err = a.CheckErrorFunc()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = addArtistFunc(client, a) // 아티스트 추가
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Token 가져오기
	token, _ := getTokenFromHeaderFunc(w, r)
	log := Log{}
	log.UserID = token.ID
	log.CreatedAt = time.Now()
	log.Content = fmt.Sprintf("VFX 아티스트 ID %s가 추가되었습니다.(ShotgunEvent)", id)

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
