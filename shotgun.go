// 프로젝트 결산 프로그램
//
// Description : shotgun 관련 스크립트

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// accessTokensFunc 함수는 shotgun의 rest API를 이용하여 acess token을 가져오는 함수이다.
func accessTokensFunc() string {
	headers := map[string][]string{
		"Content-Type": []string{"application/x-www-form-urlencoded"},
		"Accept":       []string{"application/json"},
	}

	jsonReq := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {"authentication_script"},
		"client_secret": {"vZnk#jbmopl6oqispvodazfyn"},
	}
	data := bytes.NewBufferString(jsonReq.Encode())

	req, err := http.NewRequest("POST", "https://road101.shotgunstudio.com/api/v1/auth/access_token", data) // request 구조체 생성
	if err != nil {
		log.Println(err)
	}
	req.Header = headers // 헤더값 설정

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
	}

	token, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}
	resp.Body.Close()

	result := make(map[string]interface{})
	err = json.Unmarshal(token, &result)
	if err != nil {
		log.Println(err)
		return ""
	}

	// token을 string형으로 변환
	tokenType := fmt.Sprintf("%v", result["token_type"])
	accessToken := fmt.Sprintf("%v", result["access_token"])
	strToken := strings.Join([]string{tokenType, accessToken}, " ")

	return strToken
}

// sgGetArtistFunc 함수는 Shotgun에서 입력받은 id와 일치하는 아티스트를 찾아 반환하는 함수이다.
func sgGetArtistFunc(id string) (Artist, error) {
	token := accessTokensFunc()

	headers := map[string][]string{
		"Accept":        []string{"application/json"},
		"Authorization": []string{token},
	}

	// 원래 department.Department.tags.Tag.name 이거로 잘 가져왔는데 department.Department.tags로 해야 가져와진다.
	jsonReq := fmt.Sprintf(`
	{
		"filters": [
			["id", "is", %s]
		],
		"fields": ["name", "department.Department.tags", "department.Department.name"]
	}
	`, id)

	data := bytes.NewBuffer([]byte(jsonReq))
	req, err := http.NewRequest("POST", "https://road101.shotgunstudio.com/api/v1/entity/human_users/_search", data)
	if err != nil {
		return Artist{}, err
	}
	req.Header = headers

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return Artist{}, err
	}
	defer resp.Body.Close()

	bytes, _ := ioutil.ReadAll(resp.Body)

	type Tag struct {
		Name string `json:"name" bson:"name"`
	}

	type Attribute struct {
		Name string `json:"name" bson:"name"`
		Dept []Tag  `json:"department.Department.tags" bson:"department.Department.tags"`
		Team string `json:"department.Department.name" bson:"department.Department.name"`
	}

	type ArtistJson struct {
		Type       string    `json:"type" bson:"type"`
		Attributes Attribute `json:"attributes" bson:"attributes"`
	}

	type Recipe struct {
		Data []ArtistJson `json:"data" bson:"data"`
	}

	var rcp Recipe
	json.Unmarshal(bytes, &rcp)

	var result Artist

	if len(rcp.Data) == 0 {
		return Artist{}, errors.New(fmt.Sprintf("Shotgun에 %s ID를 가진 아티스트를 찾을 수 없습니다", id))
	}

	r := rcp.Data[0]
	result.ID = id
	result.Name = r.Attributes.Name
	if len(r.Attributes.Dept) != 0 { // 팀 태그가 설정이 안 되어 있을 수도 있다.
		result.Dept = r.Attributes.Dept[0].Name
	}
	result.Team = r.Attributes.Team

	return result, nil
}

