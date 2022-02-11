// 프로젝트 결산 프로그램
//
// Description : cmd Admin Setting 관련 스크립트

package main

import (
	"context"
	"log"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func setMonthlyStatusCmdFunc() {
	ms := MonthlyStatus{}
	ms.Date = *flagDate
	status, err := strconv.ParseBool(*flagStatus) // string형으로 받은 값을 bool형으로 변환
	if err != nil {
		log.Fatal(err)
	}
	ms.Status = status

	err = ms.CheckErrorFunc()
	if err != nil {
		log.Fatal(err)
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

	err = setMonthlyStatusFunc(client, ms)
	if err != nil {
		log.Print(err)
	}
}
