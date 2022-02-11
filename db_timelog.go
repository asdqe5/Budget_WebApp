// 프로젝트 결산 프로그램
//
// Description : DB 타임로그 관련 스크립트

package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// addTimelogFunc 함수는 타임로그 시간을 업데이트하는 함수이다.
func addTimelogFunc(client *mongo.Client, t Timelog) error {
	collection := client.Database(*flagDBName).Collection("timelogs")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 분기 설정
	quarter, err := monthToQuaterFunc(t.Month)
	if err != nil {
		return err
	}
	t.Quarter = quarter

	t.Project = strings.ToUpper(t.Project) // 프로젝트명을 대문자로 변경

	var timelog Timelog
	// 타임로그가 존재하는지 검색한다.
	err = collection.FindOne(ctx, bson.M{"userid": t.UserID, "year": t.Year, "month": t.Month, "project": t.Project}).Decode(&timelog)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			_, err = collection.InsertOne(ctx, t) // 타임로그가 존재하지 않으면 추가한다.
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}
	// 타임로그가 존재하면 duration 을 업데이트한다.
	timelog.Duration = t.Duration
	_, err = collection.UpdateOne(
		ctx,
		bson.M{"userid": t.UserID, "year": t.Year, "month": t.Month, "project": t.Project},
		bson.D{{Key: "$set", Value: timelog}},
	)
	if err != nil {
		return err
	}
	return nil
}

// getTimelogFunc 함수는 DB에서 입력받은 아티스트, 날짜, 프로젝트 정보가 일치하는 타임로그를 가져오는 함수이다.
func getTimelogFunc(client *mongo.Client, artistID string, year int, month int, projectName string) (Timelog, error) {
	collection := client.Database(*flagDBName).Collection("timelogs")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result Timelog
	err := collection.FindOne(ctx, bson.M{"userid": artistID, "year": year, "month": month, "project": projectName}).Decode(&result)
	if err != nil {
		return result, err
	}
	return result, nil
}

// searchTimelogFunc 함수는 DB에서 타임로그를 검색하는 함수이다.
func searchTimelogFunc(client *mongo.Client, searchWord string) ([]Timelog, error) {
	var results []Timelog
	if searchWord == "" {
		return results, nil
	}
	collection := client.Database(*flagDBName).Collection("timelogs")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wordQueries := []bson.M{}
	for _, word := range strings.Split(searchWord, " ") {
		if word == "" {
			continue
		}
		querys := []bson.M{}
		if strings.HasPrefix(word, "userid:") {
			querys = append(querys, bson.M{"userid": strings.TrimPrefix(word, "userid:")})
		} else if strings.HasPrefix(word, "quarter:") {
			quarter, _ := strconv.Atoi(strings.TrimPrefix(word, "quarter:"))
			querys = append(querys, bson.M{"quarter": quarter})
		} else if strings.HasPrefix(word, "year:") {
			year, _ := strconv.Atoi(strings.TrimPrefix(word, "year:"))
			querys = append(querys, bson.M{"year": year})
		} else if strings.HasPrefix(word, "month:") {
			month, _ := strconv.Atoi(strings.TrimPrefix(word, "month:"))
			querys = append(querys, bson.M{"month": month})
		} else if strings.HasPrefix(word, "project:") {
			querys = append(querys, bson.M{"project": strings.TrimPrefix(word, "project:")})
		} else if strings.HasPrefix(word, "duration:") {
			duration, _ := strconv.ParseFloat(strings.TrimPrefix(word, "duration:"), 64)
			querys = append(querys, bson.M{"duration": duration})
		}
		wordQueries = append(wordQueries, bson.M{"$or": querys})
	}
	q := bson.M{"$and": wordQueries} // 최종 쿼리는 bson type 오브젝트가 되어야 한다.
	opts := options.Find()
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

// rmTimelogFunc 함수는 DB에서 타임로그 정보를 삭제하는 함수이다.
func rmTimelogFunc(client *mongo.Client, t Timelog) error {
	collection := client.Database(*flagDBName).Collection("timelogs")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	n, err := collection.CountDocuments(ctx, bson.M{"userid": t.UserID, "year": t.Year, "month": t.Month, "project": t.Project})
	if err != nil {
		return err
	}
	if n == 0 {
		return mongo.ErrNoDocuments
	}

	_, err = collection.DeleteOne(ctx, bson.M{"userid": t.UserID, "year": t.Year, "month": t.Month, "project": t.Project})
	if err != nil {
		return err
	}
	return nil
}

// rmTimelogByIDFunc 함수는 DB에서 입력받은 ID가 작성한 타임로그를 모두 삭제하는 함수이다.
func rmTimelogByIDFunc(client *mongo.Client, id []string) error {
	collection := client.Database(*flagDBName).Collection("timelogs")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	n, err := collection.CountDocuments(ctx, bson.M{"userid": bson.M{"$in": id}})
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("삭제할 타임로그가 없습니다")
	}

	_, err = collection.DeleteMany(ctx, bson.M{"userid": bson.M{"$in": id}})
	if err != nil {
		return err
	}
	return nil
}