// sgGetProjectsFunc 함수는 Shotgun에서 진행중인 프로젝트 목록리스트를 반환하는 함수이다.
func sgGetProjectsFunc(excludeProjects []string) ([]string, error) {
	token := accessTokensFunc()

	headers := map[string][]string{
		"Content-Type":  []string{"application/vnd+shotgun.api3_array+json"},
		"Accept":        []string{"application/json"},
		"Authorization": []string{token},
	}

	// 제외할 프로젝트 설정
	ep := make([]string, len(excludeProjects))
	copy(ep, excludeProjects)
	if len(ep) != 0 {
		for i, p := range ep {
			ep[i] = fmt.Sprintf(`"%s"`, p)
		}
	} else {
		ep = append(ep, `""`)
	}

	var jsonReq string
	jsonReq = fmt.Sprintf(`
	{
		"filters": [
			["archived", "is", false],
			["is_demo", "is", false],
			["is_template", "is", false],
			["name", "not_in", [%s]]
		],
		"fields": ["name"],
		"sort": "id"
	}
	`, strings.Join(ep, ","))

	data := bytes.NewBuffer([]byte(jsonReq))
	req, err := http.NewRequest("POST", "https://road101.shotgunstudio.com/api/v1/entity/project/_search", data)
	if err != nil {
		return []string{}, err
	}
	req.Header = headers

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return []string{}, err
	}
	defer resp.Body.Close()

	bytes, _ := ioutil.ReadAll(resp.Body)

	type Attribute struct {
		Name string `json:"name" bson:"name"`
	}

	type ProjectJson struct {
		Type       string    `json:"type" bson:"type"`
		Attributes Attribute `json:"attributes" bson:"attributes"`
	}

	type Recipe struct {
		Data []ProjectJson `json:"data" bson:"data"`
	}

	var rcp Recipe
	json.Unmarshal(bytes, &rcp)

	var result []string
	for _, r := range rcp.Data {
		result = append(result, r.Attributes.Name)
	}

	sort.Sort(sort.StringSlice(result)) // 오름차순으로 정렬
	return result, nil
}

