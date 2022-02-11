// 프로젝트 결산 프로그램
//
// Description : 프로젝트 재무 관리 툴 내의 아티스트 관련 함수

package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// realSalaryFunc 함수는 입력받은 연도와 월, 아티스트의 정보를 기준으로 실지급액을 계산하는 함수이다.
func realSalaryFunc(artist Artist, year int, month int) (float64, error) {
	startDay := artist.StartDay // 아티스트 입사일
	if startDay == "" {         // 입사일이 없는 경우
		return 0, nil
	}
	startDate, err := time.Parse("2006-01-02", startDay) // 입사일 Date
	if err != nil {
		return 0, err
	}
	startFirstDate := time.Date(startDate.Year(), startDate.Month(), 1, 0, 0, 0, 0, time.UTC) // 입사월의 시작일 Date
	startLastDate := startFirstDate.AddDate(0, 1, -1)                                         // 입사월의 말일 Date
	startMonthDays := int(startLastDate.Sub(startFirstDate).Hours()/24) + 1                   // 입사월의 일수

	tempFirstDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC) // 입력받은 연월에 해당하는 시작일 Date
	lastDate := tempFirstDate.AddDate(0, 1, -1)                                  // 입력받은 연월의 말일 Date

	var endDate time.Time      // 퇴사일 Date
	var endFirstDate time.Time // 퇴사월의 시작일 Date
	var endMonthDays int       // 퇴사월의 일수
	if artist.Resination {     // 아티스트가 퇴사를 한 경우
		endDate, err = time.Parse("2006-01-02", artist.EndDay) // 퇴사일 Date
		if err != nil {
			return 0, err
		}
		endFirstDate = time.Date(endDate.Year(), endDate.Month(), 1, 0, 0, 0, 0, time.UTC) // 퇴사월의 첫날 Date
		endLastDate := endFirstDate.AddDate(0, 1, -1)                                      // 퇴사월의 말일 Date
		endMonthDays = int(endLastDate.Sub(endFirstDate).Hours()/24) + 1
	}
	var changeDate time.Time      // 동일 연도 연봉 변경일 Date
	var strChangeSalary string    // 동일 연도 연봉 변경 전 연봉
	var changeFirstDate time.Time // 동일 연도 연봉 변경일 월의 첫날 Date
	var diffChangeDay int         // 동일 연도 연봉 변경일까지의 일수(변경일 제외)
	var changeMonthDays int       // 동일 연도 연봉 변경월의 일수
	if artist.Changed {           // 아티스트 동일 연도 연봉이 변경된 경우
		for key, value := range artist.ChangedSalary {
			changeDate, err = time.Parse("2006-01-02", key) // 연봉 변경 날짜
			if err != nil {
				return 0, err
			}
			strChangeSalary = value
		}
		changeFirstDate = time.Date(changeDate.Year(), changeDate.Month(), 1, 0, 0, 0, 0, time.UTC)
		changeLastDate := changeFirstDate.AddDate(0, 1, -1) // 동일 연도 연봉 변경일 월의 마지막날 Date
		changeMonthDays = int(changeLastDate.Sub(changeFirstDate).Hours()/24) + 1
		diffChangeDay = int(changeDate.Sub(changeFirstDate).Hours() / 24)
	}
	firstDate := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC) // 입력받은 연도의 첫날 Date
	sumSalary := 0.0                                                    // Total 실지급액

	// 조건별 필요 데이터
	// difference of Months
	diffCSMonth := int(changeDate.Month() - startDate.Month()) // 연봉 변경일과 입사일의 월차이
	diffLSMonth := int(lastDate.Month() - startDate.Month())   // 입사일과 입력받은 연도기준 말일의 월차이
	diffLCMonth := int(lastDate.Month() - changeDate.Month())  // 연봉 변경일과 입력받은 연도기준 말일의 월차이
	diffECMonth := int(endDate.Month() - changeDate.Month())   // 퇴사일과 연봉 변경일의 월차이
	diffESMonth := int(endDate.Month() - startDate.Month())    // 퇴사일과 입사일의 월차이
	diffLFMonth := int(lastDate.Month() - firstDate.Month())   // 연 첫날과 입력받은 연도기준 말일의 월차이
	diffCFMonth := int(changeDate.Month() - firstDate.Month()) // 연 첫날과 연봉 변경일의 월차이
	diffEFMonth := int(endDate.Month() - firstDate.Month())    // 연 첫날과 퇴사일의 월차이

	// difference of Days
	diffStartDay := int(startDate.Sub(startFirstDate).Hours() / 24) // 입사월의 시작일부터 입사일까지의 Day
	diffEndDay := int(endDate.Sub(endFirstDate).Hours()/24) + 1     // 퇴사월의 시작일부터 퇴사일까지의 Day
	diffCSDay := int(changeDate.Sub(startDate).Hours() / 24)        // 입사일과 연봉 변경일의 차이 Day
	diffECDay := int(endDate.Sub(changeDate).Hours()/24) + 1        // 연봉 변경일과 퇴사일의 차이 Day
	diffESDay := int(endDate.Sub(startDate).Hours()/24) + 1         // 입사일과 퇴사일의 차이 Day

	// little Salary
	littleCSSalary, err := realMonthlySalaryFunc(strChangeSalary, startMonthDays, diffCSDay) // 입사일과 연봉 변경일 사이의 급여
	if err != nil {
		return 0, err
	}
	littleECSalary, err := realMonthlySalaryFunc(artist.Salary[strconv.Itoa(year)], changeMonthDays, diffECDay) // 연봉 변경일과 퇴사일 사이의 급여
	if err != nil {
		return 0, err
	}
	littleESSalary, err := realMonthlySalaryFunc(artist.Salary[strconv.Itoa(year)], startMonthDays, diffESDay) // 입사일과 퇴사일 사이의 급여
	if err != nil {
		return 0, err
	}

	// all Salary
	allLSBeforeSalary, err := allMonthlySalaryFunc(strChangeSalary, diffLSMonth) // 입사일과 이번달의 말일 사이의 온전한 급여 -> 변경 전
	if err != nil {
		return 0, err
	}
	allCSSalary, err := allMonthlySalaryFunc(strChangeSalary, diffCSMonth-1) // 입사일과 연봉 변경일 사이의 온전한 급여
	if err != nil {
		return 0, err
	}
	allECSalary, err := allMonthlySalaryFunc(artist.Salary[strconv.Itoa(year)], diffECMonth-1) // 연봉 변경일과 퇴사일 사이의 온전한 급여
	if err != nil {
		return 0, err
	}
	allESSalary, err := allMonthlySalaryFunc(artist.Salary[strconv.Itoa(year)], diffESMonth-1) // 입사일과 퇴사일 사이의 온전한 급여
	if err != nil {
		return 0, err
	}
	allLCSalary, err := allMonthlySalaryFunc(artist.Salary[strconv.Itoa(year)], diffLCMonth) // 연봉 변경일과 이번달의 말일 사이의 온전한 급여
	if err != nil {
		return 0, err
	}
	allLSSalary, err := allMonthlySalaryFunc(artist.Salary[strconv.Itoa(year)], diffLSMonth) // 입사일과 이번달의 말일 사이의 온전한 급여 -> 변경 후
	if err != nil {
		return 0, err
	}
	allLFSalary, err := allMonthlySalaryFunc(strChangeSalary, diffLFMonth+1) // 연 첫날과 입력받은 연도기준 말일 사이의 온전한 월급
	if err != nil {
		return 0, err
	}
	allLFAfterSalary, err := allMonthlySalaryFunc(artist.Salary[strconv.Itoa(year)], diffLFMonth+1) // 연 첫날과 입력받은 연도기준 말일 사이의 온전한 월급
	if err != nil {
		return 0, err
	}
	allCFSalary, err := allMonthlySalaryFunc(strChangeSalary, diffCFMonth) // 연 첫날과 연봉 변경일 사이의 온전한 월급
	if err != nil {
		return 0, err
	}
	allEFSalary, err := allMonthlySalaryFunc(artist.Salary[strconv.Itoa(year)], diffEFMonth) // 연 첫날과 퇴사일 사이의 온전한 월급
	if err != nil {
		return 0, err
	}

	// Basic Salary
	startSalary, err := realMonthlySalaryFunc(artist.Salary[strconv.Itoa(year)], startMonthDays, startMonthDays-diffStartDay) // 입사일 기준 급여
	if err != nil {
		return 0, err
	}
	startChangeSalary, err := realMonthlySalaryFunc(strChangeSalary, startMonthDays, startMonthDays-diffStartDay) // 입사일 기준 급여 -> 변경 전
	if err != nil {
		return 0, err
	}
	changeBeforeSalary, err := realMonthlySalaryFunc(strChangeSalary, changeMonthDays, diffChangeDay) // 변경일 기준 급여 -> 변경 전
	if err != nil {
		return 0, err
	}
	changeAfterSalary, err := realMonthlySalaryFunc(artist.Salary[strconv.Itoa(year)], changeMonthDays, changeMonthDays-diffChangeDay) // 변경일 기준 급여 -> 변경 후
	if err != nil {
		return 0, err
	}
	endSalary, err := realMonthlySalaryFunc(artist.Salary[strconv.Itoa(year)], endMonthDays, diffEndDay) // 퇴사일 기준 급여
	if err != nil {
		return 0, err
	}

	if startDate.Year() == year { // 입사 연도와 입력받은 연도가 같은 경우 -> 입사일 기준
		if artist.Resination { // 아티스트가 퇴사를 했는지 안했는지
			if endDate.After(lastDate) { // 입력받은 날 기준
				if artist.Changed { // 동일 연도 연봉 변경
					if changeDate.After(lastDate) { // 변경 전 연봉으로 계산
						if changeDate.Year() == lastDate.Year() { // 변경 연도가 같은 경우
							return startChangeSalary + allLSBeforeSalary, nil
						} else { // 변경 연도가 다른 경우
							return startSalary + allLSSalary, nil
						}
					} else { // 변경일 적용해서 계산
						if diffCSMonth == 0 {
							sumSalary += littleCSSalary
						} else {
							sumSalary += startChangeSalary + allCSSalary + changeBeforeSalary
						}

						sumSalary += allLCSalary + changeAfterSalary

						return sumSalary, nil
					}
				}

				// 동일 연도 연봉 변경 X
				return startSalary + allLSSalary, nil
			} else { // 퇴사일 기준
				if artist.Changed { // 동일 연도 연봉 변경
					if diffCSMonth == 0 {
						sumSalary += littleCSSalary
					} else {
						sumSalary += startChangeSalary + allCSSalary + changeBeforeSalary
					}

					if diffECMonth == 0 {
						sumSalary += littleECSalary
					} else {
						sumSalary += changeAfterSalary + allECSalary + endSalary
					}

					return sumSalary, nil
				}

				// 동일 연도 연봉 변경 X
				if diffESMonth == 0 {
					return littleESSalary, nil
				} else {
					return startSalary + allESSalary + endSalary, nil
				}
			}
		}
		// 아티스트 퇴사 X
		if artist.Changed { // 동일 연도 연봉 변경
			if changeDate.After(lastDate) {
				if changeDate.Year() == lastDate.Year() { // 변경 연도가 같은 경우
					return startChangeSalary + allLSBeforeSalary, nil
				} else { // 변경 연도가 다른 경우
					return startSalary + allLSSalary, nil
				}
			} else {
				if diffCSMonth == 0 {
					sumSalary += littleCSSalary
				} else {
					sumSalary += startChangeSalary + allCSSalary + changeBeforeSalary
				}

				sumSalary += allLCSalary + changeAfterSalary

				return sumSalary, nil
			}
		}

		// 동일 연도 연봉 변경 X
		return startSalary + allLSSalary, nil
	} else if startDate.Year() < year { // 입사 연도가 입력받은 연도보다 작은 경우 -> 해당 연도의 첫날 기준
		if artist.Resination { // 아티스트가 퇴사한 경우
			if endDate.After(lastDate) { // 퇴사일이 입력받은 말일 이후인 경우 -> 말일 기준
				if artist.Changed { // 동일 연도 연봉이 변경된 경우
					if changeDate.After(lastDate) { // 동일 연도 연봉 변경일이 말일 이후인 경우
						if changeDate.Year() == lastDate.Year() { // 동일 연도 연봉 변경일 연도가 같은 경우
							return allLFSalary, nil
						} else { // 동일 연도 연봉 변경일 연도가 다른 경우
							return allLFAfterSalary, nil
						}
					} else {
						if changeDate.Year() == lastDate.Year() {
							return allCFSalary + changeBeforeSalary + changeAfterSalary + allLCSalary, nil
						} else {
							return allLFAfterSalary, nil
						}
					}
				}
				// 동일 연도 연봉 변경 X
				return allLFAfterSalary, nil
			} else {
				if endDate.Year() == lastDate.Year() { // 퇴사 연도가 입력받은 연도와 같은 경우
					if artist.Changed {
						if changeDate.Year() == lastDate.Year() {
							sumSalary += allCFSalary + changeBeforeSalary

							if diffECMonth == 0 {
								sumSalary += littleECSalary
							} else {
								sumSalary += changeAfterSalary + allECSalary + endSalary
							}

							return sumSalary, nil
						} else {
							return allEFSalary + endSalary, nil
						}
					}

					return allEFSalary + endSalary, nil
				} else { // 퇴사 연도가 입력받은 연도와 다른 경우
					return 0, errors.New(fmt.Sprintf("ID %s, 이름 %s 입력받은 연도 이전에 퇴사한 아티스트입니다.", artist.ID, artist.Name))
				}
			}
		}

		// 아티스트가 퇴사를 하지 않음
		if artist.Changed { // 동일 연도 연봉 변경
			if changeDate.After(lastDate) {
				if changeDate.Year() == lastDate.Year() {
					return allLFSalary, nil
				} else {
					return allLFAfterSalary, nil
				}
			} else {
				if changeDate.Year() == lastDate.Year() {
					return allCFSalary + changeBeforeSalary + changeAfterSalary + allLCSalary, nil
				} else {
					return allLFAfterSalary, nil
				}
			}
		}

		return allLFAfterSalary, nil
	} else { // 입사 연도가 입력받은 연도보다 큰 경우 -> 에러
		return 0, errors.New(fmt.Sprintf("ID %s, 이름 %s 아티스트의 입사 연도가 잘못되었습니다.", artist.ID, artist.Name))
	}
}

