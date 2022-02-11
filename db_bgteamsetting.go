// 프로젝트 결산 프로그램
//
// Description : DB Team Setting 관련 스크립트

package main

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// getBGTeamSettingFunc 함수는 DB에서 예산 Team Setting 정보를 가져오는 함수이다.
func getBGTeamSettingFunc(client *mongo.Client) (BGTeamSetting, error) {
	collection := client.Database(*flagDBName).Collection("setting.bgteam")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var result BGTeamSetting
	err := collection.FindOne(ctx, bson.M{"id": "setting.bgteam"}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments { // document가 존재하지 않는 경우
			return BGTeamSetting{}, nil
		}
		return BGTeamSetting{}, err
	}
	return result, nil
}

// setBGTeamSettingFunc 함수는 DB에서 예산 Team Setting 정보를 업데이트하는 함수이다.
func setBGTeamSettingFunc(client *mongo.Client, ts BGTeamSetting) error {
	collection := client.Database(*flagDBName).Collection("setting.bgteam")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	n, err := collection.CountDocuments(ctx, bson.M{"id": "setting.bgteam"})
	if err != nil {
		return err
	}
	ts.ID = "setting.bgteam"
	if n == 0 {
		_, err = collection.InsertOne(ctx, ts)
		if err != nil {
			return err
		}
		return nil
	}
	_, err = collection.UpdateOne(
		ctx,
		bson.M{"id": "setting.bgteam"},
		bson.D{{Key: "$set", Value: ts}},
	)
	if err != nil {
		return err
	}
	return nil
}