// sgGetTimelogsFunc 함수는 입력받은 상태를 기준으로 타임로그를 반환하는 함수이다.
func sgGetTimelogsFunc(timelogID string, excludeID []string, excludeProjects []string, taskProjects []string, checkStatus bool) ([]Timelog, string, error) {
	token := accessTokensFunc()

	headers := map[string][]string{
		"Content-Type":  []string{"application/vnd+shotgun.api3_array+json"},
		"Accept":        []string{"application/json"},
		"Authorization": []string{token},
	}

	// 검색할 날짜 설정
	var firstDate string
	lasttimelogID := timelogID
	nowTime := time.Now()                   // 현재 날짜 및 시간
	nowDate := nowTime.Format("2006-01-02") // 현재 날짜만 나오도록 포맷 설정
	if checkStatus {
		firstDate = time.Date(nowTime.Year(), nowTime.Month(), 1, 0, 0, 0, 0, time.Local).Format("2006-01-02") // 그 달의 첫번째 날 설정
	} else {
		ld := nowTime.AddDate(0, -1, 0)
		firstDate = time.Date(ld.Year(), ld.Month(), 1, 0, 0, 0, 0, time.Local).Format("2006-01-02") // 지난 달의 첫번째 날 설정
	}

	// 제외할 아티스트의 ID 설정
	ei := make([]string, len(excludeID))
	copy(ei, excludeID)
	if len(ei) != 0 {
		for i, id := range ei {
			ei[i] = fmt.Sprintf("%s", id)
		}
	} else {
		ei = append(ei, "0")
	}

	// 제외할 프로젝트 설정
	ep := make([]string, len(excludeProjects))
	copy(ep, excludeProjects)
	if len(ep) != 0 {
		for i, p := range ep {
			ep[i] = fmt.Sprintf(`"%s"`, p)
		}
	} else {
		ep = append(ep, `""`)
	}

	var jsonReq string
	jsonReq = fmt.Sprintf(`
	{
		"filters": [
			["date", "between", ["%s", "%s"]],
			["project.Project.name", "not_in", [%s]],
			["user.HumanUser.id", "not_in", [%s]],
			["id", "greater_than", %s]
		],
		"fields": ["date", "duration", "user.HumanUser.id", "project.Project.name", "created_at", "entity.Task.content"],
		"sort": "id",
		"options": {            
			"include_archived_projects": true        
		}
	}
	`, firstDate, nowDate, strings.Join(ep, ","), strings.Join(ei, ","), lasttimelogID)

	data := bytes.NewBuffer([]byte(jsonReq))
	req, err := http.NewRequest("POST", "https://road101.shotgunstudio.com/api/v1/entity/time_log/_search", data)
	if err != nil {
		return []Timelog{}, lasttimelogID, err
	}
	req.Header = headers

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return []Timelog{}, lasttimelogID, err
	}
	defer resp.Body.Close()

	bytes, _ := ioutil.ReadAll(resp.Body)

	type Attribute struct {
		Date        string  `json:"date" bson:"date"`
		Duration    float64 `json:"duration" bson:"duration"`
		UserID      int     `json:"user.HumanUser.id" bson:"user.HumanUser.id"`
		ProjectName string  `json:"project.Project.name" bson:"project.Project.name"`
		CreatedAt   string  `json:"created_at" bson:"created_at"`
		TaskName    string  `json:"entity.Task.content" bson:"entity.Task.content"`
	}

	type TimelogJson struct {
		Type       string    `json:"type" bson:"type"`
		Attributes Attribute `json:"attributes" bson:"attributes"`
		ID         int       `json:"id" bson:"id"`
	}

	type Recipe struct {
		Data []TimelogJson `json:"data" bson:"data"`
	}

	var rcp Recipe
	json.Unmarshal(bytes, &rcp)

	var result []Timelog

	for _, r := range rcp.Data {
		year, err := strconv.Atoi(strings.Split(r.Attributes.Date, "-")[0])
		if err != nil {
			return []Timelog{}, lasttimelogID, err
		}
		month, err := strconv.Atoi(strings.Split(r.Attributes.Date, "-")[1])
		if err != nil {
			return []Timelog{}, lasttimelogID, err
		}

		t := Timelog{}
		t.UserID = strconv.Itoa(r.Attributes.UserID)
		t.Year = year
		t.Month = month

		// 태스크로 프로젝트를 구분하는 프로젝트들 처리
		include := checkStringInListFunc(r.Attributes.ProjectName, taskProjects)
		if include {
			t.Project = strings.ToUpper(r.Attributes.TaskName)
		} else {
			t.Project = r.Attributes.ProjectName
		}
		t.Duration = r.Attributes.Duration
		lasttimelogID = strconv.Itoa(r.ID)
		result = append(result, t)
	}
	return result, lasttimelogID, nil
}

// sgGetTeamsFunc 함수는 입력받은 팀 태그에 해당하는 팀들을 반환하는 함수이다.
func sgGetTeamsFunc(teamTagList []string) ([]string, error) {
	token := accessTokensFunc()

	headers := map[string][]string{
		"Content-Type":  []string{"application/vnd+shotgun.api3_array+json"},
		"Accept":        []string{"application/json"},
		"Authorization": []string{token},
	}

	jsonReq := `
	{
		"filters": [
			["sg_status_list", "is", "act"]
		],
		"fields": ["name", "tags"]
	}
	`

	data := bytes.NewBuffer([]byte(jsonReq))
	req, err := http.NewRequest("POST", "https://road101.shotgunstudio.com/api/v1/entity/department/_search", data)
	if err != nil {
		return []string{}, err
	}
	req.Header = headers

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return []string{}, err
	}
	defer resp.Body.Close()

	bytes, _ := ioutil.ReadAll(resp.Body)

	type Tag struct {
		ID   int64  `json:"id" bson:"id"`
		Name string `json:"name" bson:"name"`
		Type string `json:"type" bson:"type"`
	}

	type TagData struct {
		Data []Tag `json:"data" bson:"data"`
	}

	type Relationship struct {
		Tags TagData `json:"tags" bson:"tags"`
	}

	type Attribute struct {
		Name string `json:"name" bson:"name"`
	}

	type TagJson struct {
		Type          string       `json:"type" bson:"type"`
		Attributes    Attribute    `json:"attributes" bson:"attributes"`
		Relationships Relationship `json:"relationships" bson:"relationships"`
	}

	type Recipe struct {
		Data []TagJson `json:"data" bson:"data"`
	}

	var rcp Recipe
	json.Unmarshal(bytes, &rcp)

	var result []string
	for _, r := range rcp.Data {
		for _, tag := range r.Relationships.Tags.Data {
			// 전달받은 팀 태그 리스트에 포함되어 있는지 확인
			existed := checkStringInListFunc(tag.Name, teamTagList)
			if !existed {
				continue
			}

			// result 변수에 없으면 추가
			existed = checkStringInListFunc(r.Attributes.Name, result)
			if !existed {
				result = append(result, r.Attributes.Name)
			}
		}
	}

	sort.Sort(sort.StringSlice(result)) // 오름차순으로 정렬
	return result, err
}

