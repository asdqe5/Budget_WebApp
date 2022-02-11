// 프로젝트 결산 프로그램
//
// Description : template에서 사용하는 비용 관련 스크립트

package main

import (
	"fmt"
	"math"
	"strconv"

	"github.com/dustin/go-humanize"
)

// decryptCostFunc 함수는 프로젝트 비용을 복호화하여 반환하는 함수이다.
func decryptCostFunc(cost string, withComma bool) string {
	result, err := decryptAES256Func(cost)
	if err != nil {
		return ""
	}

	if withComma {
		intResult, _ := strconv.Atoi(result)
		return humanize.Comma(int64(intResult)) // 100 단위로 콤마 찍음
	}

	return result
}

// decryptPaymentFunc 함수는 매출 비용의 합을 복호화하여 반환하는 함수이다.
func decryptPaymentFunc(payment []Payment, withComma bool) string {
	total := 0
	for _, pay := range payment {
		decrypted, err := decryptAES256Func(pay.Expenses)
		if err != nil {
			return ""
		}
		if decrypted != "" {
			decryptedInt, err := strconv.Atoi(decrypted)
			if err != nil {
				return ""
			}
			total += decryptedInt
		}
	}

	if withComma {
		return humanize.Comma(int64(total))
	}

	return strconv.Itoa(total)
}

// totalOfPurchaseCostFunc 함수는 입력받은 date에 해당하는 구매 내역의 총액을 반환하는 함수이다.
func totalOfPurchaseCostFunc(purchaseCostMap map[string][]PurchaseCost, date string, withComma bool) string {
	total := 0
	for _, purchaseCost := range purchaseCostMap[date] {
		expenses := decryptCostFunc(purchaseCost.Expenses, false)
		expensesInt, _ := strconv.Atoi(expenses)
		total = total + expensesInt
	}

	if withComma {
		return humanize.Comma(int64(total)) // 100 단위로 콤마 찍음
	}

	if total == 0 {
		return ""
	}

	return humanize.Comma(int64(total)) // 100 단위로 콤마 찍음
}

// totalLaborOfCostSumFunc 함수는 디테일 페이지에서 내부 인건비의 총합을 반환하는 함수이다.
func totalLaborOfCostSumFunc(costSum map[string]string) string {
	vfxLaborCost := decryptCostFunc(costSum["VFX"], false)
	vfxLaborCostInt, err := strconv.Atoi(vfxLaborCost)
	if err != nil {
		vfxLaborCostInt = 0
	}
	cmLaborCost := decryptCostFunc(costSum["CM"], false)
	cmLaborCostInt, err := strconv.Atoi(cmLaborCost)
	if err != nil {
		cmLaborCostInt = 0
	}
	withoutLaborCost := vfxLaborCostInt + cmLaborCostInt

	return humanize.Comma(int64(withoutLaborCost))
}

// totalOfFinishedLaborCostFunc 함수는 정산 완료된 프로젝트의 내부 인건비 총액을 반환하는 함수이다.
func totalOfFinishedLaborCostFunc(laborCost LaborCost, withComma bool) string {
	vfx := decryptCostFunc(laborCost.VFX, false)
	vfxInt, _ := strconv.Atoi(vfx)
	cm := decryptCostFunc(laborCost.CM, false)
	cmInt, _ := strconv.Atoi(cm)
	total := vfxInt + cmInt

	if total == 0 {
		return ""
	}

	if withComma {
		return humanize.Comma(int64(total)) // 100 단위로 콤마 찍음
	}

	return strconv.Itoa(total)
}

// calRatioFunc 함수는 총 매출 대비 cost의 비율을 계산하여 반화하는 함수이다.
func calRatioFunc(cost string, payment []Payment) string {
	decryptedCost := decryptCostFunc(cost, false)
	costFloat, _ := strconv.ParseFloat(decryptedCost, 64)
	decryptedPayment := decryptPaymentFunc(payment, false)
	paymentFloat, _ := strconv.ParseFloat(decryptedPayment, 64)
	if paymentFloat == 0 {
		return "0"
	}

	ratio := math.Round(costFloat / paymentFloat * 100.0) // 소수점 첫째자리에서 반올림

	return fmt.Sprintf("%.f", ratio)
}