// workingDayFunc 함수는 입력받은 연도와 월, 아티스트의 정보를 기준으로 총 근무일수를 계산하는 함수이다,
func workingDayFunc(artist Artist, year int, month int) (int, error) {
	startDay := artist.StartDay // 아티스트 입사일
	if startDay == "" {         // 입사일이 없는 경우
		return 0, nil
	}
	startDate, err := time.Parse("2006-01-02", startDay) // 입사일 Date
	if err != nil {
		return 0, err
	}
	tempFirstDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC) // 입력받은 연월에 해당하는 시작일 Date
	lastDate := tempFirstDate.AddDate(0, 1, -1)                                  // 입력받은 연월의 말일 Date

	firstDate := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC) // 입력받은 연도의 시작일 Date
	var endDate time.Time
	if artist.Resination { //퇴사를 한 경우
		endDate, err = time.Parse("2006-01-02", artist.EndDay) // 퇴사일 Date
		if err != nil {
			return 0, err
		}
	}

	if startDate.After(lastDate) {
		return 0, errors.New(fmt.Sprintf("ID %s, 이름 %s 입사일이 잘못되었습니다.", artist.ID, artist.Name))
	} else {
		if startDate.Before(firstDate) { // 1월 1일 기준
			if artist.Resination { // 퇴사를 한 경우
				if endDate.Before(firstDate) {
					return 0, nil
				} else {
					if endDate.After(lastDate) {
						return int(lastDate.Sub(firstDate).Hours()/24) + 1, nil
					} else {
						return int(endDate.Sub(firstDate).Hours()/24) + 1, nil
					}
				}
			}

			return int(lastDate.Sub(firstDate).Hours()/24) + 1, nil
		} else { // 입사일 기준
			if artist.Resination {
				if endDate.After(lastDate) {
					return int(lastDate.Sub(startDate).Hours()/24) + 1, nil
				} else {
					return int(endDate.Sub(startDate).Hours()/24) + 1, nil
				}
			}

			return int(lastDate.Sub(startDate).Hours()/24) + 1, nil
		}
	}
}

