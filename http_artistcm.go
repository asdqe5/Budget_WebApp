// 프로젝트 결산 프로그램
//
// Description : http CM 아티스트 관련 스크립트

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// handleArtistsCMFunc 함수는 CM 아티스트 관리 페이지를 띄우는 함수이다.
func handleArtistsCMFunc(w http.ResponseWriter, r *http.Request) {
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
		Token      Token
		User       User
		Year       string
		Resination bool // 퇴사자 토글 옵션값
		Artists    []Artist
	}
	rcp := Recipe{}
	rcp.Token = token
	rcp.User, err = getUserFunc(client, token.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	year := r.FormValue("year")
	if year == "" { // year 값이 없으면 올해로 검색
		y, _, _ := time.Now().Date()
		year = strconv.Itoa(y)

	}
	rcp.Year = year
	sort := r.FormValue("sort")
	resination := r.FormValue("resination")
	if resination == "true" {
		rcp.Resination = true
	} else {
		rcp.Resination = false
	}

	if rcp.Resination { // 퇴사자도 함께 보여야하기 때문에 모든 CM 아티스트들을 가져온다.
		rcp.Artists, err = getCMArtistsFunc(client, sort, rcp.Year)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else { // 퇴사자를 제외한 모든 CM 아티스트들을 가져온다.
		rcp.Artists, err = getCMArtistsWithoutRetireeFunc(client, sort, rcp.Year)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	err = genArtistsCMExcelFunc(rcp.Artists, token.ID) // 엑셀 파일 미리 생성
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "artists-cm", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

// handleEditArtistCMFunc 함수는 CM 아티스트 정보를 수정할 수 있는 페이지를 연다. edit 버튼을 누를 때 실행된다.
func handleEditArtistCMFunc(w http.ResponseWriter, r *http.Request) {
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

	q := r.URL.Query()
	id := q.Get("id")
	if id == "" {
		http.Error(w, "URL에 id를 입력해주세요", http.StatusBadRequest)
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
		Token  Token
		User   User
		Artist Artist
	}
	rcp := Recipe{}
	rcp.Token = token
	rcp.User, err = getUserFunc(client, token.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rcp.Artist, err = getArtistFunc(client, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 연봉 복호화
	for key, val := range rcp.Artist.Salary {
		result, err := decryptAES256Func(val)
		if err != nil {
			return
		}
		rcp.Artist.Salary[key] = result
	}

	// Edit 페이지를 띄운다
	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "edit-artistcm", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleEditArtistCMSubmitFunc 함수는 CM 아티스트 정보를 수정하는 페이지에서 UPDATE 버튼을 누르면 작동하는 함수다.
func handleEditArtistCMSubmitFunc(w http.ResponseWriter, r *http.Request) {
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
	id := r.FormValue("id")

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

	artist, err := getArtistFunc(client, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	artist.Team = r.FormValue("team")
	artist.Name = r.FormValue("name")
	artist.StartDay = r.FormValue("startday")
	artist.EndDay = r.FormValue("endday")

	salary := r.FormValue("salary")
	if salary != "" {
		if !regexSalary.MatchString(salary) {
			http.Error(w, "salary가 2019:2400,2020:2400 형식이 아닙니다", http.StatusBadRequest)
			return
		}
	}
	artist.Salary, _ = stringToMapFunc(salary)
	// 연봉 암호화
	for key, value := range artist.Salary {
		encrypted, err := encryptAES256Func(value)
		artist.Salary[key] = encrypted
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// 동일 연도 연봉 변경일이 정해진 경우
	if r.FormValue("changedate") != "" {
		artist.Changed = true
		artist.ChangedSalary = make(map[string]string)
		encrypted, err := encryptAES256Func(r.FormValue("changesalary"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		artist.ChangedSalary[r.FormValue("changedate")] = encrypted

		// 입사일, 퇴사일, 동일 연도 연봉 변경일 체크
		startDate, err := time.Parse("2006-01-02", artist.StartDay) // 입사일 Date
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		changeDate, err := time.Parse("2006-01-02", r.FormValue("changedate")) // 연봉 변경 날짜
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !startDate.Before(changeDate) { // 연봉 변경일이 잘못 입력된 경우
			http.Error(w, "동일 연도 연봉 변경일이 잘못 입력되었습니다.", http.StatusInternalServerError)
			return
		}
		if artist.EndDay != "" { // 퇴사일이 정해진 경우
			endDate, err := time.Parse("2006-01-02", artist.EndDay) // 퇴사일 Date
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if !startDate.Before(changeDate) || changeDate.After(endDate) { // 연봉 변경일이 잘못 입력된 경우
				http.Error(w, "동일 연도 연봉 변경일이 잘못 입력되었습니다.", http.StatusInternalServerError)
				return
			}
		}
	} else {
		artist.Changed = false
		artist.ChangedSalary = nil
	}
	err = artist.CheckErrorFunc()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = setArtistFunc(client, artist)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log := Log{}
	log.UserID = token.ID
	log.CreatedAt = time.Now()
	log.Content = fmt.Sprintf("CM 아티스트 ID %s의 정보가 수정되었습니다.", id)

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/editartistcm-success?id=%s", id), http.StatusSeeOther)
}

// handleEditArtistCMSuccessFunc 함수는 CM 아티스트 정보 수정을 성공했다는 페이지를 연다.
func handleEditArtistCMSuccessFunc(w http.ResponseWriter, r *http.Request) {
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

	q := r.URL.Query()
	id := q.Get("id")
	if id == "" {
		http.Error(w, "URL에 id를 입력해주세요", http.StatusBadRequest)
		return
	}

	type Recipe struct {
		Token
		ID string // 아티스트 ID
	}
	rcp := Recipe{
		Token: token,
		ID:    id,
	}

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "editartistcm-success", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleUpdateArtistsCMFunc 함수는 CM 아티스트를 업데이트하는 페이지를 띄운다
func handleUpdateArtistsCMFunc(w http.ResponseWriter, r *http.Request) {
	// 로그인 상태인지 확인
	token, err := getTokenFromHeaderFunc(w, r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// admin 레벨 미만이면 invalidaccess 페이지로 리다이렉트
	if token.AccessLevel < AdminLevel {
		http.Redirect(w, r, "invalidaccess", http.StatusSeeOther)
		return
	}

	type Recipe struct {
		Token
	}
	rcp := Recipe{}
	rcp.Token = token

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "updateartists-cm", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleArtistsCMExcelDownloadFunc 함수는 CM 아티스트 정보를 입력할 엑셀 파일의 템플릿을 생성하여 다운로드하는 함수이다.
func handleArtistsCMExcelDownloadFunc(w http.ResponseWriter, r *http.Request) {
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

	if r.Method != http.MethodGet {
		http.Error(w, "Get Only", http.StatusMethodNotAllowed)
		return
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
	f.SetCellValue(sheet, "A1", "CM ID\n(숫자만 입력)")
	f.SetCellValue(sheet, "B1", "팀(Team)")
	f.SetCellValue(sheet, "C1", "이름\n(한글 및 영어만 입력)")
	f.SetCellValue(sheet, "D1", "입사일\n(ex. 2020-05-01)")
	f.SetCellValue(sheet, "E1", "퇴사일\n(ex. 2020-09-01)")
	f.SetCellValue(sheet, "F1", "연봉\n(ex. 2019:2400,2020:2400)")
	f.SetCellValue(sheet, "G1", "동일 연도 연봉 변경\n(ex. 2020-04-01:1920\n변경일과 변경 전 연봉 입력)")
	f.SetColWidth(sheet, "A", "A", 20)
	f.SetColWidth(sheet, "B", "B", 15)
	f.SetColWidth(sheet, "C", "G", 30)
	f.SetColWidth(sheet, "D", "E", 20)
	f.SetRowHeight(sheet, 1, 45)
	f.SetCellStyle(sheet, "A1", "G1", style)

	tempDir, err := ioutil.TempDir("", "excel")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tempDir) // 다운로드 후 임시 파일 삭제

	filename := "cm_artist_template.xlsx"
	err = f.SaveAs(tempDir + "/" + filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Disposition", fmt.Sprintf("Attachment; filename=%s", filename))
	http.ServeFile(w, r, tempDir+"/"+filename)
}

// handleArtistsCMExcelSubmitFunc 함수는 엑셀 파일에서 데이터를 가져와 체크하는 페이지로 전달하는 함수이다.
func handleArtistsCMExcelSubmitFunc(w http.ResponseWriter, r *http.Request) {
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

	if r.Method != http.MethodGet {
		http.Error(w, "Get Method Only", http.StatusMethodNotAllowed)
		return
	}

	path := os.TempDir() + "/budget/" + token.ID + "/artists"

	// path에 있는 파일들을 가져온다.
	fileInfo, err := ioutil.ReadDir(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// path에 파일의 개수가 하나가 아니면 엑셀 파일을 다시 업로드하도록 리다이렉트
	if len(fileInfo) != 1 {
		http.Redirect(w, r, "/updateartists-cm", http.StatusSeeOther)
		return
	}

	// 파일의 확장자가 xlsx가 아니면 엑셀 파일을 다시 업로드하도록 리다이렉트
	ext := filepath.Ext(fileInfo[0].Name())
	if ext != ".xlsx" {
		http.Redirect(w, r, "/updateartists-cm", http.StatusSeeOther)
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

	var results []Artist
	for n, line := range excelRows {
		// 첫번째 줄
		if n == 0 {
			if len(line) != 7 {
				http.Error(w, "엑셀 파일의 Cell 개수는 7개이어야 합니다", http.StatusBadRequest)
				return
			}
			continue
		}

		if len(line) == 0 { // 행을 추가하거나 삭제하면 데이터가 없는 행도 가져오게된다.
			break
		}

		artist := Artist{}

		// CM ID
		cmID, err := f.GetCellValue("Sheet1", fmt.Sprintf("A%d", n+1))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if cmID == "" {
			continue
		}
		artist.ID = changeToCMIDFunc(cmID)

		// 중복되는 데이터이면 continue
		found := false
		for _, r := range results {
			if r.ID == artist.ID {
				found = true
			}
		}
		if found {
			continue
		}

		// 입력받은 셀의 CM 아티스트 정보를 가져온다.
		artist.Dept = "cm"
		artist.Team, err = f.GetCellValue("Sheet1", fmt.Sprintf("B%d", n+1))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if artist.Team == "" {
			continue
		}
		artist.Name, err = f.GetCellValue("Sheet1", fmt.Sprintf("C%d", n+1))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if artist.Name == "" {
			continue
		}

		// 입사일
		artist.StartDay, err = f.GetCellValue("Sheet1", fmt.Sprintf("D%d", n+1))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// 퇴사일
		artist.EndDay, err = f.GetCellValue("Sheet1", fmt.Sprintf("E%d", n+1))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// 연봉
		salary, err := f.GetCellValue("Sheet1", fmt.Sprintf("F%d", n+1))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		artist.Salary, _ = stringToMapFunc(salary)

		// 동일 연도 연봉 변경
		change, err := f.GetCellValue("Sheet1", fmt.Sprintf("G%d", n+1))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		artist.ChangedSalary, _ = stringToMapFunc(change)

		results = append(results, artist)
	}

	type Recipe struct {
		Artists []Artist
		Token
	}
	rcp := Recipe{
		Artists: results,
		Token:   token,
	}

	err = TEMPLATES.ExecuteTemplate(w, "updateartistscm-check", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleUpdateArtistsCMSubmitFunc 함수는 DB의 CM 아티스트를 업데이트한 후 모두 완료되면 /updateartistscm-success로 리다이렉트하는 함수이다.
func handleUpdateArtistsCMSubmitFunc(w http.ResponseWriter, r *http.Request) {
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

	artistNum, err := strconv.Atoi(r.FormValue("artistNum"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	errArtist := make(map[string]string)
	for i := 0; i < artistNum; i++ {
		a := Artist{}
		a.ID = r.FormValue(fmt.Sprintf("id%d", i))
		a.Dept = r.FormValue(fmt.Sprintf("dept%d", i))
		a.Team = r.FormValue(fmt.Sprintf("team%d", i))
		a.Name = r.FormValue(fmt.Sprintf("name%d", i))
		a.StartDay = r.FormValue(fmt.Sprintf("startday%d", i))
		a.EndDay = r.FormValue(fmt.Sprintf("endday%d", i))

		// 연봉 정보 체크
		salary := r.FormValue(fmt.Sprintf("salary%d", i))
		if salary != "" {
			if !regexSalary.MatchString(salary) {
				errArtist[a.ID] = "연봉이 2019:2400,2020:2400 형식이 아닙니다"
				continue
			}
		}
		a.Salary, _ = stringToMapFunc(salary)

		// 연봉 암호화
		for key, value := range a.Salary {
			encrypt, err := encryptAES256Func(value)
			a.Salary[key] = encrypt
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		// 동일 연도 연봉 변경 정보 체크
		change := r.FormValue(fmt.Sprintf("change%d", i))
		var changeDate string
		if change != "" {
			if !regexChangeSalary.MatchString(change) {
				errArtist[a.ID] = "동일 연도 연봉 변경이 2020-04-01:1920 형식이 아닙니다"
				continue
			}
			a.ChangedSalary, _ = stringToMapFunc(change)
			for key, value := range a.ChangedSalary {
				encrypted, err := encryptAES256Func(value)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				changeDate = key
				a.ChangedSalary[key] = encrypted
			}
			a.Changed = true
		}

		err = a.CheckErrorFunc()
		if err != nil {
			errArtist[a.ID] = err.Error()
			continue
		}

		// 입사일, 퇴사일, 동일 연도 연봉 변경일 체크
		startDate, err := time.Parse("2006-01-02", a.StartDay) // 입사일 Date
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if a.EndDay != "" { // 퇴사일이 설정되어 있으면
			endDate, err := time.Parse("2006-01-02", a.EndDay) // 퇴사일 Date
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if endDate.Before(startDate) { // 퇴사일이 입사일 전인 경우 -> 에러 처리
				errArtist[a.ID] = "퇴사일이 잘못 입력되었습니다"
				continue
			}
			if a.Changed { // 아티스트 동일 연도 연봉이 변경된 경우
				changeDate, err := time.Parse("2006-01-02", changeDate) // 연봉 변경 날짜
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if !startDate.Before(changeDate) || changeDate.After(endDate) { // 연봉 변경일이 잘못 입력된 경우
					errArtist[a.ID] = "동일 연도 연봉 변경일이 잘못 입력되었습니다."
					continue
				}
			}
		}
		if a.Changed { // 아티스트 동일 연도 연봉이 변경된 경우
			changeDate, err := time.Parse("2006-01-02", changeDate) // 연봉 변경 날짜
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if !startDate.Before(changeDate) { // 연봉 변경일이 잘못 입력된 경우
				errArtist[a.ID] = "동일 연도 연봉 변경일이 잘못 입력되었습니다."
				continue
			}
		}

		err = a.CheckErrorFunc()
		if err != nil {
			errArtist[a.ID] = err.Error()
			continue
		}

		err = updateArtistFunc(client, a) // DB에 아티스트 추가
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	log := Log{}
	log.UserID = token.ID
	log.CreatedAt = time.Now()

	// DB에 추가하지 못한 아티스트가 존재할 경우 에러 페이지로 이동
	if len(errArtist) != 0 {
		type Recipe struct {
			Artists map[string]string
			Token
		}
		rcp := Recipe{
			Artists: errArtist,
			Token:   token,
		}

		log.Content = "CM 아티스트 업데이트에 실패했습니다."

		err = addLogsFunc(client, log)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		err = TEMPLATES.ExecuteTemplate(w, "updateartistscm-fail", rcp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		log.Content = "CM 아티스트 업데이트를 완료했습니다."

		err = addLogsFunc(client, log)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "updateartistscm-success", http.StatusSeeOther)
	}
}

// handleUpdateArtistsCMSuccessFunc 함수는 CM팀 아티스트 정보 업데이트를 성공했다는 페이지를 연다.
func handleUpdateArtistsCMSuccessFunc(w http.ResponseWriter, r *http.Request) {
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

	type Recipe struct {
		Token
	}
	rcp := Recipe{}
	rcp.Token = token

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "updateartistscm-success", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// genArtistsCMExcelFunc 함수는 CM팀의 아티스트 데이터를 엑셀 파일로 생성하는 함수이다.
func genArtistsCMExcelFunc(artists []Artist, userID string) error {
	path := os.TempDir() + "/budget/" + userID + "/artistscm"
	fileName := "cm_artists.xlsx"

	err := createFolderFunc(path)
	if err != nil {
		return err
	}
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

	// 제목 입력
	f.SetCellValue(sheet, "A1", "CM ID")
	f.SetCellValue(sheet, "B1", "팀")
	f.SetCellValue(sheet, "C1", "이름")
	f.SetCellValue(sheet, "D1", "입사일")
	f.SetCellValue(sheet, "E1", "퇴사일")
	f.SetCellValue(sheet, "F1", "연봉")
	f.SetCellValue(sheet, "G1", "동일 연도 연봉 변경\n(변경 전 연봉 입력)")
	f.SetColWidth(sheet, "A", "G", 15)
	f.SetColWidth(sheet, "F", "F", 40)
	f.SetColWidth(sheet, "G", "G", 40)
	f.SetRowHeight(sheet, 1, 35)
	f.SetCellStyle(sheet, "A1", "G1", style)

	// 데이터 입력
	for n, a := range artists {
		// CM ID
		aPos, err := excelize.CoordinatesToCellName(1, n+2) // ex) pos = "A2"
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, aPos, strings.Trim(a.ID, "cm"))

		// Team
		pos, err := excelize.CoordinatesToCellName(2, n+2) // ex) pos = "B2"
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, a.Team)

		// Name
		pos, err = excelize.CoordinatesToCellName(3, n+2) // ex) pos = "C2"
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, a.Name)

		// 입사일
		pos, err = excelize.CoordinatesToCellName(4, n+2) // ex) pos = "D2"
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, a.StartDay)

		// 퇴사일
		pos, err = excelize.CoordinatesToCellName(5, n+2) // ex) ePos = "E2"
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, pos, a.EndDay)

		// Salary
		pos, err = excelize.CoordinatesToCellName(6, n+2) // ex) pos = "F2"
		if err != nil {
			return err
		}
		// 연봉 복호화
		salary := make(map[string]string)
		for key, val := range a.Salary {
			result, err := decryptAES256Func(val)
			if err != nil {
				return err
			}
			salary[key] = result
		}
		f.SetCellValue(sheet, pos, mapToStringFunc(salary))

		// 동일 연도 연병 변경 정보
		gPos, err := excelize.CoordinatesToCellName(7, n+2) // ex) hPos = "G2"
		if err != nil {
			return err
		}
		// 연봉 복호화
		change := make(map[string]string)
		for key, val := range a.ChangedSalary {
			result, err := decryptAES256Func(val)
			if err != nil {
				return err
			}
			change[key] = result
		}
		f.SetCellValue(sheet, gPos, mapToStringFunc(change))

		f.SetCellStyle(sheet, aPos, gPos, style)
	}

	// 엑셀 파일 저장
	err = f.SaveAs(path + "/" + fileName)
	if err != nil {
		return err
	}

	return nil
}

// handleExportArtistsCMFunc 함수는 임시 폴더에 저장된 엑셀 파일을 다운로드하는 함수이다.
func handleExportArtistsCMFunc(w http.ResponseWriter, r *http.Request) {
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

	path := os.TempDir() + "/budget/" + token.ID + "/artistscm"

	// path에 있는 파일들을 가져온다.
	fileInfo, err := ioutil.ReadDir(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 파일의 확장자가 xlsx가 아니면 엑셀 파일이 다시 생성되도록 리다이렉트
	ext := filepath.Ext(fileInfo[0].Name())
	if ext != ".xlsx" {
		http.Redirect(w, r, "artists-cm", http.StatusSeeOther)
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

	log := Log{
		UserID:    token.ID,
		CreatedAt: time.Now(),
		Content:   "CM 아티스트 페이지에서 아티스트 데이터를 다운로드하였습니다.",
	}

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Disposition", fmt.Sprintf("Attachment; filename=%s", fileInfo[0].Name()))
	http.ServeFile(w, r, path+"/"+fileInfo[0].Name())
}
