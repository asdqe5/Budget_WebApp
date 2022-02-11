package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// rmArtistCmdFunc 함수는 cmd를 통해 아티스트를 삭제하는 함수이다.
func rmArtistCmdFunc() {
	id := *flagID
	if id == "" {
		log.Fatal("삭제할 아티스트의 ID를 입력해주세요")
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

	err = rmArtistFunc(client, id)
	if err != nil {
		log.Print(err)
	}
}

// getArtistCmdFunc 함수는 cmd를 통해 id가 일치하는 아티스트를 가져오는 함수이다.
func getArtistCmdFunc() {
	id := *flagID
	if id == "" {
		log.Fatal("가져올 아티스트의 ID를 입력해주세요")
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

	artist, err := getArtistFunc(client, id)
	if err != nil {
		log.Print(err)
	}

	// 아티스트가 존재하면 출력해준다.
	if artist.ID != "" {
		fmt.Println(artist)
	}
}

// searchArtistCmdFunc 함수는 cmd를 통해 아티스트를 검색하는 함수이다.
func searchArtistCmdFunc() {
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
		searchWord = searchWord + " id:" + *flagID
	}
	if *flagName != "" {
		searchWord = searchWord + "name:" + *flagName
	}
	if *flagDept != "" {
		searchWord = searchWord + "dept:" + *flagDept
	}
	if *flagTeam != "" {
		searchWord = searchWord + "team:" + *flagTeam
	}

	artists, err := searchArtistFunc(client, searchWord)
	if err != nil {
		log.Print(err)
	}

	fmt.Println(artists)
}

// setResinationCmdFunc 함수는 아티스트의 퇴사일과 현재 날짜를 비교하여 퇴사 여부를 설정하는 함수이다.
func setResinationCmdFunc() {
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

	artists, err := getAllArtistFunc(client)
	if err != nil {
		log.Fatal(err)
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
				log.Fatal(err)
			}
		}
	}
}
