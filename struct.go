// 프로젝트 결산 프로그램
//
// Description : 결산 관련 자료구조 스크립트

package main

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Timelog 자료구조
type Timelog struct {
	UserID   string  `json:"userid" bson:"userid"`     // 아티스트의 Shotgun ID
	Quarter  int     `json:"quarter" bson:"quarter"`   // 분기
	Year     int     `json:"year" bson:"year"`         // 연도
	Month    int     `json:"month" bson:"month"`       // 월
	Project  string  `json:"project" bson:"project"`   // 프로젝트
	Duration float64 `json:"duration" bson:"duration"` // 타임로그 시간
}

// FinishedTimelogStatus 자료구조는 정산 완료된 프로젝트에 타임로그를 작성했을 경우 ETC로 처리할지의 여부를 담는 자료구조이다.
type FinishedTimelogStatus struct {
	Year        int                `json:"year" bson:"year"`               // 연도
	Month       int                `json:"month" bson:"month"`             // 월
	Project     string             `json:"project" bson:"project"`         // 프로젝트
	Status      bool               `json:"status" bson:"status"`           // true: ETC로 처리, false: 프로젝트로 처리
	TimelogInfo map[string]float64 `json:"timeloginfo" bson:"timeloginfo"` // 아티스트와 아티스트가 작성한 타임로그의 duration 정보
}

// Artist 자료구조
type Artist struct {
	// 아티스트 기본 정보
	ID   string // ID(VFX : Shotgun ID, CM : 1부터 시작)
	Name string // 이름
	Dept string // 부서
	Team string // 팀

	// 아티스트 입사 및 퇴사 정보
	StartDay   string // 입사일 2020-01-01
	EndDay     string // 퇴사일 2020-09-01
	Resination bool   // 퇴사 여부

	// 아티스트 연봉 관련 정보
	Salary        map[string]string // 연봉 {"2019": 2000000, "2020": 2000000}
	Changed       bool              // 같은 해에 연봉이 바뀌었는지 체크
	ChangedSalary map[string]string // 같은 해에 연봉이 바뀐 경우 바뀐 날짜에 해당하는 연봉 {"2020-03-15":2400}
}

// Cost 자료구조는 프로젝트를 진행하면서 지출되는 비용의 자료구조이다.
type Cost struct {
	LaborCost    LaborCost // 내부 인건비 "{"VFX": 100000, "CM": 100000, "RND": 100000}"
	ProgressCost string    // 진행비 "100000"
	PurchaseCost string    // 구매비 "100000"
}

// PurchaseCost 자료구조는 프로젝트를 진행하면서 지출되는 구매비의 자료구조이다.
type PurchaseCost struct {
	CompanyName string // 업체 이름
	Detail      string // 내역
	Expenses    string // 금액
}

// LaborCost 자료구조는 프로젝트를 진행하면서 지출되는 인건비의 자료구조인다.
type LaborCost struct {
	VFX string // VFX 인건비
	CM  string // CM 인건비
	RND string // RND 인건비
}

// Project 자료구조
type Project struct {
	// 초기 프로젝트 추가에 필요한 요소
	ID           string    // 프로젝트 영문 약자
	Name         string    // 프로젝트 한글 이름
	Payment      []Payment // 총 매출(계약금), 추가 계약금
	StartDate    string    // 작업 시작일
	SMEndDate    string    // 결산시 작업 마감일
	DirectorName string    // 감독 이름
	ProducerName string    // 제작사 이름

	IsFinished   bool   // 정산 완료 여부(이미 정산 완료된 프로젝트를 추가할 때 true)
	TotalAmount  string // 정산 완료된 프로젝트의 총 내부 비용(이미 정산 완료된 프로젝트를 추가할 때 입력하는 내부 비용)
	FinishedCost Cost   // 정산 완료된 프로젝트 내부 비용 세부 정보

	// 결산에 필요한 요소
	SMStatus              map[string]string         // 상태 {"2020-06":"WIP", "2020-07":"HOLD"}
	SMMonthlyPayment      map[string][]Payment      // 결산시 월별 매출(수익)
	SMMonthlyProgressCost map[string]string         // 결산시 월별 진행비
	SMMonthlyLaborCost    map[string]LaborCost      // 결산시 월별 인건비
	SMMonthlyPurchaseCost map[string][]PurchaseCost // 결산시 월별 구매비
	SMDifference          string                    // 경영관리실에서 입력하는 차액(퇴직금, 감가상각비, 공통 노무비, 공통 경비 등)

	// 프로젝트 부가 정보
	ContractCuts int // 프로젝트 계약 컷수
	WorkingCuts  int // 프로젝트 작업 컷수
}

