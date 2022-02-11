// 프로젝트 결산 프로그램
//
// Description : http 예산 디테일 페이지 관련 스크립트

package main

import (
	"context"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// handleBGDetailFunc 함수는 프로젝트의 예산 디테일 페이지를 여는 함수이다.
func handleBGDetailFunc(w http.ResponseWriter, r *http.Request) {
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
		Token       Token
		TeamSetting BGTeamSetting
	}
	rcp := Recipe{}
	rcp.Token = token
	rcp.TeamSetting, err = getBGTeamSettingFunc(client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = TEMPLATES.ExecuteTemplate(w, "bgdetail", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
