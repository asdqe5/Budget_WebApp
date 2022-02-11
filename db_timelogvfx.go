// 프로젝트 결산 프로그램
//
// Description : DB VFX 타임로그 관련 스크립트

package main

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// getTimelogOfTheMonthVFXFunc 함수는 VFX팀에서 입력받은 날짜에 작성한 타임로그를 반환하는 함수이다.
func getTimelogOfTheMonthVFXFunc(client *mongo.Client, year int, month int) ([]Timelog, error) {
	collection := client.Database(*flagDBName).Collection("timelogs")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var results []Timelog
	q := bson.M{"userid": primitive.Regex{Pattern: "^(?!cm)", Options: "i"}, "year": year, "month": month}
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

// getTimelogOfTheProjectVFXFunc 함수는 VFX팀에서 입력받은 날짜와 프로젝트에 해당하는 타임로그를 반환하는 함수이다.
func getTimelogOfTheProjectVFXFunc(client *mongo.Client, year int, month int, project string) ([]Timelog, error) {
	collection := client.Database(*flagDBName).Collection("timelogs")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var results []Timelog
	q := bson.M{"userid": primitive.Regex{Pattern: "^(?!cm)", Options: "i"}, "year": year, "month": month, "project": project}
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

// rmVFXTimelogFunc 함수는 DB에서 입력받은 날짜에 작성한 VFX 아티스트의 타임로그만 삭제하는 함수이다
func rmVFXTimelogFunc(client *mongo.Client, year int, month int, sup []string) error {
	collection := client.Database(*flagDBName).Collection("timelogs")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	queries := []bson.M{}
	queries = append(queries, bson.M{"year": year})
	queries = append(queries, bson.M{"month": month})
	queries = append(queries, bson.M{"userid": primitive.Regex{Pattern: "^(?!cm)", Options: "i"}})

	if sup != nil {
		queries = append(queries, bson.M{"userid": bson.M{"$nin": sup}})
	}

	query := bson.M{"$and": queries}

	_, err := collection.DeleteMany(ctx, query)
	if err != nil {
		return err
	}

	return nil
}

// rmVFXAllTimelogFunc 함수는 DB에서 VFX 아티스트의 타임로그만 삭제하는 함수이다.
func rmVFXAllTimelogFunc(client *mongo.Client, sup []string) error {
	collection := client.Database(*flagDBName).Collection("timelogs")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	queries := []bson.M{}
	queries = append(queries, bson.M{"userid": primitive.Regex{Pattern: "^(?!cm)", Options: "i"}})

	if sup != nil {
		queries = append(queries, bson.M{"userid": bson.M{"$nin": sup}})
	}

	query := bson.M{"$and": queries}

	_, err := collection.DeleteMany(ctx, query)
	if err != nil {
		return err
	}
	return nil
}

// getTimelogUntilTheMonthVFXFunc 함수는 타임로그 누계 페이지에서 입력받은 달까지 작성한 VFX 아티스트의 타임로그를 반환하는 함수이다,
func getTimelogUntilTheMonthVFXFunc(client *mongo.Client, year int, month int) ([]Timelog, error) {
	collection := client.Database(*flagDBName).Collection("timelogs")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var results []Timelog
	q := bson.M{"userid": primitive.Regex{Pattern: "^(?!cm)", Options: "i"}, "year": year, "month": bson.M{"$lte": month}}
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
