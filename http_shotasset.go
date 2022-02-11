// 프로젝트 결산 프로그램
//
// Description : http 예산 샷, 어셋 관련 스크립트

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
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

// handelShotAssetFunc 함수는 샷, 어셋 관리 페이지를 여는 함수이다.
func handelShotAssetFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// member 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < MemberLevel {
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

	type Recipe struct {
		Token             Token
		Year              string      // 연도
		AllBGProjects     []BGProject // 해당 연도의 모든 예산 프로젝트
		SearchWord        string      // 검색어
		SelectedProjectID string      // 선택한 프로젝트 ID
		BGStatus          string      // 예산 프로젝트 Status
		BGProjects        []BGProject // 예산 프로젝트 리스트
	}
	rcp := Recipe{}
	rcp.Token = token
	q := r.URL.Query()
	year := q.Get("year")
	if year == "" { // year 값이 없으면 올해로 검색
		y, _, _ := time.Now().Date()
		year = strconv.Itoa(y)
	}
	rcp.Year = year
	rcp.SearchWord = q.Get("searchword")
	bgs := q.Get("status")
	if bgs == "" {
		bgs = "all"
	}
	rcp.BGStatus = bgs
	rcp.SelectedProjectID = q.Get("project")

	// 프로젝트 검색 - 모든 프로젝트
	searchword := "year:" + rcp.Year + " " + "status:" + bgs
	rcp.AllBGProjects, err = searchBGProjectFunc(client, searchword, "id")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Search에 따른 프로젝트
	if rcp.SelectedProjectID != "" {
		searchword = searchword + " " + "id:" + rcp.SelectedProjectID
	}
	if rcp.SearchWord != "" {
		searchword = searchword + " " + rcp.SearchWord
	}
	if rcp.SelectedProjectID == "" && rcp.SearchWord == "" {
		rcp.BGProjects = rcp.AllBGProjects
	} else {
		rcp.BGProjects, err = searchBGProjectFunc(client, searchword, "id") // DB에서 searchword로 프로젝트 검색
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "shotasset", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleSearchShotAssetFunc 함수는 Shot, Asset 관리 페이지에서 Search 버튼을 눌렀을 때 검색을 실행하는 함수이다,
func handleSearchShotAssetFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// member 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < MemberLevel {
		http.Redirect(w, r, "/invalidaccess", http.StatusSeeOther)
		return
	}

	status := ""
	if r.FormValue("truestatus") == "on" && r.FormValue("falsestatus") == "on" {
		status = "all"
	} else if r.FormValue("truestatus") == "on" {
		status = "true"
	} else if r.FormValue("falsestatus") == "on" {
		status = "false"
	}
	year := r.FormValue("year")
	project := r.FormValue("project")
	searchword := r.FormValue("searchword")

	http.Redirect(w, r, fmt.Sprintf("/shotasset?status=%s&year=%s&project=%s&searchword=%s", status, year, project, searchword), http.StatusSeeOther)
}

