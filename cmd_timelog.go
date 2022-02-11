package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// addTimelogCmdFunc 함수는 cmd를 통해 타임로그를 추가하는 함수이다.
func addTimelogCmdFunc() {
	t := Timelog{}
	t.UserID = *flagID
	t.Year = *flagYear
	t.Month = *flagMonth
	t.Project = *flagProject
	t.Duration = *flagDuration

	err := t.CheckErrorFunc()
	if err != nil {
		log.Fatal(err)
	}

	t.Duration = t.Duration * 60

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

	err = addTimelogFunc(client, t)
	if err != nil {
		log.Print(err)
	}
}

// updateTimelogCmdFunc 함수는 cmd를 통해 타임로그 데이터를 업데이트하는 함수이다.
func updateTimelogCmdFunc() {
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

	// DB에서 Admin setting 데이터를 가져온다.
	adminSetting, err := getAdminSettingFunc(client)
	if err != nil {
		log.Fatal(err)
	}

	// DB에서 MonthlyStatus 데이터를 가져온다.
	var checkStatus bool
	y, m, _ := time.Now().Date()
	ld := time.Now().AddDate(0, -1, 0)
	thisdate := fmt.Sprintf("%04d-%02d", y, m)
	thisMonthStatus, err := getMonthlyStatusFunc(client, thisdate)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			thisMonthStatus.Date = thisdate
			thisMonthStatus.Status = false
			err = thisMonthStatus.CheckErrorFunc()
			if err != nil {
				log.Fatal(err)
			}
			err = setMonthlyStatusFunc(client, thisMonthStatus)
			if err != nil {
				log.Print(err)
			}
		} else {
			log.Fatal(err)
		}
	}
	if thisMonthStatus.Status {
		log.Fatal("이번 달의 결산이 완료되었습니다.\nAdminSetting을 확인해주세요.")
	} else {
		lastdate := fmt.Sprintf("%04d-%02d", ld.Year(), ld.Month())
		lastMonthStatus, err := getMonthlyStatusFunc(client, lastdate)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				lastMonthStatus.Date = lastdate
				lastMonthStatus.Status = false
				err = lastMonthStatus.CheckErrorFunc()
				if err != nil {
					log.Fatal(err)
				}
				err = setMonthlyStatusFunc(client, lastMonthStatus)
				if err != nil {
					log.Print(err)
				}
			} else {
				log.Fatal(err)
			}
		}
		if lastMonthStatus.Status {
			checkStatus = true
		} else {
			checkStatus = false
		}
	}

	excludeID := adminSetting.SGExcludeID
	excludeProjects := adminSetting.SGExcludeProjects
	taskProjects := adminSetting.TaskProjects
	updateErr := ""
	lasttimelogID := "0" // 마지막 타임로그 아이디 값이 들어갈 변수
	var timelogList []Timelog
	for { // 업데이트할 타임로그가 없을때까지 반복
		timelogs, timelogID, err := sgGetTimelogsFunc(lasttimelogID, excludeID, excludeProjects, taskProjects, checkStatus)
		lasttimelogID = timelogID
		if err != nil {
			updateErr = fmt.Sprintf("%s", err)
			break
		}
		if len(timelogs) == 0 {
			break
		}

		for _, t := range timelogs {
			err = t.CheckErrorFunc()
			if err != nil {
				updateErr = fmt.Sprintf("%s", err)
				break
			}
			if timelogList == nil { // 처음에 timelogList가 비어있을 경우
				timelogList = append(timelogList, t)
				continue
			}
			for i, l := range timelogList {
				if t.UserID == l.UserID && t.Project == l.Project && t.Year == l.Year && t.Month == l.Month { // user id와 project name이 같은 경우
					timelogList[i].Duration = l.Duration + t.Duration
					break
				} else if i == len(timelogList)-1 {
					timelogList = append(timelogList, t) // 마지막까지 user id와 project name이 같지 않은 경우
					break
				}
			}
		}
		if updateErr != "" {
			break
		}
	}

	// DB에 저장하기 전에 VFX 타임로그 정보를 삭제한다.
	if checkStatus {
		err = rmVFXTimelogFunc(client, y, int(m), adminSetting.SMSupervisorIDs)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		err = rmVFXTimelogFunc(client, y, int(m), adminSetting.SMSupervisorIDs)
		if err != nil {
			log.Fatal(err)
		}
		err = rmVFXTimelogFunc(client, ld.Year(), int(ld.Month()), adminSetting.SMSupervisorIDs)
		if err != nil {
			log.Fatal(err)
		}
	}

	for _, t := range timelogList { // timelogList를 db에 저장
		err = addTimelogFunc(client, t)
		if err != nil {
			updateErr = fmt.Sprintf("%s", err)
			break
		}
		if updateErr != "" {
			break
		}
	}

	adminSetting.SGUpdatedTime = time.Now().Format(time.RFC3339) // admin setting의 업데이트 시간을 현재 시간으로 설정
	err = updateAdminSettingFunc(client, adminSetting)
	if err != nil {
		log.Fatal(err)
	}

	if updateErr != "" {
		log.Fatal(updateErr)
	}
}

