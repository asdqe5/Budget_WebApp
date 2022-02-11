// 프로젝트 결산 프로그램
//
// Description : template에서 사용하는 스크립트

package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
)

// intToAccessLevelFunc 함수는 AccessLevel을 string형으로 변환하는 함수이다.
func intToAccessLevelFunc(level AccessLevel) string {
	return level.String()
}

// addInt 함수는 int형 변수 두개를 입력받아 합산하여 반환하는 함수이다.
func addIntFunc(value1, value2 int) int {
	return (value1 + value2)
}

// stringToDateFunc 함수는 "2020-09", "2020-09-01" 형태의 문자열을 "2020년 9월", "2020년 9월 1일" 형태로 변환하는 함수이다.
func stringToDateFunc(date string) string {
	if date == "" {
		return ""
	}

	if regexDate.MatchString(date) {
		return fmt.Sprintf("%s년 %s월", strings.Split(date, "-")[0], strings.Split(date, "-")[1])
	} else if regexDate2.MatchString(date) {
		return fmt.Sprintf("%s년 %s월 %s일", strings.Split(date, "-")[0], strings.Split(date, "-")[1], strings.Split(date, "-")[2])
	} else {
		return "지원하는 날짜 포맷이 아닙니다"
	}
}

// getColorOfRevenueFunc 함수는 수익에 맞는 텍스트 컬러를 반환하는 함수이다.
func getColorOfRevenueFunc(encryptedRevenue string) string {
	revenue := decryptCostFunc(encryptedRevenue, false)
	revenueInt, _ := strconv.Atoi(revenue)
	if revenueInt < 0 {
		return "text-danger"
	}
	return "text-primary"
}

// checkLineChangeFunc 함수는 로그 내용을 띄어쓰기 해야하는지 체크하는 함수이다.
func checkLineChangeFunc(content string) bool {
	if strings.Contains(content, "\n") {
		return true
	}
	return false
}

// splitLineFunc 함수는 줄바꿈을 기준으로 문자열을 나누는 함수이다.
func splitLineFunc(content string) []string {
	return strings.Split(content, "\n")
}

// durationToTimeFunc 함수는 duration을 시간으로 반환해주는 함수이다.
func durationToTimeFunc(duration float64) string {
	if duration == 0 {
		return ""
	}

	time := fmt.Sprintf("%.1f", duration) + "h"
	return time
}

// supDurationToTimeFunc 함수는 supervisor timelog duration을 시간으로 구해주는 함수이다.
func supDurationToTimeFunc(duration float64) float64 {
	result := math.Round(duration/60*10) / 10
	return result
}

// hasStatusFunc 함수는 status가 선택한 status에 포함되는지 확인하는 함수이다.
func hasStatusFunc(selectedStatus string, status string) bool {
	statusList := stringToListFunc(selectedStatus, ",")
	return checkStringInListFunc(status, statusList)
}

// getStatusFunc 함수는 해당하는 Status를 반환하는 함수이다.
func getStatusFunc(statusList []Status, status string) Status {
	ps := Status{}
	for _, s := range statusList {
		if s.ID == status {
			return s
		}
	}
	return ps
}

// getMonthlyPaymentInfoFunc 함수는 프로젝트 월별 매출에 관한 정보를 반환하는 함수이다.
func getMonthlyPaymentInfoFunc(monthlyPayment map[string][]Payment, date string) map[string]string {
	paymentMap := make(map[string]string)
	total := 0
	tooltip := ""
	in := "true" // 해당 월에 매출들이 모두 입금되었는지 확인하기 위함

	for _, payment := range monthlyPayment[date] {
		decrypted, err := decryptAES256Func(payment.Expenses)
		if err != nil {
			return nil
		}
		if decrypted != "" {
			decryptedInt, err := strconv.Atoi(decrypted)
			if err != nil {
				return nil
			}
			total += decryptedInt

			// 입금이 되었다면 툴팁에 입금일과 OK 문구를 넣어주고, 입금이 되지 않았다면 in을 false로 설정한다.
			if payment.Status {
				tooltip += fmt.Sprintf("%s : %s (발행일 : %s, 입금일 : %s) OK\n", payment.Type, humanize.Comma(int64(decryptedInt)), stringToDateFunc(payment.Date), stringToDateFunc(payment.DepositDate))
			} else {
				tooltip += fmt.Sprintf("%s : %s (발행일 : %s)\n", payment.Type, humanize.Comma(int64(decryptedInt)), stringToDateFunc(payment.Date))
				in = "false"
			}

			// 입금일을 작성하지 않았다면 in을 false로 설정한다.
			if payment.DepositDate == "" {
				in = "false"
			}
		}
	}

	// 해당 월의 총 비용 암호화
	encryptedExpenses, err := encryptAES256Func(strconv.Itoa(total))
	if err != nil {
		return nil
	}
	paymentMap["payment"] = encryptedExpenses
	paymentMap["tooltip"] = tooltip
	paymentMap["in"] = in

	return paymentMap
}

// putCommaFunc 함수는 입력받은 정수에 세자릿수 단위로 콤마를 찍어서 반환하는 함수이다.
func putCommaFunc(num int) string {
	return humanize.Comma(int64(num))
}

// changeDateFormatFunc 함수는 "2021-08-10T14:50:50+09:00" 형식의 string을 "" 형식으로 변환하여 반환하는 함수이다.
func changeDateFormatFunc(date string) string {
	return strings.Replace(strings.Split(date, "+")[0], "T", " ", 1)
}
