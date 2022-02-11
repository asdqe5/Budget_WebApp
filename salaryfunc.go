// 프로젝트 결산 프로그램
//
// Description : 연봉 데이터와 관련된 스크립트

package main

import (
	"math"
	"strconv"
)

// realMonthlySalaryFunc 함수는 실제 급여를 계산하여 반환하는 함수이다.
func realMonthlySalaryFunc(salary string, whole int, days int) (float64, error) {
	if salary == "" {
		return 0, nil
	}

	decryptedSalary, err := decryptAES256Func(salary) // 연봉 정보 복호화
	if err != nil {
		return 0, err
	}

	// 근무일수에 맞는 월급 구하기
	intSalary, err := strconv.Atoi(decryptedSalary)
	if err != nil {
		return 0, err
	}
	monthSalary := math.Round(float64(intSalary) * 10000.0 / 12.0)
	realSalary := math.Round(monthSalary / float64(whole) * float64(days))

	return realSalary, nil
}

// allMonthlySalaryFunc 함수는 실제 월 총 급여를 계산하여 반환하는 함수이다.
func allMonthlySalaryFunc(salary string, count int) (float64, error) {
	if salary == "" {
		return 0, nil
	}

	decryptedSalary, err := decryptAES256Func(salary) // 연봉 정보 복호화
	if err != nil {
		return 0, err
	}
	intSalary, err := strconv.Atoi(decryptedSalary)
	if err != nil {
		return 0, err
	}
	monthlySalary := math.Round(float64(intSalary) * 10000.0 / 12.0)

	return monthlySalary * float64(count), nil
}
