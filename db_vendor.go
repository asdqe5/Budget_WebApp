// 프로젝트 결산 프로그램
//
// Description : DB 벤더 관련 스크립트

package main

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// addVendorFunc 함수는 DB에 Vendor를 추가하는 함수이다.
func addVendorFunc(client *mongo.Client, v Vendor) error {
	collection := client.Database(*flagDBName).Collection("vendors")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := collection.InsertOne(ctx, v)
	if err != nil {
		return err
	}
	return nil
}

// searchVendorFunc 함수는 DB에서 해당하는 Vendor를 검색하는 함수이다.
func searchVendorFunc(client *mongo.Client, searchWord string) ([]Vendor, error) {
	var results []Vendor
	if searchWord == "" {
		return results, nil
	}
	collection := client.Database(*flagDBName).Collection("vendors")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wordQueries := []bson.M{}
	for _, word := range strings.Split(searchWord, " ") {
		if word == "" {
			continue
		}
		querys := []bson.M{}
		if strings.HasPrefix(word, "project:") {
			querys = append(querys, bson.M{"project": strings.TrimPrefix(word, "project:")})
		} else if strings.HasPrefix(word, "name:") {
			querys = append(querys, bson.M{"name": strings.TrimPrefix(word, "name:")})
		} else if strings.HasPrefix(word, "status:") {
			status := strings.TrimPrefix(word, "status:")
			if status == "downpayment" {
				querys = append(querys, bson.M{"downpayment.expenses": bson.M{"$ne": ""}, "downpayment.status": false}) // 계약금이 존재하고 status가 false인지 확인 -> 아무것도 넣지 않아도 status가 false로 저장됨
			} else if status == "mediumplating" {
				querys = append(querys, bson.M{"mediumplating.status": false}) // 중도금은 존재하는 경우에만 저장됨
			} else if status == "balance" {
				querys = append(querys, bson.M{"balance.expenses": bson.M{"$ne": ""}, "balance.status": false}) // 잔금이 존재하고 status가 false인지 확인 -> 아무것도 넣지 않아도 status가 false로 저장됨
			} else {
				querys = append(querys, bson.M{})
			}
		} else {
			querys = append(querys, bson.M{"project": primitive.Regex{Pattern: word, Options: "i"}})
			querys = append(querys, bson.M{"name": primitive.Regex{Pattern: word, Options: "i"}})
			querys = append(querys, bson.M{"projectname": primitive.Regex{Pattern: word, Options: "i"}})
		}
		wordQueries = append(wordQueries, bson.M{"$or": querys})
	}
	q := bson.M{"$and": wordQueries}
	opts := options.Find()
	cursor, err := collection.Find(ctx, q, opts)
	if err != nil {
		return results, err
	}
	err = cursor.All(ctx, &results)
	if err != nil {
		return results, err
	}
	return results, nil
}

// rmVendorFunc 함수는 DB에서 입력받은 프로젝트의 입력받은 벤더를 삭제하는 함수이다.
func rmVendorFunc(client *mongo.Client, project string, name string, id string) error {
	collection := client.Database(*flagDBName).Collection("vendors")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if id == "" { // 프로젝트 ID 및 벤더 이름으로 삭제하는 경우
		if name == "" { // 프로젝트 ID로 삭제하는 경우
			n, err := collection.CountDocuments(ctx, bson.M{"project": project})
			if err != nil {
				return err
			}
			if n == 0 {
				return nil
			}
			_, err = collection.DeleteMany(ctx, bson.M{"project": project})
			if err != nil {
				return err
			}
		} else {
			n, err := collection.CountDocuments(ctx, bson.M{"project": project, "name": name})
			if err != nil {
				return err
			}
			if n == 0 {
				return errors.New("삭제할 벤더가 없습니다")
			}
			_, err = collection.DeleteMany(ctx, bson.M{"project": project, "name": name})
			if err != nil {
				return err
			}
		}
	} else { // 웹페이지에서 벤더 ID로 삭제하는 경우
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return err
		}
		_, err = collection.DeleteOne(ctx, bson.M{"_id": objID})
		if err != nil {
			return err
		}
	}
	return nil
}

// getVendorFunc 함수는 해당 ID의 벤더를 가져오는 함수이다.
func getVendorFunc(client *mongo.Client, id string) (Vendor, error) {
	collection := client.Database(*flagDBName).Collection("vendors")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var result Vendor
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return result, err
	}
	err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&result)
	if err != nil {
		return result, err
	}
	return result, nil
}

// setVendorFunc 함수는 해당 벤더 정보를 업데이트하는 함수이다.
func setVendorFunc(client *mongo.Client, vendor Vendor) error {
	collection := client.Database(*flagDBName).Collection("vendors")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": vendor.ID},
		bson.D{{Key: "$set", Value: vendor}},
	)
	if err != nil {
		return err
	}
	return nil
}

// getVendorsByYearFunc 함수는 DB에서 해당하는 연도에 계약금 및 잔금 지출일이 존재하는 벤더를 가져오는 함수이다.
func getVendorsByYearFunc(client *mongo.Client, year string) ([]Vendor, error) {
	var results []Vendor
	if year == "" {
		return results, nil
	}
	collection := client.Database(*flagDBName).Collection("vendors")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wordQueries := []bson.M{}
	querys := []bson.M{}
	querys = append(querys, bson.M{"downpayment.date": primitive.Regex{Pattern: year, Options: "i"}})
	querys = append(querys, bson.M{"mediumplating.date": primitive.Regex{Pattern: year, Options: "i"}})
	querys = append(querys, bson.M{"balance.date": primitive.Regex{Pattern: year, Options: "i"}})
	wordQueries = append(wordQueries, bson.M{"$or": querys})

	q := bson.M{"$and": wordQueries}
	opts := options.Find()
	cursor, err := collection.Find(ctx, q, opts)
	if err != nil {
		return results, err
	}
	err = cursor.All(ctx, &results)
	if err != nil {
		return results, err
	}
	return results, nil
}

// getVendorsByTodayFunc 함수는 DB에서 계약금, 중도금, 잔금 세금계산서 발행일이 오늘인 벤더를 가져오는 함수이다.
func getVendorsByTodayFunc(client *mongo.Client) ([]Vendor, error) {
	collection := client.Database(*flagDBName).Collection("vendors")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	nowDate := time.Now().Format("2006-01-02") // 오늘 날짜

	wordQueries := []bson.M{}
	querys := []bson.M{}
	querys = append(querys, bson.M{"downpayment.date": nowDate})
	querys = append(querys, bson.M{"mediumplating.date": nowDate})
	querys = append(querys, bson.M{"balance.date": nowDate})
	wordQueries = append(wordQueries, bson.M{"$or": querys})

	var results []Vendor
	q := bson.M{"$and": wordQueries}
	opts := options.Find()
	cursor, err := collection.Find(ctx, q, opts)
	if err != nil {
		return results, err
	}
	err = cursor.All(ctx, &results)
	if err != nil {
		return results, err
	}
	return results, nil
}
