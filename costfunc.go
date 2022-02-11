// 프로젝트 결산 프로그램
//
// Description : 프로젝트 재무 관리 툴 내의 비용을 계산하는 함수 모음

package main

import (
	"context"
	"math"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// calMonthlyVFXLaborCostFunc 함수는 프로젝트의 월별 VFX 본부 인건비를 계산하는 함수이다.
func calMonthlyVFXLaborCostFunc(projectID string, year int, month int) (int, error) {
	// mongoDB client 연결
	credential := options.Credential{
		Username: *flagDBID,
		Password: *flagDBPW,
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(*flagMongoDBURI).SetAuth(credential))
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		return 0, err
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return 0, err
	}

	// VFX 아티스트들의 타임로그 정보를 가져온다.
	vfxTimelogs, err := getTimelogOfTheProjectVFXFunc(client, year, month, projectID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return 0, nil
		}
		return 0, err
	}

	monthlyVFXLaborCost := 0.0
	for _, timelog := range vfxTimelogs {
		// VFX 아티스트들의 타임로그 Duration 정보
		artistDuration := math.Round(timelog.Duration/60*10) / 10

		// USER ID에 해당하는 아티스트를 가져온다.
		artist, err := getArtistFunc(client, timelog.UserID)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				continue
			}
			return 0, err
		}
		hourlyWage := 0.0
		if artist.Salary[strconv.Itoa(year)] != "" {
			hourlyWage, err = hourlyWageFunc(artist, year, month) // 시급 계산
			if err != nil {
				return 0, err
			}
		}

		monthlyVFXLaborCost += artistDuration * hourlyWage
	}

	return int(math.Round(monthlyVFXLaborCost)), nil
}

// calMonthlyCMLaborCostFunc 함수는 프로젝트의 월별 CM 본부 인건비를 계산하는 함수이다,
func calMonthlyCMLaborCostFunc(projectID string, year int, month int) (int, error) {
	// mongoDB client 연결
	credential := options.Credential{
		Username: *flagDBID,
		Password: *flagDBPW,
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(*flagMongoDBURI).SetAuth(credential))
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		return 0, err
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return 0, err
	}

	// CM 아티스트들의 타임로그 정보를 가져온다.
	cmTimelogs, err := getTimelogOfTheProjectCMFunc(client, year, month, projectID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return 0, nil
		}
		return 0, err
	}

	monthlyCMLaborCost := 0.0
	for _, value := range cmTimelogs {
		// cm 아티스트의 타임로그 Duration 정보
		artistDuration := math.Round(value.Duration/60*10) / 10

		// USER ID에 해당하는 아티스트를 가져온다
		artist, err := getArtistFunc(client, value.UserID)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				continue
			}
			return 0, err
		}

		hourlyWage := 0.0
		if artist.Salary[strconv.Itoa(year)] != "" {
			hourlyWage, err = hourlyWageFunc(artist, year, month) // 시급 계산
			if err != nil {
				return 0, err
			}
		}

		monthlyCMLaborCost += artistDuration * hourlyWage
	}

	return int(math.Round(monthlyCMLaborCost)), nil
}
