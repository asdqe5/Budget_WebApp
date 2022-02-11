package main

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// addArtistCMCmdFunc 함수는 cmd를 통해 CM 아티스트를 추가하는 함수이다.
func addArtistCMCmdFunc() {
	a := Artist{}
	a.ID = changeToCMIDFunc(*flagID)
	a.Name = *flagName
	a.Dept = "cm"
	a.Team = *flagTeam
	a.EndDay = *flagEndDate
	a.Resination = *flagResination

	if *flagSalary != "" {
		if !regexSalary.MatchString(*flagSalary) {
			log.Fatal("salary가 2019:2400,2020:2400 형식이 아닙니다")
		}
	}
	a.Salary, _ = stringToMapFunc(*flagSalary)

	// 연봉 암호화
	for key, value := range a.Salary {
		encrypted, err := encryptAES256Func(value)
		a.Salary[key] = encrypted
		if err != nil {
			log.Fatal(err)
		}
	}

	err := a.CheckErrorFunc()
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

	err = addArtistFunc(client, a)
	if err != nil {
		log.Print(err)
	}
}
