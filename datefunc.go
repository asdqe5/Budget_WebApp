package main

import (
	"fmt"
	"time"
)

// getDatesFunc 함수는 두 날짜 사이의 date 리스트를 반환하는 함수이다.
func getDatesFunc(startDate string, endDate string) ([]string, error) {
	var dateList []string
	sDate, err := time.Parse("2006-01", startDate)
	if err != nil {
		return nil, err
	}
	dateList = append(dateList, startDate)

	eDate, err := time.Parse("2006-01", endDate)
	if err != nil {
		return nil, err
	}

	for {
		if sDate == eDate { // 시작일과 마감일이 같을 경우 반복문을 빠져나간다.
			break
		}
		sDate = sDate.AddDate(0, 1, 0)
		year, month, _ := sDate.Date()
		date := fmt.Sprintf("%04d-%02d", year, month)
		dateList = append(dateList, date)
	}

	return dateList, nil
}
