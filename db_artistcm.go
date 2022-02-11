// 프로젝트 결산 프로그램
//
// Description : DB 아티스트 관련 스크립트

package main

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// getCMArtistsFunc 함수는 DB에서 CM팀의 아티스트를 찾아 반환하는 함수이다.
func getCMArtistsFunc(client *mongo.Client, sort string, year string) ([]Artist, error) {
	collection := client.Database(*flagDBName).Collection("artists")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	lastDay := fmt.Sprintf("%s-12-31", year) // 해당 연도의 마지막 날

	var results []Artist
	queries := []bson.M{}
	queries = append(queries, bson.M{
		"id":       primitive.Regex{Pattern: "^cm", Options: "i"}, // id가 cm으로 시작하면
		"startday": bson.M{"$lte": lastDay},                       // 입력받은 연도를 포함하여 입사일이 그 이전인 경우
	})
	queries = append(queries, bson.M{
		"id":       primitive.Regex{Pattern: "^cm", Options: "i"}, // id가 cm으로 시작하면
		"startday": "",                                            // 입사일이 없는 경우
	})
	query := bson.M{"$or": queries}
	opts := options.Find()
	if sort != "" {
		opts.SetSort(bson.M{sort: 1}) // 전달받은 sort가 빈 문자열이 아니면 sort를 기준으로 오름차순 정렬
	} else {
		opts.SetSort(bson.M{"name": 1}) // 전달받은 sort가 빈 문자열이라면 이름을 기준으로 오름차순 정렬
	}
	cursor, err := collection.Find(ctx, query, opts)
	if err != nil {
		return results, err
	}
	err = cursor.All(ctx, &results)
	if err != nil {
		return results, err
	}
	return results, nil
}

// getCMArtistsWithoutRetireeFunc 함수는 퇴사자를 제외한 모든 CM팀의 아티스트를 찾아 반환하는 함수이다.
func getCMArtistsWithoutRetireeFunc(client *mongo.Client, sort string, year string) ([]Artist, error) {
	collection := client.Database(*flagDBName).Collection("artists")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	lastDay := fmt.Sprintf("%s-12-31", year) // 해당 연도의 마지막 날

	var results []Artist
	queries := []bson.M{}

	queries = append(queries, bson.M{
		"id":       primitive.Regex{Pattern: "^cm", Options: "i"},
		"startday": bson.M{"$lte": lastDay}, // 입력받은 연도를 포함하여 입사일이 그 이전인 경우
		"endday":   bson.M{"$gte": year},    // 입력받은 연도를 포함하여 퇴사일이 그 이후인 경우
	})
	queries = append(queries, bson.M{
		"id":       primitive.Regex{Pattern: "^cm", Options: "i"},
		"startday": bson.M{"$lte": lastDay}, // 입력받은 연도를 포함하여 입사일이 그 이전인 경우
		"endday":   "",                      // 퇴사일이 없는 경우
	})
	queries = append(queries, bson.M{
		"id":       primitive.Regex{Pattern: "^cm", Options: "i"},
		"startday": "",                   // 입사일이 없는 경우
		"endday":   bson.M{"$gte": year}, // 입력받은 연도를 포함하여 퇴사일이 그 이후인 경우
	})
	queries = append(queries, bson.M{
		"id":       primitive.Regex{Pattern: "^cm", Options: "i"},
		"startday": "", // 입사일이 없는 경우
		"endday":   "", // 퇴사일이 없는 경우
	})
	query := bson.M{"$or": queries}

	opts := options.Find()
	if sort != "" {
		opts.SetSort(bson.M{sort: 1}) // 전달받은 sort가 빈 문자열이 아니면 sort를 기준으로 오름차순 정렬
	} else {
		opts.SetSort(bson.M{"name": 1}) // 전달받은 sort가 빈 문자열이라면 이름을 기준으로 오름차순 정렬
	}
	cursor, err := collection.Find(ctx, query, opts)
	if err != nil {
		return results, err
	}
	err = cursor.All(ctx, &results)
	if err != nil {
		return results, err
	}
	return results, nil
}

// rmAllCMArtists 함수는 DB에서 CM 아티스트를 모두 삭제하는 함수이다.
func rmAllCMArtists(client *mongo.Client) error {
	collection := client.Database(*flagDBName).Collection("artists")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	q := bson.M{"id": primitive.Regex{Pattern: "^cm", Options: "i"}} // id가 cm으로 시작하면

	_, err := collection.DeleteMany(ctx, q)
	if err != nil {
		return err
	}
	return nil
}
