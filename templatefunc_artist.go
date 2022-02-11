// 프로젝트 결산 프로그램
//
// Description : template에서 사용하는 아티스트 관련 함수

package main

import (
	"math"
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
)

// workingDayByYearFunc 함수는 입력받은 연도와 아티스트 정보를 기준으로 총 근무일수를 반환하는 함수이다.
func workingDayByYearFunc(artist Artist, year string) string {
	startDay := artist.StartDay // 아티스트 입사일
	if startDay == "" {
		return "0"
	}
	startDate, err := time.Parse("2006-01-02", startDay) // 입사일 Date
	if err != nil {
		return "0"
	}
	intYear, err := strconv.Atoi(year) // 입력받은 연도
	if err != nil {
		return "0"
	}
	var lastDate time.Time
	if intYear == time.Now().Year() { // 입력받은 연도가 올해인 경우
		tempFirstDate := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
		lastDate = tempFirstDate.AddDate(0, 1, -1) // 이번달 말일 Date
	} else if intYear < time.Now().Year() { // 입력받은 연도가 올해 전인 경우
		lastDate = time.Date(intYear, time.December, 31, 0, 0, 0, 0, time.UTC) // 입력받은 연도의 말일 Date
	} else {
		return "0"
	}
	firstDate := time.Date(intYear, time.January, 1, 0, 0, 0, 0, time.UTC) // 입력받은 연도의 시작일 Date
	var endDate time.Time
	if artist.Resination { //퇴사를 한 경우
		endDate, err = time.Parse("2006-01-02", artist.EndDay) // 퇴사일 Date
		if err != nil {
			return "0"
		}
	}
	if intYear == time.Now().Year() { // 입력받은 연도가 올해인 경우
		if startDate.After(lastDate) { // 입사일이 입력받은 날짜 이후인 경우
			return "0"
		} else {
			if startDate.Before(firstDate) { // 1월 1일 기준
				if artist.Resination {
					if endDate.Before(firstDate) {
						return "0"
					} else {
						days := int(endDate.Sub(firstDate).Hours()/24) + 1
						return strconv.Itoa(days)
					}
				}
				days := int(lastDate.Sub(firstDate).Hours()/24) + 1
				return strconv.Itoa(days)
			} else { // 입사일 기준
				if artist.Resination {
					days := int(endDate.Sub(startDate).Hours()/24) + 1
					return strconv.Itoa(days)
				}
				days := int(lastDate.Sub(startDate).Hours()/24) + 1
				return strconv.Itoa(days)
			}
		}
	} else if lastDate.Year() < time.Now().Year() { // 입력 받은 연도가 올해 전인 경우
		if startDate.After(lastDate) { // 입사일이 입력받은 날짜 이후인 경우
			return "0"
		} else {
			if startDate.Before(firstDate) { // 1월 1일 기준
				if artist.Resination {
					if endDate.Before(firstDate) {
						return "0"
					} else {
						if endDate.After(lastDate) {
							days := int(lastDate.Sub(firstDate).Hours()/24) + 1
							return strconv.Itoa(days)
						} else {
							days := int(endDate.Sub(firstDate).Hours()/24) + 1
							return strconv.Itoa(days)
						}
					}
				}
				days := int(lastDate.Sub(firstDate).Hours()/24) + 1
				return strconv.Itoa(days)
			} else { // 입사일 기준
				if artist.Resination {
					if endDate.After(lastDate) {
						days := int(lastDate.Sub(startDate).Hours()/24) + 1
						return strconv.Itoa(days)
					} else {
						days := int(endDate.Sub(startDate).Hours()/24) + 1
						return strconv.Itoa(days)
					}
				}
				days := int(lastDate.Sub(startDate).Hours()/24) + 1
				return strconv.Itoa(days)
			}
		}
	} else { // 입력 받은 연도가 올해 이후인 경우
		return "0"
	}
}

