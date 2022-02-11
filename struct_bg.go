// 프로젝트 결산 프로그램
//
// Description : 예산 관련 자료구조 스크립트

package main

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// BGTeamSetting 자료구조 - 기본적인 본부, 부서, 태스크, 팀에 대한 트리 구조
type BGTeamSetting struct {
	// default
	ID          string `json:"id" bson:"id"`                   // DB에서 값을 가지고 오기 위한 ID(setting.bgteam)
	UpdatedTime string `json:"updatedtime" bson:"updatedtime"` // 마지막으로 업데이트된 시간

	Headquarters []string               `json:"headquarters" bson:"headquarters"` // 본부 ex) [VFX, CM]
	Departments  map[string][]BGDept    `json:"departments" bson:"departments"`   // 본부별 부서 ex) VFX:[pre-production, Asset, 3D+FX, COMP, SUP+PROD], CM:[CM]
	Controls     map[string][]BGControl `json:"controls" bson:"controls"`         // 본부별 해당 Supervisor, Production, Management 자료구조
	Teams        map[string][]string    `json:"teams" bson:"teams"`               // 태스크별 해당 팀 ex) texture:Asset & Lookdev, model:Asset & Lookdev, MM:MatchMove, Ani:Animation, CM_Matte:cm_Matte
}

// BGPart 자료구조 - 본부별 부서세팅에 관련된 Part 자료구조
type BGPart struct {
	Name  string   `json:"name" bson:"name"`   // Previsual
	Tasks []string `json:"tasks" bson:"tasks"` // [previz]
}

// BGDept 자료구조 - 본부별 부서세팅에 관련된 Dept 자료구조
type BGDept struct {
	Name  string   `json:"name" bson:"name"`   // pre-production
	Parts []BGPart `json:"parts" bson:"parts"` // Previsual:[previz], CM_Concept:[CM_Concept], model:[texture, model, rigging]
	Type  bool     `json:"type" bson:"type"`   // true: 어셋, false: 샷
}

// BGControlPart 자료구조 - 본부에 해당하는 슈퍼바이저, 프로덕션, 매니지먼트에 관련된 Part 자료구조
type BGControlPart struct {
	Name  string   `json:"name" bson:"name"`   // Supervisor
	Teams []string `json:"teams" bson:"teams"` // [Executive Supervisor, supervisor]
}

// BGControl 자료구조 - 본부에 해당하는 슈퍼바이저, 프로덕션, 매니지먼트 관련 BGControl 자료구조
type BGControl struct {
	Name  string          `json:"name" bson:"name"`   // SUP+PROD
	Parts []BGControlPart `json:"parts" bson:"parts"` // SUP, PROD, MNG에 해당하는 팀 목록 Supervisor:[Executive Supervisor, Supervisor] -> 팀 목록은 AdminSetting 에서 가져온다.
}

// BGProject - 예산 프로젝트 자료구조
type BGProject struct {
	// 초기 프로젝트 추가에 필요한 요소
	ID        string // 프로젝트 영문 약자 ex) KIJ
	Name      string // 프로젝트 한글 이름
	StartDate string // 예상 작업 시작일
	EndDate   string // 예상 작업 마감일
	Type      string // 영화인지 드라마인지 타입 ex) 영화:movie, 드라마:drama

	// 프로젝트 부가 정보
	DirectorName string // 감독 이름
	ProducerName string // 제작사 이름

	// 예산 프로젝트 관련 정보
	Status   bool                  // true: 계약 완료, false: 사전 검토 or string "계약 완료", "사전 검토"
	TypeList []string              // 예산안 타입리스트
	MainType string                // 예산안 타입들 중에서 메인으로 결정된 타입
	TypeData map[string]BGTypeData // 예산안 데이터

	UpdatedTime string `json:"updatedtime" bson:"updatedtime"` // 마지막으로 업데이트된 시간
}

// BGTypeData - 예산안 자료구조
type BGTypeData struct {
	// 예산안 기본 정보
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"` // 예산안을 구분하기 위한 ID
	TeamSetting  BGTeamSetting      // 예산안 업데이트 당시의 팀세팅
	ContractDate string             // 계약일
	Proposal     string             // 제안 견적
	Decision     string             // 계약 결정액
	ContractCuts int                // 프로젝트 계약 컷수
	WorkingCuts  int                // 프로젝트 작업 컷수

	// 실예산안 계산 정보
	RetakeRatio   float64 // Retake율
	ProgressRatio float64 // 진행비율
	VendorRatio   float64 // 외주비율

	// 비용 정보
	LaborCosts  []BGLaborCost     // 예산안 비용
	EpisodeCost map[string]string // 에피소드별 비용 ex) EP01: 100000, EP02: 200000 ...

	// 슈퍼바이저, 프로덕션, 매니지먼트 리스트
	Supervisors []BGManagement // 슈퍼바이저
	Production  []BGManagement // 프로덕션
	Management  []BGManagement // 매니지먼트

	// 샷, 어셋 정보
	ShotList  []BGShotAsset // 샷 정보
	AssetList []BGShotAsset // 어셋 정보
}

// BGLaborCost 예산안 비용 자료구조
type BGLaborCost struct {
	Headquarter    string            // 본부명 ex) VFX, CM ...
	DepartmentCost map[string]string // 부서별 비용 ex) 3D+FX: 100000, COMP: 200000 ...
	Management     string            // 매니지먼트 비용
}

// BGManagement 자료구조
type BGManagement struct {
	UserID string  // 아티스트 샷건 ID
	Period int     // 기간
	Work   string  // 업무
	Ratio  float64 // 업무 퍼센티지
}

// BGShotAsset 자료구조
type BGShotAsset struct {
	// Asset, Shot 공통 자료
	Name   string             // 명칭 및 Shot name ex) Asset: 시뮬레이션, Shot: s0010_c0010
	Manday map[string]float64 // 태스크별 bid ex) {"MM":1, "Comp":0.5}
	Note   string             // 노트 정보 ex) Asset: 번개섬 야자수 해변, 번개섬 봉우리, Shot(drama): EP01, EP02 ...

	// Asset 자료
	Class string // 어셋 분류 ex) Asset, Concept
	Shot  int    // 어셋의 샷 개수
}
