// 프로젝트 결산 프로그램
//
// Description : 레귤러 익스프레션 스크립트
//
// Main Author : aerim.shim @ RD101
// Sub Author :

package main

import (
	"regexp"
)

var (
	regexIPv4         = regexp.MustCompile(`^([01]?\d?\d|2[0-4]\d|25[0-5]).([01]?\d?\d|2[0-4]\d|25[0-5]).([01]?\d?\d|2[0-4]\d|25[0-5]).([01]?\d?\d|2[0-4]\d|25[0-5])$`) // 0.0.0.0 ~ 255.255.255.255
	regexLower        = regexp.MustCompile(`[a-z]+$`)
	regexName         = regexp.MustCompile(`^[가-힣a-zA-Z]+$`)
	regexSalary       = regexp.MustCompile(`^\d{4}:[0-9]+(,\d{4}:[0-9]+)*$`)                                   // 2020:2400,2020:2400
	regexChangeSalary = regexp.MustCompile(`^\d{4}-(0?[1-9]|1[012])-(0?[1-9]|[12][0-9]|3[01]):[0-9]+$`)        // 2020-04-01:1920
	regexMap          = regexp.MustCompile(`^[a-zA-Z0-9-]+:[a-zA-Z0-9.-_]+(,[a-zA-Z0-9-]+:[a-zA-Z0-9.-_]+)*$`) // key:value,key:value
	regexDate         = regexp.MustCompile(`^\d{4}-(0?[1-9]|1[012])$`)                                         // 2020-07
	regexDate2        = regexp.MustCompile(`^\d{4}-(0?[1-9]|1[012])-(0?[1-9]|[12][0-9]|3[01])$`)               // 2020-07-01
	regexDept         = regexp.MustCompile(`^[a-zA-Z0-9+]+$`)
	regexWord         = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	regexProject      = regexp.MustCompile(`^[A-Z0-9_]+$`) // BEE, RND2020, CM_ART
	regexDigit        = regexp.MustCompile(`^[0-9]+$`)     // 숫자
	regexWebColor     = regexp.MustCompile(`^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$`)
)