// sgResetTimelogsFunc 함수는 모든 타임로그를 반환하는 함수이다.
func sgResetTimelogsFunc(timelogID string, excludeID []string, excludeProjects []string, taskProjects []string) ([]Timelog, string, error) {
	token := accessTokensFunc()

	headers := map[string][]string{
		"Content-Type":  []string{"application/vnd+shotgun.api3_array+json"},
		"Accept":        []string{"application/json"},
		"Authorization": []string{token},
	}

	lasttimelogID := timelogID

	// 제외할 아티스트의 ID 설정
	ei := make([]string, len(excludeID))
	copy(ei, excludeID)
	if len(ei) != 0 {
		for i, id := range ei {
			ei[i] = fmt.Sprintf("%s", id)
		}
	} else {
		ei = append(ei, "0")
	}

	// 제외할 프로젝트 설정
	ep := make([]string, len(excludeProjects))
	copy(ep, excludeProjects)
	if len(ep) != 0 {
		for i, p := range ep {
			ep[i] = fmt.Sprintf(`"%s"`, p)
		}
	} else {
		ep = append(ep, `""`)
	}

	var jsonReq string
	jsonReq = fmt.Sprintf(`
	{
		"filters": [
			["id", "greater_than", "%s"],
			["project.Project.name", "not_in", [%s]],
			["user.HumanUser.id", "not_in", [%s]]
		],
		"fields": ["date", "duration", "user.HumanUser.id", "project.Project.name", "created_at", "entity.Task.content"],
		"sort": "id",
		"options": {            
			"include_archived_projects": true        
		}
	}
	`, lasttimelogID, strings.Join(ep, ","), strings.Join(ei, ","))

	data := bytes.NewBuffer([]byte(jsonReq))
	req, err := http.NewRequest("POST", "https://road101.shotgunstudio.com/api/v1/entity/time_log/_search", data)
	if err != nil {
		return []Timelog{}, lasttimelogID, err
	}
	req.Header = headers

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return []Timelog{}, lasttimelogID, err
	}
	defer resp.Body.Close()

	bytes, _ := ioutil.ReadAll(resp.Body)

	type Attribute struct {
		Date        string  `json:"date" bson:"date"`
		Duration    float64 `json:"duration" bson:"duration"`
		UserID      int     `json:"user.HumanUser.id" bson:"user.HumanUser.id"`
		ProjectName string  `json:"project.Project.name" bson:"project.Project.name"`
		CreatedAt   string  `json:"created_at" bson:"created_at"`
		TaskName    string  `json:"entity.Task.content" bson:"entity.Task.content"`
	}

	type TimelogJson struct {
		Type       string    `json:"type" bson:"type"`
		Attributes Attribute `json:"attributes" bson:"attributes"`
		ID         int       `json:"id" bson:"id"`
	}

	type Recipe struct {
		Data []TimelogJson `json:"data" bson:"data"`
	}

	var rcp Recipe
	json.Unmarshal(bytes, &rcp)

	var result []Timelog

	for _, r := range rcp.Data {
		year, err := strconv.Atoi(strings.Split(r.Attributes.Date, "-")[0])
		if err != nil {
			return []Timelog{}, lasttimelogID, err
		}
		month, err := strconv.Atoi(strings.Split(r.Attributes.Date, "-")[1])
		if err != nil {
			return []Timelog{}, lasttimelogID, err
		}

		t := Timelog{}
		t.UserID = strconv.Itoa(r.Attributes.UserID)
		t.Year = year
		t.Month = month
		// 태스크로 프로젝트를 구분하는 프로젝트들 처리
		include := checkStringInListFunc(r.Attributes.ProjectName, taskProjects)
		if include {
			t.Project = strings.ToUpper(r.Attributes.TaskName)
		} else {
			t.Project = r.Attributes.ProjectName
		}
		t.Duration = r.Attributes.Duration
		lasttimelogID = strconv.Itoa(r.ID)
		result = append(result, t)
	}
	return result, lasttimelogID, nil
}

