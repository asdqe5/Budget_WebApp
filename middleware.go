// 프로젝트 결산 프로그램
//
// Description : 미들웨어 관련 스크립트

package main

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// getTokenFromHeaderFunc 함수는 쿠키에서 Token 값을 반환한다.
func getTokenFromHeaderFunc(w http.ResponseWriter, r *http.Request) (Token, error) {
	// Token을 열기위해서 헤더 쿠키에서 필요한 정보를 불러온다.
	sessionToken := ""
	sessionSignkey := ""
	for _, cookie := range r.Cookies() {
		if cookie.Name == "SessionToken" {
			sessionToken = cookie.Value
			continue
		}
		if cookie.Name == "SessionSignKey" {
			sessionSignkey = cookie.Value
			continue
		}
	}
	tk := Token{}
	// Signkey로 Token 정보를 연다.
	token, err := jwt.ParseWithClaims(sessionToken, &tk, func(token *jwt.Token) (interface{}, error) {
		return []byte(sessionSignkey), nil
	})
	if err != nil {
		return tk, err
	}
	if !token.Valid {
		return tk, errors.New("Token key is not valid")
	}
	if tk.ToolName != "budget" {
		return tk, errors.New("Token key is not for budget")
	}

	// mongoDB client 연결
	credential := options.Credential{
		Username: *flagDBID,
		Password: *flagDBPW,
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(*flagMongoDBURI).SetAuth(credential))
	if err != nil {
		return tk, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		return tk, err
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return tk, err
	}

	// DB에 저장된 토큰키와 일치하는지 확인
	user, err := getUserFunc(client, tk.ID)
	if err != nil {
		return tk, err
	}
	if user.Token != token.Raw {
		return tk, errors.New("토큰키가 일치하지 않습니다")
	}
	return tk, nil
}

// getAccessLevelFromHeaderFunc 함수는 rest api 사용시 토큰을 확인하고 access level을 반환하는 함수이다.
func getAccessLevelFromHeaderFunc(r *http.Request, client *mongo.Client) (AccessLevel, error) {
	// header에서 token을 가져온다.
	auth := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(auth) != 2 || auth[0] != "Basic" {
		return GuestLevel, errors.New("Authorization failed")
	}
	token := auth[1]

	// DB 검색
	collection := client.Database(*flagDBName).Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user := User{}
	err := collection.FindOne(ctx, bson.M{"token": token}).Decode(&user)
	if err != nil {
		return GuestLevel, err
	}
	return user.AccessLevel, nil
}

// GetObjectIDfromRequestHeader 미들웨어는 리퀘스트헤더에서 ObjectID를 가지고 온다.
func GetObjectIDfromRequestHeader(r *http.Request) (string, error) {
	// 리퀘스트헤더에서 ObjectID를 가지고 온다.
	uri := r.Header.Get("Referer")
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	urlValues, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return "", err
	}
	var objectID string
	// urlValues에 objectid가 존재하는지 채크한다.
	if value, has := urlValues["objectid"]; has {
		// urlValues["objectid"] 갯수가 1개인지 체크한다.
		if len(value) != 1 {
			return "", errors.New("objectid 값이 1개가 아닙니다")
		}
		objectID = value[0]
	}
	if objectID == "" {
		return "", errors.New("objectid 값이 빈 문자열입니다")
	}
	return objectID, nil
}
