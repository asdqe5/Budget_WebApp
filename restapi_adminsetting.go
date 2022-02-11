package main

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// handleAPIVFXTeamsFunc 함수는 admin setting에서 입력받은 부서에 해당하는 VFX 팀을 반환하는 함수이다.
func handleAPIVFXTeamsFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Get method only", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	dept := q.Get("dept")
	if dept == "" {
		http.Error(w, "URL에 dept를 입력해주세요", http.StatusBadRequest)
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

	adminSetting, err := getAdminSettingFunc(client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var team []string
	if strings.ToLower(dept) == "all" {
		for _, value := range adminSetting.VFXTeams {
			team = append(team, value...)
		}
	} else {
		team = adminSetting.VFXTeams[dept]
	}

	sort.Strings(team)
	// json으로 결과 전송
	data, err := json.Marshal(team)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// handleAPITotalTeamsFunc 함수는 Total 팀에서 입력받은 부서에 해당하는 팀을 반환하는 함수이다.
func handleAPITotalTeamsFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Get method only", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	dept := q.Get("dept")
	if dept == "" {
		http.Error(w, "URL에 dept를 입력해주세요", http.StatusBadRequest)
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

	adminSetting, err := getAdminSettingFunc(client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var team []string
	if strings.ToLower(dept) == "all" {
		for _, value := range adminSetting.VFXTeams {
			team = append(team, value...)
		}
		for _, value := range adminSetting.CMTeams {
			if !checkStringInListFunc(value, team) {
				team = append(team, value)
			}
		}
	} else if strings.ToLower(dept) == "cm" {
		team = adminSetting.CMTeams
	} else {
		team = adminSetting.VFXTeams[dept]
	}

	sort.Strings(team)
	// json으로 결과 전송
	data, err := json.Marshal(team)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
