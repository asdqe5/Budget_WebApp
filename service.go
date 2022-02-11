// 프로젝트 결산 프로그램
//
// Description : 서비스 관련 스크립트

package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/smtp"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"gopkg.in/robfig/cron.v2"
)

// serviceFunc 함수는 정기적으로 돌아야하는 서비스를 설정하는 함수이다.
func serviceFunc() {
	c := cron.New()

	// 매일 자정에 아티스트의 퇴사일을 확인하여 퇴사 여부를 설정하는 서비스
	c.AddFunc("@daily", func() {
		log.Println("아티스트 퇴사 여부 설정 서비스 실행")
		setResinationFunc()
	})

	// 매일 오전 10시에 프로젝트 발행일을 확인하여 메일을 보내는 서비스
	c.AddFunc("0 10 * * *", func() {
		log.Println("프로젝트 매출 발행일 메일 서비스 실행")
		sendMailForProjectFunc()
	})

	// 매일 오전 10시에 벤더 발행일을 확인하여 메일을 보내는 서비스
	c.AddFunc("0 10 * * *", func() {
		log.Println("벤더 비용 발행일 메일 서비스 실행")
		sendMailForVendorFunc()
	})

	c.Start()
}

// setResinationFunc 함수는 아티스트의 퇴사일과 현재 날짜를 비교하여 퇴사 여부를 설정하는 함수이다.
func setResinationFunc() {
	// mongoDB client 연결
	credential := options.Credential{
		Username: *flagDBID,
		Password: *flagDBPW,
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(*flagMongoDBURI).SetAuth(credential))
	if err != nil {
		log.Print(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		log.Print(err)
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Print(err)
	}

	if *flagDBIP != "" {
		// 입력받은 DB IP의 형식이 맞는지 확인
		if !regexIPv4.MatchString(*flagDBIP) {
			log.Print(err)
		}
	}

	artists, err := getAllArtistFunc(client)
	if err != nil {
		log.Print(err)
	}

	for _, artist := range artists {
		if artist.EndDay == "" { // 아티스트의 퇴사일이 비어있으면 continue
			continue
		}
		t, _ := time.Parse("2006-01-02", artist.EndDay)
		duration := time.Now().Sub(t).Hours() / 24
		if duration > 0 {
			err = setArtistFunc(client, artist)
			if err != nil {
				log.Print(err)
			}
		}
	}
}

// sendMailForVendorFunc 함수는 벤더 비용 발행일을 확인하여 오늘 날짜이면 메일을 보내는 함수이다.
func sendMailForVendorFunc() {
	// mongoDB client 연결
	credential := options.Credential{
		Username: *flagDBID,
		Password: *flagDBPW,
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(*flagMongoDBURI).SetAuth(credential))
	if err != nil {
		log.Print(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		log.Print(err)
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Print(err)
	}

	if *flagDBIP != "" {
		// 입력받은 DB IP의 형식이 맞는지 확인
		if !regexIPv4.MatchString(*flagDBIP) {
			log.Print(err)
		}
	}

	// 벤더 비용이 오늘인 벤더 정보를 가져온다.
	vendors, err := getVendorsByTodayFunc(client)
	if err != nil {
		log.Print(err)
	}
	if len(vendors) == 0 { // 벤더 정보가 없는 경우 서비스를 리턴한다.
		return
	}

	// 이메일을 보낼 그룹웨어 ID를 가져온다.
	adminsetting, err := getAdminSettingFunc(client)
	if err != nil {
		log.Print(err)
	}
	if len(adminsetting.GWIDs) == 0 { // 메일을 보낼 그룹웨어 ID 정보가 없는 경우 서비스를 리턴한다.
		return
	}

	// 메시지 설정
	from := "BUDGET"
	to := adminsetting.GWIDs

	smtpHost := "gw.rd101.co.kr"
	smtpPort := "25"

	var msg []byte
	var body bytes.Buffer

	// Set Header
	headerSubject := "Subject: " + encodeRFC2047Func("[BUDGET] 벤더 세금계산서 발행일 알람") + "\r\n"
	headerFrom := "From: BUDGET\r\n"
	headerTo := "To: " + listToStringFunc(to, false) + "\r\n"
	msg = append(msg, []byte(headerSubject+headerFrom+headerTo)...)

	// Set Body
	type Recipe struct {
		Vendors []Vendor
		Date    string
	}
	rcp := Recipe{}
	rcp.Vendors = vendors
	rcp.Date = time.Now().Format("2006-01-02") // 오늘 날짜

	mimeHeaders := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body.Write([]byte(fmt.Sprintf("%s", mimeHeaders)))

	// 템플릿 로딩
	vfsTemplate, err := loadTemplatesFunc()
	if err != nil {
		log.Fatal(err)
	}
	TEMPLATES = vfsTemplate
	err = TEMPLATES.ExecuteTemplate(&body, "mail-vendor", rcp)
	if err != nil {
		log.Fatal(err)
		return
	}
	msg = append(msg, body.Bytes()...)

	// 메시지 보내기
	err = smtp.SendMail(smtpHost+":"+smtpPort, nil, from, to, msg)
	if err != nil {
		log.Println(err)
	}
}

// sendMailForProjectFunc 함수는 프로젝트 발행일을 확인하여 오늘 날짜이면 메일을 보내는 함수이다.
func sendMailForProjectFunc() {
	// mongoDB client 연결
	credential := options.Credential{
		Username: *flagDBID,
		Password: *flagDBPW,
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(*flagMongoDBURI).SetAuth(credential))
	if err != nil {
		log.Print(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		log.Print(err)
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Print(err)
	}

	if *flagDBIP != "" {
		// 입력받은 DB IP의 형식이 맞는지 확인
		if !regexIPv4.MatchString(*flagDBIP) {
			log.Print(err)
		}
	}

	projects, err := getProjectsByTodayFunc(client)
	if err != nil {
		log.Print(err)
	}
	if len(projects) == 0 { // 프로젝트 정보가 없는 경우 서비스를 리턴한다.
		return
	}

	// 이메일을 보낼 그룹웨어 ID를 가져온다.
	adminsetting, err := getAdminSettingFunc(client)
	if err != nil {
		log.Print(err)
	}
	if len(adminsetting.GWIDsForProject) == 0 { // 메일을 보낼 그룹웨어 ID 정보가 없는 경우 서비스를 리턴한다.
		return
	}

	// 메시지 설정
	from := "BUDGET"
	to := adminsetting.GWIDsForProject

	smtpHost := "gw.rd101.co.kr"
	smtpPort := "25"

	var msg []byte
	var body bytes.Buffer

	// Set Header
	headerSubject := "Subject: " + encodeRFC2047Func("[BUDGET] 프로젝트 세금계산서 발행일 알람") + "\r\n"
	headerFrom := "From: BUDGET\r\n"
	headerTo := "To: " + listToStringFunc(to, false) + "\r\n"
	msg = append(msg, []byte(headerSubject+headerFrom+headerTo)...)

	// Set Body
	type ProjectInfo struct {
		Name string   // 프로젝트 이름
		Type []string // 발행일이 오늘인 매출의 타입
	}

	type Recipe struct {
		Projects []ProjectInfo
		Date     string
	}
	rcp := Recipe{}
	rcp.Date = time.Now().Format("2006-01-02") // 오늘 날짜
	month := dateToMonthFunc(rcp.Date)
	for _, project := range projects {
		var typeList []string
		for _, payment := range project.SMMonthlyPayment[month] {
			if payment.Date == rcp.Date {
				typeList = append(typeList, payment.Type)
			}
		}
		pi := ProjectInfo{
			Name: project.Name,
			Type: typeList,
		}
		rcp.Projects = append(rcp.Projects, pi)
	}

	mimeHeaders := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body.Write([]byte(fmt.Sprintf("%s", mimeHeaders)))

	// 템플릿 로딩
	vfsTemplate, err := loadTemplatesFunc()
	if err != nil {
		log.Fatal(err)
	}
	TEMPLATES = vfsTemplate
	err = TEMPLATES.ExecuteTemplate(&body, "mail-project", rcp)
	if err != nil {
		log.Fatal(err)
		return
	}
	msg = append(msg, body.Bytes()...)

	// 메시지 보내기
	err = smtp.SendMail(smtpHost+":"+smtpPort, nil, from, to, msg)
	if err != nil {
		log.Println(err)
	}
}
