// 프로젝트 결산 프로그램
//
// Description : 프로젝트 관련 스크립트

package main

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// getLastStatusOfProjectFunc 함수는 프로젝트의 제일 마지막 status를 반환하는 함수이다.
func getLastStatusOfProjectFunc(project Project) (string, error) {
	dateList, err := getDatesFunc(project.StartDate, project.SMEndDate)
	if err != nil {
		return "", err
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dateList))) // 역순으로 정렬

	for _, date := range dateList {
		status := project.SMStatus[date]
		if status != "" {
			return status, nil
		}
	}

	return "", nil
}

// getThisMonthStatusOfProjectFunc 함수는 이번달의 프로젝트 Status를 반환하는 함수이다.
func getThisMonthStatusOfProjectFunc(project Project) string {
	y, m, _ := time.Now().Date()
	date := fmt.Sprintf("%04d-%02d", y, m)
	status := project.SMStatus[date]

	if status == "" { // Status가 비어있는 경우 마지막 Status를 가져온다.
		lastStatus, err := getLastStatusOfProjectFunc(project)
		if err != nil {
			return ""
		}
		status = lastStatus
	}

	return status
}

// getLaborCostVFXFunc 함수는 프로젝트의 VFX 인건비 총합을 반환하는 함수이다.
func getLaborCostVFXFunc(project Project) (int, error) {
	dateList, err := getDatesFunc(project.StartDate, project.SMEndDate) // 프로젝트의 작업시작과 작업마감 사이의 Date를 가져온다.
	if err != nil {
		return 0, err
	}

	totalLaborCost := 0
	for _, d := range dateList {
		laborCost := project.SMMonthlyLaborCost[d]
		vfxCostInt := 0
		if laborCost.VFX != "" {
			vfxCost, err := decryptAES256Func(laborCost.VFX)
			if err != nil {
				return 0, err
			}
			vfxCostInt, err = strconv.Atoi(vfxCost)
			if err != nil {
				return 0, err
			}
		}
		totalLaborCost = totalLaborCost + vfxCostInt
	}
	return totalLaborCost, nil
}

// getLaborCostCMFunc 함수는 프로젝트의 CM 인건비 총합을 반환하는 함수이다.
func getLaborCostCMFunc(project Project) (int, error) {
	dateList, err := getDatesFunc(project.StartDate, project.SMEndDate) // 프로젝트의 작업시작과 작업마감 사이의 Date를 가져온다.
	if err != nil {
		return 0, err
	}

	totalLaborCost := 0
	for _, d := range dateList {
		laborCost := project.SMMonthlyLaborCost[d]
		cmCostInt := 0
		if laborCost.CM != "" {
			cmCost, err := decryptAES256Func(laborCost.CM)
			if err != nil {
				return 0, err
			}
			cmCostInt, err = strconv.Atoi(cmCost)
			if err != nil {
				return 0, err
			}
		}
		totalLaborCost = totalLaborCost + cmCostInt
	}
	return totalLaborCost, nil
}

// getMonthlyLaborCostFunc 함수는 입력받은 달의 총 인건비를 계산하여 반환하는 함수이다.
func getMonthlyLaborCostFunc(project Project, date string) (int, error) {
	// 월별 인건비 - VFX
	intMonthlyVFXLaborCost := 0
	if project.SMMonthlyLaborCost[date].VFX != "" {
		monthlyVFXLaborCost, err := decryptAES256Func(project.SMMonthlyLaborCost[date].VFX)
		if err != nil {
			return 0, err
		}
		intMonthlyVFXLaborCost, err = strconv.Atoi(monthlyVFXLaborCost)
		if err != nil {
			return 0, err
		}
	}

	// 월별 인건비 - CM
	intMonthlyCMLaborCost := 0
	if project.SMMonthlyLaborCost[date].CM != "" {
		monthlyCMLaborCost, err := decryptAES256Func(project.SMMonthlyLaborCost[date].CM)
		if err != nil {
			return 0, err
		}
		intMonthlyCMLaborCost, err = strconv.Atoi(monthlyCMLaborCost)
		if err != nil {
			return 0, err
		}
	}

	totalLaborCost := intMonthlyVFXLaborCost + intMonthlyCMLaborCost

	return totalLaborCost, nil
}

// getTotalLaborCostOfFPFunc 함수는 정산 완료된 프로젝트의 총 인건비를 반환하는 함수이다.
func getTotalLaborCostOfFPFunc(project Project) (int, error) {
	// VFX 인건비
	vfxCost, err := decryptAES256Func(project.FinishedCost.LaborCost.VFX)
	if err != nil {
		return 0, err
	}
	vfxCostInt := 0
	if vfxCost != "" {
		vfxCostInt, err = strconv.Atoi(vfxCost)
		if err != nil {
			return 0, err
		}
	}

	// CM 인건비
	cmCost, err := decryptAES256Func(project.FinishedCost.LaborCost.CM)
	if err != nil {
		return 0, err
	}
	cmCostInt := 0
	if cmCost != "" {
		cmCostInt, err = strconv.Atoi(cmCost)
		if err != nil {
			return 0, err
		}
	}

	totalLaborCost := vfxCostInt + cmCostInt

	return totalLaborCost, nil
}