// Payment 자료구조는 프로젝트의 매출 정보를 담을 때 사용하는 자료구조이다.
type Payment struct {
	Type        string // 계약금, 중도금, 잔금
	Date        string // 날짜
	Expenses    string // 비용
	Status      bool   // 받았는지 여부
	DepositDate string // 입금일 날짜
}

// Vendor 자료구조는 외주 업체 정보를 담을 때 사용하는 자료구조이다.
type Vendor struct {
	// 기본 정보
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`        // 벤더를 구분하기 위한 ID
	Project     string             `json:"project" bson:"project"`         // 프로젝트 이름
	ProjectName string             `json:"projectname" bson:"projectname"` // 프로젝트 한글명
	Name        string             `json:"name" bson:"name"`               // 벤더 이름
	Expenses    string             `json:"expenses" bson:"expenses"`       // 총 외주비
	Date        string             `json:"date" bson:"date"`               // 벤더 계약일

	// 비용 정보
	Downpayment   VendorCost   `json:"downpayment" bson:"downpayment"`     // 계약금
	MediumPlating []VendorCost `json:"mediumplating" bson:"mediumplating"` // 중도금 - 1개가 아닐 수 있다.
	Balance       VendorCost   `json:"balance" bson:"balance"`             // 잔금

	// 부가 정보
	Cuts  int      `json:"cuts" bson:"cuts"`   // 컷 수
	Tasks []string `json:"tasks" bson:"tasks"` // 벤더 태스크
}

// VendorCost 자료구조는 외주 업체 비용 정보를 담을 때 사용하는 자료구조이다.
type VendorCost struct {
	Expenses  string // 비용
	Date      string // 세금 계산서 발행일 ex) 2020-12-01
	PayedDate string // 벤더 비용 지급일 ex) 2021-01-15
	Status    bool   // 정산 여부
}

// Status 자료구조
type Status struct {
	ID        string `json:"id"`        // ID, 상태코드
	TextColor string `json:"textcolor"` // TEXT 색상
	BGColor   string `json:"bgcolor"`   // BG 상태 색상
}

// AdminSetting 자료구조
type AdminSetting struct {
	// default
	ID string `json:"id" bson:"id"` // DB에서 값을 가지고 오기 위한 ID(setting.admin)

	// VFX
	VFXDepts []string            `json:"vfxdepts" bson:"vfxdepts"` // VFX 부서 리스트
	VFXTeams map[string][]string `json:"vfxteams" bson:"vfxteams"` // VFX 팀 리스트

	// CM
	CMTeams []string `json:"cmteams" bson:"cmteams"` // CM 팀 리스트

	// Shotgun
	SGUpdatedTime     string   `json:"sgupdatedtime" bson:"sgupdatedtime"`         // Shotgun에서 타임로그 데이터가 업데이트된 시간
	SGExcludeID       []string `json:"sgexcludeid" bson:"sgexcludeid"`             // Shotgun에서 타임로그를 가져올 때 제외할 아티스트 ID(ex. 90)
	SGExcludeProjects []string `json:"sgexcludeprojects" bson:"sgexcludeprojects"` // Shotgun에서 타임로그를 가져올 때 제외할 프로젝트 리스트(ex. td2)

	// Project
	ProjectStatus []Status `json:"projectstatus" bson:"projectstatus"` // 프로젝트 상태 리스트
	RNDProjects   []string `json:"rndprojects" bson:"rndprojects"`     // 타임로그를 업데이트할 때 RND 프로젝트로 처리힐 프로젝트 리스트(ex. RND, RND2020)
	ETCProjects   []string `json:"etcprojects" bson:"etcprojects"`     // 타임로그를 업데이트할 때 ETC 프로젝트로 처리할 프로젝트 리스트(ex. ETC)
	TaskProjects  []string `json:"taskprojects" bson:"taskprojects"`   // 태스크로 프로젝트를 구분하는 프로젝트 리스트

	// 결산(SettleMent)
	SMSupervisorIDs []string `json:"smsupervisorids" bson:"smsupervisorids"` // 프로젝트 관리 페이지에서 따로 타임로그를 작성할 수퍼바이저들의 ID 리스트
	GWIDs           []string `json:"gwids" bson:"gwids"`                     // 벤더 발행일에 메일을 전송할 그룹웨어 ID
	GWIDsForProject []string `json:"gwidsforproject" bson:"gwidsforproject"` // 프로젝트 발행일에 메일을 전송할 그룹웨어 ID

	// 예산(Budget)
	BGSupervisorTeams []string `json:"bgsupervisorteams" bson:"bgsupervisorteams"` // 예산안 및 예산 관련 팀 세팅에서 사용될 슈퍼바이저 Team 리스트
	BGProductionTeams []string `json:"bgproductionteams" bson:"bgproductionteams"` // 예산안 및 예산 관련 팀 세팅에서 사용될 프로덕션 Team 리스트
	BGManagementTeams []string `json:"bgmanagementteams" bson:"bgmanagementteams"` // 예산안 및 예산 관련 팀 세팅에서 사용될 매니지먼트 Team 리스트
}

// MonthlyStatus 자료구조
type MonthlyStatus struct {
	Date   string `json:"date" bson:"date"`     // 2020-09
	Status bool   `json:"status" bson:"status"` // 결산이 완료되었으면 true, 결산이 완료되지 않았다면 false
}

// Log 자료구조
type Log struct {
	UserID    string    `json:"userid" bson:"userid"`         // 유저 ID
	CreatedAt time.Time `json:"created_at" bson:"created_at"` // 로그가 생성된 시간
	Content   string    `json:"content" bson:"content"`       // 로그 내용
}

// CheckErrorFunc 메소드는 Artist 자료구조에 값이 정확히 들어갔는지 확인하는 함수이다.
func (a Artist) CheckErrorFunc() error {
	if a.ID == "" {
		return errors.New("ID를 입력해주세요")
	}
	if a.Name == "" {
		return errors.New("이름을 입력해주세요")
	}
	if !regexName.MatchString(a.Name) {
		return errors.New("이름에는 한글, 영문만 사용가능합니다")
	}
	if a.Dept == "" {
		return errors.New("Shotgun에서 팀 태그를 확인하거나 부서를 입력해주세요")
	}
	if a.Team == "" {
		return errors.New("팀을 입력해주세요")
	}
	if a.StartDay != "" {
		if !regexDate2.MatchString(a.StartDay) {
			return errors.New("입사일이 2020-09-01 형식이 아닙니다")
		}
	}
	if a.EndDay != "" {
		if !regexDate2.MatchString(a.EndDay) {
			return errors.New("퇴사일이 2020-09-01 형식이 아닙니다")
		}
	}
	return nil
}

// CheckErrorFunc 메소드는 Timelog 자료구조에 값이 정확히 들어갔는지 확인하는 함수이다.
func (t Timelog) CheckErrorFunc() error {
	if t.UserID == "" {
		return errors.New("ID를 입력해주세요")
	}
	if t.Year == 0 {
		return errors.New("연도를 입력해주세요")
	}
	if t.Month == 0 {
		return errors.New("월을 입력해주세요")
	}
	if t.Project == "" {
		return errors.New("프로젝트를 입력해주세요")
	}
	if !regexWord.MatchString(t.Project) {
		return errors.New("프로젝트명은 영문, 숫자만 사용가능합니다")
	}
	if t.Duration == -1 {
		return errors.New("타임로그 시간을 입력해주세요")
	}
	return nil
}

// CheckErrorFunc 메소드는 Project 자료구조에 값이 정확히 들어갔는지 확인하는 함수이다.
func (p Project) CheckErrorFunc() error {
	if p.ID == "" {
		return errors.New("ID를 입력해주세요")
	}
	if !regexProject.MatchString(p.ID) {
		return errors.New("프로젝트 ID에는 영문(대문자), 숫자, 특수문자(_)만 입력 가능합니다")
	}
	if p.Name == "" {
		return errors.New("이름을 입력해주세요")
	}
	if !regexDate.MatchString(p.StartDate) {
		return errors.New("작업 시작일이 2020-07 형식이 아닙니다")
	}
	if !regexDate.MatchString(p.SMEndDate) {
		return errors.New("작업 마감일이 2020-07 형식이 아닙니다")
	}
	return nil
}

// CheckErrorFunc 메소드는 Vendor 지료구조에 값이 정확히 들어갔는지 확인하는 함수이다.
func (v Vendor) CheckErrorFunc() error {
	if v.Project == "" {
		return errors.New("프로젝트 ID를 입력해주세요")
	}
	if !regexProject.MatchString(v.Project) {
		return errors.New("프로젝트 ID에는 영문(대문자), 숫자, 특수문자(_)만 입력 가능합니다")
	}
	if v.Name == "" {
		return errors.New("Vendor 이름을 입력해주세요")
	}
	if v.Downpayment.Date != "" {
		if !regexDate2.MatchString(v.Downpayment.Date) {
			return errors.New("계약금 지출 날짜가 2020-12-15 형식이 아닙니다")
		}
	}
	for _, mp := range v.MediumPlating {
		if mp.Date != "" {
			if !regexDate2.MatchString(mp.Date) {
				return errors.New("잔금 지출 날짜가 2020-12-15 형식이 아닙니다")
			}
		}
	}
	if v.Balance.Date != "" {
		if !regexDate2.MatchString(v.Balance.Date) {
			return errors.New("잔금 지출 날짜가 2020-12-15 형식이 아닙니다")
		}
	}
	return nil
}

// CheckErrorFunc 메소드는 AdminSetting 자료구조에 값이 정확히 들어갔는지 확인하는 함수이다.
func (a AdminSetting) CheckErrorFunc() error {
	for _, id := range a.SGExcludeID {
		if !regexDigit.MatchString(id) {
			return errors.New("제외할 아티스트의 ID는 숫자만 가능합니다")
		}
	}
	for _, ep := range a.SGExcludeProjects {
		if !regexProject.MatchString(ep) {
			return errors.New("제외할 프로젝트에는 영문(대문자), 숫자, 특수문자(_)만 입력 가능합니다")
		}
	}
	for _, s := range a.ProjectStatus {
		if !regexWord.MatchString(s.ID) {
			return errors.New("status ID에는 영문, 숫자만 입력 가능합니다")
		}
	}
	for _, rp := range a.RNDProjects {
		if !regexProject.MatchString(rp) {
			return errors.New("기타 프로젝트에는 영문(대문자), 숫자, 특수문자(_)만 입력 가능합니다")
		}
	}
	for _, tp := range a.TaskProjects {
		if !regexProject.MatchString(tp) {
			return errors.New("태스크로 구분할 프로젝트에는 영문(대문자), 숫자, 특수문자(_)만 입력 가능합니다")
		}
	}
	for _, id := range a.SMSupervisorIDs {
		if !regexDigit.MatchString(id) {
			return errors.New("수퍼바이저의 ID는 숫자만 가능합니다")
		}
	}
	return nil
}

// CheckErrorFunc 메소드는 MonthlyStatus 자료구조에 값이 정확히 들어갔는지 확인하는 함수이다.
func (ms MonthlyStatus) CheckErrorFunc() error {
	if !regexDate.MatchString(ms.Date) {
		return errors.New("2020-07 형식이 아닙니다")
	}
	return nil
}

// CheckErrorFunc 메소드는 Status 자료구조의 에러를 체크한다.
func (s Status) CheckErrorFunc() error {
	if s.ID == "" {
		return errors.New("ID가 빈 문자열 입니다")
	}
	if !regexWebColor.MatchString(s.TextColor) {
		return errors.New("웹컬러 문자열이 아닙니다")
	}
	if !regexWebColor.MatchString(s.BGColor) {
		return errors.New("웹컬러 문자열이 아닙니다")
	}
	return nil
}
