// 프로젝트 결산 프로그램
//
// Description : DB 로그 관련 스크립트

package main

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo/options"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// addLogsFunc 함수는 DB에 로그 정보를 저장하는 함수이다.
func addLogsFunc(client *mongo.Client, log Log) error {
	collection := client.Database(*flagDBName).Collection("logs")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := collection.InsertOne(ctx, log)
	if err != nil {
		return err
	}
	return nil
}

// SearchLogsFunc 함수는 로그 데이터를 반환한다.
func SearchLogsFunc(client *mongo.Client, page int64, limitnum int64) (int64, int64, []Log, error) {
	var results []Log

	collection := client.Database(*flagDBName).Collection("logs")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	q := bson.D{}
	opts := options.Find()
	opts.SetSort(bson.M{"created_at": -1})
	opts.SetSkip(int64((page - 1) * limitnum))
	opts.SetLimit(int64(limitnum))
	cursor, err := collection.Find(ctx, q, opts)
	if err != nil {
		return 0, 0, nil, err
	}
	err = cursor.All(ctx, &results)
	if err != nil {
		return 0, 0, nil, err
	}
	totalNum, err := collection.CountDocuments(ctx, q)
	if err != nil {
		return 0, 0, nil, err
	}
	return TotalPageFunc(totalNum, limitnum), totalNum, results, nil
}
