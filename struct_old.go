// 프로젝트 결산 프로그램
//
// Description : 예전 자료구조 스크립트

package main

// OldProject 자료구조
type OldProject struct {
	// 초기 프로젝트 추가에 필요한 요소
	ID           string // 프로젝트 영문 약자
	Name         string // 프로젝트 한글 이름
	Payment      string // 총 매출(계약금)
	StartDate    string // 작업 시작일
	BGEndDate    string // 예산시 작업 마감일, 초기에 결산 작업 마감일과 동일하게 설정
	SMEndDate    string // 결산시 작업 마감일
	DirectorName string // 감독 이름
	ProducerName string // 제작사 이름

	IsFinished   bool   // 정산 완료 여부(이미 정산 완료된 프로젝트를 추가할 때 true)
	TotalAmount  string // 정산 완료된 프로젝트의 총 내부 비용(이미 정산 완료된 프로젝트를 추가할 때 입력하는 내부 비용)
	FinishedCost Cost   // 정산 완료된 프로젝트 내부 비용 세부 정보

	// 결산에 필요한 요소
	SMStatus              map[string]string         // 상태 {"2020-06":"WIP", "2020-07":"HOLD"}
	SMMonthlyPayment      map[string]string         // 결산시 월별 매출(수익)
	SMMonthlyProgressCost map[string]string         // 결산시 월별 진행비
	SMMonthlyLaborCost    map[string]LaborCost      // 결산시 월별 인건비
	SMMonthlyPurchaseCost map[string][]PurchaseCost // 결산시 월별 구매비
	SMDifference          string                    // 경영관리실에서 입력하는 차액(퇴직금, 감가상각비 등)

	// // 예산에 필요한 요소
	// BGStatus bool // 예산이 끝난지 아닌지 판단
	// BGInfo Budget //  예산 정보
	// BGVendor string
}