// rmTimelogCmdFunc 함수는 cmd를 통해 타임로그 데이터를 삭제하는 함수이다.
func rmTimelogCmdFunc() {
	t := Timelog{}
	t.UserID = *flagID
	t.Year = *flagYear
	t.Month = *flagMonth
	t.Project = *flagProject

	err := t.CheckErrorFunc()
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

	err = rmTimelogFunc(client, t)
	if err != nil {
		log.Print(err)
	}
}

// subTimelogCmdFunc 함수는 cmd를 통해 타임로그의 duration 에서 *flagDuration만큼 빼는 함수이다.
func subTimelogCmdFunc() {
	t := Timelog{}
	t.UserID = *flagID
	t.Year = *flagYear
	t.Month = *flagMonth
	t.Project = *flagProject
	t.Duration = *flagDuration

	err := t.CheckErrorFunc()
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

	err = subTimelogFunc(client, t)
	if err != nil {
		log.Print(err)
	}
}

// getTimelogCmdFunc 함수는 cmd를 통해 입력한 아티스트, 날짜, 프로젝트 정보가 일치하는 타임로그를 가져오는 함수이다.
func getTimelogCmdFunc() {
	t := Timelog{}
	t.UserID = *flagID
	t.Year = *flagYear
	t.Month = *flagMonth
	t.Project = *flagProject

	err := t.CheckErrorFunc()
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

	timelog, err := getTimelogFunc(client, t.UserID, t.Year, t.Month, t.Project)
	if err != nil {
		log.Print(err)
	}

	// 타임로그가 존재하면 출력해준다.
	if timelog.UserID != "" {
		fmt.Println(timelog)
	}
}

// searchTimelogCmdFunc 함수는 cmd를 통해 타임로그를 검색하는 함수이다.
func searchTimelogCmdFunc() {
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
		searchWord = searchWord + " userid:" + *flagID
	}
	if *flagQuarter != 0 {
		quarter := strconv.Itoa(*flagQuarter)
		searchWord = searchWord + " quarter:" + quarter
	}
	if *flagYear != 0 {
		year := strconv.Itoa(*flagYear)
		searchWord = searchWord + " year:" + year
	}
	if *flagMonth != 0 {
		month := strconv.Itoa(*flagMonth)
		searchWord = searchWord + " month:" + month
	}
	if *flagProject != "" {
		searchWord = searchWord + " project:" + *flagProject
	}

	timelogs, err := searchTimelogFunc(client, searchWord)
	if err != nil {
		log.Print(err)
	}

	fmt.Println(timelogs)
}

// getAllTimelogCmdFunc 함수는 현재 샷건의 모든 타임로그 정보를 가져오는 함수이다.
func resetAllTimelogCmdFunc() {
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

	// DB에서 Admin setting 데이터를 가져온다.
	adminSetting, err := getAdminSettingFunc(client)
	if err != nil {
		log.Fatal(err)
	}

	excludeID := adminSetting.SGExcludeID
	excludeProjects := adminSetting.SGExcludeProjects
	taskProjects := adminSetting.TaskProjects
	updateErr := ""
	lasttimelogID := "0"      // 마지막 타임로그 아이디 값이 들어갈 변수
	var timelogList []Timelog // 타임로그 정리
	for {                     // 업데이트할 타임로그가 없을때까지 반복
		timelogs, timelogID, err := sgResetTimelogsFunc(lasttimelogID, excludeID, excludeProjects, taskProjects)
		lasttimelogID = timelogID
		if err != nil {
			updateErr = fmt.Sprintf("%s", err)
			break
		}
		if len(timelogs) == 0 {
			break
		}

		for _, t := range timelogs {
			err = t.CheckErrorFunc()
			if err != nil {
				updateErr = fmt.Sprintf("%s", err)
				break
			}
			if timelogList == nil { // 처음에 timelogList가 비어있을 경우
				timelogList = append(timelogList, t)
				continue
			}
			for i, l := range timelogList {
				if t.UserID == l.UserID && t.Project == l.Project && t.Year == l.Year && t.Month == l.Month { // id, project, year, month가 같은 경우
					timelogList[i].Duration = l.Duration + t.Duration
					break
				} else if i == len(timelogList)-1 {
					timelogList = append(timelogList, t) // 마지막까지 user id와 project name이 같지 않은 경우
					break
				}
			}
		}
		if updateErr != "" {
			break
		}
	}

	// DB에 저장하기 전에 VFX 타임로그 정보를 삭제한다.
	err = rmVFXAllTimelogFunc(client, adminSetting.SMSupervisorIDs)
	if err != nil {
		log.Fatal(err)
		return
	}

	for _, t := range timelogList { // timelogList(모든)를 db에 저장
		err = addTimelogFunc(client, t)
		if err != nil {
			updateErr = fmt.Sprintf("%s", err)
			break
		}
		if updateErr != "" {
			break
		}
	}

	adminSetting.SGUpdatedTime = time.Now().Format(time.RFC3339)
	err = updateAdminSettingFunc(client, adminSetting)
	if err != nil {
		log.Fatal(err)
	}

	if updateErr != "" {
		log.Fatal(updateErr)
	}

}
