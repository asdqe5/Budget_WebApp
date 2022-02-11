// 프로젝트 결산 프로그램
//
// Description : http 유저 관련 스크립트

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
	"golang.org/x/crypto/bcrypt"
)

// handleSignupFunc 함수는 회원 가입 페이지를 띄우는 함수이다.
func handleSignupFunc(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	err := TEMPLATES.ExecuteTemplate(w, "signup", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleSignupSubmitFunc 함수는 유저를 추가하고 회원가입 완료 페이지로 리다이렉트하는 함수이다.
func handleSignupSubmitFunc(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("ID")
	if id == "" {
		http.Error(w, "ID 값이 빈 문자열입니다", http.StatusBadRequest)
		return
	}
	pw := r.FormValue("Password")
	if pw == "" {
		http.Error(w, "Password 값이 빈 문자열입니다", http.StatusBadRequest)
		return
	}
	if pw != r.FormValue("ConfirmPassword") {
		http.Error(w, "입력받은 2개의 패스워드가 서로 다릅니다", http.StatusBadRequest)
		return
	}
	name := r.FormValue("Name")
	if name == "" {
		http.Error(w, "Name 값이 빈 문자열입니다", http.StatusBadRequest)
		return
	}
	team := r.FormValue("Team")
	if team == "" {
		http.Error(w, "Team 값이 빈 문자열입니다", http.StatusBadRequest)
		return
	}
	encryptedPW, err := encryptFunc(pw)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	u := User{}
	u.AccessLevel = GuestLevel
	u.ID = id
	u.Password = encryptedPW
	u.Name = name
	u.Team = team
	err = u.CreateToken()
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

	err = addUserFunc(client, u) // DB에 유저 추가
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log := Log{}
	log.UserID = u.ID
	log.CreatedAt = time.Now()
	log.Content = "회원가입하였습니다."

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/signup-success", http.StatusSeeOther)
}

// handleSignupSuccessFunc 함수는 회원 가입 완료 페이지를 띄우는 함수이다.
func handleSignupSuccessFunc(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	err := TEMPLATES.ExecuteTemplate(w, "signup-success", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleSigninFunc 함수는 로그인 페이지를 띄우는 함수이다.
func handleSigninFunc(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	err := TEMPLATES.ExecuteTemplate(w, "signin", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleSigninSubmitFunc 함수는 로그인 정보가 DB와 일치하는지 확인하고 쿠키에 토큰을 저장하는 함수이다.
func handleSigninSubmitFunc(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("ID")
	if id == "" {
		http.Error(w, "ID 값이 빈 문자열 입니다", http.StatusBadRequest)
		return
	}
	pw := r.FormValue("Password")
	if pw == "" {
		http.Error(w, "Password 값이 빈 문자열 입니다", http.StatusBadRequest)
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

	u, err := getUserFunc(client, id) // DB에서 유저 정보를 가져온다.
	if err != nil {
		if err == mongo.ErrNoDocuments { // DB에 저장된 유저가 없을 때 로그인 실패 페이지를 띄운다.
			err := TEMPLATES.ExecuteTemplate(w, "signin-fail", nil)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 입력한 비밀번호와 DB에 저장된 비밀번호가 일치하는지 확인
	err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(pw))
	if err != nil {
		err := TEMPLATES.ExecuteTemplate(w, "signin-fail", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	// Token을 쿠키에 저장한다.
	c := http.Cookie{
		Name:    "SessionToken",
		Value:   u.Token,
		Expires: time.Now().Add(time.Duration(*flagCookieAge) * time.Hour),
	}
	http.SetCookie(w, &c)
	signKey := http.Cookie{
		Name:    "SessionSignKey",
		Value:   u.SignKey,
		Expires: time.Now().Add(time.Duration(*flagCookieAge) * time.Hour),
	}
	http.SetCookie(w, &signKey)

	// / 로 리다이렉션 한다.
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleSignOutFunc 함수는 쿠키에 저장된 토큰을 삭제하는 함수이다.
func handleSignOutFunc(w http.ResponseWriter, r *http.Request) {
	tokenKey := http.Cookie{
		Name:   "SessionToken",
		Value:  "",
		MaxAge: -1,
	}
	http.SetCookie(w, &tokenKey)
	signKey := http.Cookie{
		Name:   "SessionSignKey",
		Value:  "",
		MaxAge: -1,
	}
	http.SetCookie(w, &signKey)
	http.Redirect(w, r, "/signin", http.StatusSeeOther)
}

// handleEditProfileFunc 함수는 프로필 페이지를 띄우는 함수이다.
func handleEditProfileFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
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
		User  User
		Token Token
	}
	rcp := Recipe{}
	rcp.Token = token

	rcp.User, err = getUserFunc(client, token.ID) // DB에서 유저 정보를 가져온다.
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Profile 페이지를 띄운다.
	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "editprofile", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleEditProfileSubmitFunc 함수는 유저 정보를 업데이트하고 수정 완료 페이지로 리다이렉트하는 함수이다.
func handleEditProfileSubmitFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
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

	id := token.ID
	u, err := getUserFunc(client, id) // DB에서 유저 정보를 가져온다.
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	u.Team = r.FormValue("team")
	u.Name = r.FormValue("name")
	err = u.CheckError()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = setUserFunc(client, u) // DB에 저장된 유저 정보를 업데이트한다.
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log := Log{}
	log.UserID = token.ID
	log.CreatedAt = time.Now()
	log.Content = "프로필이 수정되었습니다."

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/editprofile-success", http.StatusSeeOther)
}

// handleEditProfileSuccessFunc 함수는 유저 정보 수정 완료 페이지를 띄우는 함수이다.
func handleEditProfileSuccessFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	type Recipe struct {
		Token
	}
	rcp := Recipe{}
	rcp.Token = token

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "editprofile-success", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleUpdatePasswordFunc 함수는 비밀번호 업데이트하는 페이지를 띄우는 함수이다.
func handleUpdatePasswordFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	type Recipe struct {
		Token
	}
	rcp := Recipe{}
	rcp.Token = token

	err = TEMPLATES.ExecuteTemplate(w, "updatepassword", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleUpdatePasswordSubmitFunc 함수는 새로운 패스워드로 업데이트하고 쿠키를 수정하는 함수이다.
func handleUpdatePasswordSubmitFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	id := token.ID
	nowPW := r.FormValue("nowPassword")
	if nowPW == "" {
		http.Error(w, "현재 사용중인 패스워드 값이 빈 문자열입니다", http.StatusBadRequest)
		return
	}
	newPW := r.FormValue("newPassword")
	if newPW == "" {
		http.Error(w, "새 패스워드 값이 빈 문자열입니다", http.StatusBadRequest)
		return
	}
	if newPW != r.FormValue("confirmNewPassword") {
		http.Error(w, "입력받은 2개의 패스워드가 서로 다릅니다", http.StatusBadRequest)
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

	u, err := getUserFunc(client, id) // DB에서 유저 정보를 가져온다.
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 입력한 비밀번호와 DB에 저장된 비밀번호가 일치하는지 확인
	err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(nowPW))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	encryptedPW, err := encryptFunc(newPW) // 입력한 비밀번호를 암호화한다.
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	u.Password = encryptedPW

	// token 재생성
	err = u.CreateToken()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = setUserFunc(client, u) // DB에 저장된 유저 정보를 업데이트한다.
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Token을 쿠키에 저장한다.
	c := http.Cookie{
		Name:    "SessionToken",
		Value:   u.Token,
		Expires: time.Now().Add(time.Duration(*flagCookieAge) * time.Hour),
	}
	http.SetCookie(w, &c)
	signKey := http.Cookie{
		Name:    "SessionSignKey",
		Value:   u.SignKey,
		Expires: time.Now().Add(time.Duration(*flagCookieAge) * time.Hour),
	}
	http.SetCookie(w, &signKey)

	log := Log{}
	log.UserID = token.ID
	log.CreatedAt = time.Now()
	log.Content = "비밀번호가 변경되었습니다."

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/updatepassword-success", http.StatusSeeOther)
}

// handleUpdatePasswordSuccessFunc 함수는 비밀번호 업데이트 완료 페이지를 띄우는 함수이다.
func handleUpdatePasswordSuccessFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	type Recipe struct {
		Token
	}
	rcp := Recipe{}
	rcp.Token = token

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "updatepassword-success", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleInvalidAccessFunc 함수는 권한 없음 페이지를 띄우는 함수이다.
func handleInvalidAccessFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	type Recipe struct {
		Token
	}
	rcp := Recipe{}
	rcp.Token = token

	err = TEMPLATES.ExecuteTemplate(w, "invalidaccess", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleUsersFunc 함수는 유저 관리 페이지를 띄우는 함수이다.
func handleUsersFunc(w http.ResponseWriter, r *http.Request) {
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
		Token Token
		User  User   // 현재 로그인된 유저
		Users []User // 유저 리스트
	}
	rcp := Recipe{}
	rcp.Token = token
	rcp.User, err = getUserFunc(client, token.ID) // DB에서 현재 로그인된 유저의 정보를 가져온다.
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rcp.Users, err = getAllUsersFunc(client) // DB에 저장된 모든 유저 정보를 가져온다.
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "users", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleUpdateUsersFunc 함수는 유저들의 정보를 업데이트하는 함수이다.
func handleUpdateUsersFunc(w http.ResponseWriter, r *http.Request) {
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

	userNum, err := strconv.Atoi(r.FormValue("userNum"))
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

	for i := 0; i < userNum; i++ {
		id := r.FormValue(fmt.Sprintf("id%d", i))
		user, err := getUserFunc(client, id) // DB에서 id로 아티스트 검색
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 수정한 레벨로 변경
		accessLevel, err := strconv.Atoi(r.FormValue(fmt.Sprintf("accesslevel%d", i)))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		user.AccessLevel = AccessLevel(accessLevel)

		// token 재생성
		err = user.CreateToken()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = setUserFunc(client, user) // DB에 저장된 유저 정보를 업데이트한다.
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 현재 로그인되어 있는 계정인 경우 새로 재생성한 Token을 쿠키에 저장한다.
		if id == token.ID {
			SessionToken := http.Cookie{
				Name:    "SessionToken",
				Value:   user.Token,
				Expires: time.Now().Add(time.Duration(*flagCookieAge) * time.Hour),
			}
			http.SetCookie(w, &SessionToken)
			SessionSignKey := http.Cookie{
				Name:    "SessionSignKey",
				Value:   user.SignKey,
				Expires: time.Now().Add(time.Duration(*flagCookieAge) * time.Hour),
			}
			http.SetCookie(w, &SessionSignKey)
		}
	}

	log := Log{}
	log.UserID = token.ID
	log.CreatedAt = time.Now()
	log.Content = "유저 권한이 수정되었습니다."

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/updateusers-success", http.StatusSeeOther)
}

// handleUpdateUsersSuccess 함수는 유저 정보 업데이트 완료 페이지를 띄우는 함수이다.
func handleUpdateUsersSuccess(w http.ResponseWriter, r *http.Request) {
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
	err = TEMPLATES.ExecuteTemplate(w, "updateusers-success", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleChangePasswordFunc 함수는 관리자가 User 관리 페이지에서 비밀번호를 변경하는 함수이다.
func handleChangePasswordFunc(w http.ResponseWriter, r *http.Request) {
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
		ID    string
		Token Token
	}
	rcp := Recipe{}
	rcp.Token = token

	q := r.URL.Query()
	rcp.ID = q.Get("id")
	if rcp.ID == "" {
		http.Error(w, "URL에 id를 입력해주세요", http.StatusBadRequest)
		return
	}

	err = TEMPLATES.ExecuteTemplate(w, "changepassword", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleChangePasswordSubmitFunc 함수는 관리자가 ChangePassword 페이지에서 Submit을 했을 때 실행하는 함수이다.
func handleChangePasswordSubmitFunc(w http.ResponseWriter, r *http.Request) {
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

	// 비밀번호를 변경할 User의 ID를 가져온다.
	q := r.URL.Query()
	id := q.Get("id")
	if id == "" {
		http.Error(w, "URL에 id를 입력해주세요", http.StatusBadRequest)
		return
	}

	// 현재 관리자 계정이고 비밀번호를 변경하려는 ID와 자신의 ID가 같은 경우 현재 비밀번호를 가져온다.
	nowPW := ""
	if token.AccessLevel == AdminLevel && id == token.ID {
		nowPW = r.FormValue("nowPassword")
		if nowPW == "" {
			http.Error(w, "현재 사용중인 패스워드 값이 빈 문자열입니다", http.StatusBadRequest)
			return
		}
	}

	// 입력받은 새로운 비밀번호를 가져온다.
	newPW := r.FormValue("newPassword")
	if newPW == "" {
		http.Error(w, "새 패스워드 값이 빈 문자열입니다", http.StatusBadRequest)
		return
	}
	if newPW != r.FormValue("confirmNewPassword") {
		http.Error(w, "입력받은 2개의 패스워드가 서로 다릅니다", http.StatusBadRequest)
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

	u, err := getUserFunc(client, id) // DB에서 유저 정보를 가져온다.
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 관리자가 본인의 비밀번호를 변경하는 경우 입력한 비밀번호와 DB에 저장된 비밀번호가 일치하는지 확인
	if token.AccessLevel == AdminLevel && id == token.ID {
		err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(nowPW))
		if err != nil {
			if err == bcrypt.ErrMismatchedHashAndPassword {
				http.Error(w, "현재 비밀번호가 맞지 않습니다. 다시 확인해주세요.", http.StatusInternalServerError)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// 입력받은 새로운 비밀번호를 암호화한다.
	encryptedPW, err := encryptFunc(newPW)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	u.Password = encryptedPW

	// 새로 토큰을 생성하고 User를 Set한다.
	err = u.CreateToken()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = setUserFunc(client, u)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 현재 로그인되어있는 관리자의 비밀번호를 변경하는 경우 Token을 쿠키에 저장한다.
	if token.AccessLevel == AdminLevel && id == token.ID {
		SessionToken := http.Cookie{
			Name:    "SessionToken",
			Value:   u.Token,
			Expires: time.Now().Add(time.Duration(*flagCookieAge) * time.Hour),
		}
		http.SetCookie(w, &SessionToken)
		SessionSignKey := http.Cookie{
			Name:    "SessionSignKey",
			Value:   u.SignKey,
			Expires: time.Now().Add(time.Duration(*flagCookieAge) * time.Hour),
		}
		http.SetCookie(w, &SessionSignKey)
	}

	log := Log{}
	log.UserID = token.ID
	log.CreatedAt = time.Now()
	log.Content = fmt.Sprintf("유저 %s의 비밀번호가 수정되었습니다.", id)

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/changepassword-success", http.StatusSeeOther)
}

// handleChangePasswordSuccessFunc 함수는 User 비밀번호 변경이 성공했다는 페이지를 띄운다.
func handleChangePasswordSuccessFunc(w http.ResponseWriter, r *http.Request) {
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
	err = TEMPLATES.ExecuteTemplate(w, "changepassword-success", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
