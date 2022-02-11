// 프로젝트 결산 프로그램
//
// Description : template에서 사용하는 벤더 관련 스크립트

package main

import (
	"fmt"
	"strconv"

	"github.com/dustin/go-humanize"
)

// lenOfVendorsMapFunc 함수는 벤더맵의 길이를 반환하는 함수이다.
func lenOfVendorsMapFunc(vendorsMap map[string][]Vendor, typ bool) int {
	result := 0
	if typ {
		for _, v := range vendorsMap {
			for _, mp := range v {
				if len(mp.MediumPlating) == 0 {
					result++
				}
				result += len(mp.MediumPlating)
			}
		}
	} else {
		for _, v := range vendorsMap {
			result += len(v)
		}
	}
	return result
}

// lenOfVendorsListFunc 함수는 벤더리스트의 길이를 반환하는 함수이다.
func lenOfVendorsListFunc(vendorsList []Vendor) int {
	result := 0
	for _, v := range vendorsList {
		if len(v.MediumPlating) == 0 {
			result++
		}
		result += len(v.MediumPlating)
	}
	return result
}

// setVendorInfoMapFunc 함수는 해당 월에 대한 벤더 비용 및 지출 여부 등의 정보를 맵 형태로 반환하는 함수이다,
func setVendorInfoMapFunc(vendor Vendor, date string) map[string]string {
	vendorInfoMap := make(map[string]string)
	expenses := 0 // 벤더 비용
	tooltip := "" // 툴팁에 적힐 문구
	out := "true" // 해당 월에 비용들이 모두 입금되었는지 확인하기 위함

	// 해당 월의 계약금 확인
	if date == dateToMonthFunc(vendor.Downpayment.Date) {
		downpayment, err := decryptAES256Func(vendor.Downpayment.Expenses)
		if err != nil {
			return nil
		}
		downpaymentInt, err := strconv.Atoi(downpayment)
		if err != nil {
			return nil
		}
		expenses += downpaymentInt

		// 계약금 지급이 되었다면 툴팁에 지급일과 OK 문구를 넣어주고, 지급이 되지 않았다면 out을 false로 설정한다.
		if vendor.Downpayment.Status == true {
			tooltip += fmt.Sprintf("계약금 : %s (발행일 : %s, 지급일 : %s) OK\n", humanize.Comma(int64(downpaymentInt)), stringToDateFunc(vendor.Downpayment.Date), stringToDateFunc(vendor.Downpayment.PayedDate))
		} else {
			tooltip += fmt.Sprintf("계약금 : %s (발행일 : %s)\n", humanize.Comma(int64(downpaymentInt)), stringToDateFunc(vendor.Downpayment.Date))
			out = "false"
		}

		// 계약금 지급일을 작성하지 않았다면 out을 false로 설정한다.
		if vendor.Downpayment.PayedDate == "" {
			out = "false"
		}
	}

	// 해당 월의 중도금 확인 - 중도금이 여러개 존재할 수 있음
	for num, mp := range vendor.MediumPlating {
		if date == dateToMonthFunc(mp.Date) { //  해당 월의 중도금 확인
			mediumplating, err := decryptAES256Func(mp.Expenses)
			if err != nil {
				return nil
			}
			mediumplatingInt, err := strconv.Atoi(mediumplating)
			if err != nil {
				return nil
			}
			expenses += mediumplatingInt

			// 증도금 지급이 되었다면 툴팁에 지급일과 OK 문구를 넣어주고, 지급이 되지 않았다면 out을 false로 설정한다.
			if mp.Status == true {
				tooltip += fmt.Sprintf("중도금%d : %s (발행일 : %s, 지급일 : %s) OK\n", num+1, humanize.Comma(int64(mediumplatingInt)), stringToDateFunc(mp.Date), stringToDateFunc(mp.PayedDate))
			} else {
				tooltip += fmt.Sprintf("중도금%d : %s (발행일 : %s)\n", num+1, humanize.Comma(int64(mediumplatingInt)), stringToDateFunc(mp.Date))
				out = "false"
			}

			// 중도금 지급일을 작성하지 않았다면 out을 false로 설정한다.
			if mp.PayedDate == "" {
				out = "false"
			}
		}
	}

	// 해당 월의 잔금 확인
	if date == dateToMonthFunc(vendor.Balance.Date) {
		balance, err := decryptAES256Func(vendor.Balance.Expenses)
		if err != nil {
			return nil
		}
		balanceInt, err := strconv.Atoi(balance)
		if err != nil {
			return nil
		}
		expenses += balanceInt

		// 잔금 지급이 되었다면 툴팁에 지급일과 OK 문구를 넣어주고, 지급이 되지 않았다면 out을 false로 설정한다.
		if vendor.Balance.Status == true {
			tooltip += fmt.Sprintf("잔금 : %s (발행일 : %s, 지급일 : %s) OK\n", humanize.Comma(int64(balanceInt)), stringToDateFunc(vendor.Balance.Date), stringToDateFunc(vendor.Balance.PayedDate))
		} else {
			tooltip += fmt.Sprintf("잔금 : %s (발행일 : %s)\n", humanize.Comma(int64(balanceInt)), stringToDateFunc(vendor.Balance.Date))
			out = "false"
		}

		// 잔금 지급일을 작성하지 않았다면 out을 false로 설정한다.
		if vendor.Balance.PayedDate == "" {
			out = "false"
		}
	}

	// 해당 월의 총 비용 암호화
	encryptedExpenses, err := encryptAES256Func(strconv.Itoa(expenses))
	if err != nil {
		return nil
	}
	vendorInfoMap["expenses"] = encryptedExpenses
	vendorInfoMap["tooltip"] = tooltip
	vendorInfoMap["out"] = out

	return vendorInfoMap
}