// hourlyWageFunc 함수는 아티스트의 시급을 계산하는 함수이다.
func hourlyWageFunc(artist Artist, year int, month int) (float64, error) {
	realSalary, err := realSalaryFunc(artist, year, month) // 연 실지급액
	if err != nil {
		return 0, err
	}
	workingDay, err := workingDayFunc(artist, year, month)
	if err != nil {
		return 0, err
	}
	if workingDay == 0 {
		return 0, nil
	}

	return math.Round((realSalary / float64(workingDay)) / 8), nil
}

// averageWageByTeamsFunc 함수는 해당하는 팀들에 속하는 아티스트들의 평균 인건비를 계산하는 함수이다.
func averageWageByTeamsFunc(task string, teams []string) (float64, error) {
	// mongoDB client 연결
	credential := options.Credential{
		Username: *flagDBID,
		Password: *flagDBPW,
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(*flagMongoDBURI).SetAuth(credential))
	if err != nil {
		return 0.0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		return 0.0, err
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return 0.0, err
	}

	// 입력받은 팀에 해당하는 아티스트를 가져온다.
	if teams == nil {
		return 0.0, fmt.Errorf("%s에 해당하는 팀이 존재하지 않습니다. 팀세팅을 확인해주세요", task)
	}
	artists, err := getArtistByTeamsFunc(client, teams)
	if err != nil {
		return 0.0, err
	}

	// 가져온 아티스트를 통해 평균 인건비를 계산한다.
	totalSalary := 0.0
	for _, artist := range artists {
		salary := artist.Salary[strconv.Itoa(time.Now().Year())] // 아티스트의 올해 연봉 정보

		// 아티스트의 동일 연도 연봉 변경 정보가 있는지 체크
		if artist.Changed {
			// 오늘 날짜와 동일 연도 연봉 변경일 비교
			thisDate := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC)
			for key, value := range artist.ChangedSalary {
				changedDate, err := time.Parse("2006-01-02", key)
				if err != nil {
					return 0.0, err
				}
				if thisDate.Before(changedDate) { // 변경 전 연봉으로 계산
					salary = value
				}
			}
		}

		// 연봉 정보가 있다면 계산
		if salary != "" {
			decryptSalary, err := decryptAES256Func(salary)
			if err != nil {
				return 0.0, err
			}
			if decryptSalary != "" { // 복호화된 금액 정보가 있다면
				intSalary, err := strconv.Atoi(decryptSalary)
				if err != nil {
					return 0.0, err
				}
				cost := float64(intSalary) * 10000 / 12
				totalSalary += cost
			}
		}
	}
	averageCost := (totalSalary / 30) / float64(len(artists))

	return math.Round(averageCost), nil
}
