// 프로젝트 결산 프로그램
//
// Description : DB 아티스트 관련 스크립트

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

// addArtistFunc 함수는 DB에 아티스트를 추가하는 함수이다.
func addArtistFunc(client *mongo.Client, a Artist) error {
	collection := client.Database(*flagDBName).Collection("artists")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 퇴사일이 설정되어 있으면 오늘 날짜랑 비교하여 지났으면 퇴사 여부를 true로 설정한다.
	if a.EndDay != "" {
		t, _ := time.Parse("2006-01-02", a.EndDay)
		duration := time.Now().Sub(t).Hours() / 24
		if duration > 0 {
			a.Resination = true
		} else {
			a.Resination = false
		}
	}

	var artist Artist
	// 아티스트가 존재하는지 검색한 후 없으면 추가한다.
	err := collection.FindOne(ctx, bson.M{"id": a.ID}).Decode(&artist)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			_, err := collection.InsertOne(ctx, a)
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}
	return errors.New(fmt.Sprintf("Shotgun / CM ID가 %s인 아티스트가 이미 DB에 존재합니다", a.ID))
}

// getArtistFunc 함수는 DB에서 id가 일치하는 아티스트를 찾아 반환하는 함수이다.
func getArtistFunc(client *mongo.Client, id string) (Artist, error) {
	collection := client.Database(*flagDBName).Collection("artists")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result Artist
	err := collection.FindOne(ctx, bson.M{"id": id}).Decode(&result)
	if err != nil {
		return result, err
	}
	return result, nil
}

// getAllArtistFunc 함수는 DB에 저장된 모든 아티스트를 반환하는 함수이다.
func getAllArtistFunc(client *mongo.Client) ([]Artist, error) {
	collection := client.Database(*flagDBName).Collection("artists")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var results []Artist
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	err = cursor.All(ctx, &results)
	if err != nil {
		return nil, err
	}
	return results, nil
}

// getArtistByTeamsFunc 함수는 해당하는 팀들의 퇴사히지 않은 아티스트들을 반환하는 함수이다.
func getArtistByTeamsFunc(client *mongo.Client, teams []string) ([]Artist, error) {
	collection := client.Database(*flagDBName).Collection("artists")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var results []Artist
	if teams == nil { // 초기 상태에 팀이 설정되지 않은 경우 $in 쿼리를 사용할 때 에러가 발생한다.
		return results, nil
	}
	for _, team := range teams {
		var result []Artist
		if strings.HasPrefix(team, "CM_") { // CM 아티스트에서 팀 찾기
			cursor, err := collection.Find(ctx, bson.M{
				"id":         primitive.Regex{Pattern: "^cm", Options: "i"}, // id가 cm으로 시작하면
				"team":       strings.TrimPrefix(team, "CM_"),
				"resination": false,
			})
			if err != nil {
				return nil, err
			}
			err = cursor.All(ctx, &result)
			if err != nil {
				return nil, err
			}
		} else {
			cursor, err := collection.Find(ctx, bson.M{
				"id":         primitive.Regex{Pattern: "^(?!cm)", Options: "i"}, // id가 cm으로 시작하지 않으면
				"team":       team,
				"resination": false,
			})
			if err != nil {
				return nil, err
			}
			err = cursor.All(ctx, &result)
			if err != nil {
				return nil, err
			}
		}
		results = append(results, result...)
	}
	return results, nil
}

// rmArtistFunc 함수는 DB에서 id가 일치하는 아티스트를 찾아 삭제하는 함수이다.
func rmArtistFunc(client *mongo.Client, id string) error {
	collection := client.Database(*flagDBName).Collection("artists")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	n, err := collection.CountDocuments(ctx, bson.M{"id": id})
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("삭제할 아티스트가 없습니다")
	}

	_, err = collection.DeleteOne(ctx, bson.M{"id": id})
	if err != nil {
		return err
	}
	return nil
}

// searchArtistFunc 함수는 묶음으로 가져올 때 검색하는 함수이다.
func searchArtistFunc(client *mongo.Client, searchWord string) ([]Artist, error) {
	var results []Artist
	if searchWord == "" {
		return results, nil
	}
	collection := client.Database(*flagDBName).Collection("artists")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wordQueries := []bson.M{}
	for _, word := range strings.Split(searchWord, " ") {
		if word == "" {
			continue
		}
		querys := []bson.M{}
		if strings.HasPrefix(word, "id:") {
			querys = append(querys, bson.M{"id": strings.TrimPrefix(word, "id:")})
		} else if strings.HasPrefix(word, "name:") {
			querys = append(querys, bson.M{"name": strings.TrimPrefix(word, "name:")})
		} else if strings.HasPrefix(word, "dept:") {
			querys = append(querys, bson.M{"dept": strings.TrimPrefix(word, "dept:")})
		} else if strings.HasPrefix(word, "team:") {
			querys = append(querys, bson.M{"team": strings.TrimPrefix(word, "team:")})
		} else { // ~~: ~~ 형식이 아닌 채로 검색을 할 경우, 우선은 타임로그에서 아티스트 검색 시 id와 이름만
			querys = append(querys, bson.M{"id": primitive.Regex{Pattern: word, Options: "i"}})
			querys = append(querys, bson.M{"name": primitive.Regex{Pattern: word, Options: "i"}})
			querys = append(querys, bson.M{"dept": primitive.Regex{Pattern: word, Options: "i"}})
			querys = append(querys, bson.M{"team": primitive.Regex{Pattern: word, Options: "i"}})
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

// setArtistFunc 함수는 아티스트 정보를 업데이트하는 함수이다.
func setArtistFunc(client *mongo.Client, artist Artist) error {
	collection := client.Database(*flagDBName).Collection("artists")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 퇴사일이 설정되어 있으면 오늘 날짜랑 비교하여 지났으면 퇴사 여부를 true로 설정한다.
	if artist.EndDay != "" {
		t, _ := time.Parse("2006-01-02", artist.EndDay)
		duration := time.Now().Sub(t).Hours() / 24
		if duration > 0 {
			artist.Resination = true
		} else {
			artist.Resination = false
		}
	} else {
		artist.Resination = false
	}

	_, err := collection.UpdateOne(
		ctx,
		bson.M{"id": artist.ID},
		bson.D{{Key: "$set", Value: artist}},
	)
	if err != nil {
		return err
	}
	return nil
}

// updateArtistFunc 함수는 DB에 아티스트를 추가하거나 업데이트하는 함수이다.
func updateArtistFunc(client *mongo.Client, artist Artist) error {
	collection := client.Database(*flagDBName).Collection("artists")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 퇴사일이 설정되어 있으면 오늘 날짜랑 비교하여 지났으면 퇴사 여부를 true로 설정한다.
	if artist.EndDay != "" {
		t, _ := time.Parse("2006-01-02", artist.EndDay)
		duration := time.Now().Sub(t).Hours() / 24
		if duration > 0 {
			artist.Resination = true
		} else {
			artist.Resination = false
		}
	}

	var a Artist
	// 아티스트가 존재하는지 검색한 후 없으면 추가한다.
	err := collection.FindOne(ctx, bson.M{"id": artist.ID}).Decode(&a)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			_, err := collection.InsertOne(ctx, artist)
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}

	// 아티스트가 존재하면 Update한다.
	_, err = collection.UpdateOne(
		ctx,
		bson.M{"id": artist.ID},
		bson.D{{Key: "$set", Value: artist}},
	)
	if err != nil {
		return err
	}
	return nil
}