// rmTimelogByProjectFunc 함수는 DB에서 입력받은 프로젝트에 작성한 타임로그를 모두 삭제하는 함수이다.
func rmTimelogByProjectFunc(client *mongo.Client, project []string) error {
	collection := client.Database(*flagDBName).Collection("timelogs")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	n, err := collection.CountDocuments(ctx, bson.M{"project": bson.M{"$in": project}})
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("삭제할 타임로그가 없습니다")
	}

	_, err = collection.DeleteMany(ctx, bson.M{"project": bson.M{"$in": project}})
	if err != nil {
		return err
	}
	return nil
}

// subTimelogFunc 함수는 DB에서 타임로그를 검색하여 전달받은 duration 만큼 빼는 함수이다.
func subTimelogFunc(client *mongo.Client, t Timelog) error {
	collection := client.Database(*flagDBName).Collection("timelogs")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var timelog Timelog
	// 타임로그가 존재하는지 검색한다.
	err := collection.FindOne(ctx, bson.M{"userid": t.UserID, "year": t.Year, "month": t.Month, "project": t.Project}).Decode(&timelog)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.New("타임로그를 찾을 수 없습니다")
		}
		return err
	}

	value := timelog.Duration - t.Duration
	if value < 0 { // DB에 저장된 duration 보다 큰 숫자를 빼려고 한다면 에러
		return errors.New(fmt.Sprintf("%.2f보다 작아야 합니다", timelog.Duration))
	} else if value == 0 { // 뺐을 때 0이 되면 삭제
		err = rmTimelogFunc(client, t)
		if err != nil {
			return err
		}
	} else {
		timelog.Duration = value
		_, err = collection.UpdateOne(
			ctx,
			bson.M{"userid": t.UserID, "year": t.Year, "month": t.Month, "project": t.Project},
			bson.D{{Key: "$set", Value: timelog}},
		)
	}
	return nil
}

// updateFinishedTimelogStatusFunc 함수는 FinishedTimelogStatus를 업데이트하는 함수이다.
func updateFinishedTimelogStatusFunc(client *mongo.Client, fts FinishedTimelogStatus) error {
	collection := client.Database(*flagDBName).Collection("finishedtimelogstatus")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	n, err := collection.CountDocuments(ctx, bson.M{"year": fts.Year, "month": fts.Month, "project": fts.Project})
	if err != nil {
		return err
	}
	if n == 0 {
		_, err = collection.InsertOne(ctx, fts)
		if err != nil {
			return err
		}
		return nil
	}
	_, err = collection.UpdateOne(
		ctx,
		bson.M{"year": fts.Year, "month": fts.Month, "project": fts.Project},
		bson.D{{Key: "$set", Value: fts}},
	)
	if err != nil {
		return err
	}
	return nil
}

// getFinishedTimelogStatusFunc 함수는 DB에서 FinishedTimelogStatus를 가져오는 함수이다.
func getFinishedTimelogStatusFunc(client *mongo.Client, year int, month int, project string) (FinishedTimelogStatus, error) {
	collection := client.Database(*flagDBName).Collection("finishedtimelogstatus")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result FinishedTimelogStatus
	err := collection.FindOne(ctx, bson.M{"year": year, "month": month, "project": project}).Decode(&result)
	if err != nil {
		return FinishedTimelogStatus{}, err
	}
	return result, nil
}

// getFTStatusByMonth 함수는 DB에서 연월로 FinishedTimelogStatus를 가져오는 함수이다.
func getFTStatusByMonth(client *mongo.Client, year int, month int) ([]FinishedTimelogStatus, error) {
	collection := client.Database(*flagDBName).Collection("finishedtimelogstatus")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var results []FinishedTimelogStatus
	q := bson.M{"year": year, "month": month}
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