// getTotalProgressCostFunc 함수는 프로젝트의 총 진행비를 계산하는 함수이다.
func getTotalProgressCostFunc(project Project) (int, error) {
	dateList, err := getDatesFunc(project.StartDate, project.SMEndDate) // 프로젝트의 작업시작과 작업마감 사이의 Date를 가져온다.
	if err != nil {
		return 0, err
	}

	// 월별 진행비 합산
	totalProgressCost := 0
	for _, d := range dateList {
		value := project.SMMonthlyProgressCost[d]
		progressCost, err := decryptAES256Func(value)
		if err != nil {
			return 0, err
		}
		intProgressCost := 0
		if progressCost != "" {
			intProgressCost, err = strconv.Atoi(progressCost)
			if err != nil {
				return 0, err
			}
		}
		totalProgressCost += intProgressCost
	}

	return totalProgressCost, nil
}

// getTotalPurchaseCostFunc 함수는 프로젝트의 총 구매비를 계산하는 함수이다.
func getTotalPurchaseCostFunc(project Project) (int, error) {
	dateList, err := getDatesFunc(project.StartDate, project.SMEndDate) // 프로젝트의 작업시작과 작업마감 사이의 Date를 가져온다.
	if err != nil {
		return 0, err
	}

	// 월별 구매비 합산
	totalPurchaseCost := 0
	for _, d := range dateList {
		value := project.SMMonthlyPurchaseCost[d]
		for _, pcost := range value {
			purchaseCost, err := decryptAES256Func(pcost.Expenses)
			if err != nil {
				return 0, err
			}
			intPurchaseCost := 0
			if purchaseCost != "" {
				intPurchaseCost, err = strconv.Atoi(purchaseCost)
				if err != nil {
					return 0, err
				}
			}
			totalPurchaseCost += intPurchaseCost
		}
	}

	return totalPurchaseCost, nil
}

// calTotalAmountOfFPFunc 함수는 정산 완료된 프로젝트의 총 내부비용을 계산하여 반환하는 함수이다.
func calTotalAmountOfFPFunc(project Project) (int, error) {
	// 진행비
	totalProgressCost, err := decryptAES256Func(project.FinishedCost.ProgressCost)
	if err != nil {
		return 0, err
	}
	totalProgressCostInt := 0
	if totalProgressCost != "" {
		totalProgressCostInt, err = strconv.Atoi(totalProgressCost)
		if err != nil {
			return 0, err
		}
	}

	// 구매비
	totalPurchaseCost, err := decryptAES256Func(project.FinishedCost.PurchaseCost)
	if err != nil {
		return 0, err
	}
	totalPurchaseCostInt := 0
	if totalPurchaseCost != "" {
		totalPurchaseCostInt, err = strconv.Atoi(totalPurchaseCost)
		if err != nil {
			return 0, err
		}
	}

	// 내부 인건비
	totalLaborCost := 0
	totalLaborCost, err = getTotalLaborCostOfFPFunc(project)
	if err != nil {
		return 0, err
	}

	totalAmount := totalProgressCostInt + totalPurchaseCostInt + totalLaborCost

	return totalAmount, nil
}

// calFinishedProjectCostFunc 함수는 정산 완료된 프로젝트의 총 내부비용을 계산하여 업데이트하는 함수이다.
func calFinishedProjectCostFunc(project Project) error {
	// mongoDB client 연결
	credential := options.Credential{
		Username: *flagDBID,
		Password: *flagDBPW,
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(*flagMongoDBURI).SetAuth(credential))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		return err
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return err
	}

	// 총 진행비 계산
	totalProgressCost, err := getTotalProgressCostFunc(project)
	if err != nil {
		return err
	}
	project.FinishedCost.ProgressCost, err = encryptAES256Func(strconv.Itoa(totalProgressCost))
	if err != nil {
		return err
	}

	// 총 인건비 계산
	vfxTotalLaborCost, err := getLaborCostVFXFunc(project) // VFX
	if err != nil {
		return err
	}
	project.FinishedCost.LaborCost.VFX, err = encryptAES256Func(strconv.Itoa(vfxTotalLaborCost))
	if err != nil {
		return err
	}
	cmTotalLaborCost, err := getLaborCostCMFunc(project) // CM
	if err != nil {
		return err
	}
	project.FinishedCost.LaborCost.CM, err = encryptAES256Func(strconv.Itoa(cmTotalLaborCost))
	if err != nil {
		return err
	}
	totalLaborCost := vfxTotalLaborCost + cmTotalLaborCost

	// 총 구매비 계산
	totalPurchaseCost, err := getTotalPurchaseCostFunc(project)
	if err != nil {
		return err
	}
	project.FinishedCost.PurchaseCost, err = encryptAES256Func(strconv.Itoa(totalPurchaseCost))
	if err != nil {
		return err
	}

	// 총 내부비용 계산
	project.TotalAmount, err = encryptAES256Func(strconv.Itoa(totalProgressCost + totalPurchaseCost + totalLaborCost))
	if err != nil {
		return err
	}

	err = setProjectFunc(client, project)
	if err != nil {
		return err
	}

	return nil
}

