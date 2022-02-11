// 프로젝트 결산 프로그램
//
// Description : string 타입을 처리하는 스크립트

package main

import (
	"errors"
	"fmt"
	"strings"
)

// stringToMapFunc 함수는 문자열을 map 형으로 변환하는 함수이다.
func stringToMapFunc(str string) (map[string]string, error) {
	str = strings.TrimSpace(str)
	if str == "" {
		return nil, nil
	}

	var result map[string]string
	result = make(map[string]string)

	if !regexMap.MatchString(str) {
		return nil, errors.New("map 형식이 아닙니다")
	}

	for _, s := range strings.Split(str, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		key := strings.Split(s, ":")[0]
		value := strings.Split(s, ":")[1]
		result[key] = value
	}
	return result, nil
}

// monthToQuaterFunc 함수는 입력받은 월의 분기를 반환하는 함수이다.
func monthToQuaterFunc(month int) (int, error) {
	switch month {
	case 1, 2, 3:
		return 1, nil
	case 4, 5, 6:
		return 2, nil
	case 7, 8, 9:
		return 3, nil
	case 10, 11, 12:
		return 4, nil
	default:
		return 0, errors.New("월은 1부터 12까지의 숫자만 가능합니다")
	}
}

// listToStringFunc 함수는 입력받은 리스트를 띄어쓰기로 구분되는 문자열을 반환하는 함수이다.
func listToStringFunc(list []string, withComma bool) string {
	if withComma {
		return strings.Join(list, ",")
	}
	return strings.Join(list, " ")
}

// stringToListFunc 함수는 입력받은 문자열을 sep으로 나누어 리스트로 반환하는 함수이다.
func stringToListFunc(str string, sep string) []string {
	var result []string
	for _, s := range strings.Split(str, sep) {
		if s == "" {
			continue
		}
		result = append(result, s)
	}
	return result
}

// checkStringInList 함수는 문자열이 리스트에 포함되어 있는지 여부를 반환하는 함수이다.
func checkStringInListFunc(str string, list []string) bool {
	for _, l := range list {
		if l == str {
			return true
		}
	}
	return false
}

// mapToStringFunc 함수는 {2019:2400, 2020:2400}와 같은 map 형을 2019:2400,2020:2400와 같은 문자열로 변화하는 함수이다.
func mapToStringFunc(m map[string]string) string {
	result := ""
	for key, value := range m {
		s := key + ":" + value
		if result == "" {
			result = s
			continue
		}
		result = strings.Join([]string{result, s}, ",")
	}
	return result
}

// changeToCMIDFunc 함수는 cm001 형식으로 바꿔주는 함수이다.
func changeToCMIDFunc(id string) string {
	zeroID := fmt.Sprintf("%03s", id)
	cmID := "cm" + zeroID
	return cmID
}

// dateToMonthFunc 함수는 2020-12-12 형식에서 월까지만 반환하는 함수이다.
func dateToMonthFunc(date string) string {
	if date == "" {
		return ""
	}
	return strings.Split(date, "-")[0] + "-" + strings.Split(date, "-")[1]
}
