// 프로젝트 결산 프로그램
//
// Description : http 결산 인건비 관련 스크립트

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

// handleSMDetailLaborCostFunc 함수는 세부 인건비 페이지를 여는 함수이다.
func handleSMDetailLaborCostFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// Admin 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < AdminLevel {
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
		Token                 Token
		User                  User
		Date                  string                       // 2020-12 형태의 날짜
		Type                  string                       // VFX, CM
		Projects              []string                     // 타임로그가 존재하는 프로젝트
		Artist                []Artist                     // 존재하는 아티스트
		NoneArtists           []string                     // DB에 존재하지 않는 아티스트 목록
		DetailLaborCost       map[string]map[string]string // 아티스트별 인건비
		TotalArtistLaborCost  map[string]string            // 아티스트별 총 인건비
		TotalProjectLaborCost map[string]string            // 프로젝트별 총 인건비
		TotalLaborCost        string                       // 총 인건비
	}

	rcp := Recipe{}
	rcp.Token = token
	rcp.User, err = getUserFunc(client, token.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	date := r.FormValue("date")
	if date == "" { // year 값이 없으면 올해로 검색
		y, m, _ := time.Now().Date()
		date = fmt.Sprintf("%04d-%02d", y, int(m))
	}
	rcp.Date = date
	typ := r.FormValue("type")
	if typ == "" {
		typ = "vfx"
	}
	rcp.Type = typ

	// VFX 아티스트별 인건비 계산
	year, err := strconv.Atoi(strings.Split(rcp.Date, "-")[0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	month, err := strconv.Atoi(strings.Split(rcp.Date, "-")[1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var timelogs []Timelog
	if rcp.Type == "vfx" {
		timelogs, err = getTimelogOfTheMonthVFXFunc(client, year, month)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		timelogs, err = getTimelogOfTheMonthCMFunc(client, year, month)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// 가져온 타임로그 정보를 기반으로 USER ID에 해당하는 타임로그들을 map 형식으로 정리한다.
	timelogsMap := make(map[string][]Timelog)
	for _, value := range timelogs {
		if !checkStringInListFunc(value.Project, rcp.Projects) {
			rcp.Projects = append(rcp.Projects, value.Project)
		}
		timelogsMap[value.UserID] = append(timelogsMap[value.UserID], value)
	}
	sort.Strings(rcp.Projects)

	// 아티스트 별로 정리된 타임로그를 기반으로 프로젝트별 인건비를 계산한다.
	rcp.DetailLaborCost = make(map[string]map[string]string)
	detailLaborCost := make(map[string]map[string]int)
	totalArtistLaborCost := make(map[string]int)
	totalProjectLaborCost := make(map[string]int)
	totalLaborCost := 0
	for key, value := range timelogsMap {
		artist, err := getArtistFunc(client, key) // DB에서 아티스트를 검색한다.
		if err != nil {
			if err == mongo.ErrNoDocuments { // DB에 없을 경우 errArtistID에 추가한다.
				if !checkStringInListFunc(key, rcp.NoneArtists) {
					rcp.NoneArtists = append(rcp.NoneArtists, key)
					continue
				}
				continue
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		rcp.Artist = append(rcp.Artist, artist)

		// 아티스트가 존재하면 해당 아티스트의 인건비를 계산한다.
		detailLaborCost[key] = make(map[string]int)

		// VFX 인건비
		if rcp.Type == "vfx" {
			// 아티스트의 프로젝트별 타임로그 계산 후 비율에 맞게 인건비 계산
			for _, t := range value {
				duration := math.Round(t.Duration/60*10) / 10
				hourlyWage := 0.0
				if artist.Salary[strconv.Itoa(year)] != "" {
					hourlyWage, err = hourlyWageFunc(artist, year, month) // 시급 계산
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}
				laborCost := duration * hourlyWage
				detailLaborCost[key][t.Project] = int(math.Round(laborCost))
				totalProjectLaborCost[t.Project] += int(math.Round(laborCost))
			}
		} else { // CM 인건비
			for _, t := range value {
				duration := math.Round(t.Duration/60*10) / 10
				hourlyWage := 0.0
				if artist.Salary[strconv.Itoa(year)] != "" {
					hourlyWage, err = hourlyWageFunc(artist, year, month) // 시급 계산
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}
				laborCost := duration * hourlyWage
				detailLaborCost[key][t.Project] = int(math.Round(laborCost))
				totalProjectLaborCost[t.Project] += int(math.Round(laborCost))
			}
		}

		// 아티스트의 프로젝트별 인건비 암호화
		rcp.DetailLaborCost[key] = make(map[string]string)
		for project, cost := range detailLaborCost[key] {
			rcp.DetailLaborCost[key][project], err = encryptAES256Func(strconv.Itoa(cost))
			totalArtistLaborCost[key] += cost
			totalLaborCost += cost
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
	sort.Slice(rcp.Artist, func(i, j int) bool {
		return rcp.Artist[i].Name < rcp.Artist[j].Name
	})

	// 합계 인건비 암호화
	rcp.TotalArtistLaborCost = make(map[string]string)
	rcp.TotalProjectLaborCost = make(map[string]string)
	for key, value := range totalArtistLaborCost {
		rcp.TotalArtistLaborCost[key], err = encryptAES256Func(strconv.Itoa(value))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	for key, value := range totalProjectLaborCost {
		rcp.TotalProjectLaborCost[key], err = encryptAES256Func(strconv.Itoa(value))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	rcp.TotalLaborCost, err = encryptAES256Func(strconv.Itoa(totalLaborCost))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 타입에 맞게 엑셀 파일 생성
	err = genSMDetailLaborCostExcelFunc(strings.ToUpper(rcp.Type)+"_"+rcp.Date, rcp.Artist, rcp.Projects, rcp.DetailLaborCost, rcp.TotalArtistLaborCost, rcp.TotalProjectLaborCost, rcp.TotalLaborCost, token.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "smdetail-laborcost", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// genSMDetailLaborCostExcelFunc 함수는 세부 인건비 엑셀 파일을 만드는 함수이다.
func genSMDetailLaborCostExcelFunc(fileName string, artists []Artist, projects []string, detail map[string]map[string]string, totalArtist map[string]string, totalProject map[string]string, total string, userID string) error {
	path := os.TempDir() + "/budget/" + userID + "/smdetaillaborcost/"
	excelFileName := fmt.Sprintf("smdetaillaborcost_%s.xlsx", fileName)

	err := createFolderFunc(path)
	if err != nil {
		return err
	}
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
	numberStyle, err := f.NewStyle(`{"alignment":{"horizontal":"right","vertical":"center","wrap_text":true}, "number_format": 3}`)
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
	totalNumStyle, err := f.NewStyle(
		`
		{"alignment":{"horizontal":"right","vertical":"center","wrap_text":true},
		"font":{"bold":true},
		"fill":{"type":"pattern","color":["#FFC000"],"pattern":1},
		"number_format": 3}
		`)
	if err != nil {
		return err
	}

	// 제목 입력
	f.SetCellValue(sheet, "A1", "ID")
	f.MergeCell(sheet, "A1", "A2")
	f.SetCellValue(sheet, "B1", "이름")
	f.MergeCell(sheet, "B1", "B2")
	f.SetCellValue(sheet, "C1", "프로젝트")
	pos, err := excelize.CoordinatesToCellName(len(projects)+2, 1)
	if err != nil {
		return err
	}
	f.MergeCell(sheet, "C1", pos)
	for i, p := range projects {
		projectName, err := getNameOfProjectFunc(client, p)
		if err != nil {
			if err == mongo.ErrNoDocuments { // DB에 없는 프로젝트라면 ID로 보여준다.
				projectName = p
			} else {
				return err
			}
		}
		pos, err = excelize.CoordinatesToCellName(i+3, 2)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, projectName)
	}
	tapos, err := excelize.CoordinatesToCellName(len(projects)+3, 1)
	if err != nil {
		return err
	}
	f.SetCellValue(sheet, tapos, "Total")
	tampos, err := excelize.CoordinatesToCellName(len(projects)+3, 2)
	if err != nil {
		return err
	}
	f.MergeCell(sheet, tapos, tampos)
	f.SetColWidth(sheet, "A", strings.Split(tapos, "1")[0], 20)
	f.SetColWidth(sheet, "A", "B", 10)
	f.SetRowHeight(sheet, 1, 20)
	f.SetRowHeight(sheet, 2, 40)

	// 데이터 입력
	for i, artist := range artists {
		// 아티스트 ID
		pos, err = excelize.CoordinatesToCellName(1, i+3)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, artist.ID)

		// 아티스트 이름
		pos, err = excelize.CoordinatesToCellName(2, i+3)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, artist.Name)

		// 프로젝트별 세부 인건비
		for j, project := range projects {
			pos, err = excelize.CoordinatesToCellName(3+j, i+3)
			if err != nil {
				return err
			}
			detailLaborCost, err := decryptAES256Func(detail[artist.ID][project])
			if err != nil {
				return err
			}
			detailLaborCostInt := 0
			if detailLaborCost != "" {
				detailLaborCostInt, err = strconv.Atoi(detailLaborCost)
				if err != nil {
					return err
				}
			}
			f.SetCellValue(sheet, pos, detailLaborCostInt)
		}

		// 아티스트별 디테일 Total
		pos, err = excelize.CoordinatesToCellName(len(projects)+3, i+3)
		if err != nil {
			return err
		}
		artistTotal, err := decryptAES256Func(totalArtist[artist.ID])
		if err != nil {
			return err
		}
		artistTotalInt := 0
		if artistTotal != "" {
			artistTotalInt, err = strconv.Atoi(artistTotal)
			if err != nil {
				return err
			}
		}
		f.SetCellValue(sheet, pos, artistTotalInt)

		f.SetRowHeight(sheet, i+3, 20)
	}

	// 프로젝트별 Total 입력
	tppos, err := excelize.CoordinatesToCellName(1, len(artists)+3)
	if err != nil {
		return err
	}
	f.SetCellValue(sheet, tppos, "Total")
	tpmpos, err := excelize.CoordinatesToCellName(2, len(artists)+3)
	if err != nil {
		return err
	}
	f.MergeCell(sheet, tppos, tpmpos)

	for i, project := range projects {
		pos, err = excelize.CoordinatesToCellName(i+3, len(artists)+3)
		if err != nil {
			return err
		}
		projectTotal, err := decryptAES256Func(totalProject[project])
		if err != nil {
			return err
		}
		projectTotalInt := 0
		if projectTotal != "" {
			projectTotalInt, err = strconv.Atoi(projectTotal)
			if err != nil {
				return err
			}
		}
		f.SetCellValue(sheet, pos, projectTotalInt)
	}
	// 총 Total 입력
	pos, err = excelize.CoordinatesToCellName(len(projects)+3, len(artists)+3)
	if err != nil {
		return err
	}
	t, err := decryptAES256Func(total)
	if err != nil {
		return err
	}
	totalInt := 0
	if t != "" {
		totalInt, err = strconv.Atoi(t)
		if err != nil {
			return err
		}
	}
	f.SetCellValue(sheet, pos, totalInt)

	f.SetRowHeight(sheet, len(artists)+3, 20)

	f.SetCellStyle(sheet, "A1", pos, style)
	f.SetCellStyle(sheet, "C3", pos, numberStyle)
	f.SetCellStyle(sheet, tapos, tampos, totalStyle)
	f.SetCellStyle(sheet, tppos, tpmpos, totalStyle)
	tapos, err = excelize.CoordinatesToCellName(len(projects)+3, 3)
	if err != nil {
		return err
	}
	tppos, err = excelize.CoordinatesToCellName(3, len(artists)+3)
	if err != nil {
		return err
	}
	f.SetCellStyle(sheet, tapos, pos, totalNumStyle)
	f.SetCellStyle(sheet, tppos, pos, totalNumStyle)

	// 엑셀 파일 저장
	err = f.SaveAs(path + "/" + excelFileName)
	if err != nil {
		return err
	}

	return nil
}

// handleExportSMDetailLaborCostFunc 함수는 임시 폴더에 저장된 엑셀 파일을 다운로드하는 함수이다.
func handleExportSMDetailLaborCostFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// admin 레벨 미만이면 invalideaccess 페이지로 리다이렉트
	if token.AccessLevel < AdminLevel {
		http.Redirect(w, r, "/invalidaccess", http.StatusSeeOther)
		return
	}

	// Post 메소드가 아니면 에러
	if r.Method != http.MethodPost {
		http.Error(w, "Post Method Only", http.StatusMethodNotAllowed)
		return
	}

	path := os.TempDir() + "/budget/" + token.ID + "/smdetaillaborcost"

	// path에 있는 파일들을 가져온다.
	fileInfo, err := ioutil.ReadDir(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 파일의 확장자가 xlsx가 아니면 엑셀 파일이 다시 생성되도록 리다이렉트
	ext := filepath.Ext(fileInfo[0].Name())
	if ext != ".xlsx" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
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
		Content:   fmt.Sprintf("%s 세부 인건비 페이지에서 %s년 %s월의 데이터를 다운로드하였습니다.", filename[1], strings.Split(filename[2], "-")[0], strings.Split(filename[2], "-")[1]),
	}

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Disposition", fmt.Sprintf("Attachment; filename=%s", fileInfo[0].Name()))
	http.ServeFile(w, r, path+"/"+fileInfo[0].Name())
}

// handleSMTotalLaborCostFunc 함수는 Total 인건비 페이지를 여는 함수이다.
func handleSMTotalLaborCostFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// Admin 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < AdminLevel {
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

	type ProjectData struct {
		Name        string            // 프로젝트 이름
		MonthlyCost map[string]string // 월별 인건비
		Total       string            // 프로젝트의 총 인건비
	}

	type Recipe struct {
		Token    Token
		Year     string // 연도
		Dates    []string
		Projects []ProjectData
		DateSum  map[string]string // 월별 인건비 합계
		Total    string            // 인건비 총 합계
	}

	rcp := Recipe{}
	rcp.Token = token
	year := r.FormValue("year")
	if year == "" { // year 값이 없으면 올해로 검색
		y, _, _ := time.Now().Date()
		year = strconv.Itoa(y)
	}
	rcp.Year = year
	dateList := []string{}
	for i := 1; i <= 12; i++ {
		dateList = append(dateList, fmt.Sprintf("%s-%02d", rcp.Year, i))
	}
	rcp.Dates = dateList
	projects, err := getProjectsByYearFunc(client, rcp.Year)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})

	// 프로젝트별, 월별 인건비 합계를 구한다.
	dateSum := make(map[string]int)
	total := 0

	for _, p := range projects {
		var projectData ProjectData
		projectData.Name = p.Name
		projectData.MonthlyCost = make(map[string]string)

		projectDates, err := getDatesFunc(p.StartDate, p.SMEndDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		totalLaborCost := 0 // 프로젝트에 투입된 총 인건비
		for _, d := range rcp.Dates {
			// 프로젝트 기간에 포함되는지 확인한다.
			if !checkStringInListFunc(d, projectDates) {
				continue
			}

			laborCost, err := getMonthlyLaborCostFunc(p, d)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// 프로젝트의 월별 인건비에 암호화하여 넣어준다.
			projectData.MonthlyCost[d], err = encryptAES256Func(strconv.Itoa(laborCost))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			totalLaborCost += laborCost // 프로젝트에 투입된 총 인건비에 더해준다.
			dateSum[d] += laborCost     // 월별로 투입된 총 인건비에 더해준다.
		}

		projectData.Total, err = encryptAES256Func(strconv.Itoa(totalLaborCost))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		rcp.Projects = append(rcp.Projects, projectData)
		total += totalLaborCost
	}

	// 프로젝트별 및 월별 인건비 합계 암호화
	rcp.DateSum = make(map[string]string)
	for key, value := range dateSum {
		rcp.DateSum[key], err = encryptAES256Func(strconv.Itoa(value))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	rcp.Total, err = encryptAES256Func(strconv.Itoa(total))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// ProjectData 자료구조를 인자로 넘길 수 없어서 json 파일을 생성한다.
	path := os.TempDir() + "/budget/" + token.ID + "/smtotallaborcost/" // json으로 바꾼 프로젝트 데이터를 저장할 임시 폴더 경로
	err = createFolderFunc(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonData, _ := json.Marshal(rcp.Projects)
	_ = ioutil.WriteFile(path+"/projects.json", jsonData, 0644)

	// 엑셀 파일 생성
	err = genSMTotalLaborCostExcelFunc(rcp.Dates, rcp.DateSum, rcp.Total, token.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "smtotal-laborcost", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// genSMTotalLaborCostExcelFunc 함수는 결산 인건비 현황의 엑셀파일을 만들어주는 함수이다.
func genSMTotalLaborCostExcelFunc(dates []string, dateSum map[string]string, encryptedTotal string, userID string) error {
	path := os.TempDir() + "/budget/" + userID + "/smtotallaborcost/"
	excelFileName := fmt.Sprintf("smtotallaborcost_%s.xlsx", strings.Split(dates[0], "-")[0])

	// json 파일에서 프로젝트 데이터를 가져온다.
	jsonData, err := ioutil.ReadFile(path + "projects.json")
	if err != nil {
		return err
	}

	type ProjectData struct {
		Name        string            // 프로젝트 이름
		MonthlyCost map[string]string // 월별 인건비
		Total       string            // 프로젝트의 총 인건비
	}

	var projects []ProjectData
	json.Unmarshal(jsonData, &projects)

	err = delAllFilesFunc(path)
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
	numberStyle, err := f.NewStyle(`{"alignment":{"horizontal":"right","vertical":"center","wrap_text":true}, "number_format": 3}`)
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
	totalNumStyle, err := f.NewStyle(
		`
		{"alignment":{"horizontal":"right","vertical":"center","wrap_text":true},
		"font":{"bold":true},
		"fill":{"type":"pattern","color":["#FFC000"],"pattern":1},
		"number_format": 3}
		`)
	if err != nil {
		return err
	}

	// 제목 입력
	f.SetCellValue(sheet, "A1", "프로젝트")
	f.MergeCell(sheet, "A1", "A2")
	f.SetCellValue(sheet, "B1", strings.Split(dates[0], "-")[0]+"년")
	f.MergeCell(sheet, "B1", "M1")
	for i := 1; i <= 12; i++ {
		pos, err := excelize.CoordinatesToCellName(i+1, 2)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, strconv.Itoa(i)+"월")
	}
	f.SetCellValue(sheet, "N1", "Total")
	f.MergeCell(sheet, "N1", "N2")

	f.SetColWidth(sheet, "A", "N", 15)
	f.SetColWidth(sheet, "A", "A", 30)
	f.SetRowHeight(sheet, 1, 25)
	f.SetRowHeight(sheet, 2, 25)

	// 데이터 입력
	pos := ""
	for i, p := range projects {
		// 프로젝트명
		pos, err = excelize.CoordinatesToCellName(1, i+3)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, p.Name)

		// 프로젝트에 해당하는 월별 인건비
		for j, d := range dates {
			pos, err = excelize.CoordinatesToCellName(j+2, i+3)
			if err != nil {
				return err
			}
			laborCost, err := decryptAES256Func(p.MonthlyCost[d])
			if err != nil {
				return err
			}
			laborCostInt := 0
			if laborCost != "" {
				laborCostInt, err = strconv.Atoi(laborCost)
				if err != nil {
					return err
				}
			}
			f.SetCellValue(sheet, pos, laborCostInt)
		}

		// 프로젝트별 합계
		pos, err = excelize.CoordinatesToCellName(14, i+3)
		if err != nil {
			return err
		}
		total, err := decryptAES256Func(p.Total)
		if err != nil {
			return err
		}
		totalInt := 0
		if total != "" {
			totalInt, err = strconv.Atoi(total)
			if err != nil {
				return err
			}
		}
		f.SetCellValue(sheet, pos, totalInt)

		f.SetRowHeight(sheet, i+3, 20)
	}

	tdpos, err := excelize.CoordinatesToCellName(1, len(projects)+3)
	if err != nil {
		return err
	}
	f.SetCellValue(sheet, tdpos, "Total")
	for i, d := range dates {
		pos, err = excelize.CoordinatesToCellName(i+2, len(projects)+3)
		if err != nil {
			return err
		}
		total, err := decryptAES256Func(dateSum[d])
		if err != nil {
			return err
		}
		totalInt := 0
		if total != "" {
			totalInt, err = strconv.Atoi(total)
			if err != nil {
				return err
			}
		}
		f.SetCellValue(sheet, pos, totalInt)
	}

	pos, err = excelize.CoordinatesToCellName(len(dates)+2, len(projects)+3)
	if err != nil {
		return err
	}
	total, err := decryptAES256Func(encryptedTotal)
	if err != nil {
		return err
	}
	totalInt := 0
	if total != "" {
		totalInt, err = strconv.Atoi(total)
		if err != nil {
			return err
		}
	}
	f.SetCellValue(sheet, pos, totalInt)

	f.SetRowHeight(sheet, len(projects)+3, 20)

	f.SetCellStyle(sheet, "A1", pos, style)
	f.SetCellStyle(sheet, "B3", pos, numberStyle)
	f.SetCellStyle(sheet, "N1", "N1", totalStyle)
	f.SetCellStyle(sheet, tdpos, pos, totalNumStyle)
	f.SetCellStyle(sheet, tdpos, tdpos, totalStyle)
	f.SetCellStyle(sheet, "N3", pos, totalNumStyle)

	// 엑셀 파일 저장
	err = f.SaveAs(path + "/" + excelFileName)
	if err != nil {
		return err
	}

	return nil
}

// handleExportSMTotalLaborCostFunc 함수는 임시 폴더에 저장된 엑셀 파일을 다운로드하는 함수이다.
func handleExportSMTotalLaborCostFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// admin 레벨 미만이면 invalideaccess 페이지로 리다이렉트
	if token.AccessLevel < AdminLevel {
		http.Redirect(w, r, "/invalidaccess", http.StatusSeeOther)
		return
	}

	// Post 메소드가 아니면 에러
	if r.Method != http.MethodPost {
		http.Error(w, "Post Method Only", http.StatusMethodNotAllowed)
		return
	}

	path := os.TempDir() + "/budget/" + token.ID + "/smtotallaborcost"

	// path에 있는 파일들을 가져온다.
	fileInfo, err := ioutil.ReadDir(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 파일의 확장자가 xlsx가 아니면 엑셀 파일이 다시 생성되도록 리다이렉트
	ext := filepath.Ext(fileInfo[0].Name())
	if ext != ".xlsx" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
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
		Content:   fmt.Sprintf("전체 인건비 페이지에서 %s년의 데이터를 다운로드하였습니다.", filename[1]),
	}

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Disposition", fmt.Sprintf("Attachment; filename=%s", fileInfo[0].Name()))
	http.ServeFile(w, r, path+"/"+fileInfo[0].Name())
}
