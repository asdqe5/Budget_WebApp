// 프로젝트 결산 프로그램
//
// Description : cmd User 관련 스크립트

package main

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func addUserCmdFunc() {
	u := User{}

	u.ID = *flagID
	u.Password = *flagPW
	u.Name = *flagName
	u.Team = *flagTeam
	u.AccessLevel = AccessLevel(*flagAccessLevel)
	err := u.CheckError()
	if err != nil {
		log.Print(err)
		return
	}

	encryptedPW, err := encryptFunc(*flagPW)
	if err != nil {
		log.Print(err)
		return
	}
	u.Password = encryptedPW
	err = u.CreateToken()
	if err != nil {
		log.Print(err)
		return
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

	err = addUserFunc(client, u)
	if err != nil {
		log.Print(err)
		return
	}
}