// handleUploadShotFunc 함수는 예산 프로젝트의 예산안에 해당하는 샷 엑셀 파일을 임포트하는 페이지를 불러오는 함수이다.
func handleUploadShotFunc(w http.ResponseWriter, r *http.Request) {
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
	id := q.Get("id")
	bgtype := q.Get("bgtype")
	typ := q.Get("type")
	if id == "" || bgtype == "" || typ == "" {
		http.Redirect(w, r, "/shotasset", http.StatusSeeOther)
		return
	}

	type Recipe struct {
		Token  Token
		ID     string
		BGType string
		Type   string
	}
	rcp := Recipe{
		Token:  token,
		ID:     id,
		BGType: bgtype,
		Type:   typ,
	}

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "uploadshot", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleShotExcelDownloadFunc 함수는 Shot 업로드를 위한 엑셀 파일의 템플릿을 생성하여 다운로드하는 함수이다.
func handleShotExcelDownloadFunc(w http.ResponseWriter, r *http.Request) {
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

	q := r.URL.Query()
	id := q.Get("id")
	bgtype := q.Get("bgtype")
	typ := q.Get("type")

	bgp, err := getBGProjectFunc(client, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var taskList []string
	bgTeamSetting := bgp.TypeData[bgtype].TeamSetting
	for _, deptList := range bgTeamSetting.Departments {
		for _, dept := range deptList {
			if dept.Type { // Asset 관련 팀인 경우 패스
				continue
			}
			for _, part := range dept.Parts {
				for _, task := range part.Tasks {
					if !checkStringInListFunc(task, taskList) {
						taskList = append(taskList, task)
					}
				}
			}
		}
	}
	sort.Strings(taskList)

	// 엑셀 파일 생성
	f := excelize.NewFile()
	sheet := "Sheet1"
	index := f.NewSheet(sheet)
	f.SetActiveSheet(index)

	// 스타일
	style, err := f.NewStyle(`{"alignment":{"horizontal":"center","vertical":"center","wrap_text":true}, "font":{"bold":true}}`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	f.SetCellValue(sheet, "A1", "Shot Name")
	pos := ""
	if typ == "drama" {
		f.SetCellValue(sheet, "B1", "Episode")
	}
	for i, task := range taskList {
		if typ == "movie" {
			pos, err = excelize.CoordinatesToCellName(i+2, 1)
		} else {
			pos, err = excelize.CoordinatesToCellName(i+3, 1)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		f.SetCellValue(sheet, pos, task)
	}
	f.SetColWidth(sheet, "A", strings.Split(pos, "1")[0], 15)
	f.SetRowHeight(sheet, 1, 30)
	f.SetCellStyle(sheet, "A1", pos, style)

	tempDir, err := ioutil.TempDir("", "excel")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tempDir) // 다운로드 후 임시 파일 삭제

	filename := "shot_template.xlsx"
	if typ == "movie" {
		filename = "movie_" + filename
	} else {
		filename = "drama_" + filename
	}
	err = f.SaveAs(tempDir + "/" + filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Disposition", fmt.Sprintf("Attachment; filename=%s", filename))
	http.ServeFile(w, r, tempDir+"/"+filename)
}

// handleUploadShotExcelFunc 함수는 업로드한 엑셀 파일을 임시 폴더로 복사하는 함수이다.
func handleUploadShotExcelFunc(w http.ResponseWriter, r *http.Request) {
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

	path := os.TempDir() + "/budget/" + token.ID + "/shot/" // 엑셀 파일을 저장할 임시 폴더 경로
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

// handleShotExcelSubmitFunc 함수는 엑셀 파일에서 데이터를 가져와 체크하는 페이지로 전달하는 함수이다.
func handleShotExcelSubmitFunc(w http.ResponseWriter, r *http.Request) {
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

	path := os.TempDir() + "/budget/" + token.ID + "/shot"

	q := r.URL.Query()
	id := q.Get("id")
	bgtype := q.Get("bgtype")
	typ := q.Get("type")

	// path에 있는 파일들을 가져온다.
	fileInfo, err := ioutil.ReadDir(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// path에 파일의 개수가 하나가 아니면 엑셀 파일을 다시 업로드하도록 리다이렉트
	if len(fileInfo) != 1 {
		http.Redirect(w, r, fmt.Sprintf("/uploadshot?id=%s&bgtype=%s&type=%s", id, bgtype, typ), http.StatusSeeOther)
		return
	}

	// 파일의 확장자가 xlsx가 아니면 엑셀 파일을 다시 업로드하도록 리다이렉트
	ext := filepath.Ext(fileInfo[0].Name())
	if ext != ".xlsx" {
		http.Redirect(w, r, fmt.Sprintf("/uploadshot?id=%s&bgtype=%s&type=%s", id, bgtype, typ), http.StatusSeeOther)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(excelRows) == 0 {
		http.Error(w, "엑셀 파일의 Sheet1 값이 비어있습니다", http.StatusBadRequest)
		return
	}

	type Recipe struct {
		Token      Token
		ID         string
		BGType     string
		Type       string
		BGShotList []BGShotAsset
		BGTaskList []string
		TotalBid   map[string]float64
	}
	rcp := Recipe{}
	rcp.Token = token
	rcp.ID = id
	rcp.BGType = bgtype
	rcp.Type = typ
	rcp.TotalBid = make(map[string]float64)
	for n, line := range excelRows {
		// 첫번째 줄
		if n == 0 {
			if typ == "movie" { // 영화인 경우
				if len(line) < 2 {
					http.Error(w, "엑셀 파일의 Cell 개수는 2개 이상이어야 합니다", http.StatusBadRequest)
					return
				}
				rcp.BGTaskList = line[1:]
			} else { // 드라마인 경우
				if len(line) < 3 {
					http.Error(w, "엑셀 파일의 Cell 개수는 3개 이상이어야 합니다", http.StatusBadRequest)
					return
				}
				rcp.BGTaskList = line[2:]
			}
			continue
		}

		if len(line) == 0 { // 행을 추가하거나 삭제하면 데이터가 없는 행도 가져오게된다.
			break
		}

		// Shot Name
		shotName, err := f.GetCellValue("Sheet1", fmt.Sprintf("A%d", n+1))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if shotName == "" {
			continue
		}

		// 드라마 에피소드 정보
		episode := ""
		if typ == "drama" { // 드라마인 경우
			episode, err = f.GetCellValue("Sheet1", fmt.Sprintf("B%d", n+1))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		// Shot Bid
		var shotInfo BGShotAsset
		for num, info := range line {
			if num == 0 {
				shotInfo.Name = shotName
				shotInfo.Manday = make(map[string]float64)
				continue
			}
			if typ == "drama" {
				if num == 1 {
					shotInfo.Note = episode
					continue
				}
			}
			if info != "" {
				bid, err := strconv.ParseFloat(info, 64)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if typ == "movie" {
					shotInfo.Manday[rcp.BGTaskList[num-1]] = bid
					rcp.TotalBid[rcp.BGTaskList[num-1]] += bid
				} else {
					shotInfo.Manday[rcp.BGTaskList[num-2]] = bid
					rcp.TotalBid[rcp.BGTaskList[num-2]] += bid
				}
			}
			if num == len(line)-1 {
				rcp.BGShotList = append(rcp.BGShotList, shotInfo)
			}
		}
	}

	// json 파일 생성
	jsonData, _ := json.Marshal(rcp.BGShotList)
	_ = ioutil.WriteFile(path+"/shot.json", jsonData, 0644)

	err = TEMPLATES.ExecuteTemplate(w, "uploadshot-check", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleUploadShotSubmitFunc 함수는 Shot 정보를 DB에 Upload 하는 함수이다.
func handleUploadShotSubmitFunc(w http.ResponseWriter, r *http.Request) {
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

	q := r.URL.Query()
	id := q.Get("id")
	bgtype := q.Get("bgtype")
	typ := q.Get("type")

	bgp, err := getBGProjectFunc(client, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	bgTypeData := bgp.TypeData[bgtype]

	var taskList []string // 예산 팀세팅의 샷 관련 태스크 리스트
	bgTeamSetting := bgTypeData.TeamSetting
	for _, deptList := range bgTeamSetting.Departments {
		for _, dept := range deptList {
			if dept.Type { // Asset 관련 팀인 경우 패스
				continue
			}
			for _, part := range dept.Parts {
				for _, task := range part.Tasks {
					if !checkStringInListFunc(task, taskList) {
						taskList = append(taskList, task)
					}
				}
			}
		}
	}
	sort.Strings(taskList)

	// json 파일에서 shot 정보 가져오기
	path := os.TempDir() + "/budget/" + token.ID + "/shot/shot.json" // json 파일 경로
	jsonData, err := ioutil.ReadFile(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var shotData []BGShotAsset
	json.Unmarshal(jsonData, &shotData)

	bgTypeData.ShotList = []BGShotAsset{}
	totalBid := make(map[string]float64)
	totalBidByEpisode := make(map[string]map[string]float64) // 드라마 프로젝트의 경우 에피소드별 TotalBid

	var shotList []string              // 샷 코드 리스트
	errShot := make(map[string]string) // 에러가 있는 샷 정보
	for _, shot := range shotData {
		if checkStringInListFunc(shot.Name, shotList) {
			errShot[shot.Name] += "해당 샷코드가 중복되었습니다."
			continue
		}
		shotList = append(shotList, shot.Name)

		var shotInfo BGShotAsset
		shotInfo.Name = shot.Name
		if typ == "drama" {
			shotInfo.Note = shot.Note
			if totalBidByEpisode[shot.Note] == nil {
				totalBidByEpisode[shot.Note] = make(map[string]float64)
			}
		}
		shotInfo.Manday = make(map[string]float64)
		for task, bid := range shot.Manday {
			if !checkStringInListFunc(task, taskList) {
				errShot[shot.Name] += fmt.Sprintf("태스크 %s가 예산 팀세팅에 등록되지 않았습니다.", task)
				continue
			}
			if bid < 0 {
				errShot[shot.Name] += fmt.Sprintf("태스크 %s Bid가 0보다 작습니다.", task)
				continue
			}
			shotInfo.Manday[task] = bid
			totalBid[task] += bid
			if typ == "drama" {
				totalBidByEpisode[shot.Note][task] += bid
			}
		}

		if shotInfo.Manday != nil {
			bgTypeData.ShotList = append(bgTypeData.ShotList, shotInfo)
		}
	}

	// 규칙에 맞지 않는 샷 정보가 존재할 경우 샷 업로드 실패 페이지로 이동한다.
	if len(errShot) != 0 {
		type Recipe struct {
			Token    Token
			ShotInfo map[string]string
		}
		rcp := Recipe{
			Token:    token,
			ShotInfo: errShot,
		}

		log := Log{}
		log.UserID = token.ID
		log.CreatedAt = time.Now()
		log.Content = fmt.Sprintf("프로젝트 %s의 %s 샷 정보를 업로드하지 못했습니다.", id, bgtype)

		err = addLogsFunc(client, log)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		err = TEMPLATES.ExecuteTemplate(w, "uploadshot-fail", rcp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		return
	}

	// 샷 정보를 기반으로 인건비를 계산한 후에 태스크별 비용을 정리한다.
	taskCost := make(map[string]float64)
	for task, bid := range totalBid {
		wage, err := averageWageByTeamsFunc(task, bgTeamSetting.Teams[task])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		taskCost[task] = math.Round(wage * bid)
	}

	// 현재 예산안의 팀세팅 정보를 기반으로 본부별 샷 관련 태스크를 정리한다.
	taskByHead := make(map[string]map[string][]string)
	for head, deptList := range bgTypeData.TeamSetting.Departments {
		taskByHead[head] = make(map[string][]string)
		for _, dept := range deptList {
			if !dept.Type {
				for _, parts := range dept.Parts {
					taskByHead[head][dept.Name] = append(taskByHead[head][dept.Name], parts.Tasks...)
				}
			}
		}
	}

	// 본부별로 정리된 태스크에 따라서 비용을 정리한다.
	costByHead := make(map[string]map[string]float64)
	for head, tasksByDept := range taskByHead {
		costByHead[head] = make(map[string]float64)
		for dept, tasks := range tasksByDept {
			for task, cost := range taskCost {
				if checkStringInListFunc(task, tasks) {
					costByHead[head][dept] += cost
				}
			}
		}
	}

	// 기존 예산안이 있는지 확인후 비교하여 예산안 비용을 암호화하여 업데이트한다.
	bglsList := []BGLaborCost{}
	for head, costByDept := range costByHead {
		bgls := BGLaborCost{}
		bgls.Headquarter = head
		if bgTypeData.LaborCosts != nil { // 기존 예산안이 있는 경우
			for _, ls := range bgTypeData.LaborCosts {
				if ls.Headquarter == head { // 기존에 저장된 본부의 비용의 경우
					bgls = ls
					break
				}
			}
		}

		// 이미 부서별 비용이 있는 경우 -> 샷 관련 부서 비용만 초기화
		if bgls.DepartmentCost == nil {
			bgls.DepartmentCost = make(map[string]string)
		} else {
			var shotDept []string
			for _, parts := range bgTypeData.TeamSetting.Departments { // 현재 팀세팅의 Shot 관련 부서
				for _, part := range parts {
					if !part.Type {
						shotDept = append(shotDept, part.Name)
					}
				}
			}
			for dept := range bgls.DepartmentCost {
				if checkStringInListFunc(dept, shotDept) {
					delete(bgls.DepartmentCost, dept)
				}
			}
		}

		// 계산된 부서별 비용 암호화하여 저장
		for dept, cost := range costByDept {
			encryptedCost, err := encryptAES256Func(strconv.Itoa(int(cost)))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			bgls.DepartmentCost[dept] = encryptedCost
		}
		bglsList = append(bglsList, bgls)
	}
	bgTypeData.LaborCosts = bglsList

	// 드라마인 경우 에피소드별 비용을 계산한다.
	bgTypeData.EpisodeCost = make(map[string]string)
	if typ == "drama" {
		for ep, tBid := range totalBidByEpisode {
			costByEpisode := 0.0
			for task, bid := range tBid {
				wage, err := averageWageByTeamsFunc(task, bgTeamSetting.Teams[task])
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				costByEpisode += math.Round(wage * bid)
			}
			encryptedCost, err := encryptAES256Func(strconv.Itoa(int(costByEpisode)))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			bgTypeData.EpisodeCost[ep] = encryptedCost
		}
	}

	// 샷 정보와 비용 정보를 저장한 후 프로젝트를 업데이트한다.
	bgp.TypeData[bgtype] = bgTypeData
	err = setBGProjectFunc(client, bgp, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 프로젝트 예산안 샷 업로드 성공 페이지로 리다이렉트
	log := Log{}
	log.UserID = token.ID
	log.CreatedAt = time.Now()
	log.Content = fmt.Sprintf("프로젝트 %s의 %s 샷 정보를 업로드하였습니다.", id, bgtype)

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("uploadshot-success?id=%s&bgtype=%s", id, bgtype), http.StatusSeeOther)
}

// handleUploadShotSuccessFunc 함수는 예산 프로젝트 예산안 샷 정보 업로드를 성공했다는 페이지를 연다.
func handleUploadShotSuccessFunc(w http.ResponseWriter, r *http.Request) {
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
		Token  Token
		ID     string
		BGType string
	}
	rcp := Recipe{}
	rcp.Token = token
	q := r.URL.Query()
	rcp.ID = q.Get("id")
	rcp.BGType = q.Get("bgtype")

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "uploadshot-success", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleDetailShotFunc 함수는 Shot, Asset 관리 페이지에서 Shot Detail 페이지를 여는 함수이다.
func handleDetailShotFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// member 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < MemberLevel {
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

	q := r.URL.Query()
	id := q.Get("id")
	bgtype := q.Get("bgtype")
	typ := q.Get("type")

	bgp, err := getBGProjectFunc(client, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	bgTypeData := bgp.TypeData[bgtype]

	type Recipe struct {
		Token    Token
		ID       string
		Name     string
		BGType   string
		Type     string
		ShotList []BGShotAsset
		TaskList []string
		TotalBid map[string]float64
	}
	rcp := Recipe{}
	rcp.Token = token
	rcp.ID = id
	rcp.Name = bgp.Name
	rcp.BGType = bgtype
	rcp.Type = typ
	rcp.ShotList = bgTypeData.ShotList
	rcp.TotalBid = make(map[string]float64)
	for _, shot := range rcp.ShotList {
		for task, bid := range shot.Manday {
			if !checkStringInListFunc(task, rcp.TaskList) {
				rcp.TaskList = append(rcp.TaskList, task)
			}
			rcp.TotalBid[task] += bid
		}
	}
	sort.Strings(rcp.TaskList)

	err = genDetailShotExcelFunc(id, bgtype, typ, rcp.ShotList, rcp.TaskList, rcp.TotalBid, token.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = TEMPLATES.ExecuteTemplate(w, "detail-shot", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// genDetailShotExcelFunc 함수는 샷 디테일 정보 엑셀 파일을 생성하는 함수이다.
func genDetailShotExcelFunc(id string, bgtype string, typ string, shotList []BGShotAsset, taskList []string, totalBid map[string]float64, userID string) error {
	path := os.TempDir() + "/budget/" + userID + "/detailShot/"
	excelFileName := fmt.Sprintf("detailShot_%s_%s.xlsx", id, bgtype)

	err := createFolderFunc(path)
	if err != nil {
		return err
	}
	err = delAllFilesFunc(path) // 경로에 있는 모든 파일 삭제
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
	f.SetCellValue(sheet, "A1", "Shot Name")
	if typ == "drama" {
		f.SetCellValue(sheet, "B1", "Episode")
	}
	tpos := ""
	for n, task := range taskList {
		if typ == "movie" {
			tpos, err = excelize.CoordinatesToCellName(n+2, 1)
			if err != nil {
				return err
			}
		} else {
			tpos, err = excelize.CoordinatesToCellName(n+3, 1)
			if err != nil {
				return err
			}
		}
		f.SetCellValue(sheet, tpos, task)
	}
	f.SetColWidth(sheet, "A", strings.Split(tpos, "1")[0], 20)
	f.SetColWidth(sheet, "A", "A", 25)
	f.SetRowHeight(sheet, 1, 25)

	// 데이터 입력
	pos := ""
	for i, shot := range shotList {
		// Shot Name
		pos, err = excelize.CoordinatesToCellName(1, i+2) // ex) pos = "A2"
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, shot.Name)

		// Episode - 드라마인 경우
		if typ == "drama" {
			pos, err = excelize.CoordinatesToCellName(2, i+2) // ex) pos = "B2"
			if err != nil {
				return err
			}
			f.SetCellValue(sheet, pos, shot.Note)
		}

		// Shot Bid
		for n, task := range taskList {
			if typ == "movie" {
				pos, err = excelize.CoordinatesToCellName(n+2, i+2) // ex) pos = "B2", "C2" ...
				if err != nil {
					return err
				}
			} else {
				pos, err = excelize.CoordinatesToCellName(n+3, i+2) // ex) pos = "B2", "C2" ...
				if err != nil {
					return err
				}
			}
			if shot.Manday[task] == 0.0 {
				f.SetCellValue(sheet, pos, "")
			} else {
				f.SetCellValue(sheet, pos, shot.Manday[task])
			}
		}

		f.SetRowHeight(sheet, i+2, 20)
	}

	// 합계 데이터 입력
	topos, err := excelize.CoordinatesToCellName(1, len(shotList)+2)
	if err != nil {
		return err
	}
	f.SetCellValue(sheet, topos, "Total")
	if typ == "drama" {
		tompos, err := excelize.CoordinatesToCellName(2, len(shotList)+2)
		if err != nil {
			return err
		}
		f.MergeCell(sheet, topos, tompos)
	}
	for n, task := range taskList {
		if typ == "movie" {
			tpos, err = excelize.CoordinatesToCellName(n+2, len(shotList)+2)
			if err != nil {
				return err
			}
		} else {
			tpos, err = excelize.CoordinatesToCellName(n+3, len(shotList)+2)
			if err != nil {
				return err
			}
		}
		if totalBid[task] == 0.0 {
			f.SetCellValue(sheet, tpos, "")
		} else {
			f.SetCellValue(sheet, tpos, totalBid[task])
		}
	}
	f.SetRowHeight(sheet, len(shotList)+2, 20)

	f.SetCellStyle(sheet, "A1", tpos, style)
	f.SetCellStyle(sheet, topos, tpos, totalStyle)

	// 엑셀 파일 저장
	err = f.SaveAs(path + "/" + excelFileName)
	if err != nil {
		return err
	}

	return nil
}

// handleExportDetailShotFunc 함수는 임시 폴더에 저장된 샷 정보 엑셀 파일을 다운로드하는 함수이다.
func handleExportDetailShotFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// admin 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < AdminLevel {
		http.Redirect(w, r, "/invalidaccess", http.StatusSeeOther)
		return
	}

	// Post 메소드가 아니면 에러
	if r.Method != http.MethodPost {
		http.Error(w, "Post Method Only", http.StatusMethodNotAllowed)
		return
	}

	path := os.TempDir() + "/budget/" + token.ID + "/detailShot"

	// path에 있는 파일들을 가져온다.
	fileInfo, err := ioutil.ReadDir(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	q := r.URL.Query()
	id := q.Get("id")
	bgtype := q.Get("bgtype")
	typ := q.Get("type")

	// path에 파일의 개수가 하나가 아니면 엑셀 파일이 다시 생성되도록 리다이렉트
	if len(fileInfo) != 1 {
		http.Redirect(w, r, fmt.Sprintf("/detail-shot?id=%s&bgtype=%s&type=%s", id, bgtype, typ), http.StatusSeeOther)
		return
	}

	// 파일의 확장자가 xlsx가 아니면 엑셀 파일이 다시 생성되도록 리다이렉트
	ext := filepath.Ext(fileInfo[0].Name())
	if ext != ".xlsx" {
		http.Redirect(w, r, fmt.Sprintf("/detail-shot?id=%s&bgtype=%s&type=%s", id, bgtype, typ), http.StatusSeeOther)
		return
	}

	filename := strings.Split(strings.Split(fileInfo[0].Name(), ".")[0], "_")

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

	log := Log{
		UserID:    token.ID,
		CreatedAt: time.Now(),
		Content:   fmt.Sprintf("프로젝트 %s %s의 세부 샷 정보를 다운로드하였습니다..", filename[1], filename[2]),
	}

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Disposition", fmt.Sprintf("Attachment; filename=%s", fileInfo[0].Name()))
	http.ServeFile(w, r, path+"/"+fileInfo[0].Name())
}

// handleUploadAssetFunc 함수는 예산 프로젝트의 예산안에 해당하는 어셋 엑셀 파일을 임포트하는 페이지를 불러오는 함수이다.
func handleUploadAssetFunc(w http.ResponseWriter, r *http.Request) {
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
	id := q.Get("id")
	bgtype := q.Get("bgtype")
	if id == "" || bgtype == "" {
		http.Redirect(w, r, "/shotasset", http.StatusSeeOther)
		return
	}

	type Recipe struct {
		Token  Token
		ID     string
		BGType string
	}
	rcp := Recipe{
		Token:  token,
		ID:     id,
		BGType: bgtype,
	}

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "uploadasset", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleAssetExcelDownloadFunc 함수는 Asset 업로드를 위한 엑셀 파일의 템플릿을 생성하여 다운로드하는 함수이다.
func handleAssetExcelDownloadFunc(w http.ResponseWriter, r *http.Request) {
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

	q := r.URL.Query()
	id := q.Get("id")
	bgtype := q.Get("bgtype")

	bgp, err := getBGProjectFunc(client, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var taskList []string
	bgTeamSetting := bgp.TypeData[bgtype].TeamSetting
	for _, deptList := range bgTeamSetting.Departments {
		for _, dept := range deptList {
			if dept.Type { // Asset 관련 태스크만 포함
				for _, part := range dept.Parts {
					for _, task := range part.Tasks {
						if !checkStringInListFunc(task, taskList) {
							taskList = append(taskList, task)
						}
					}
				}
			}
		}
	}
	sort.Strings(taskList)

	// 엑셀 파일 생성
	f := excelize.NewFile()
	sheet := "Sheet1"
	index := f.NewSheet(sheet)
	f.SetActiveSheet(index)

	// 스타일
	style, err := f.NewStyle(`{"alignment":{"horizontal":"center","vertical":"center","wrap_text":true}, "font":{"bold":true}}`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	f.SetCellValue(sheet, "A1", "분류")
	f.SetCellValue(sheet, "B1", "명칭")
	f.SetCellValue(sheet, "C1", "VFX_note")
	f.SetCellValue(sheet, "D1", "Shot")
	pos := ""
	for i, task := range taskList {
		pos, err = excelize.CoordinatesToCellName(i+5, 1)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		f.SetCellValue(sheet, pos, task)
	}
	f.SetColWidth(sheet, "A", strings.Split(pos, "1")[0], 15)
	f.SetColWidth(sheet, "B", "B", 30)
	f.SetColWidth(sheet, "C", "C", 45)
	f.SetRowHeight(sheet, 1, 25)
	f.SetCellStyle(sheet, "A1", pos, style)

	tempDir, err := ioutil.TempDir("", "excel")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tempDir) // 다운로드 후 임시 파일 삭제

	filename := "asset_template.xlsx"
	err = f.SaveAs(tempDir + "/" + filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Disposition", fmt.Sprintf("Attachment; filename=%s", filename))
	http.ServeFile(w, r, tempDir+"/"+filename)
}

// handleUploadAssetExcelFunc 함수는 업로드한 엑셀 파일을 임시 폴더로 복사하는 함수이다,
func handleUploadAssetExcelFunc(w http.ResponseWriter, r *http.Request) {
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

	path := os.TempDir() + "/budget/" + token.ID + "/asset/" // 엑셀 파일을 저장할 임시 폴더 경로
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

// handleAssetExcelSubmitFunc 함수는 엑셀 파일에서 데이터를 가져와 체크하는 페이지로 전달하는 함수이다.
func handleAssetExcelSubmitFunc(w http.ResponseWriter, r *http.Request) {
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

	path := os.TempDir() + "/budget/" + token.ID + "/asset"

	q := r.URL.Query()
	id := q.Get("id")
	bgtype := q.Get("bgtype")

	// path에 있는 파일들을 가져온다.
	fileInfo, err := ioutil.ReadDir(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// path에 파일의 개수가 하나가 아니면 엑셀 파일을 다시 업로드하도록 리다이렉트
	if len(fileInfo) != 1 {
		http.Redirect(w, r, fmt.Sprintf("/uploadasset?id=%s&bgtype=%s", id, bgtype), http.StatusSeeOther)
		return
	}

	// 파일의 확장자가 xlsx가 아니면 엑셀 파일을 다시 업로드하도록 리다이렉트
	ext := filepath.Ext(fileInfo[0].Name())
	if ext != ".xlsx" {
		http.Redirect(w, r, fmt.Sprintf("/uploadasset?id=%s&bgtype=%s", id, bgtype), http.StatusSeeOther)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(excelRows) == 0 {
		http.Error(w, "엑셀 파일의 Sheet1 값이 비어있습니다", http.StatusBadRequest)
		return
	}

	type Recipe struct {
		Token       Token
		ID          string
		BGType      string
		BGAssetList []BGShotAsset
		BGTaskList  []string
		TotalBid    map[string]float64
	}
	rcp := Recipe{}
	rcp.Token = token
	rcp.ID = id
	rcp.BGType = bgtype
	rcp.TotalBid = make(map[string]float64)
	for n, line := range excelRows {
		// 첫번째 줄
		if n == 0 {
			if len(line) < 5 {
				http.Error(w, "엑셀 파일의 Cell 개수는 5개 이상이어야 합니다", http.StatusBadRequest)
				return
			}
			rcp.BGTaskList = line[4:]
			continue
		}

		if len(line) == 0 { // 행을 추가하거나 삭제하면 데이터가 없는 행도 가져오게된다.
			break
		}

		// 어셋 명칭이 적혀있는지 확인
		assetName, err := f.GetCellValue("Sheet1", fmt.Sprintf("B%d", n+1))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if assetName == "" {
			continue
		}

		// Asset 기입 정보 확인 후 저장
		var assetInfo BGShotAsset
		assetInfo.Manday = make(map[string]float64)
		for num, info := range line {
			if num == 0 { // 어셋 분류 입력
				assetInfo.Class = info
				continue
			}
			if num == 1 { // 어셋 명칭 입력
				assetInfo.Name = info
				continue
			}
			if num == 2 { // 어셋 노트 입력
				assetInfo.Note = info
				continue
			}
			if num == 3 { // 어셋 샷 개수 입력
				if info != "" {
					shot, err := strconv.Atoi(info)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					assetInfo.Shot = shot
				}
				continue
			}

			// 어셋 Bid 정보 입력
			if info != "" {
				bid, err := strconv.ParseFloat(info, 64)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				assetInfo.Manday[rcp.BGTaskList[num-4]] = bid
				rcp.TotalBid[rcp.BGTaskList[num-4]] += bid
			}
			if num == len(line)-1 {
				rcp.BGAssetList = append(rcp.BGAssetList, assetInfo)
			}

		}
	}

	// json 파일 생성
	jsonData, _ := json.Marshal(rcp.BGAssetList)
	_ = ioutil.WriteFile(path+"/asset.json", jsonData, 0644)

	err = TEMPLATES.ExecuteTemplate(w, "uploadasset-check", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleUploadAssetSubmitFunc 함수는 Asset 정보를 DB에 Upload 하는 함수이다.
func handleUploadAssetSubmitFunc(w http.ResponseWriter, r *http.Request) {
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

	q := r.URL.Query()
	id := q.Get("id")
	bgtype := q.Get("bgtype")

	bgp, err := getBGProjectFunc(client, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	bgTypeData := bgp.TypeData[bgtype]

	var taskList []string
	bgTeamSetting := bgTypeData.TeamSetting
	for _, deptList := range bgTeamSetting.Departments {
		for _, dept := range deptList {
			if dept.Type { // Asset 관련 태스크만 포함
				for _, part := range dept.Parts {
					for _, task := range part.Tasks {
						if !checkStringInListFunc(task, taskList) {
							taskList = append(taskList, task)
						}
					}
				}
			}
		}
	}
	sort.Strings(taskList)

	// json 파일에서 asset 정보 가져오기
	path := os.TempDir() + "/budget/" + token.ID + "/asset/asset.json" // json 파일 경로
	jsonData, err := ioutil.ReadFile(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var assetData []BGShotAsset
	json.Unmarshal(jsonData, &assetData)

	bgTypeData.AssetList = []BGShotAsset{}
	totalBid := make(map[string]float64)

	type ErrAsset struct {
		Class string
		Name  string
		Note  string
		Shot  int
		Err   string
	}
	var errAssetList []ErrAsset

	for _, asset := range assetData {
		var assetInfo BGShotAsset
		assetInfo.Class = asset.Class
		assetInfo.Name = asset.Name
		assetInfo.Note = asset.Note
		assetInfo.Shot = asset.Shot
		assetInfo.Manday = make(map[string]float64)
		for task, bid := range asset.Manday {
			if !checkStringInListFunc(task, taskList) {
				errAsset := ErrAsset{
					Class: asset.Class,
					Name:  asset.Name,
					Note:  asset.Note,
					Shot:  asset.Shot,
					Err:   fmt.Sprintf("태스크 %s가 예산 팀세팅에 등록되지 않았습니다.", task),
				}
				errAssetList = append(errAssetList, errAsset)
				continue
			}
			if bid < 0 {
				errAsset := ErrAsset{
					Class: asset.Class,
					Name:  asset.Name,
					Note:  asset.Note,
					Shot:  asset.Shot,
					Err:   fmt.Sprintf("태스크 %s Bid가 0보다 작습니다.", task),
				}
				errAssetList = append(errAssetList, errAsset)
				continue
			}
			assetInfo.Manday[task] = bid
			totalBid[task] += bid
		}

		if assetInfo.Manday != nil {
			bgTypeData.AssetList = append(bgTypeData.AssetList, assetInfo)
		}
	}

	// 규칙에 맞지 않는 어셋 정보가 존재할 경우 어셋 업로듣 실패 페이지로 이동한다.
	if len(errAssetList) != 0 {
		type Recipe struct {
			Token        Token
			ErrAssetList []ErrAsset
		}
		rcp := Recipe{
			Token:        token,
			ErrAssetList: errAssetList,
		}

		log := Log{}
		log.UserID = token.ID
		log.CreatedAt = time.Now()
		log.Content = fmt.Sprintf("프로젝트 %s의 %s 어셋 정보를 업로드하지 못했습니다.", id, bgtype)

		err = addLogsFunc(client, log)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		err = TEMPLATES.ExecuteTemplate(w, "uploadasset-fail", rcp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		return
	}

	// 어셋 정보를 기반으로 인건비를 계산한 후에 태스크별 비용을 정리한다.
	taskCost := make(map[string]float64)
	for task, bid := range totalBid {
		wage, err := averageWageByTeamsFunc(task, bgTeamSetting.Teams[task])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		taskCost[task] = math.Round(wage * bid)
	}

	// 현재 예산안의 팀세팅 정보를 기반으로 본부별 샷 관련 태스크를 정리한다.
	taskByHead := make(map[string]map[string][]string)
	for head, deptList := range bgTypeData.TeamSetting.Departments {
		taskByHead[head] = make(map[string][]string)
		for _, dept := range deptList {
			if dept.Type {
				for _, parts := range dept.Parts {
					taskByHead[head][dept.Name] = append(taskByHead[head][dept.Name], parts.Tasks...)
				}
			}
		}
	}

	// 본부별로 정리된 태스크에 따라서 비용을 정리한다.
	costByHead := make(map[string]map[string]float64)
	for head, tasksByDept := range taskByHead {
		costByHead[head] = make(map[string]float64)
		for dept, tasks := range tasksByDept {
			for task, cost := range taskCost {
				if checkStringInListFunc(task, tasks) {
					costByHead[head][dept] += cost
				}
			}
		}
	}

	// 기존 예산안이 있는지 확인후 비교하여 예산안 비용을 암호화하여 업데이트한다.
	bglsList := []BGLaborCost{}
	for head, costByDept := range costByHead {
		bgls := BGLaborCost{}
		bgls.Headquarter = head
		if bgTypeData.LaborCosts != nil { // 기존 예산안이 있는 경우
			for _, ls := range bgTypeData.LaborCosts {
				if ls.Headquarter == head { // 기존에 저장된 본부의 비용의 경우
					bgls = ls
					break
				}
			}
		}

		// 이미 부서별 비용이 있는 경우 -> 샷 관련 부서 비용만 초기화
		if bgls.DepartmentCost == nil {
			bgls.DepartmentCost = make(map[string]string)
		} else {
			var assetDept []string
			for _, parts := range bgTypeData.TeamSetting.Departments { // 현재 팀세팅의 Asset 관련 부서
				for _, part := range parts {
					if part.Type {
						assetDept = append(assetDept, part.Name)
					}
				}
			}
			for dept := range bgls.DepartmentCost {
				if checkStringInListFunc(dept, assetDept) {
					delete(bgls.DepartmentCost, dept)
				}
			}
		}

		// 계산된 부서별 비용 암호화하여 저장
		for dept, cost := range costByDept {
			encryptedCost, err := encryptAES256Func(strconv.Itoa(int(cost)))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			bgls.DepartmentCost[dept] = encryptedCost
		}
		bglsList = append(bglsList, bgls)
	}
	bgTypeData.LaborCosts = bglsList

	// 어셋 정보를 저장한 후 프로젝트를 업데이트한다.
	bgp.TypeData[bgtype] = bgTypeData
	err = setBGProjectFunc(client, bgp, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 프로젝트 예산안 샷 업로드 성공 페이지로 리다이렉트
	log := Log{}
	log.UserID = token.ID
	log.CreatedAt = time.Now()
	log.Content = fmt.Sprintf("프로젝트 %s의 %s 어셋 정보를 업로드하였습니다.", id, bgtype)

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("uploadasset-success?id=%s&bgtype=%s", id, bgtype), http.StatusSeeOther)
}

// handleUploadAssetSuccessFunc 함수는 예산 프로젝트 예산안 어셋 정보 업로드를 성공했다는 페이지를 연다.
func handleUploadAssetSuccessFunc(w http.ResponseWriter, r *http.Request) {
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
		Token  Token
		ID     string
		BGType string
	}
	rcp := Recipe{}
	rcp.Token = token
	q := r.URL.Query()
	rcp.ID = q.Get("id")
	rcp.BGType = q.Get("bgtype")

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "uploadasset-success", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleDetailAssetFunc 함수는 Shot, Asset 관리 페이지에서 Asset Detail 페이지를 여는 함수이다.
func handleDetailAssetFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// member 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < MemberLevel {
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

	q := r.URL.Query()
	id := q.Get("id")
	bgtype := q.Get("bgtype")

	bgp, err := getBGProjectFunc(client, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	bgTypeData := bgp.TypeData[bgtype]

	type Recipe struct {
		Token     Token
		ID        string
		Name      string
		BGType    string
		AssetList []BGShotAsset
		TaskList  []string
		TotalBid  map[string]float64
	}
	rcp := Recipe{}
	rcp.Token = token
	rcp.ID = id
	rcp.Name = bgp.Name
	rcp.BGType = bgtype
	rcp.AssetList = bgTypeData.AssetList
	rcp.TotalBid = make(map[string]float64)
	for _, asset := range rcp.AssetList {
		for task, bid := range asset.Manday {
			if !checkStringInListFunc(task, rcp.TaskList) {
				rcp.TaskList = append(rcp.TaskList, task)
			}
			rcp.TotalBid[task] += bid
		}
	}
	sort.Strings(rcp.TaskList)

	err = genDetailAssetExcelFunc(id, bgtype, rcp.AssetList, rcp.TaskList, rcp.TotalBid, token.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = TEMPLATES.ExecuteTemplate(w, "detail-asset", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// genDetailAssetExcelFunc 함수는 어셋 디테일 정보 엑셀 파일을 생성하는 함수이다.
func genDetailAssetExcelFunc(id string, bgtype string, assetList []BGShotAsset, taskList []string, totalBid map[string]float64, userID string) error {
	path := os.TempDir() + "/budget/" + userID + "/detailAsset/"
	excelFileName := fmt.Sprintf("detailAsset_%s_%s.xlsx", id, bgtype)

	err := createFolderFunc(path)
	if err != nil {
		return err
	}
	err = delAllFilesFunc(path) // 경로에 있는 모든 파일 삭제
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
	f.SetCellValue(sheet, "A1", "분류")
	f.SetCellValue(sheet, "B1", "명칭")
	f.SetCellValue(sheet, "C1", "VFX_note")
	f.SetCellValue(sheet, "D1", "Shot")
	tpos := ""
	for n, task := range taskList {
		tpos, err = excelize.CoordinatesToCellName(n+5, 1)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, tpos, task)
	}
	f.SetColWidth(sheet, "A", strings.Split(tpos, "1")[0], 15)
	f.SetColWidth(sheet, "B", "B", 40)
	f.SetColWidth(sheet, "C", "C", 50)
	f.SetColWidth(sheet, "D", "D", 15)
	f.SetRowHeight(sheet, 1, 25)

	// 데이터 입력
	pos := ""
	for i, asset := range assetList {
		// Asset 분류
		pos, err = excelize.CoordinatesToCellName(1, i+2)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, asset.Class)

		// Asset 명칭
		pos, err = excelize.CoordinatesToCellName(2, i+2)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, asset.Name)

		// Asset Note
		pos, err = excelize.CoordinatesToCellName(3, i+2)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, asset.Note)

		// Asset Shot
		pos, err = excelize.CoordinatesToCellName(4, i+2)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, asset.Shot)

		// Asset Bid
		for n, task := range taskList {
			pos, err = excelize.CoordinatesToCellName(n+5, i+2)
			if err != nil {
				return err
			}
			if asset.Manday[task] == 0.0 {
				f.SetCellValue(sheet, pos, "")
			} else {
				f.SetCellValue(sheet, pos, asset.Manday[task])
			}
		}

		f.SetRowHeight(sheet, i+2, 20)
	}

	// 합계 데이터 입력
	topos, err := excelize.CoordinatesToCellName(1, len(assetList)+2)
	if err != nil {
		return err
	}
	f.SetCellValue(sheet, topos, "Total")
	tompos, err := excelize.CoordinatesToCellName(4, len(assetList)+2)
	if err != nil {
		return err
	}
	f.MergeCell(sheet, topos, tompos)

	for n, task := range taskList {
		tpos, err = excelize.CoordinatesToCellName(n+5, len(assetList)+2)
		if err != nil {
			return err
		}
		if totalBid[task] == 0.0 {
			f.SetCellValue(sheet, tpos, "")
		} else {
			f.SetCellValue(sheet, tpos, totalBid[task])
		}
	}
	f.SetRowHeight(sheet, len(assetList)+2, 20)

	f.SetCellStyle(sheet, "A1", tpos, style)
	f.SetCellStyle(sheet, topos, tpos, totalStyle)

	// 엑셀 파일 저장
	err = f.SaveAs(path + "/" + excelFileName)
	if err != nil {
		return err
	}

	return nil
}

// handleExportDetailAssetFunc 함수는 임시 폴더에 저장된 어셋 정보 엑셀 파일을 다운로드하는 함수이다.
func handleExportDetailAssetFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// admin 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < AdminLevel {
		http.Redirect(w, r, "/invalidaccess", http.StatusSeeOther)
		return
	}

	// Post 메소드가 아니면 에러ㄴ
	if r.Method != http.MethodPost {
		http.Error(w, "Post Method Only", http.StatusMethodNotAllowed)
		return
	}

	path := os.TempDir() + "/budget/" + token.ID + "/detailAsset"

	// path에 있는 파일들을 가져온다.
	fileInfo, err := ioutil.ReadDir(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	q := r.URL.Query()
	id := q.Get("id")
	bgtype := q.Get("bgtype")

	// path에 파일의 개수가 하나가 아니면 엑셀 파일이 다시 생성되도록 리다이렉트
	if len(fileInfo) != 1 {
		http.Redirect(w, r, fmt.Sprintf("/detail-asset?id=%s&bgtype=%s", id, bgtype), http.StatusSeeOther)
		return
	}

	// 파일의 확장자가 xlsx가 아니면 엑셀 파일이 다시 생성되도록 리다이렉트
	ext := filepath.Ext(fileInfo[0].Name())
	if ext != ".xlsx" {
		http.Redirect(w, r, fmt.Sprintf("/detail-asset?id=%s&bgtype=%s", id, bgtype), http.StatusSeeOther)
		return
	}

	filename := strings.Split(strings.Split(fileInfo[0].Name(), ".")[0], "_")

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

	log := Log{
		UserID:    token.ID,
		CreatedAt: time.Now(),
		Content:   fmt.Sprintf("프로젝트 %s %s의 세부 어셋 정보를 다운로드하였습니다..", filename[1], filename[2]),
	}

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Disposition", fmt.Sprintf("Attachment; filename=%s", fileInfo[0].Name()))
	http.ServeFile(w, r, path+"/"+fileInfo[0].Name())
}
