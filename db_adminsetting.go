// 프로젝트 결산 프로그램
//
// Description : DB Admin Setting 관련 스크립트

package main

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// getAdminSettingFunc 함수는 DB에서 Admin Setting 정보를 가져오는 함수이다.
func getAdminSettingFunc(client *mongo.Client) (AdminSetting, error) {
	collection := client.Database(*flagDBName).Collection("setting.admin")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var result AdminSetting
	err := collection.FindOne(ctx, bson.M{"id": "setting.admin"}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments { // document가 존재하지 않는 경우
			return AdminSetting{}, nil
		}
		return AdminSetting{}, err
	}
	return result, nil
}

// updateAdminSettingFunc 함수는 admin setting을 업데이트하는 함수이다.
func updateAdminSettingFunc(client *mongo.Client, a AdminSetting) error {
	collection := client.Database(*flagDBName).Collection("setting.admin")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	n, err := collection.CountDocuments(ctx, bson.M{"id": "setting.admin"})
	if err != nil {
		return err
	}
	a.ID = "setting.admin"
	if n == 0 {
		_, err = collection.InsertOne(ctx, a)
		if err != nil {
			return err
		}
		return nil
	}
	_, err = collection.UpdateOne(
		ctx,
		bson.M{"id": "setting.admin"},
		bson.D{{Key: "$set", Value: a}},
	)
	if err != nil {
		return err
	}
	return nil
}

// setMonthlyStatusFunc 함수는 결산의 월별 상태를 저장한다.
func setMonthlyStatusFunc(client *mongo.Client, ms MonthlyStatus) error {
	collection := client.Database(*flagDBName).Collection("monthlystatus")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var monthlyStatus MonthlyStatus
	// DB에 존재하는지 검색한다.
	err := collection.FindOne(ctx, bson.M{"date": ms.Date}).Decode(&monthlyStatus)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			_, err = collection.InsertOne(ctx, ms) // DB에 존재하지 않으면 추가한다.
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}
	// DB에 존재하면 status를 업데이트한다.
	monthlyStatus.Status = ms.Status
	_, err = collection.UpdateOne(
		ctx,
		bson.M{"date": ms.Date},
		bson.D{{Key: "$set", Value: monthlyStatus}},
	)
	if err != nil {
		return err
	}
	return nil
}

// getMonthlyStatusFunc 함수는 결산의 월별 상태를 가져온다.
func getMonthlyStatusFunc(client *mongo.Client, month string) (MonthlyStatus, error) {
	collection := client.Database(*flagDBName).Collection("monthlystatus")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result MonthlyStatus
	err := collection.FindOne(ctx, bson.M{"date": month}).Decode(&result)
	if err != nil {
		return MonthlyStatus{}, err
	}
	return result, nil
}
