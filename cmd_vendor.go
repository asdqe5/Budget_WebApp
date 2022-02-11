// 프로젝트 결산 프로그램
//
// Description : cmd Vendor 관련 스크립트

package main

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// addVendorCmdFunc 함수는 cmd에서 Vendor를 추가하는 함수이다.
func addVendorCmdFunc() {
	v := Vendor{}

	v.ID = primitive.NewObjectID()
	v.Project = strings.ToUpper(*flagProject) // 프로젝트명
	v.Name = *flagName                        // 벤더명

	// 계약금 및 잔금을 내는 날짜가 정해진 경우 입력
	v.Downpayment.Date = *flagStartDate
	v.Balance.Date = *flagEndDate

	err := v.CheckErrorFunc()
	if err != nil {
		log.Fatal(err)
	}

	// 총 지출 암호화
	if *flagPayment == 0 {
		log.Fatal("총 지출을 입력해주세요")
	}
	expenses, err := encryptAES256Func(strconv.Itoa(*flagPayment))
	if err != nil {
		log.Fatal(err)
	}
	v.Expenses = expenses

	// 계약금 지출 날짜와 잔금 지출 날짜 비교
	if v.Downpayment.Date != "" && v.Balance.Date != "" {
		dpDate, err := time.Parse("2006-01-02", v.Downpayment.Date)
		if err != nil {
			log.Fatal(err)
		}
		balDate, err := time.Parse("2006-01-02", v.Balance.Date)
		if err != nil {
			log.Fatal(err)
		}
		if balDate.Before(dpDate) {
			log.Fatal("잔금 지출 날짜가 계약금 지출 날짜 전으로 입력되었습니다")
		}
	}

	// 컷수와 태스크가 정해진 경우 입력
	v.Cuts = *flagCuts
	if *flagTasks != "" {
		// 태스크는 ,로 구분하여 입력한다.
		v.Tasks = stringToListFunc(*flagTasks, ",")
		sort.Strings(v.Tasks)
	}

	// mongoDB client 연결
	credential := options.Credential{
		Username: *flagDBID,
		Password: *flagDBPW,
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(*flagMongoDBURI).SetAuth(credential))
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Fatal(err)
	}

	// 프로젝트가 존재하는지 체크
	project, err := getProjectFunc(client, v.Project)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Fatal(fmt.Sprintf("%s 프로젝트가 존재하지 않습니다. 프로젝트를 먼저 추가해주세요.", v.Project))
		}
		log.Fatal(err)
	}
	v.ProjectName = project.Name

	if *flagDBIP != "" {
		// 입력받은 DB IP의 형식이 맞는지 확인
		if !regexIPv4.MatchString(*flagDBIP) {
			log.Fatal(err)
		}
	}

	// 벤더 정보를 DB에 추가한다.
	err = addVendorFunc(client, v)
	if err != nil {
		log.Print(err)
		return
	}
}

// rmVendorCmdFunc 함수는 cmd에서 Vendor를 삭제하는 함수이다.
func rmVendorCmdFunc() {
	if *flagProject == "" {
		log.Fatal("삭제할 벤더의 프로젝트ID를 입력해주세요")
	}
	if *flagName == "" {
		log.Fatal("삭제할 벤더명을 입력해주세요")
	}

	project := strings.ToUpper(*flagProject)
	if !regexProject.MatchString(project) {
		log.Fatal("프로젝트 ID에는 영문(대문자), 숫자, 특수문자(_)만 입력 가능합니다")
	}
	name := *flagName

	// mongoDB client 연결
	credential := options.Credential{
		Username: *flagDBID,
		Password: *flagDBPW,
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(*flagMongoDBURI).SetAuth(credential))
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Fatal(err)
	}

	if *flagDBIP != "" {
		// 입력받은 DB IP의 형식이 맞는지 확인
		if !regexIPv4.MatchString(*flagDBIP) {
			log.Fatal(err)
		}
	}

	err = rmVendorFunc(client, project, name, "")
	if err != nil {
		log.Print(err)
		return
	}
}

// searchVendorCmdFunc 함수는 cmd에서 Vendor를 검색하는 함수이다.
func searchVendorCmdFunc() {
	// 프로젝트ID 혹은 벤더명으로는 검색을 하도록 체크
	if *flagProject == "" && *flagName == "" {
		log.Fatal("프로젝트 ID, 벤더명 중에 하나는 입력해주세요")
	}
	// mongoDB client 연결
	credential := options.Credential{
		Username: *flagDBID,
		Password: *flagDBPW,
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(*flagMongoDBURI).SetAuth(credential))
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Fatal(err)
	}

	if *flagDBIP != "" {
		// 입력받은 DB IP의 형식이 맞는지 확인
		if !regexIPv4.MatchString(*flagDBIP) {
			log.Fatal(err)
		}
	}

	searchWord := ""
	if *flagProject != "" {
		searchWord = searchWord + "project:" + strings.ToUpper(*flagProject)
	}
	if *flagName != "" {
		searchWord = searchWord + " name:" + *flagName
	}

	vendors, err := searchVendorFunc(client, searchWord)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(vendors)
}
