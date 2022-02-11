// 프로젝트 결산 프로그램
//
// Description : cmd Project 관련 스크립트

package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func addProjectCmdFunc() {
	p := Project{}

	p.ID = strings.ToUpper(*flagID)
	p.Name = *flagName
	p.StartDate = *flagStartDate
	p.SMEndDate = *flagEndDate

	// 총매출(계약금) 암호화
	var err error
	if *flagPayment == 0 {
		log.Fatal("총매출을 입력해주세요")
	}
	payment := strconv.Itoa(*flagPayment)
	var pay Payment
	pay.Date = fmt.Sprintf("%04d-%02d-%02d", time.Now().Year(), time.Now().Month(), time.Now().Day())
	pay.Expenses, err = encryptAES256Func(payment)
	if err != nil {
		log.Fatal(err)
	}
	p.Payment = append(p.Payment, pay)

	err = p.CheckErrorFunc()
	if err != nil {
		log.Fatal(err)
	}

	// 입력 받은 작업시작일과 작업마감일을 비교한다.
	sDate, err := time.Parse("2006-01", p.StartDate)
	eDate, err := time.Parse("2006-01", p.SMEndDate)
	if eDate.Before(sDate) {
		log.Fatal("작업마감일이 작업시작일 전으로 입력되었습니다. 다시 확인해주세요.")
	}

	if *flagIsFinished == true {
		p.IsFinished = true
		if *flagTotalAmount == 0 {
			log.Fatal("정산 완료된 프로젝트의 총 내부비용을 입력해주세요")
		}
		totalamount := strconv.Itoa(*flagTotalAmount)
		p.TotalAmount, err = encryptAES256Func(totalamount)
		if err != nil {
			log.Fatal(err)
		}
		if *flagLaborCost != 0 {
			laborcost := strconv.Itoa(*flagLaborCost)
			p.FinishedCost.LaborCost.VFX, err = encryptAES256Func(laborcost)
			if err != nil {
				log.Fatal(err)
			}
		}
		if *flagProgressCost != 0 {
			progresscost := strconv.Itoa(*flagProgressCost)
			p.FinishedCost.ProgressCost, err = encryptAES256Func(progresscost)
			if err != nil {
				log.Fatal(err)
			}
		}
		if *flagPurchaseCost != 0 {
			purchasecost := strconv.Itoa(*flagPurchaseCost)
			p.FinishedCost.PurchaseCost, err = encryptAES256Func(purchasecost)
			if err != nil {
				log.Fatal(err)
			}
		}
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

	err = addProjectFunc(client, p)
	if err != nil {
		log.Print(err)
		return
	}
}

func rmProjectCmdFunc() {
	id := strings.ToUpper(*flagID)
	if id == "" {
		log.Fatal("삭제할 프로젝트의 ID를 입력해주세요")
	} else if !regexProject.MatchString(id) {
		log.Fatal("프로젝트 ID에는 영문(대문자), 숫자, 특수문자(_)만 입력 가능합니다")
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

	err = rmProjectFunc(client, id)
	if err != nil {
		log.Print(err)
	}
}

func getProjectCmdFunc() {
	id := strings.ToUpper(*flagID)
	if id == "" {
		log.Fatal("가져올 프로젝트의 ID를 입력해주세요")
	} else if !regexProject.MatchString(id) {
		log.Fatal("프로젝트 ID에는 영문(대문자), 숫자, 특수문자(_)만 입력 가능합니다")
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

	project, err := getProjectFunc(client, id)
	if err != nil {
		log.Print(err)
	}

	fmt.Println(project)
}

func setProjectCmdFunc() {
	id := strings.ToUpper(*flagID)
	if id == "" {
		log.Fatal("프로젝트의 ID를 입력해주세요")
	} else if !regexProject.MatchString(id) {
		log.Fatal("프로젝트 ID에는 영문(대문자), 숫자, 특수문자(_)만 입력 가능합니다")
	}

	//mongoDB client 연결
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

	project, err := getProjectFunc(client, id)
	if err != nil {
		log.Fatal(err)
	}
	if *flagName != "" {
		project.Name = *flagName
	}
	project.Payment = nil
	if *flagPayment != 0 {
		payment := strconv.Itoa(*flagPayment)
		var pay Payment
		pay.Date = fmt.Sprintf("%04d-%02d-%02d", time.Now().Year(), time.Now().Month(), time.Now().Day())
		pay.Expenses, err = encryptAES256Func(payment)
		if err != nil {
			log.Fatal(err)
		}
		project.Payment = append(project.Payment, pay)
	}

	err = setProjectFunc(client, project)
	if err != nil {
		log.Fatal(err)
	}
}

func searchProjectCmdFunc() {
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
	if *flagID != "" {
		searchWord = searchWord + " id:" + strings.ToUpper(*flagID)
	}
	if *flagName != "" {
		searchWord = searchWord + " name:" + *flagName
	}
	if *flagDate != "" {
		searchWord = searchWord + " date:" + *flagDate
	}

	projects, err := searchProjectFunc(client, searchWord, "")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(projects)
}

// updateProjectCmdFunc 함수는 프로젝트의 자료구조가 바뀌었을 때 예전 자료구조에서 새롱누 자료구조로 데이터를 업데이트시켜주는 함수이다.
func updateProjectCmdFunc() {
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

	idList, err := getIDOfProjectsFunc(client)
	if err != nil {
		log.Fatal(err)
	}

	for _, id := range idList {
		op, err := getOldProjectFunc(client, id)
		if err != nil {
			log.Print(id + " >> " + err.Error())
			continue
		}
		var project Project

		project.ID = op.ID
		project.Name = op.Name
		var payment Payment
		payment.Expenses = op.Payment
		project.Payment = append(project.Payment, payment)
		project.StartDate = op.StartDate
		project.SMEndDate = op.SMEndDate
		project.DirectorName = op.DirectorName
		project.ProducerName = op.ProducerName
		project.IsFinished = op.IsFinished
		project.TotalAmount = op.TotalAmount
		project.FinishedCost = op.FinishedCost
		project.SMStatus = op.SMStatus
		project.SMMonthlyPayment = make(map[string][]Payment)
		project.SMMonthlyProgressCost = op.SMMonthlyProgressCost
		project.SMMonthlyLaborCost = op.SMMonthlyLaborCost
		project.SMMonthlyPurchaseCost = op.SMMonthlyPurchaseCost
		project.SMDifference = op.SMDifference

		// 월별 매출
		for date, monthlyPayment := range op.SMMonthlyPayment {
			var p Payment
			p.Date = date + "-01"
			p.Expenses = monthlyPayment
			project.SMMonthlyPayment[date] = append(project.SMMonthlyPayment[date], p)
		}

		// 프로젝트 삭제
		err = rmProjectFunc(client, op.ID)
		if err != nil {
			log.Fatal(err)
		}

		// 프로젝트 추가
		err = addProjectFunc(client, project)
		if err != nil {
			log.Fatal(err)
		}

		log.Print(project.ID, "  updated")
	}
}
