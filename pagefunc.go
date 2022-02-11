// 프로그램 결산 프로그램
//
// Description : 페이지 관련 스크립트

package main

import (
	"strconv"
)

// TotalPageFunc 함수는 아이템의 갯수를 입력받아 필요한 총 페이지 수를 구한다.
func TotalPageFunc(itemNum, limitnum int64) int64 {
	page := itemNum / limitnum
	if itemNum%limitnum != 0 {
		page++
	}
	return page
}

// PageToIntFunc 함수는 페이지 문자를 받아서 Int형 페이지수를 반환한다.
func PageToIntFunc(page string) int64 {
	n, err := strconv.ParseInt(page, 10, 64)
	if err != nil {
		return 1 // 변환할 수 없는 문자라면, 1페이지를 반환한다.
	}
	return n
}

// PageToStringFunc 함수는 페이지 문자를 받아서 String형 페이지수를 반환한다.
func PageToStringFunc(page string) string {
	// url에서 "&page=1"이 아닌 "&page=1#" 로 실수로 입력했을 때,
	// 변환할 수 없는 문자라면, 1페이지를 반환하도록 하기 위해 이 함수가 존재한다.
	_, err := strconv.Atoi(page)
	if err != nil {
		return "1"
	}
	return page
}

// PreviousPageFunc 함수는 이전 페이지를 반환한다.
func PreviousPageFunc(current, maxnum int64) int64 {
	if maxnum < current {
		return maxnum
	}
	if current == 1 {
		return 1
	}
	return current - 1
}

// NextPageFunc 함수는 다음 페이지를 반환한다.
func NextPageFunc(current, maxnum int64) int64 {
	if maxnum <= current {
		return maxnum
	}
	return current + 1
}

// SplitPageFunc 함수는 로그 페이지 하단의 페이지 수를 나누는 함수이다,
func SplitPageFunc(current, total int64) []int64 {
	var pages []int64
	count := (current - 1) / 10 // 1~10 까지만 보이도록
	for i := 10*count + 1; i <= total; i++ {
		pages = append(pages, i)
		if i%10 == 0 {
			break
		}
	}
	return pages
}
