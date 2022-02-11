// 프로젝트 결산 프로그램
//
// Description : DB 예산 프로젝트 관련 스크립트

package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// getBGProjectFunc 함수는 DB에서 id가 일치하는 예산 프로젝트를 찾아 반환하는 함수이다.
func getBGProjectFunc(client *mongo.Client, id string) (BGProject, error) {
	collection := client.Database(*flagDBName).Collection("bgprojects")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result BGProject
	err := collection.FindOne(ctx, bson.M{"id": id}).Decode(&result)
	if err != nil {
		return result, err
	}
	return result, nil
}

// addBGProjectFunc 함수는 DB에 예산 프로젝트를 추가하는 함수이다.
func addBGProjectFunc(client *mongo.Client, bgp BGProject) error {
	collection := client.Database(*flagDBName).Collection("bgprojects")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	bgp.UpdatedTime = time.Now().Format(time.RFC3339) // 프로젝트의 마지막 업데이트된 시간을 현재 시간으로 설정

	var bgProject BGProject
	// 프로젝트가 존재하는지 검색한 후 없으면 추가한다.
	err := collection.FindOne(ctx, bson.M{"id": bgp.ID}).Decode(&bgProject)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			_, err := collection.InsertOne(ctx, bgp)
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}
	return errors.New("프로젝트가 이미 DB에 존재합니다")
}

// rmBGProjectFunc 함수는 DB에서 예산 프로젝트를 삭제하는 함수이다.
func rmBGProjectFunc(client *mongo.Client, id string) error {
	collection := client.Database(*flagDBName).Collection("bgprojects")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	n, err := collection.CountDocuments(ctx, bson.M{"id": id})
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("삭제할 프로젝트가 없습니다")
	}

	_, err = collection.DeleteOne(ctx, bson.M{"id": id})
	if err != nil {
		return err
	}
	return nil
}

// searchBGProjectFunc 함수는 DB에서 예산 프로젝트를 검색하는 함수이다.
func searchBGProjectFunc(client *mongo.Client, searchWord string, sort string) ([]BGProject, error) {
	var results []BGProject
	if searchWord == "" {
		return results, nil
	}
	collection := client.Database(*flagDBName).Collection("bgprojects")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wordQueries := []bson.M{}
	for _, word := range strings.Split(searchWord, " ") {
		if word == "" {
			continue
		}
		querys := []bson.M{}
		if strings.HasPrefix(word, "id:") {
			if strings.TrimPrefix(word, "id:") == "" {
				querys = append(querys, bson.M{})
			} else {
				querys = append(querys, bson.M{"id": strings.TrimPrefix(word, "id:")})
			}
		} else if strings.HasPrefix(word, "name:") {
			querys = append(querys, bson.M{"name": strings.TrimPrefix(word, "name:")})
		} else if strings.HasPrefix(word, "date:") {
			querys = append(querys, bson.M{
				"startdate": bson.M{"$lte": strings.TrimPrefix(word, "date:")},
				"enddate":   bson.M{"$gte": strings.TrimPrefix(word, "date:")},
			})
		} else if strings.HasPrefix(word, "year:") {
			year := strings.TrimPrefix(word, "year:")
			querys = append(querys, bson.M{"startdate": primitive.Regex{Pattern: year, Options: "i"}})
			querys = append(querys, bson.M{"enddate": primitive.Regex{Pattern: year, Options: "i"}})

			// 그 연도의 첫달과 마지막달을 프로젝트 시작과 끝과 비교
			yearStart := fmt.Sprintf("%s-01", year)
			yearEnd := fmt.Sprintf("%s-12", year)
			querys = append(querys, bson.M{"startdate": bson.M{"$lte": yearStart}, "enddate": bson.M{"$gte": yearEnd}})
		} else if strings.HasPrefix(word, "status:") {
			status := strings.TrimPrefix(word, "status:")
			if status == "all" {
				continue
			} else if status == "true" { // Status 계약 완료
				querys = append(querys, bson.M{"status": true})
			} else { // Status 사전 검토
				querys = append(querys, bson.M{"status": false})
			}
		} else {
			querys = append(querys, bson.M{"id": primitive.Regex{Pattern: word, Options: "i"}})
			querys = append(querys, bson.M{"name": primitive.Regex{Pattern: word, Options: "i"}})
			querys = append(querys, bson.M{
				"startdate": bson.M{"$lte": word},
				"enddate":   bson.M{"$gte": word},
			})
		}
		wordQueries = append(wordQueries, bson.M{"$or": querys})
	}
	q := bson.M{"$and": wordQueries}
	opts := options.Find()
	if sort != "" {
		opts.SetSort(bson.M{sort: 1}) // 전달받은 sort가 빈 문자열이 아니면 sort를 기준으로 오름차순 정렬
	} else {
		opts.SetSort(bson.M{"name": 1}) // 전달받은 sort가 빈 문자열이라면 이름을 기준으로 오름차순 정렬
	}
	cursor, err := collection.Find(ctx, q, opts)
	if err != nil {
		return nil, err
	}
	err = cursor.All(ctx, &results)
	if err != nil {
		return nil, err
	}
	return results, nil
}

// setBGProjectFunc 함수는 DB에서 예산 프로젝트 정보를 업데이트하는 함수이다.
func setBGProjectFunc(client *mongo.Client, bgProject BGProject, id string) error {
	collection := client.Database(*flagDBName).Collection("bgprojects")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := collection.UpdateOne(
		ctx,
		bson.M{"id": id},
		bson.D{{Key: "$set", Value: bgProject}},
	)
	if err != nil {
		return err
	}
	return nil
}
