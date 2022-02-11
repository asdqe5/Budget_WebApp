// 프로젝트 결산 프로그램
//
// Description : DB 프로젝트 관련 스크립트

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

// addProjectFunc 함수는 DB에 결산 프로젝트를 추가하는 함수이다.
func addProjectFunc(client *mongo.Client, p Project) error {
	collection := client.Database(*flagDBName).Collection("projects")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var project Project
	// 프로젝트가 존재하는지 검색한 후 없으면 추가한다.
	err := collection.FindOne(ctx, bson.M{"id": p.ID}).Decode(&project)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			_, err := collection.InsertOne(ctx, p)
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}
	return errors.New("프로젝트가 이미 DB에 존재합니다")
}

// rmProjectFunc 함수는 DB에서 결산 프로젝트를 삭제하는 함수이다.
func rmProjectFunc(client *mongo.Client, id string) error {
	collection := client.Database(*flagDBName).Collection("projects")
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

// getProjectFunc 함수는 DB에서 id가 일치하는 결산 프로젝트를 찾아 반환하는 함수이다.
func getProjectFunc(client *mongo.Client, id string) (Project, error) {
	collection := client.Database(*flagDBName).Collection("projects")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result Project
	err := collection.FindOne(ctx, bson.M{"id": id}).Decode(&result)
	if err != nil {
		return result, err
	}
	return result, nil
}

// getAllProjectsFunc 함수는 DB에서 모든 결산 프로젝트의 정보를 가져오는 함수이다.
func getAllProjectsFunc(client *mongo.Client) ([]Project, error) {
	collection := client.Database(*flagDBName).Collection("projects")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var results []Project
	opts := options.Find()
	opts.SetSort(bson.M{"name": 1}) // 이름을 기준으로 오름차순 정렬
	cursor, err := collection.Find(ctx, bson.D{}, opts)
	if err != nil {
		return results, err
	}
	err = cursor.All(ctx, &results)
	if err != nil {
		return results, err
	}
	return results, nil
}

// setProjectFunc 함수는 DB에서 결산 프로젝트 정보를 업데이트하는 함수이다.
func setProjectFunc(client *mongo.Client, project Project) error {
	collection := client.Database(*flagDBName).Collection("projects")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := collection.UpdateOne(
		ctx,
		bson.M{"id": project.ID},
		bson.D{{Key: "$set", Value: project}},
	)
	if err != nil {
		return err
	}
	return nil
}

// searchProjectFunc 함수는 DB에서 결산 프로젝트를 검색하는 함수이다.
func searchProjectFunc(client *mongo.Client, searchWord string, sort string) ([]Project, error) {
	var results []Project
	if searchWord == "" {
		return results, nil
	}
	collection := client.Database(*flagDBName).Collection("projects")
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
				"smenddate": bson.M{"$gte": strings.TrimPrefix(word, "date:")},
			})
		} else if strings.HasPrefix(word, "finishedstatus:") {
			fs := strings.TrimPrefix(word, "finishedstatus:")
			if fs == "ing" {
				querys = append(querys, bson.M{"isfinished": false})
			} else if fs == "end" {
				querys = append(querys, bson.M{"isfinished": true})
			} else if fs == "none" {
				querys = append(querys, bson.M{"isfinished": "none"})
			} else {
				querys = append(querys, bson.M{})
			}
		} else {
			querys = append(querys, bson.M{"id": primitive.Regex{Pattern: word, Options: "i"}})
			querys = append(querys, bson.M{"name": primitive.Regex{Pattern: word, Options: "i"}})
			querys = append(querys, bson.M{
				"startdate": bson.M{"$lte": word},
				"smenddate": bson.M{"$gte": word},
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

// getProjectOfTheMonthFunc 함수는 DB에서 date가 작업기간에 해당하는 모든 결산 프로젝트를 찾아 반환하는 함수이다.
func getProjectOfTheMonthFunc(client *mongo.Client, date string) ([]Project, error) {
	collection := client.Database(*flagDBName).Collection("projects")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var results []Project
	queries := []bson.M{
		bson.M{"startdate": bson.M{"$lte": date}},
	}
	queries = append(queries, bson.M{"smenddate": bson.M{"$gte": date}})

	query := bson.M{"$and": queries}
	opts := options.Find()
	cursor, err := collection.Find(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	err = cursor.All(ctx, &results)
	if err != nil {
		return nil, err
	}
	return results, nil
}

// getProjectsByYearFunc 함수는 DB에서 해당하는 연도에 작업 중인 결산 프로젝트를 가져오는 함수이다.
func getProjectsByYearFunc(client *mongo.Client, year string) ([]Project, error) {
	var results []Project
	if year == "" {
		return results, nil
	}
	collection := client.Database(*flagDBName).Collection("projects")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wordQueries := []bson.M{}
	querys := []bson.M{}
	querys = append(querys, bson.M{"startdate": primitive.Regex{Pattern: year, Options: "i"}})
	querys = append(querys, bson.M{"smenddate": primitive.Regex{Pattern: year, Options: "i"}})

	// 그 연도의 첫달과 마지막달을 프로젝트 시작과 끝과 비교
	yearStart := fmt.Sprintf("%s-01", year)
	yearEnd := fmt.Sprintf("%s-12", year)
	querys = append(querys, bson.M{"startdate": bson.M{"$lte": yearStart}, "smenddate": bson.M{"$gte": yearEnd}})
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

// getNameOfProjectFunc 함수는 DB에서 결산 프로젝트를 찾아 프로젝트의 한글명을 반환하는 함수이다.
func getNameOfProjectFunc(client *mongo.Client, id string) (string, error) {
	collection := client.Database(*flagDBName).Collection("projects")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var project Project
	err := collection.FindOne(ctx, bson.M{"id": id}).Decode(&project)
	if err != nil {
		return "", err
	}
	return project.Name, nil
}

// getIDOfProjectsFunc 함수는 DB에 저장된 모든 결산 프로젝트들의 id만 모아서 반환하는 함수이다.
func getIDOfProjectsFunc(client *mongo.Client) ([]string, error) {
	collection := client.Database(*flagDBName).Collection("projects")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var results []string
	idList, err := collection.Distinct(ctx, "id", bson.M{})
	if err != nil {
		return results, err
	}

	// []interface{}형을 []string형으로 변환한다.
	for _, id := range idList {
		results = append(results, fmt.Sprint(id))
	}
	return results, nil
}

// getProjectsByTodayFunc 함수는 DB에서 세금계산서 발행일이 오늘인 결산 프로젝트를 가져오는 함수이다.
func getProjectsByTodayFunc(client *mongo.Client) ([]Project, error) {
	collection := client.Database(*flagDBName).Collection("projects")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	nowDate := time.Now().Format("2006-01-02") // 오늘 날짜

	q := fmt.Sprintf("smmonthlypayment.%04d-%02d.date", time.Now().Year(), time.Now().Month())

	var results []Project
	opts := options.Find()
	cursor, err := collection.Find(ctx, bson.M{q: nowDate}, opts)
	if err != nil {
		return results, err
	}
	err = cursor.All(ctx, &results)
	if err != nil {
		return results, err
	}
	return results, nil
}

// getOldProjectFunc 함수는 DB에서 예전 자료구조의 결산 프로젝트 정보를 가져오는 함수이다.
func getOldProjectFunc(client *mongo.Client, id string) (OldProject, error) {
	collection := client.Database(*flagDBName).Collection("projects")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result OldProject
	err := collection.FindOne(ctx, bson.M{"id": id}).Decode(&result)
	if err != nil {
		return result, err
	}
	return result, nil
}
