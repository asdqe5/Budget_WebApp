// 프로젝트 결산 프로그램
//
// Description : template에서 사용하는 예산 관련 스크립트

package main

import (
	"math"
	"strconv"
)

// getBGPartInfoMapFunc 함수는 BGTeamSetting에서 본부의 Departments와 Controls 각각의 파트 개수와 합한 개수를 반환하는 함수이다.
func getBGPartInfoMapFunc(teamSetting BGTeamSetting, head string) map[string]int {
	result := make(map[string]int)
	result["dept"] = 0
	result["control"] = 0
	result["total"] = 0

	for _, dept := range teamSetting.Departments[head] {
		result["dept"] += len(dept.Parts)
		result["total"] += len(dept.Parts)
	}
	for _, control := range teamSetting.Controls[head] {
		result["control"] += len(control.Parts)
		result["total"] += len(control.Parts)
	}
	return result
}

// calNegoRatioFunc 함수는 예산안 정보에서 제안 견적과 계약 결정액을 통해 네고율을 계산하는 함수이다.
func calNegoRatioFunc(typedata BGTypeData) string {
	if typedata.Proposal == "" {
		return ""
	}
	if typedata.Decision == "" {
		return ""
	}
	proposal, err := decryptAES256Func(typedata.Proposal)
	if err != nil {
		return ""
	}
	proposalFloat, err := strconv.ParseFloat(proposal, 64)
	if err != nil {
		return ""
	}
	decision, err := decryptAES256Func(typedata.Decision)
	if err != nil {
		return ""
	}
	decisionFloat, err := strconv.ParseFloat(decision, 64)
	if err != nil {
		return ""
	}
	negoRatio := math.Round((proposalFloat - decisionFloat) / proposalFloat * 100)

	return strconv.FormatFloat(negoRatio, 'f', -1, 64)
}
