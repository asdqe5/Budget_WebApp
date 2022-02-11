// 프로젝트 결산 프로그램
//
// Description : DB CM 타임로그 관련 스크립트

package main

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// getTimelogCMFunc 함수는 CM팀에서 입력받은 날짜에 작성한 타임로그를 반환하는 함수이다.
func getTimelogOfTheMonthCMFunc(client *mongo.Client, year int, month int) ([]Timelog, error) {
	collection := client.Database(*flagDBName).Collection("timelogs")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var results []Timelog
	q := bson.M{"userid": primitive.Regex{Pattern: "^cm", Options: "i"}, "year": year, "month": month}
	opts := options.Find()
	cursor, err := collection.Find(ctx, q, opts)
	if err != nil {
		return results, err
	}
	err = cursor.All(ctx, &results)
	if err != nil {
		return nil, err
	}
	return results, nil
}

// rmCMTimelogFunc 함수는 DB에서 CM 아티스트의 타임로그만 삭제하는 함수이다.
func rmCMTimelogFunc(client *mongo.Client, year int, month int) error {
	collection := client.Database(*flagDBName).Collection("timelogs")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := collection.DeleteMany(ctx, bson.M{"userid": primitive.Regex{Pattern: "^cm", Options: "i"}, "year": year, "month": month})
	if err != nil {
		return err
	}
	return nil
}

// getTimelogUntilTheMonthCMFunc 함수는 타임로그 누계 페이지에서 입력받은 달까지 작성한 CM 아티스트의 타임로그를 반환하는 함수이다,
func getTimelogUntilTheMonthCMFunc(client *mongo.Client, year int, month int) ([]Timelog, error) {
	collection := client.Database(*flagDBName).Collection("timelogs")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var results []Timelog
	q := bson.M{"userid": primitive.Regex{Pattern: "^cm", Options: "i"}, "year": year, "month": bson.M{"$lte": month}}
	opts := options.Find()
	cursor, err := collection.Find(ctx, q, opts)
	if err != nil {
		return results, err
	}
	err = cursor.All(ctx, &results)
	if err != nil {
		return nil, err
	}
	return results, nil
}

// getTimelogOfTheProjectCMFunc 함수는 내부 인건비 계산을 위한 해당 프로젝트의 월 타임로그를 반환하는 함수이다,
func getTimelogOfTheProjectCMFunc(client *mongo.Client, year int, month int, project string) ([]Timelog, error) {
	collection := client.Database(*flagDBName).Collection("timelogs")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var results []Timelog
	q := bson.M{"userid": primitive.Regex{Pattern: "^cm", Options: "i"}, "year": year, "month": month, "project": project}
	opts := options.Find()
	cursor, err := collection.Find(ctx, q, opts)
	if err != nil {
		return results, err
	}
	err = cursor.All(ctx, &results)
	if err != nil {
		return nil, err
	}
	return results, nil
}
