// 프로젝트 결산 프로그램
//
// Description : DB User 관련 스크립트

package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// addUserFunc 함수는 DB에 사용자를 추가하는 함수이다.
func addUserFunc(client *mongo.Client, u User) error {
	collection := client.Database(*flagDBName).Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	u.ID = strings.ToLower(u.ID) // id를 소문자로 변경

	var user User
	// 유저가 존재하는지 검색한 후 없으면 추가한다.
	err := collection.FindOne(ctx, bson.M{"id": u.ID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			_, err := collection.InsertOne(ctx, u)
			if err != nil {
				return err
			}
			return nil
		} else {
			return err
		}
	}
	return errors.New(fmt.Sprintf("%s 아이디를 가진 사용자가 이미 존재합니다", u.ID))
}

// getUserFunc 함수는 DB에서 사용자 정보를 가져오는 함수이다.
func getUserFunc(client *mongo.Client, id string) (User, error) {
	collection := client.Database(*flagDBName).Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var result User
	err := collection.FindOne(ctx, bson.M{"id": id}).Decode(&result)
	if err != nil {
		return result, err
	}
	return result, nil
}

// getAllUsersFunc 함수는 DB에서 모든 사용자의 정보를 가져오는 함수이다.
func getAllUsersFunc(client *mongo.Client) ([]User, error) {
	collection := client.Database(*flagDBName).Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var results []User
	cursor, err := collection.Find(ctx, bson.D{})
	if err != nil {
		return results, err
	}
	err = cursor.All(ctx, &results)
	if err != nil {
		return results, err
	}
	return results, nil
}

// setUserFunc 함수는 유저 정보를 업데이트하는 함수이다.
func setUserFunc(client *mongo.Client, u User) error {
	collection := client.Database(*flagDBName).Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := collection.UpdateOne(
		ctx,
		bson.M{"id": u.ID},
		bson.D{{Key: "$set", Value: u}},
	)
	if err != nil {
		return err
	}
	return nil
}

// rmUserFunc 함수는 id를 입력받아 아티스트 정보를 삭제하는 함수이다.
func rmUserFunc(client *mongo.Client, id string) error {
	collection := client.Database(*flagDBName).Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := collection.DeleteOne(ctx, bson.M{"id": id})
	if err != nil {
		return err
	}
	return nil
}
