// 프로젝트 결산 프로그램
//
// Description : http VFX 타임로그 관련 스크립트

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize/v2"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// handleTimelogVFXFunc 함수는 VFX 타임로그 페이지를 불러오는 함수이다.
func handleTimelogVFXFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// default 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < DefaultLevel {
		http.Redirect(w, r, "invalidaccess", http.StatusSeeOther)
		return
	}

	// mongoDB client 연결
	credential := options.Credential{
		Username: *flagDBID,
		Password: *flagDBPW,
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(*flagMongoDBURI).SetAuth(credential))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	adminSetting, err := getAdminSettingFunc(client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type ArtistData struct { // 아티스트 정보 자료구조
		Name          string             // 아티스트 이름
		Timelogs      map[string]float64 // 타임로그 정보
		TotalDuration float64            // 토탈 타임로그 정보
	}

	type Recipe struct {
		Token         Token
		User          User
		UpdatedTime   string
		MonthlyStatus string
		NowDate       string
		Date          string   // yyyy-MM
		Depts         []string // VFX팀 부서 리스트
		Teams         []string // VFX 팀 리스트
		SelectedDept  string   // 검색한 부서명
		SelectedTeam  string   // 검색한 팀명
		SearchWord    string   // 검색어

		Projects        []string              // 타임로그 정보가 있는 프로젝트 리스트
		ArtistDatas     map[string]ArtistData // 타임로그를 작성한 아티스트 리스트
		ProjectDuration map[string]float64    // 프로젝트별 타임로그 정보
		TotalDuration   float64               // 총 타임로그 시간

		NoneArtists          []string  // DB에 존재하지 않는 아티스트들의 ID
		NoneProjects         []string  // DB에 존재하지 않는 프로젝트 ID
		StartDateErrProjects []Project // 타임로그 임포트를 하는 시점과 작업 시작일이 잘못된 프로젝트
		EndDateErrProjects   []Project // 타임로그 임포트를 하는 시점과 작업 마감일이 잘못된 프로젝트
	}
	rcp := Recipe{}
	q := r.URL.Query()
	rcp.Token = token
	rcp.User, err = getUserFunc(client, token.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rcp.UpdatedTime = adminSetting.SGUpdatedTime
	date := q.Get("date")
	y, m, _ := time.Now().Date()
	if date == "" { // date 값이 없으면 올해로 검색
		date = fmt.Sprintf("%04d-%02d", y, m)
	}
	rcp.NowDate = fmt.Sprintf("%04d-%02d", y, m)
	rcp.Date = date
	ms, err := getMonthlyStatusFunc(client, date)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			rcp.MonthlyStatus = "none"
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		rcp.MonthlyStatus = strconv.FormatBool(ms.Status)
	}
	rcp.Depts = adminSetting.VFXDepts // adminsetting의 VFX 부서를 가져온다
	rcp.SelectedDept = q.Get("dept")
	rcp.SelectedTeam = q.Get("team")
	rcp.SearchWord = q.Get("searchword")
	if rcp.SelectedDept == "" {
		for _, value := range adminSetting.VFXTeams {
			rcp.Teams = append(rcp.Teams, value...)
		}
	} else {
		rcp.Teams = adminSetting.VFXTeams[rcp.SelectedDept]
	}
	sort.Strings(rcp.Teams)
	// searchword로 아티스트를 검색한다.
	searchArtists, err := searchArtistFunc(client, rcp.SearchWord)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	year, _ := strconv.Atoi(strings.Split(date, "-")[0])
	month, _ := strconv.Atoi(strings.Split(date, "-")[1])
	timelogs, err := getTimelogOfTheMonthVFXFunc(client, year, month) // 검색한 달의 타임로그 데이터를 가져온다.
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rcp.ArtistDatas = make(map[string]ArtistData)
	rcp.ProjectDuration = make(map[string]float64)
	for _, timelog := range timelogs {
		if !checkStringInListFunc(timelog.Project, rcp.Projects) {
			rcp.Projects = append(rcp.Projects, timelog.Project)
		}
		if rcp.SearchWord != "" {
			if searchArtists == nil { // seachword로 아티스트를 검색했을 때 아무도 없으면 반복문을 빠져나간다.
				break
			}
		}
		artist, err := getArtistFunc(client, timelog.UserID) // DB에서 아티스트를 검색한다.
		if err != nil {
			if err == mongo.ErrNoDocuments { // DB에 없을 경우 errArtistID에 추가한다.
				if !checkStringInListFunc(timelog.UserID, rcp.NoneArtists) {
					rcp.NoneArtists = append(rcp.NoneArtists, timelog.UserID)
					continue
				}
				continue
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if rcp.SelectedDept != "" { // search한 부서가 있는 경우
			if artist.Dept != rcp.SelectedDept { // search한 부서가 아티스트의 dept와 같지 않으면 다음으로 넘어간다.
				continue
			}
		}
		if rcp.SelectedTeam != "" { // search한 team이 있는 경우
			if artist.Team != rcp.SelectedTeam { // search한 팀이 아티스트의 team과 같지 않으면 다음으로 넘어간다.
				continue
			}
		}
		if rcp.SearchWord != "" {
			found := false
			for _, sa := range searchArtists { // searchword로 검색한 아티스트 리스트에 없으면 continue
				if artist.ID == sa.ID {
					found = true
				}
			}
			if !found {
				continue
			}
		}
		if _, exists := rcp.ArtistDatas[artist.ID]; !exists { // 리스트에 아티스트가 없으면 ArtistDatas에 추가
			a := ArtistData{}
			a.Name = artist.Name
			a.Timelogs = make(map[string]float64)
			a.TotalDuration = 0.0
			rcp.ArtistDatas[artist.ID] = a
		}

		// 아티스트의 타임로그 정보 추가
		artistData := rcp.ArtistDatas[artist.ID]
		artistData.Timelogs[timelog.Project] = math.Round(timelog.Duration/60*10) / 10
		artistData.TotalDuration += math.Round(timelog.Duration/60*10) / 10
		rcp.ArtistDatas[artist.ID] = artistData

		rcp.ProjectDuration[timelog.Project] += math.Round(timelog.Duration/60*10) / 10 // 프로젝트의 타임로그 정보 추가
		rcp.TotalDuration += math.Round(timelog.Duration/60*10) / 10
	}
	sort.Strings(rcp.Projects)

	// 현재 월의 타임로그를 기준으로 작업기간이 맞지 않는 프로젝트를 확인한다.
	var startDateErrProject []Project // 타임로그 임포트를 하는 시점과 프로젝트의 작업 시작이 잘못된 경우 에러 처리
	var endDateErrProject []Project   // 타임로그 임포트를 하는 시점과 프로젝트의 작업 끝이 잘못된 경우 에러 처리
	thisDate, err := time.Parse("2006-01", date)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, pid := range rcp.Projects {
		project, err := getProjectFunc(client, pid)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				if !checkStringInListFunc(pid, rcp.NoneProjects) {
					rcp.NoneProjects = append(rcp.NoneProjects, pid)
					continue
				}
				continue
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// 프로젝트 작업 시작일과 비교
		startDate, err := time.Parse("2006-01", project.StartDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if thisDate.Before(startDate) { // 타임로그를 임포트하는 달이 작업시작일 전인 경우
			startDateErrProject = append(startDateErrProject, project)
		}

		// 프로젝트 작업 마감일과 비교
		endDate, err := time.Parse("2006-01", project.SMEndDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if thisDate.After(endDate) { // 타임로그를 임포트하는 달이 작업마감일 이후인 경우
			endDateErrProject = append(endDateErrProject, project)
		}
	}
	rcp.StartDateErrProjects = startDateErrProject
	rcp.EndDateErrProjects = endDateErrProject

	// ArtistData 자료구조를 인자로 넘길 수 없어서 json 파일을 생성한다.
	path := os.TempDir() + "/budget/" + token.ID + "/timelogvfx/" // json으로 바꾼 타임로그 데이터를 저장할 임시 폴더 경로
	err = createFolderFunc(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonData, _ := json.Marshal(rcp.ArtistDatas)
	_ = ioutil.WriteFile(path+"/timelog.json", jsonData, 0644)

	excelFileName := fmt.Sprintf("vfx_timelog_%s", strings.ReplaceAll(date, "-", "_"))
	if rcp.SelectedDept != "" {
		excelFileName = fmt.Sprintf("%s_%s", excelFileName, rcp.SelectedDept)
	}
	if rcp.SelectedTeam != "" {
		excelFileName = fmt.Sprintf("%s_%s", excelFileName, rcp.SelectedTeam)
	}
	if rcp.SearchWord != "" {
		excelFileName = fmt.Sprintf("%s_%s", excelFileName, rcp.SearchWord)
	}
	excelFileName = strings.ReplaceAll(excelFileName, "/", "&")
	err = genTimelogVFXExcelFunc(rcp.Projects, rcp.ProjectDuration, rcp.TotalDuration, token.ID, excelFileName+".xlsx") // 엑셀 파일 미리 생성
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "timelog-vfx", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleSearchTimelogVFXSubmitFunc 함수는 VFX 타임로그 페이지에서 Search를 눌렀을 때 실행되는 함수이다.
func handleSearchTimelogVFXFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// default 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < DefaultLevel {
		http.Redirect(w, r, "/invalidaccess", http.StatusSeeOther)
		return
	}

	date := r.FormValue("date")
	dept := r.FormValue("dept")
	team := url.QueryEscape(r.FormValue("team"))
	searchword := r.FormValue("searchword")

	http.Redirect(w, r, fmt.Sprintf("/timelog-vfx?date=%s&dept=%s&team=%s&searchword=%s", date, dept, team, searchword), http.StatusSeeOther)
}

// handleUpdateTimelogVFXFunc 함수는 VFX팀 타임로그를 업데이트하기 위해 엑셀 파일을 임포트하는 페이지를 불러오는 함수이다.
func handleUpdateTimelogVFXFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// manager 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < ManagerLevel {
		http.Redirect(w, r, "/invalidaccess", http.StatusSeeOther)
		return
	}

	q := r.URL.Query()
	date := q.Get("date")
	if date == "" {
		y, m, _ := time.Now().Date()
		date = fmt.Sprintf("%04d-%02d", y, m)
	}

	type Recipe struct {
		Token Token
		Date  string
	}
	rcp := Recipe{
		Token: token,
		Date:  date,
	}

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "updatetimelog-vfx", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleTimelogVFXExcelDownloadFunc 함수는 VFX팀의 타임로그 정보를 입력할 엑셀 파일의 템플릿을 생성하여 다운로드하는 함수이다.
func handleTimelogVFXExcelDownloadFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// manager 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < ManagerLevel {
		http.Redirect(w, r, "/invalidaccess", http.StatusSeeOther)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Get Only", http.StatusMethodNotAllowed)
		return
	}

	// mongoDB client 연결
	credential := options.Credential{
		Username: *flagDBID,
		Password: *flagDBPW,
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(*flagMongoDBURI).SetAuth(credential))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	adminsetting, err := getAdminSettingFunc(client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	// 엑셀 파일 생성
	f := excelize.NewFile()
	sheet := "Sheet1"
	index := f.NewSheet(sheet)
	f.SetActiveSheet(index)

	// 스타일
	style, err := f.NewStyle(`{"alignment":{"horizontal":"center","vertical":"center","wrap_text":true}}`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Template 제목 생성
	f.SetCellValue(sheet, "A1", "Shotgun ID")
	f.SetCellValue(sheet, "B1", "이름")

	var projects []string
	sprojects, err := sgGetProjectsFunc(adminsetting.SGExcludeProjects)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, p := range sprojects { // 태스크로 구분하는 프로젝트는 프로젝트 리스트에서 제외한다.
		if checkStringInListFunc(p, adminsetting.TaskProjects) {
			continue
		}
		projects = append(projects, p)
	}
	for i, project := range projects {
		pos, err := excelize.CoordinatesToCellName(i+3, 1) // ex) C1, D1 ...
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		f.SetCellValue(sheet, pos, project)
	}

	pos, err := excelize.CoordinatesToCellName(len(projects)+2, 1)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	f.SetColWidth(sheet, "A", strings.Split(pos, "1")[0], 15)
	f.SetRowHeight(sheet, 1, 40)
	f.SetCellStyle(sheet, "A1", pos, style)

	tempDir, err := ioutil.TempDir("", "excel")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tempDir) // 다운로드 후 임시 파일 삭제

	filename := "vfx_timelog_template.xlsx"
	err = f.SaveAs(tempDir + "/" + filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Disposition", fmt.Sprintf("Attachment; filename=%s", filename))
	http.ServeFile(w, r, tempDir+"/"+filename)
}

// handleUploadTimelogVFXExcelFunc 함수는 업로드한 엑셀 파일을 임시 폴더로 복사하는 함수이다.
func handleUploadTimelogVFXExcelFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// manager 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < ManagerLevel {
		http.Redirect(w, r, "/invalidaccess", http.StatusSeeOther)
		return
	}

	err = r.ParseMultipartForm(200000)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	path := os.TempDir() + "/budget/" + token.ID + "/timelog/" // 엑셀 파일을 저장할 임시 폴더 경로
	for _, files := range r.MultipartForm.File {
		for _, f := range files {
			file, err := f.Open()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer file.Close()

			mimeType := f.Header.Get("Content-Type")
			switch mimeType {
			// MS-Excel, Google & Libre Excel 등
			case "text/csv", "application/vnd.ms-excel", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
				data, err := ioutil.ReadAll(file)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				err = createFolderFunc(path)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				err = delAllFilesFunc(path)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				path = path + f.Filename
				err = ioutil.WriteFile(path, data, 0666)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			default:
				http.Error(w, "허용하지 않는 파일 포맷입니다", http.StatusInternalServerError)
				return
			}
		}
	}
}

// handleTimelogVFXExcelSubmitFunc 함수는 엑셀 파일에서 데이터를 가져와 체크하는 페이지로 전달하는 함수이다.
func handleTimelogVFXExcelSubmitFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// manager 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < ManagerLevel {
		http.Redirect(w, r, "/invalidaccess", http.StatusSeeOther)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, err.Error(), http.StatusMethodNotAllowed)
		return
	}

	path := os.TempDir() + "/budget/" + token.ID + "/timelog"
	q := r.URL.Query()
	date := q.Get("date")

	// path에 있는 파일들을 가져온다.
	fileInfo, err := ioutil.ReadDir(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// path에 파일의 개수가 하나가 아니면 엑셀 파일을 다시 업로드하도록 리다이렉트
	if len(fileInfo) != 1 {
		http.Redirect(w, r, "/updatetimelog-vfx", http.StatusSeeOther)
		return
	}

	// 파일의 확장자가 xlsx가 아니면 엑셀 파일을 다시 업로드하도록 리다이렉트
	ext := filepath.Ext(fileInfo[0].Name())
	if ext != ".xlsx" {
		http.Redirect(w, r, "/updatetimelog-vfx", http.StatusSeeOther)
		return
	}

	// 엑셀 파일에서 데이터를 가져온다.
	f, err := excelize.OpenFile(filepath.Join(path, fileInfo[0].Name()))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	excelRows, err := f.GetRows("Sheet1")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(excelRows) == 0 {
		http.Error(w, "엑셀 파일의 Sheet1 값이 비어있습니다", http.StatusBadRequest)
		return
	}

	type ArtistData struct {
		Name     string             // 아티스트 이름
		Timelogs map[string]float64 // 타임로그 정보
	}

	type Recipe struct {
		Token       Token
		Year        int
		Month       int
		Projects    []string              // 현재 진행중인 프로젝트 리스트
		ArtistDatas map[string]ArtistData // 타임로그를 작성한 아티스트 리스트
	}
	rcp := Recipe{}
	rcp.Token = token
	rcp.Year, _ = strconv.Atoi(strings.Split(date, "-")[0])
	rcp.Month, _ = strconv.Atoi(strings.Split(date, "-")[1])

	rcp.ArtistDatas = make(map[string]ArtistData)
	for n, line := range excelRows {
		// 첫번째 줄
		if n == 0 {
			if len(line) < 3 {
				http.Error(w, "엑셀 파일의 Cell 개수는 3개 이상이어야 합니다", http.StatusBadRequest)
				return
			}
			rcp.Projects = line[2:]
			continue
		}

		if len(line) == 0 { // 행을 추가하거나 삭제하면 데이터가 없는 행도 가져오게된다.
			break
		}

		// Artist ID
		id, err := f.GetCellValue("Sheet1", fmt.Sprintf("A%d", n+1))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if id == "" {
			continue
		}

		// 아티스트 이름
		name, err := f.GetCellValue("Sheet1", fmt.Sprintf("B%d", n+1))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		for i := 3; i < len(rcp.Projects)+3; i++ {
			pos, err := excelize.CoordinatesToCellName(i, n+1) // ex) Cn, Dn ...
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			d, err := f.GetCellValue("Sheet1", pos)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if d == "" { // duration이 비어 있으면 continue
				continue
			}

			duration, err := strconv.ParseFloat(d, 64)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			projectPos, err := excelize.CoordinatesToCellName(i, 1) // ex) C1, D1 ...
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			projectName, err := f.GetCellValue("Sheet1", projectPos)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if _, exists := rcp.ArtistDatas[id]; !exists {
				a := ArtistData{}
				a.Name = name
				a.Timelogs = make(map[string]float64)
				rcp.ArtistDatas[id] = a
			}

			// 아티스트의 타임로그 정보 추가
			artistData := rcp.ArtistDatas[id]
			artistData.Timelogs[projectName] = duration
			rcp.ArtistDatas[id] = artistData
		}
	}

	// json 파일 생성
	jsonData, _ := json.Marshal(rcp.ArtistDatas)
	_ = ioutil.WriteFile(path+"/timelog.json", jsonData, 0644)

	err = TEMPLATES.ExecuteTemplate(w, "updatetimelogvfx-check", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleUpdateTimelogVFXSubmitFunc 함수는 DB에서 선택한 월에 작성한 VFX팀 타임로그 데이터를 업데이트한 후 완료되면 /updatetimelogvfx-success로 리다이렉트하는 함수이다.
func handleUpdateTimelogVFXSubmitFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// manager 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < ManagerLevel {
		http.Redirect(w, r, "/invalidaccess", http.StatusSeeOther)
		return
	}

	// mongoDB client 연결
	credential := options.Credential{
		Username: *flagDBID,
		Password: *flagDBPW,
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(*flagMongoDBURI).SetAuth(credential))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// DB에서 Admin setting 데이터를 가져온다.
	adminSetting, err := getAdminSettingFunc(client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	q := r.URL.Query()
	date := q.Get("date")
	year, err := strconv.Atoi(strings.Split(date, "-")[0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	month, err := strconv.Atoi(strings.Split(date, "-")[1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 1. json 파일에서 타임로그 가져오기
	path := os.TempDir() + "/budget/" + token.ID + "/timelog/timelog.json" // json 파일 경로
	jsonData, err := ioutil.ReadFile(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type ArtistData struct {
		Name     string             // 아티스트 이름
		Timelogs map[string]float64 // 타임로그 정보
	}

	timelogData := make(map[string]ArtistData)
	json.Unmarshal(jsonData, &timelogData)

	// 2. 정산 완료된 프로젝트에 작성한 타임로그가 있는지 확인
	// 엑셀로 임포트했을 때는 정산 완료된 프로젝트의 타임로그는 따로 처리하지 않는다.

	// 3. 내부 인건비와 타임로그 삭제
	// 이미 존재하는 타임로그의 프로젝트 내부 인건비(VFX, RND)를 비워주고, DB에 저장하기 전에 VFX 타임로그 정보를 삭제한다.
	projects, err := getProjectsByTimelogFunc(year, month, "vfx")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	date = fmt.Sprintf("%04d-%02d", year, month)
	for _, p := range projects {
		laborCost := p.SMMonthlyLaborCost[date]
		if laborCost != (LaborCost{}) { // 인건비가 비어있는지 확인한다.
			laborCost.VFX = ""
			laborCost.RND = ""
			p.SMMonthlyLaborCost[date] = laborCost
			err = setProjectFunc(client, p)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
	err = rmVFXTimelogFunc(client, year, month, nil) // 수퍼바이저의 타임로그도 수정될 수 있기 때문에 수퍼바이저를 포함한 데이터가 지워지도록 한다.
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 4. DB에 타임로그 저장
	var projectList []string                // 현재 import한 엑셀 파일 내의 프로젝트만 인건비를 계산하기 위한 리스트
	rndProjects := adminSetting.RNDProjects // RND 프로젝트 리스트
	etcProjects := adminSetting.ETCProjects // ETC 프로젝트 리스트
	errTimelog := make(map[string]string)
	for artistID, artistData := range timelogData {
		for projectName, duration := range artistData.Timelogs {
			// rnd 프로젝트 리스트에 포함되어 있다면 타임로그의 프로젝트를 RND2021 형태로 수정하고, rnd 프로젝트의 duration과 합쳐춘다.
			if checkStringInListFunc(projectName, rndProjects) {
				projectName = fmt.Sprintf("RND%04d", year)
				rndTimelog, err := getTimelogFunc(client, artistID, year, month, projectName)
				if err != nil {
					if err != mongo.ErrNoDocuments {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}
				duration = duration + rndTimelog.Duration/60.0
			}
			// etc 프로젝트 리스트에 포함되어 있다면 타임로그의 프로젝트를 ETC2021 형태로 수정하고, etc 프로젝트의 duration과 합쳐춘다.
			if checkStringInListFunc(projectName, etcProjects) {
				projectName = fmt.Sprintf("ETC%04d", year)
				etcTimelog, err := getTimelogFunc(client, artistID, year, month, projectName)
				if err != nil {
					if err != mongo.ErrNoDocuments {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}
				duration = duration + etcTimelog.Duration/60.0
			}

			t := Timelog{
				UserID:   artistID,
				Year:     year,
				Month:    month,
				Project:  projectName,
				Duration: duration * 60,
			}
			err = t.CheckErrorFunc()
			if err != nil {
				errTimelog[artistID] = err.Error()
				continue
			}

			if !checkStringInListFunc(projectName, projectList) {
				projectList = append(projectList, projectName)
			}

			err = addTimelogFunc(client, t)
			if err != nil {
				errTimelog[artistID] = err.Error()
			}
		}
	}

	// 규칙에 맞지 않는 타임로그가 존재할 경우 타임로그 업로드 실패 페이지로 이동한다.
	if len(errTimelog) != 0 {
		type Recipe struct {
			Token    Token
			Timelogs map[string]string
			Date     string
		}
		rcp := Recipe{
			Token:    token,
			Timelogs: errTimelog,
			Date:     date,
		}

		log := Log{}
		log.UserID = token.ID
		log.CreatedAt = time.Now()
		log.Content = fmt.Sprintf("%d년 %d월의 VFX 타임로그 임포트 중 타임로그를 추가하지 못했습니다.", year, month)

		err = addLogsFunc(client, log)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		err = TEMPLATES.ExecuteTemplate(w, "updatetimelogvfx-fail", rcp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		return
	}

	// 5. finishedtimelogstatus의 타임로그 정보 업데이트
	// 엑셀로 임포트했을 때는 정산 완료된 프로젝트의 타임로그는 따로 처리하지 않는다.

	// 6. 인건비 계산
	// DB에 추가하지 못한 타임로그가 존재할 경우 에러 페이지로 이동하고, 존재하지 않을 경우에는 인건비를 계산한다.
	var errProject []string // 프로젝트가 존재하지 않을 때의 에러 처리
	for _, p := range projectList {
		project, err := getProjectFunc(client, p) // DB에서 해당 프로젝트를 가져온다.
		if err != nil {
			if err == mongo.ErrNoDocuments { // DB에 해당 프로젝트가 존재하지 않는 경우
				if !checkStringInListFunc(p, errProject) {
					errProject = append(errProject, p)
					continue
				}
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		monthlyLaborCost := make(map[string]LaborCost)
		if project.SMMonthlyLaborCost != nil {
			monthlyLaborCost = project.SMMonthlyLaborCost
		}
		laborCost := LaborCost{}

		// VFX 인건비 계산
		vfxLaborCost, err := calMonthlyVFXLaborCostFunc(p, year, month)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		laborCost.VFX, err = encryptAES256Func(strconv.Itoa(vfxLaborCost))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// CM 인건비 계산 -> VFX 타임로그와 상관없이 변하면 안되기 때문에 저장된 값을 가져온다.
		laborCost.CM = project.SMMonthlyLaborCost[date].CM

		monthlyLaborCost[date] = laborCost
		project.SMMonthlyLaborCost = monthlyLaborCost
		err = setProjectFunc(client, project)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if len(errProject) != 0 {
		type Recipe struct {
			Token    Token
			Projects []string
		}
		rcp := Recipe{
			Token:    token,
			Projects: errProject,
		}

		log := Log{}
		log.UserID = token.ID
		log.CreatedAt = time.Now()
		log.Content = fmt.Sprintf("%d년 %d월의 VFX 타임로그 임포트 중 존재하지않는 프로젝트로 인해 인건비 계산을 못했습니다.", year, month)

		err = addLogsFunc(client, log)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		err = TEMPLATES.ExecuteTemplate(w, "updatetimelogvfx-projectfail", rcp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// 월별 결산 상태를 True로 전환
		ms := MonthlyStatus{}
		ms.Date = date
		ms.Status = true
		err = setMonthlyStatusFunc(client, ms)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log := Log{}
		log.UserID = token.ID
		log.CreatedAt = time.Now()
		log.Content = fmt.Sprintf("%d년 %d월의 VFX 타임로그를 임포트 완료했습니다.", year, month)

		err = addLogsFunc(client, log)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, fmt.Sprintf("updatetimelogvfx-success?date=%s", date), http.StatusSeeOther)
	}
}

// handleUpdateTimelogVFXSuccessFunc 함수는 VFX팀 타임로그 정보 업데이트를 성공했다는 페이지를 연다.
func handleUpdateTimelogVFXSuccessFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// manager 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < ManagerLevel {
		http.Redirect(w, r, "/invalidaccess", http.StatusSeeOther)
		return
	}

	type Recipe struct {
		Token
		Date string
	}
	rcp := Recipe{}
	rcp.Token = token
	q := r.URL.Query()
	rcp.Date = q.Get("date")

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "updatetimelogvfx-success", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// genTimelogVFXExcelFunc 함수는 VFX팀의 타임로그 데이터를 엑셀 파일로 생성하는 함수이다.
func genTimelogVFXExcelFunc(projects []string, projectDuration map[string]float64, totalDuration float64, userID string, excelFileName string) error {
	path := os.TempDir() + "/budget/" + userID + "/timelogvfx/"

	// json 파일에서 타임로그 데이터를 가져온다.
	jsonData, err := ioutil.ReadFile(path + "timelog.json")
	if err != nil {
		return err
	}

	type ArtistData struct {
		Name          string             // 아티스트 이름
		Timelogs      map[string]float64 // 타임로그 정보
		TotalDuration float64            // 아티스트가 작성한 총 타임로그 시간
	}

	timelogData := make(map[string]ArtistData)
	json.Unmarshal(jsonData, &timelogData)

	type Sort struct {
		ID   string
		Name string
	}
	var sortList []Sort
	for artistID, artistData := range timelogData {
		found := false
		for _, n := range sortList {
			if n.ID == artistID {
				found = true
				break
			}
		}
		if !found {
			sort := Sort{
				ID:   artistID,
				Name: artistData.Name,
			}
			sortList = append(sortList, sort)
		}
	}
	sort.Slice(sortList, func(i, j int) bool { // 이름으로 오름차순 정렬
		return sortList[i].Name < sortList[j].Name
	})

	err = delAllFilesFunc(path)
	if err != nil {
		return err
	}

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

	// 엑셀 파일 생성
	f := excelize.NewFile()
	sheet := "Sheet1"
	index := f.NewSheet(sheet)
	f.SetActiveSheet(index)

	// 스타일
	style, err := f.NewStyle(`{"alignment":{"horizontal":"center","vertical":"center","wrap_text":true}}`)
	if err != nil {
		return err
	}
	totalStyle, err := f.NewStyle(
		`
		{"alignment":{"horizontal":"center","vertical":"center","wrap_text":true},
		"font":{"bold":true}, 
		"fill":{"type":"pattern","color":["#FFC000"],"pattern":1}}
		`)
	if err != nil {
		return err
	}

	// 제목 입력
	projectNameDict := make(map[string]string) // {"철인왕후": "CHU"}
	f.SetCellValue(sheet, "A1", "Shotgun ID")
	f.SetCellValue(sheet, "B1", "이름")
	for i, project := range projects {
		projectName, err := getNameOfProjectFunc(client, project)
		if err != nil {
			if err == mongo.ErrNoDocuments { // DB에 없는 프로젝트라면 ID로 보여준다.
				projectName = project
			} else {
				return err
			}
		}
		projectNameDict[projectName] = project
		pos, err := excelize.CoordinatesToCellName(i+3, 1) // ex) C1, D1 ...
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, projectName)
	}
	pos, err := excelize.CoordinatesToCellName(len(projects)+3, 1)
	if err != nil {
		return err
	}
	f.SetCellValue(sheet, pos, "Total")
	f.SetColWidth(sheet, "A", strings.Split(pos, "1")[0], 15)
	f.SetRowHeight(sheet, 1, 40)
	// 데이터 입력
	row := 2
	for _, sortData := range sortList {
		artistID := sortData.ID
		artistData := timelogData[artistID]
		// Shotgun ID
		aPos, err := excelize.CoordinatesToCellName(1, row)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, aPos, artistID)

		// 이름
		pos, err := excelize.CoordinatesToCellName(2, row)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, artistData.Name)

		// 타임로그
		for projectName, duration := range artistData.Timelogs {
			for i := 0; i < len(projects); i++ {
				projectPos, err := excelize.CoordinatesToCellName(i+3, 1)
				if err != nil {
					return err
				}
				pName, err := f.GetCellValue(sheet, projectPos)
				if err != nil {
					return err
				}
				pName = projectNameDict[pName] // 프로젝트의 ID를 가져옴
				if projectName == pName {
					pos, err = excelize.CoordinatesToCellName(i+3, row)
					if err != nil {
						return err
					}
					d := duration
					f.SetCellValue(sheet, pos, d)
					break
				}
			}
		}

		// 아티스트가 작성한 총 타임로그 시간
		tPos, err := excelize.CoordinatesToCellName(len(projects)+3, row)
		if err != nil {
			return err
		}
		d := artistData.TotalDuration
		f.SetCellValue(sheet, tPos, d)

		row = row + 1
	}

	// 프로젝트의 총 타임로그 시간
	pos, err = excelize.CoordinatesToCellName(1, row)
	if err != nil {
		return err
	}
	f.SetCellValue(sheet, pos, "Total")
	mergePos, err := excelize.CoordinatesToCellName(2, row)
	if err != nil {
		return err
	}
	f.MergeCell(sheet, pos, mergePos)

	for projectName, duration := range projectDuration {
		for i := 0; i < len(projects); i++ {
			projectPos, err := excelize.CoordinatesToCellName(i+3, 1)
			if err != nil {
				return err
			}
			pName, err := f.GetCellValue(sheet, projectPos)
			if err != nil {
				return err
			}
			pName = projectNameDict[pName] // 프로젝트의 ID를 가져옴
			if projectName == pName {
				pos, err = excelize.CoordinatesToCellName(i+3, row)
				if err != nil {
					return err
				}
				d := duration
				f.SetCellValue(sheet, pos, d)
				break
			}
		}
	}

	// Total
	tPos, err := excelize.CoordinatesToCellName(len(projects)+3, row)
	if err != nil {
		return err
	}
	d := totalDuration
	f.SetCellValue(sheet, tPos, d)

	f.SetCellStyle(sheet, "A1", tPos, style)

	pos, err = excelize.CoordinatesToCellName(1, row)
	if err != nil {
		return err
	}
	f.SetCellStyle(sheet, pos, tPos, totalStyle)

	pos, err = excelize.CoordinatesToCellName(len(projects)+3, 1)
	if err != nil {
		return err
	}
	f.SetCellStyle(sheet, pos, tPos, totalStyle)

	// 엑셀 파일 저장
	err = f.SaveAs(path + "/" + excelFileName)
	if err != nil {
		return err
	}

	return nil
}

// handleExportTimelogVFXFunc 함수는 임시 폴더에 저장한 엑셀 파일을 다운로드하는 함수이다.
func handleExportTimelogVFXFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// manager 레벨 미만이면 invalideaccess 페이지로 리다이렉트
	if token.AccessLevel < ManagerLevel {
		http.Redirect(w, r, "/invalidaccess", http.StatusSeeOther)
		return
	}

	// Post 메소드가 아니면 에러
	if r.Method != http.MethodPost {
		http.Error(w, "Post Method Only", http.StatusMethodNotAllowed)
		return
	}

	path := os.TempDir() + "/budget/" + token.ID + "/timelogvfx"

	// path에 있는 파일들을 가져온다.
	fileInfo, err := ioutil.ReadDir(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 파일의 확장자가 xlsx가 아니면 엑셀 파일이 다시 생성되도록 리다이렉트
	ext := filepath.Ext(fileInfo[0].Name())
	if ext != ".xlsx" {
		http.Redirect(w, r, "/timelog-vfx", http.StatusSeeOther)
		return
	}

	// mongoDB client 연결
	credential := options.Credential{
		Username: *flagDBID,
		Password: *flagDBPW,
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(*flagMongoDBURI).SetAuth(credential))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	filename := strings.Split(strings.Split(fileInfo[0].Name(), ".")[0], "_")

	log := Log{
		UserID:    token.ID,
		CreatedAt: time.Now(),
		Content:   fmt.Sprintf("VFX 타임로그 페이지에서 %s년 %s월의 데이터를 다운로드하였습니다.", filename[2], filename[3]),
	}

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Disposition", fmt.Sprintf("Attachment; filename=%s", fileInfo[0].Name()))
	http.ServeFile(w, r, path+"/"+fileInfo[0].Name())
}