// sgGetTeamMapFunc 함수는 팀태그 리스트를 통해서 맵형식의 팀배열을 얻는 함수이다.
func sgGetTeamMapFunc(teamTagList []string) (map[string][]string, error) {
	token := accessTokensFunc()

	headers := map[string][]string{
		"Content-Type":  []string{"application/vnd+shotgun.api3_array+json"},
		"Accept":        []string{"application/json"},
		"Authorization": []string{token},
	}

	jsonReq := `
	{
		"filters": [
			["sg_status_list", "is", "act"]
		],
		"fields": ["name", "tags"]
	}
	`

	data := bytes.NewBuffer([]byte(jsonReq))
	req, err := http.NewRequest("POST", "https://road101.shotgunstudio.com/api/v1/entity/department/_search", data)
	if err != nil {
		return map[string][]string{}, err
	}
	req.Header = headers

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return map[string][]string{}, err
	}
	defer resp.Body.Close()

	bytes, _ := ioutil.ReadAll(resp.Body)

	type Tag struct {
		ID   int64  `json:"id" bson:"id"`
		Name string `json:"name" bson:"name"`
		Type string `json:"type" bson:"type"`
	}

	type TagData struct {
		Data []Tag `json:"data" bson:"data"`
	}

	type Relationship struct {
		Tags TagData `json:"tags" bson:"tags"`
	}

	type Attribute struct {
		Name string `json:"name" bson:"name"`
	}

	type TagJson struct {
		Type          string       `json:"type" bson:"type"`
		Attributes    Attribute    `json:"attributes" bson:"attributes"`
		Relationships Relationship `json:"relationships" bson:"relationships"`
	}

	type Recipe struct {
		Data []TagJson `json:"data" bson:"data"`
	}

	var rcp Recipe
	json.Unmarshal(bytes, &rcp)

	var result map[string][]string
	result = make(map[string][]string)
	for _, r := range rcp.Data {
		for _, tag := range r.Relationships.Tags.Data {
			// 전달받은 팀 태그 리스트에 포함되어 있는지 확인
			existed := checkStringInListFunc(tag.Name, teamTagList)
			if !existed {
				continue
			}

			// result 변수에 없으면 추가
			mapval, mapexisted := result[tag.Name]
			if !mapexisted {
				result[tag.Name] = append(mapval, r.Attributes.Name)
			} else {
				existed = checkStringInListFunc(r.Attributes.Name, mapval)
				if !existed {
					result[tag.Name] = append(mapval, r.Attributes.Name)
				}
			}
		}
	}

	for _, value := range result {
		sort.Sort(sort.StringSlice(value))
	}
	return result, err
}

