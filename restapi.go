// 프로젝트 결산 프로그램
//
// Description : Rest API 관련 스크립트

package main

import "errors"

func postFormValueInListFunc(key string, values []string) (string, error) {
	if len(values) != 1 {
		return "", errors.New(key + "값이 여러개입니다.")
	}
	if key == "id" && values[0] == "" {
		return "", nil
	}
	return values[0], nil
}