// realSalaryByYearFunc 함수는 입력받은 연도와 아티스트 정보를 기준으로 실지급액을 계산하는 함수이다.
func realSalaryByYearFunc(artist Artist, year string) float64 {
	startDay := artist.StartDay // 아티스트 입사일
	if startDay == "" {         // 입사일이 없는 경우
		return 0
	}
	startDate, err := time.Parse("2006-01-02", startDay) // 입사일 Date
	if err != nil {
		return 0
	}
	startFirstDate := time.Date(startDate.Year(), startDate.Month(), 1, 0, 0, 0, 0, time.UTC) // 입사월의 시작일 Date
	startLastDate := startFirstDate.AddDate(0, 1, -1)                                         // 입사월의 말일 Date
	startMonthDays := int(startLastDate.Sub(startFirstDate).Hours()/24) + 1                   // 입사월의 일수

	intYear, err := strconv.Atoi(year) // 현재 입력받은 연도
	if err != nil {
		return 0
	}
	var lastDate time.Time            // 입력받은 연도 기준 말일 Date
	if intYear == time.Now().Year() { // 입력받은 연도가 올해인 경우
		tempFirstDate := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
		lastDate = tempFirstDate.AddDate(0, 1, -1) // 이번달 말일 Date
	} else if intYear < time.Now().Year() { // 입력받은 연도가 올해 전인 경우
		lastDate = time.Date(intYear, time.December, 31, 0, 0, 0, 0, time.UTC) // 입력받은 연도의 말일 Date
	} else { // 입력받은 연도가 올해 후인 경우
		return 0
	}
	var endDate time.Time      // 퇴사일 Date
	var endFirstDate time.Time // 퇴사월의 시작일 Date
	var endMonthDays int       // 퇴사월의 일수
	if artist.Resination {     // 아티스트가 퇴사를 한 경우
		endDate, err = time.Parse("2006-01-02", artist.EndDay) // 퇴사일 Date
		if err != nil {
			return 0
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
				return 0
			}
			strChangeSalary = value
		}
		changeFirstDate = time.Date(changeDate.Year(), changeDate.Month(), 1, 0, 0, 0, 0, time.UTC)
		changeLastDate := changeFirstDate.AddDate(0, 1, -1) // 동일 연도 연봉 변경일 월의 마지막날 Date
		changeMonthDays = int(changeLastDate.Sub(changeFirstDate).Hours()/24) + 1
		diffChangeDay = int(changeDate.Sub(changeFirstDate).Hours() / 24)
	}
	firstDate := time.Date(intYear, time.January, 1, 0, 0, 0, 0, time.UTC) // 입력받은 연도의 첫날 Date
	sumSalary := 0.0                                                       // Total 실지급액

	// 조건별 필요 데이터
	diffCSMonth := int(changeDate.Month() - startDate.Month()) // 연봉 변경일과 입사일의 월차이
	diffECMonth := int(endDate.Month() - changeDate.Month())   // 퇴사일과 연봉 변경일의 월차이
	diffESMonth := int(endDate.Month() - startDate.Month())    // 퇴사일과 입사일의 월차이
	diffLSMonth := int(lastDate.Month() - startDate.Month())   // 입사일과 입력받은 연도기준 말일의 월차이
	diffLCMonth := int(lastDate.Month() - changeDate.Month())  // 연봉 변경일과 입력받은 연도기준 말일의 월차이
	diffEFMonth := int(endDate.Month() - firstDate.Month())    // 연 첫날과 퇴사일의 월차이
	diffLFMonth := int(lastDate.Month() - firstDate.Month())   // 연 첫날과 입력받은 연도기준 말일의 월차이
	diffCFMonth := int(changeDate.Month() - firstDate.Month()) // 연 첫날과 연봉 변경일의 월차이

	diffStartDay := int(startDate.Sub(startFirstDate).Hours() / 24) // 입사월의 시작일부터 입사일까지의 Day
	diffEndDay := int(endDate.Sub(endFirstDate).Hours()/24) + 1     // 퇴사월의 시작일부터 퇴사일까지의 Day
	diffCSDay := int(changeDate.Sub(startDate).Hours() / 24)        // 입사일과 연봉 변경일의 차이 Day
	diffECDay := int(endDate.Sub(changeDate).Hours()/24) + 1        // 연봉 변경일과 퇴사일의 차이 Day
	diffESDay := int(endDate.Sub(startDate).Hours()/24) + 1         // 입사일과 퇴사일의 차이 Day

	littleCSSalary, _ := realMonthlySalaryFunc(strChangeSalary, startMonthDays, diffCSDay)      // 입사일과 연봉 변경일 사이의 급여
	littleECSalary, _ := realMonthlySalaryFunc(artist.Salary[year], changeMonthDays, diffECDay) // 연봉 변경일과 퇴사일 사이의 급여
	littleESSalary, _ := realMonthlySalaryFunc(artist.Salary[year], startMonthDays, diffESDay)  // 입사일과 퇴사일 사이의 급여

	allCSSalary, _ := allMonthlySalaryFunc(strChangeSalary, diffCSMonth-1)          // 입사일과 연봉 변경일 사이의 온전한 급여
	allECSalary, _ := allMonthlySalaryFunc(artist.Salary[year], diffECMonth-1)      // 연봉 변경일과 퇴사일 사이의 온전한 급여
	allESSalary, _ := allMonthlySalaryFunc(artist.Salary[year], diffESMonth-1)      // 입사일과 퇴사일 사이의 온전한 급여
	allLSBeforeSalary, _ := allMonthlySalaryFunc(strChangeSalary, diffLSMonth)      // 입사일과 이번달의 말일 사이의 온전한 급여 -> 변경 전
	allLCSalary, _ := allMonthlySalaryFunc(artist.Salary[year], diffLCMonth)        // 연봉 변경일과 이번달의 말일 사이의 온전한 급여
	allLSSalary, _ := allMonthlySalaryFunc(artist.Salary[year], diffLSMonth)        // 입사일과 이번달의 말일 사이의 온전한 급여 -> 변경 후
	allEFSalary, _ := allMonthlySalaryFunc(artist.Salary[year], diffEFMonth)        // 연 첫날과 퇴사일 사이의 온전한 월급
	allLFSalary, _ := allMonthlySalaryFunc(strChangeSalary, diffLFMonth+1)          // 연 첫날과 입력받은 연도기준 말일 사이의 온전한 월급
	allCFSalary, _ := allMonthlySalaryFunc(strChangeSalary, diffCFMonth)            // 연 첫날과 연봉 변경일 사이의 온전한 월급
	allLFAfterSalary, _ := allMonthlySalaryFunc(artist.Salary[year], diffLFMonth+1) // 연 첫날과 입력받은 연도기준 말일 사이의 온전한 월급

	startSalary, _ := realMonthlySalaryFunc(artist.Salary[year], startMonthDays, startMonthDays-diffStartDay)          // 입사일 기준 급여
	startChangeSalary, _ := realMonthlySalaryFunc(strChangeSalary, startMonthDays, startMonthDays-diffStartDay)        // 입사일 기준 급여
	changeBeforeSalary, _ := realMonthlySalaryFunc(strChangeSalary, changeMonthDays, diffChangeDay)                    // 변경일 기준 급여 -> 변경 전
	changeAfterSalary, _ := realMonthlySalaryFunc(artist.Salary[year], changeMonthDays, changeMonthDays-diffChangeDay) // 변경일 기준 급여 -> 변경 후
	endSalary, _ := realMonthlySalaryFunc(artist.Salary[year], endMonthDays, diffEndDay)                               // 퇴사일 기준 급여

	if intYear == time.Now().Year() { // 입력받은 연도가 올해인 경우
		if startDate.Year() == time.Now().Year() { // 입사연도가 올해인 경우
			if artist.Resination { // 아티스트가 올해 퇴사를 한 경우 -> 퇴사일 기준으로 실지급액 계산
				if artist.Changed { // 올해 동일 연도 연봉이 바뀐 경우
					if diffCSMonth == 0 { // 연봉 변경일과 입사일의 월이 같은 경우
						sumSalary += littleCSSalary
					} else { // 연봉 변경일과 입사일의 월이 다른 경우
						sumSalary += startChangeSalary + allCSSalary + changeBeforeSalary
					}

					if diffECMonth == 0 { // 연봉 변경일과 퇴사일의 월이 같은 경우
						sumSalary += littleECSalary
					} else {
						sumSalary += changeAfterSalary + allECSalary + endSalary
					}

					return sumSalary
				}

				// 올해 동일 연도 연봉이 바뀌지 않은 경우
				if diffESMonth == 0 { // 입사일과 퇴사일의 월이 같은 경우 -> 퇴사일까지의 근무일수
					sumSalary += littleESSalary
				} else { // 입사일과 퇴사일의 월이 다른 경우
					sumSalary += startSalary + allESSalary + endSalary
				}

				return sumSalary
			}
			// 아티스트가 퇴사를 하지 않은 경우 -> 이번달 말의 기준으로 실지급액 계산
			if artist.Changed { // 올해 동일 연도 연봉이 바뀐 경우
				if changeDate.After(lastDate) { // 동일 연도 연봉 변경일이 이번달 말일 이후인 경우 -> 미리 입력을 한 경우, 변경 전 연봉으로 이번달 말일 까지 실지급액 계산
					sumSalary += allLSBeforeSalary + startChangeSalary

					return sumSalary
				} else { // 동일 연도 연봉 변경일이 이번달 말일 전인 경우
					if diffCSMonth == 0 {
						sumSalary += littleCSSalary
					} else {
						sumSalary += startChangeSalary + allCSSalary + changeBeforeSalary
					}

					sumSalary += allLCSalary + changeAfterSalary

					return sumSalary
				}
			}
			// 올해 동일 연도 연봉이 바뀌지 않은 경우
			sumSalary += startSalary + allLSSalary

			return sumSalary
		} else if startDate.Year() < time.Now().Year() { // 입사연도가 이번달의 연도보다 작은 경우 -> 1월 1일 기준
			if artist.Resination { // 아티스트가 올해 퇴사를 한 경우 -> 퇴사일 기준으로 실지급액 계산
				if endDate.Year() == time.Now().Year() { // 이번달의 연도에 퇴사를 한 경우
					if artist.Changed { // 동일 연도 연봉이 변경된 경우
						if changeDate.Year() == time.Now().Year() { // 연봉 변경 연도가 올해인 경우
							sumSalary += allCFSalary + changeBeforeSalary

							if diffECMonth == 0 { // 연봉 변경일과 퇴사일의 월이 같은 경우
								sumSalary += littleECSalary
							} else {
								sumSalary += changeAfterSalary + allECSalary + endSalary
							}

							return sumSalary
						} else {
							sumSalary += allEFSalary + endSalary

							return sumSalary
						}
					}
					// 동일 연도 연봉이 변경되지 않은 경우
					sumSalary += allEFSalary + endSalary

					return sumSalary
				} else {
					return 0
				}
			}
			// 아티스트가 퇴사를 하지 않은 경우 -> 이번달 말의 기준으로 실지급액 계산
			if artist.Changed { // 동일 연도 연봉이 바뀐 경우
				if changeDate.After(lastDate) { // 동일 연도 연봉 변경일이 이번달 말일 이후인 경우 -> 미리 입력을 한 경우, 변경 전 연봉으로 이번달 말일 까지 실지급액 계산
					sumSalary += allLFSalary

					return sumSalary
				} else { // 동일 연도 연봉 변경일이 이번달 말일 전인 경우
					if changeDate.Year() == time.Now().Year() { // 연봉 변경 연도가 올해인 경우
						sumSalary += allCFSalary + changeBeforeSalary + changeAfterSalary + allLCSalary

						return sumSalary
					} else { // 연봉 변경 연도가 올해 전인 경우
						sumSalary += allLFAfterSalary

						return sumSalary
					}
				}
			}
			// 올해 동일 연도 연봉이 바뀌지 않은 경우
			sumSalary += allLFAfterSalary

			return sumSalary
		} else { // 입사연도가 이번달의 연도보다 큰 경우 -> 에러
			return 0
		}
	} else if intYear < time.Now().Year() { // 입력받은 연도가 이번달의 연도보다 작은 경우 -> 입력받은 연도 말 기준 실지급액 계산
		if startDate.Year() == intYear { // 입사연도와 입력받은 연도가 같은 경우 -> 입사일 기준
			if artist.Resination { // 아티스트가 퇴사를 했는지 안했는지 -> 퇴사일이 입력받은 연도와 같은지 아닌지 확인
				if endDate.Year() == startDate.Year() { // 퇴사연도와 입사연도가 같은 경우 -> 퇴사일 기준
					if artist.Changed { // 동일 연도 연봉이 바뀐 경우
						if diffCSMonth == 0 { // 연봉 변경일과 입사일의 월이 같은 경우 -> 연봉 변경일까지의 근무일수
							sumSalary += littleCSSalary
						} else { // 연봉 변경일과 입사일의 월이 다른 경우
							sumSalary += startChangeSalary + allCSSalary + changeBeforeSalary
						}

						if diffECMonth == 0 {
							sumSalary += littleECSalary
						} else { // 연봉 변경일과 퇴사일의 월이 다른 경우
							sumSalary += changeAfterSalary + allECSalary + endSalary
						}

						return sumSalary
					}
					// 동일 연도 연봉이 바뀌지 않은 경우
					if diffESMonth == 0 { // 입사일과 퇴사일의 월이 같은 경우 -> 퇴사일까지의 근무일수
						sumSalary += littleESSalary
					} else { // 입사일과 퇴사일의 월이 다른 경우
						sumSalary += startSalary + allESSalary + endSalary
					}

					return sumSalary
				} else { // 퇴사연도와 입사연도가 다른 경우 -> 그해 말 기준
					if artist.Changed {
						if startDate.Year() == changeDate.Year() { // 입사 연도와 연봉 변경 연도가 같은 경우
							if diffCSMonth == 0 { // 연봉 변경일과 입사일의 월이 같은 경우 -> 연봉 변경일까지의 근무일수
								sumSalary += littleCSSalary
							} else { // 연봉 변경일과 입사일의 월이 다른 경우
								sumSalary += startChangeSalary + allCSSalary + changeBeforeSalary
							}
							sumSalary += changeAfterSalary + allLCSalary

							return sumSalary
						} else { // 입사 연도와 연봉 변경 연도가 다른 경우
							sumSalary += startSalary + allLSSalary

							return sumSalary
						}
					}
					// 동일 연도 연봉 변경이 되지 않은 경우
					sumSalary += startSalary + allLSSalary

					return sumSalary
				}
			}
			// 아티스트가 퇴사를 하지 않은 경우 -> 입력받은 연도의 말일 기준
			if artist.Changed { // 동일 연도 연봉이 변경된 경우 -> 변경된 연도 확인
				if changeDate.After(lastDate) { // 동일 연도 연봉 변경일이 그해 연말 후인 경우
					sumSalary += startSalary + allLSSalary

					return sumSalary
				} else { // 동일 연도 연봉 변경일이 그해 연말 이전인 경우
					if diffCSMonth == 0 { // 연봉 변경일과 입사일의 월이 같은 경우 -> 연봉 변경일까지의 근무일수
						sumSalary += littleCSSalary
					} else { // 연봉 변경일과 입사일의 월이 다른 경우
						sumSalary += startChangeSalary + allCSSalary + changeBeforeSalary
					}
					sumSalary += changeAfterSalary + allLCSalary

					return sumSalary
				}
			}
			// 동일 연도 연봉이 변경되지 않은 경우
			sumSalary += allLSSalary + startSalary

			return sumSalary
		} else if startDate.Year() < intYear { // 입사연도가 입력받은 연도보다 작은 경우 -> 1월 1일 기준
			if artist.Resination { // 아티스트가 퇴사를 했는지 안했는지 -> 퇴사일이 입력받은 연도와 같은지 아닌지 확인
				if endDate.Year() == intYear { // 퇴사일이 입력받은 연도와 같은 경우 -> 퇴사일 기준
					if artist.Changed { // 동일 연도 연봉이 변경된 경우 -> 연도 확인
						if endDate.Year() == changeDate.Year() { // 퇴사 연도와 동일 연도 연봉 변경 연도가 같은 경우 -> 변경일 적용
							sumSalary += allCFSalary + changeBeforeSalary

							if diffECMonth == 0 {
								sumSalary += littleECSalary
							} else { // 연봉 변경일과 퇴사일의 월이 다른 경우
								sumSalary += allECSalary + changeAfterSalary + endSalary
							}

							return sumSalary
						} else { // 퇴사 연도와 동일 연도 연봉 변경 연도가 다른 경우 -> 변경일 적용 X
							sumSalary += allEFSalary + endSalary

							return sumSalary
						}
					}
					// 동일 연도 연봉이 바뀌지 않은 경우
					sumSalary += allEFSalary + endSalary

					return sumSalary
				} else if endDate.Year() > intYear { // 퇴사일 연도가 입력받은 연도보다 큰 경우 -> 그해 연말 기준
					if artist.Changed { // 동일 연도 연봉이 변경된 경우 -> 연도 확인
						if intYear == changeDate.Year() { // 입력받은 연도와 변경 연도가 같은 경우 -> 변경일 적용
							sumSalary += allCFSalary + changeBeforeSalary + allLCSalary + changeAfterSalary

							return sumSalary
						} else { // 입력받은 연도와 변경 연도와 다른 경우 -> 변경일 적용 X
							sumSalary, _ = allMonthlySalaryFunc(artist.Salary[year], 12)

							return sumSalary
						}
					}
					// 동일 연도 연봉이 변경되지 않은 경우
					sumSalary, _ = allMonthlySalaryFunc(artist.Salary[year], 12)

					return sumSalary
				} else { // 퇴사일 연도가 입력받은 연도보다 작은 경우 -> 에러 처리
					return 0
				}
			}
			// 아티스트가 퇴사를 하지 않은 경우 -> 입력받은 연도의 말일 기준
			if artist.Changed { // 동일 연도 연봉이 변경된 경우 -> 변경된 연도 확인
				if intYear == changeDate.Year() { // 입력받은 연도와 동일 연도 연봉 변경 연도가 같은 경우
					sumSalary += allCFSalary + changeBeforeSalary + allLCSalary + changeAfterSalary

					return sumSalary
				} else { // 입력받은 연도와 동일 연도 연봉 변경 연도가 같은 경우 -> 그 해 연봉으로 연말까지 실지급액 계산
					sumSalary, _ = allMonthlySalaryFunc(artist.Salary[year], 12)

					return sumSalary
				}
			}
			// 동일 연도 연봉이 변경되지 않은 경우 -> 그 해 연봉으로 연말까지 실지급액 계산
			sumSalary, _ = allMonthlySalaryFunc(artist.Salary[year], 12)

			return sumSalary
		} else { // 입사연도가 입력받은 연도보다 큰 경우 -> 에러
			return 0
		}
	} else { // 입력받은 연도가 이번달의 연도보다 큰 경우 -> 에러
		return 0
	}
}

// hourlyWageByYearFunc 함수는 입력받은 연도와 아티스트 정보를 기준으로 아티스트의 시급을 계산하는 함수이다.
func hourlyWageByYearFunc(artist Artist, year string) string {
	realSalary := realSalaryByYearFunc(artist, year) // 연 실지급액
	workingDay := workingDayByYearFunc(artist, year) // 연 근무일수
	if workingDay == "0" {
		return "0"
	}

	floatWorkingDay, err := strconv.ParseFloat(workingDay, 64)
	if err != nil {
		return "0"
	}

	hourlyWage := math.Round((realSalary / floatWorkingDay) / 8)

	return humanize.Comma(int64(hourlyWage))
}