// getMonthlyRevenueFunc 함수는 프로젝트의 월별 수익을 계산하여 반환하는 함수이다.
func getMonthlyRevenueFunc(project Project, date string) (int, error) {
	// 월별 매출
	intMonthlyPayment := 0
	for _, monthlyPayment := range project.SMMonthlyPayment[date] {
		decrypted, err := decryptAES256Func(monthlyPayment.Expenses)
		if err != nil {
			return 0, err
		}
		decryptedInt, err := strconv.Atoi(decrypted)
		if err != nil {
			return 0, err
		}
		intMonthlyPayment += decryptedInt
	}

	// 월별 인건비
	laborCost, err := getMonthlyLaborCostFunc(project, date)
	if err != nil {
		return 0, err
	}

	// 월별 진행비
	intMonthlyProgressCost := 0
	if project.SMMonthlyProgressCost[date] != "" {
		monthlyProgressCost, err := decryptAES256Func(project.SMMonthlyProgressCost[date])
		if err != nil {
			return 0, err
		}
		intMonthlyProgressCost, err = strconv.Atoi(monthlyProgressCost)
		if err != nil {
			return 0, err
		}
	}

	// 월별 구매비
	monthlyTotalPurchaseCost := 0
	for _, value := range project.SMMonthlyPurchaseCost[date] {
		monthlyPurchaseCost, err := decryptAES256Func(value.Expenses)
		if err != nil {
			return 0, err
		}
		intMonthlyPurchaseCost, err := strconv.Atoi(monthlyPurchaseCost)
		if err != nil {
			return 0, err
		}
		monthlyTotalPurchaseCost += intMonthlyPurchaseCost
	}

	monthlyRevenue := intMonthlyPayment - laborCost - intMonthlyProgressCost - monthlyTotalPurchaseCost

	return monthlyRevenue, nil
}

// getRevenueOfFPFunc 함수는 정산 완료된 프로젝트의 수익을 계산하여 반환하는 함수이다.
func getRevenueOfFPFunc(project Project) (int, error) {
	// 총 매출
	paymentInt := 0
	for _, payment := range project.Payment {
		decrypted, err := decryptAES256Func(payment.Expenses)
		if err != nil {
			return 0, err
		}
		if decrypted != "" {
			decryptedInt, err := strconv.Atoi(decrypted)
			if err != nil {
				return 0, err
			}
			paymentInt += decryptedInt
		}
	}

	// 총 인건비
	laborCost := 0
	laborCost, err := getTotalLaborCostOfFPFunc(project)
	if err != nil {
		return 0, err
	}

	// 총 진행비
	progressCost, err := decryptAES256Func(project.FinishedCost.ProgressCost)
	if err != nil {
		return 0, err
	}
	progressCostInt := 0
	if progressCost != "" {
		progressCostInt, err = strconv.Atoi(progressCost)
		if err != nil {
			return 0, err
		}
	}

	// 총 구매비
	purchaseCost, err := decryptAES256Func(project.FinishedCost.PurchaseCost)
	if err != nil {
		return 0, err
	}
	purchaseCostInt := 0
	if purchaseCost != "" {
		purchaseCostInt, err = strconv.Atoi(purchaseCost)
		if err != nil {
			return 0, err
		}
	}

	// 경영관리실 비용
	difference, err := decryptAES256Func(project.SMDifference)
	if err != nil {
		return 0, err
	}
	differenceInt := 0
	if difference != "" {
		differenceInt, err = strconv.Atoi(difference)
		if err != nil {
			return 0, err
		}
	}

	revenue := paymentInt - laborCost - progressCostInt - purchaseCostInt - differenceInt

	return revenue, nil
}

// getProjectsByTimelogFunc 함수는 해당 년월에 존재하는 타입별 타임로그의 프로젝트를 반환하는 함수이다.
func getProjectsByTimelogFunc(year int, month int, typ string) ([]Project, error) {
	var results []Project
	// mongoDB client 연결
	credential := options.Credential{
		Username: *flagDBID,
		Password: *flagDBPW,
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(*flagMongoDBURI).SetAuth(credential))
	if err != nil {
		return results, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		return results, err
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return results, err
	}

	var timelogs []Timelog
	if typ == "vfx" {
		timelogs, err = getTimelogOfTheMonthVFXFunc(client, year, month)
		if err != nil {
			return results, err
		}
	} else {
		timelogs, err = getTimelogOfTheMonthCMFunc(client, year, month)
		if err != nil {
			return results, err
		}
	}

	var projects []string
	for _, t := range timelogs {
		if !checkStringInListFunc(t.Project, projects) {
			projects = append(projects, t.Project)
		}
	}

	for _, p := range projects {
		project, err := getProjectFunc(client, p)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				continue
			}
			return nil, err
		}
		results = append(results, project)
	}

	return results, nil
}
