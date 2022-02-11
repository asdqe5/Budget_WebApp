// 프로젝트 결산 프로그램
//
// Description : http 누계 타임로그 관련 스크립트

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

// handleTimelogTotalFunc 함수는 타임로그 누계 페이지를 불러오는 함수이다.
func handleTimelogTotalFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/singin", http.StatusSeeOther)
		return
	}

	// default 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < DefaultLevel {
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

	adminSetting, err := getAdminSettingFunc(client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type ArtistData struct {
		Name          string             // 아티스트 이름
		Timelogs      map[string]float64 // 타임로그 정보
		TotalDuration float64            // 아티스트가 작성한 총 타임로그 시간
	}

	type Recipe struct {
		Token        Token
		User         User
		UpdatedTime  string
		Date         string
		Depts        []string // VFX 팀 태그 + CM
		Teams        []string // VFX 팀 리스트 + CM 팀 리스트
		SelectedDept string   // 검색한 부서명
		SelectedTeam string   // 검색한 팀명
		SearchWord   string   // 검색어

		Projects        []string              // 타임로그 정보가 있는 프로젝트 리스트
		VFXArtistDatas  map[string]ArtistData // 타임로그를 작성한 VFX 아티스트 리스트
		CMArtistDatas   map[string]ArtistData // 타임로그를 작성한 CM 아티스트 리스트
		ProjectDuration map[string]float64    // 프로젝트별 타임로그 정보
		TotalDuration   float64               // 총 타임로그 시간

		NoneArtists  []string // DB에 존재하지 않는 아티스트들의 ID
		NoneProjects []string // DB에 존재하지 않는 프로젝트들의 ID
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
	if date == "" {
		y, m, _ := time.Now().Date()
		date = fmt.Sprintf("%04d-%02d", y, m)
	}
	rcp.Date = date
	rcp.Depts = append(adminSetting.VFXDepts, "cm") // adminsetting의 VFX 부서(팀 태그)와 cm부서를 가져온다
	rcp.SelectedDept = q.Get("dept")
	rcp.SelectedTeam = q.Get("team")
	rcp.SearchWord = q.Get("searchword")
	if rcp.SelectedDept == "" {
		for _, value := range adminSetting.VFXTeams {
			rcp.Teams = append(rcp.Teams, value...)
		}
		for _, value := range adminSetting.CMTeams {
			if !checkStringInListFunc(value, rcp.Teams) {
				rcp.Teams = append(rcp.Teams, value)
			}
		}
	} else if rcp.SelectedDept == "cm" {
		rcp.Teams = adminSetting.CMTeams
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
	vfxTimelogs, err := getTimelogUntilTheMonthVFXFunc(client, year, month) // 검색한 달까지의 VFX 타임로그 데이터를 가져온다.
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cmTimelogs, err := getTimelogUntilTheMonthCMFunc(client, year, month) // 검색한 연도의 CM 타임로그 데이터를 가져온다.
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rcp.VFXArtistDatas = make(map[string]ArtistData)
	rcp.CMArtistDatas = make(map[string]ArtistData)
	rcp.ProjectDuration = make(map[string]float64)
	for _, timelog := range vfxTimelogs { // VFX 타임로그
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
			if err == mongo.ErrNoDocuments { // DB에 없을 경우 NoneArtistID에 추가한다.
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
			if artist.Dept != rcp.SelectedDept {
				continue
			}
		}

		if rcp.SelectedTeam != "" {
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

		if _, exitsts := rcp.VFXArtistDatas[artist.ID]; !exitsts { // 리스트에 아티스트가 없으면 ArtistDatas에 추가
			a := ArtistData{}
			a.Name = artist.Name
			a.Timelogs = make(map[string]float64)
			a.TotalDuration = 0.0
			rcp.VFXArtistDatas[artist.ID] = a
		}

		// 아티스트의 타임로그 정보 추가
		artistData := rcp.VFXArtistDatas[artist.ID]
		artistData.Timelogs[timelog.Project] += math.Round(timelog.Duration/60*10) / 10
		artistData.TotalDuration += math.Round(timelog.Duration/60*10) / 10
		rcp.VFXArtistDatas[artist.ID] = artistData

		rcp.ProjectDuration[timelog.Project] += math.Round(timelog.Duration/60*10) / 10 // 프로젝트의 타임로그 정보 추가
		rcp.TotalDuration += math.Round(timelog.Duration/60*10) / 10
	}

	for _, timelog := range cmTimelogs { //CM 타임로그
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
			if err == mongo.ErrNoDocuments { // DB에 없을 경우 NoneArtistID에 추가한다.
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
			if artist.Dept != rcp.SelectedDept {
				continue
			}
		}

		if rcp.SelectedTeam != "" {
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

		if _, exitsts := rcp.CMArtistDatas[artist.ID]; !exitsts { // 리스트에 아티스트가 없으면 ArtistDatas에 추가
			a := ArtistData{}
			a.Name = artist.Name
			a.Timelogs = make(map[string]float64)
			a.TotalDuration = 0.0
			rcp.CMArtistDatas[artist.ID] = a
		}

		// 아티스트의 타임로그 정보 추가
		artistData := rcp.CMArtistDatas[artist.ID]
		artistData.Timelogs[timelog.Project] += math.Round(timelog.Duration/60*10) / 10
		artistData.TotalDuration += math.Round(timelog.Duration/60*10) / 10
		rcp.CMArtistDatas[artist.ID] = artistData

		rcp.ProjectDuration[timelog.Project] += math.Round(timelog.Duration/60*10) / 10 // 프로젝트의 타임로그 정보 추가
		rcp.TotalDuration += math.Round(timelog.Duration/60*10) / 10
	}
	sort.Strings(rcp.Projects)

	// DB에 없는 프로젝트 확인
	for _, pid := range rcp.Projects {
		_, err := getProjectFunc(client, pid)
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
	}

	// ArtistData 자료구조를 인자로 넘길 수 없어서 json 파일을 생성한다.
	path := os.TempDir() + "/budget/" + token.ID + "/timelogtotal/" // json으로 바꾼 타임로그 데이터를 저장할 임시 폴더 경로
	err = createFolderFunc(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonVFXData, _ := json.Marshal(rcp.VFXArtistDatas)
	_ = ioutil.WriteFile(path+"/vfxtimelog.json", jsonVFXData, 0644)
	jsonCMData, _ := json.Marshal(rcp.CMArtistDatas)
	_ = ioutil.WriteFile(path+"/cmtimelog.json", jsonCMData, 0644)

	excelFileName := fmt.Sprintf("total_timelogs_%04d_%02d", year, month)
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
	err = genTimelogTotalExcelFunc(rcp.Projects, rcp.ProjectDuration, rcp.TotalDuration, token.ID, excelFileName+".xlsx") // 엑셀 파일 미리 생성
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "timelog-total", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleSearchTimelogTotalFunc 함수는 누계 타임로그 페이지에서 Search를 눌렀을 때 실행되는 함수이다.
func handleSearchTimelogTotalFunc(w http.ResponseWriter, r *http.Request) {
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

	http.Redirect(w, r, fmt.Sprintf("/timelog-total?date=%s&dept=%s&team=%s&searchword=%s", date, dept, team, searchword), http.StatusSeeOther)
}

// genTimelogTotalExcelFunc 함수는 누계 타임로그 데이터를 엑셀 파일로 생성하는 함수이다
func genTimelogTotalExcelFunc(projects []string, projectDurtation map[string]float64, totalDuration float64, userID string, excelFileName string) error {
	path := os.TempDir() + "/budget/" + userID + "/timelogtotal/"

	// json 파일에서 타임로그 데이터를 가져온다.
	jsonVFXData, err := ioutil.ReadFile(path + "vfxtimelog.json")
	if err != nil {
		return err
	}
	jsonCMData, err := ioutil.ReadFile(path + "cmtimelog.json")
	if err != nil {
		return err
	}

	type ArtistData struct {
		Name          string             // 아티스트 이름
		Timelogs      map[string]float64 //타임로그 정보
		TotalDuration float64            // 아티스트가 작성한 총 타임로그 시간
	}

	vfxTimelogData := make(map[string]ArtistData)
	cmTimelogData := make(map[string]ArtistData)
	json.Unmarshal(jsonVFXData, &vfxTimelogData)
	json.Unmarshal(jsonCMData, &cmTimelogData)

	type Sort struct {
		ID   string
		Name string
	}
	var vfxSortList []Sort
	var cmSortList []Sort
	for artistID, artistData := range vfxTimelogData {
		found := false
		for _, n := range vfxSortList {
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
			vfxSortList = append(vfxSortList, sort)
		}
	}
	for artistID, artistData := range cmTimelogData {
		found := false
		for _, n := range cmSortList {
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
			cmSortList = append(cmSortList, sort)
		}
	}
	sort.Slice(vfxSortList, func(i, j int) bool { // 이름으로 오름차순 정렬
		return vfxSortList[i].Name < vfxSortList[j].Name
	})
	sort.Slice(cmSortList, func(i, j int) bool { // 이름으로 오름차순 정렬
		return cmSortList[i].Name < cmSortList[j].Name
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
	f.SetCellValue(sheet, "A1", "ID")
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
		pos, err := excelize.CoordinatesToCellName(i+3, 1) // ex) C1, D1
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

	row := 2
	for _, sortData := range vfxSortList { // VFX 데이터 입력
		artistID := sortData.ID
		artistData := vfxTimelogData[artistID]
		// ID
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

	for _, sortData := range cmSortList { // CM 데이터 입력
		artistID := sortData.ID
		artistData := cmTimelogData[artistID]
		// ID
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

	for projectName, duration := range projectDurtation {
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

// handleExportTimelogTotalFunc 함수는 임시 폴더에 저장한 엑셀 파일을 다운로드하는 함수이다.
func handleExportTimelogTotalFunc(w http.ResponseWriter, r *http.Request) {
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

	path := os.TempDir() + "/budget/" + token.ID + "/timelogtotal"

	// path에 있는 파일들을 가져온다.
	fileInfo, err := ioutil.ReadDir(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 파일의 확장자가 xlsx가 아니면 엑셀 파일이 다시 생성되도록 리다이렉트
	ext := filepath.Ext(fileInfo[0].Name())
	if ext != ".xlsx" {
		http.Redirect(w, r, "/timelog-total", http.StatusSeeOther)
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
		Content:   fmt.Sprintf("Total 타임로그 페이지에서 %s년 %s월의 데이터를 다운로드하였습니다.", filename[2], filename[3]),
	}

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Disposition", fmt.Sprintf("Attachment; filename=%s", fileInfo[0].Name()))
	http.ServeFile(w, r, path+"/"+fileInfo[0].Name())
}