// sgGetAllProjectsFunc 함수는 Shotgun에 등록된 모든 프로젝트를 반환하는 함수이다.
func sgGetAllProjectsFunc(excludeProjects []string) ([]Project, error) {
	token := accessTokensFunc()

	headers := map[string][]string{
		"Content-Type":  []string{"application/vnd+shotgun.api3_array+json"},
		"Accept":        []string{"application/json"},
		"Authorization": []string{token},
	}

	// 제외할 프로젝트 설정
	ep := make([]string, len(excludeProjects))
	copy(ep, excludeProjects)
	if len(ep) != 0 {
		for i, p := range ep {
			ep[i] = fmt.Sprintf(`"%s"`, p)
		}
	} else {
		ep = append(ep, `""`)
	}

	var jsonReq string
	jsonReq = fmt.Sprintf(`
	{
		"filters": [
			["name", "not_in", [%s]]
		],
		"fields": ["name", "sg_description", "start_date", "end_date", "created_at", "tags"],
		"options": {
			"include_archived_projects": true
		}
	}
	`, strings.Join(ep, ","))

	data := bytes.NewBuffer([]byte(jsonReq))
	req, err := http.NewRequest("POST", "https://road101.shotgunstudio.com/api/v1/entity/Project/_search", data)
	if err != nil {
		return []Project{}, err
	}
	req.Header = headers

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return []Project{}, err
	}
	defer resp.Body.Close()

	bytes, _ := ioutil.ReadAll(resp.Body)

	// 프로젝트 태그용 구조
	type Tag struct {
		ID   int64  `json:"id" bson:"id"`
		Name string `json:"name" bson:"name"`
		Type string `json:"type" bson:"type"`
	}

	type TagData struct {
		Data []Tag `json:"data" bson:"data"`
	}

	type Relationship struct {
		Tags TagData `json:"tags" bson:"tags"`
	}

	// 프로젝트 정보
	type Attribute struct {
		Name        string `json:"name" bson:"name"`
		Description string `json:"sg_description" bson:"sg_description"`
		StartDate   string `json:"start_date" bson:"start_date"`
		EndDate     string `json:"end_date" bson:"end_date"`
		CreatedAt   string `json:"created_at" bson:"created_at"`
	}

	type ProjectJSON struct {
		Type          string       `json:"type" bson:"type"`
		Attributes    Attribute    `json:"attributes" bson:"attributes"`
		Relationships Relationship `json:"relationships" bson:"relationshops"`
	}

	type Recipe struct {
		Data []ProjectJSON `json:"data" bson:"data"`
	}

	var rcp Recipe
	json.Unmarshal(bytes, &rcp)

	var result []Project
	for _, r := range rcp.Data {
		p := Project{}

		// PROJECT 태그인지 확인한다.
		if len(r.Relationships.Tags.Data) == 0 {
			continue
		} else if !strings.Contains(r.Relationships.Tags.Data[0].Name, "PROJECT") {
			continue
		}
		p.ID = strings.TrimSpace(strings.ToUpper(r.Attributes.Name)) // 프로젝트 ID
		// 프로젝트 이름
		if r.Attributes.Description == "" {
			p.Name = strings.TrimSpace(r.Attributes.Name) // Description이 없는 경우 ID와 동일하게 입력
		} else {
			p.Name = strings.TrimSpace(r.Attributes.Description)
		}
		// 프로젝트 작업 시작일
		if r.Attributes.StartDate == "" {
			create := strings.Split(r.Attributes.CreatedAt, "-")
			startDate := create[0] + "-" + create[1]
			p.StartDate = startDate
		} else {
			start := strings.Split(r.Attributes.StartDate, "-")
			startDate := start[0] + "-" + start[1]
			p.StartDate = startDate
		}
		//프로젝트 작업 마감일
		if r.Attributes.EndDate == "" {
			p.SMEndDate = p.StartDate
		} else {
			end := strings.Split(r.Attributes.EndDate, "-")
			endDate := end[0] + "-" + end[1]
			p.SMEndDate = endDate
		}
		// 프로젝트 총 매출
		var payment Payment
		payment.Expenses, err = encryptAES256Func("0")
		if err != nil {
			return []Project{}, err
		}
		p.Payment = append(p.Payment, payment)

		err = p.CheckErrorFunc()
		if err != nil {
			return []Project{}, err
		}

		result = append(result, p)
	}

	return result, nil
}