// getVendorTooltipFunc 함수는 벤더의 비용 정보를 툴팁으로 가져오는 함수이다.
func getVendorTooltipFunc(vendor Vendor) string {
	tooltip := ""
	downpayment, err := decryptAES256Func(vendor.Downpayment.Expenses)
	if err != nil {
		return ""
	}
	if downpayment != "" {
		downpaymentInt, err := strconv.Atoi(downpayment)
		if err != nil {
			return ""
		}
		tooltip += fmt.Sprintf("계약금 : %s (발행일 : %s)\n", humanize.Comma(int64(downpaymentInt)), stringToDateFunc(vendor.Downpayment.Date))
	}
	for n, mediumplating := range vendor.MediumPlating {
		expenses, err := decryptAES256Func(mediumplating.Expenses)
		if err != nil {
			return ""
		}
		if expenses != "" {
			expensesInt, err := strconv.Atoi(expenses)
			if err != nil {
				return ""
			}
			tooltip += fmt.Sprintf("중도금%d : %s (발행일 : %s)\n", n+1, humanize.Comma(int64(expensesInt)), stringToDateFunc(mediumplating.Date))
		}
	}
	balance, err := decryptAES256Func(vendor.Balance.Expenses)
	if err != nil {
		return ""
	}
	if balance != "" {
		balanceInt, err := strconv.Atoi(balance)
		if err != nil {
			return ""
		}
		tooltip += fmt.Sprintf("잔금 : %s (발행일 : %s)\n", humanize.Comma(int64(balanceInt)), stringToDateFunc(vendor.Balance.Date))
	}

	return tooltip
}

// calUnitPriceByCutsFunc 함수는 벤더관리페이지에서 컷별단가를 구하는 함수이다.
func calUnitPriceByCutsFunc(expenses string, cuts int) string {
	exp, err := decryptAES256Func(expenses)
	if err != nil {
		return ""
	}
	expInt, err := strconv.Atoi(exp)
	if err != nil {
		return ""
	}
	if cuts == 0 {
		return "0"
	}
	return humanize.Comma(int64(expInt / cuts))
}

// checkMediumPlatingStatusFunc 함수는 중도금 정산여부를 체크하는 함수이다.
func checkMediumPlatingStatusFunc(mediumPlating []VendorCost) bool {
	for _, mp := range mediumPlating {
		if mp.Status == false {
			return false
		}
	}
	return true
}
