// 프로젝트 결산 프로그램
//
// Description : http 벤더 관련 스크립트

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize/v2"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// handleVendorsFunc 함수는 벤더 관리 페이지를 띄우는 함수이다.
func handleVendorsFunc(w http.ResponseWriter, r *http.Request) {
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
	type Recipe struct {
		Token      Token
		User       User
		Type       string                         // 프로젝트별(project), 벤더별(vendor)
		Status     string                         // 전체, 계약금, 중도금, 잔금
		SearchWord string                         // 검색어
		Vendors    map[string]map[string][]Vendor // Type별로 정리된 Vendor 목록
		IsFinished bool                           // 정산완료 토글 옵션값
	}
	rcp := Recipe{}
	rcp.Token = token
	typ := q.Get("type")
	if typ == "" { // typ 값이 없으면 프로젝트별로 설정
		typ = "project"
	}
	rcp.User, err = getUserFunc(client, token.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rcp.Type = typ
	rcp.Status = q.Get("status")
	rcp.SearchWord = q.Get("searchword")
	isfinished := q.Get("isfinished")
	if isfinished == "true" {
		rcp.IsFinished = true
	} else {
		rcp.IsFinished = false
	}

	// Vendor 검색
	searchWord := "status:" + rcp.Status
	if rcp.SearchWord != "" {
		searchWord = searchWord + " " + rcp.SearchWord
	}
	vendors, err := searchVendorFunc(client, searchWord)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 벤더 정보의 정산 완료된 프로젝트를 확인한다.
	var realVendors []Vendor
	if !rcp.IsFinished {
		for _, v := range vendors {
			project, err := getProjectFunc(client, v.Project)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if project.IsFinished {
				continue
			}
			realVendors = append(realVendors, v)
		}
	} else {
		realVendors = vendors
	}

	// DB에서 가져온 Vendor 정리
	vendorsMap := make(map[string]map[string][]Vendor)
	if rcp.Type == "project" { // 프로젝트별로 벤더 정리
		var projectIDList []string
		for _, v := range realVendors { // Vendor에 존재하는 프로젝트 정리
			pid := fmt.Sprintf("%s-%s", v.ProjectName, v.Project)
			if !checkStringInListFunc(pid, projectIDList) {
				projectIDList = append(projectIDList, pid)
			}
		}
		for _, pid := range projectIDList {
			vendorsMap[pid] = make(map[string][]Vendor)
		}
		for _, v := range realVendors {
			pid := fmt.Sprintf("%s-%s", v.ProjectName, v.Project)
			vendorsMap[pid][v.Name] = append(vendorsMap[pid][v.Name], v)
		}
	} else { // 벤더별로 벤더 정리
		var vendorList []string
		for _, v := range realVendors {
			if !checkStringInListFunc(v.Name, vendorList) {
				vendorList = append(vendorList, v.Name)
			}
		}
		for _, v := range vendorList {
			vendorsMap[v] = make(map[string][]Vendor)
		}
		for _, v := range realVendors {
			pid := fmt.Sprintf("%s-%s", v.ProjectName, v.Project)
			vendorsMap[v.Name][pid] = append(vendorsMap[v.Name][pid], v)
		}
	}
	rcp.Vendors = vendorsMap

	err = genVendorExcelFunc(rcp.Type, rcp.Vendors, token.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "vendors", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleSearchVendorsFunc 함수는 벤더관리 페이지에서 Search 버튼을 눌렀을 때 실행하는 함수이다.
func handleSearchVendorsFunc(w http.ResponseWriter, r *http.Request) {
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

	isfinished := r.FormValue("isfinished")
	if isfinished == "on" {
		isfinished = "true"
	} else {
		isfinished = "false"
	}
	status := r.FormValue("status")
	searchword := r.FormValue("searchword")
	typ := r.FormValue("typeCheckbox1")
	if typ == "true" {
		typ = "project"
	} else {
		typ = "vendor"
	}

	http.Redirect(w, r, fmt.Sprintf("/vendors?type=%s&isfinished=%s&status=%s&searchword=%s", typ, isfinished, status, searchword), http.StatusSeeOther)
}

// genVendorExcelFunc 함수는 벤더 엑셀 파일을 생성하는 함수이다.
func genVendorExcelFunc(typ string, vendors map[string]map[string][]Vendor, userID string) error {
	path := os.TempDir() + "/budget/" + userID + "/vendor/"
	excelFileName := fmt.Sprintf("vendor_by_%s.xlsx", typ)

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
		"fill":{"type":"pattern","color":["#FFFFCC"],"pattern":1},
		"number_format": 3}
		`)
	if err != nil {
		return err
	}

	// 제목 입력
	if typ == "project" {
		f.SetCellValue(sheet, "A1", "프로젝트")
		f.SetCellValue(sheet, "B1", "벤더명")
	} else {
		f.SetCellValue(sheet, "A1", "벤더명")
		f.SetCellValue(sheet, "B1", "프로젝트")
	}
	f.MergeCell(sheet, "A1", "A2")
	f.MergeCell(sheet, "B1", "B2")
	f.SetCellValue(sheet, "C1", "계약일")
	f.MergeCell(sheet, "C1", "C2")
	f.SetCellValue(sheet, "D1", "금액")
	f.MergeCell(sheet, "D1", "G1")
	f.SetCellValue(sheet, "D2", "Total")
	f.SetCellValue(sheet, "E2", "계약금")
	f.SetCellValue(sheet, "F2", "중도금")
	f.SetCellValue(sheet, "G2", "잔금")
	f.SetCellValue(sheet, "H1", "컷수")
	f.MergeCell(sheet, "H1", "H2")
	f.SetCellValue(sheet, "I1", "태스크")
	f.MergeCell(sheet, "I1", "I2")
	f.SetCellValue(sheet, "J1", "컷별단가")
	f.MergeCell(sheet, "J1", "J2")
	f.SetCellValue(sheet, "K1", "정산여부")
	f.MergeCell(sheet, "K1", "M1")
	f.SetCellValue(sheet, "K2", "계약금")
	f.SetCellValue(sheet, "L2", "중도금")
	f.SetCellValue(sheet, "M2", "잔금")

	f.SetColWidth(sheet, "A", "M", 18)
	f.SetColWidth(sheet, "H", "H", 10)
	f.SetRowHeight(sheet, 1, 20)
	f.SetRowHeight(sheet, 2, 20)

	// 데이터 입력
	pos := ""
	mpos := ""
	tpos := ""
	i := 0
	for _, v := range vendors {
		plen := lenOfVendorsMapFunc(v, true)
		pnum := 0
		for _, vendor := range v {
			vlen := lenOfVendorsListFunc(vendor)
			vnum := 0
			for _, data := range vendor {
				mplen := len(data.MediumPlating)

				if mplen == 0 {
					if pnum == 0 {
						// 프로젝트 이름 또는 벤더명
						pos, err = excelize.CoordinatesToCellName(1, i+3)
						if err != nil {
							return err
						}
						mpos, err := excelize.CoordinatesToCellName(1, i+3+plen-1)
						if err != nil {
							return err
						}
						if typ == "project" {
							f.SetCellValue(sheet, pos, data.ProjectName)
						} else {
							f.SetCellValue(sheet, pos, data.Name)
						}
						f.MergeCell(sheet, pos, mpos)
					}
					if vnum == 0 {
						// 프로젝트 이름 또는 벤더명
						pos, err = excelize.CoordinatesToCellName(2, i+3)
						if err != nil {
							return err
						}
						mpos, err := excelize.CoordinatesToCellName(2, i+3+vlen-1)
						if err != nil {
							return err
						}
						if typ == "project" {
							f.SetCellValue(sheet, pos, data.Name)
						} else {
							f.SetCellValue(sheet, pos, data.ProjectName)
						}
						f.MergeCell(sheet, pos, mpos)
					}

					// 계약일
					pos, err = excelize.CoordinatesToCellName(3, i+3)
					if err != nil {
						return err
					}
					f.SetCellValue(sheet, pos, data.Date)

					// Total
					tpos, err = excelize.CoordinatesToCellName(4, i+3)
					if err != nil {
						return err
					}
					expenses, err := decryptAES256Func(data.Expenses)
					if err != nil {
						return err
					}
					expensesInt, err := strconv.Atoi(expenses)
					if err != nil {
						return err
					}
					f.SetCellValue(sheet, tpos, expensesInt)

					// 계약금
					pos, err = excelize.CoordinatesToCellName(5, i+3)
					if err != nil {
						return err
					}
					downpaymentInt := 0
					if data.Downpayment.Expenses != "" {
						downpayment, err := decryptAES256Func(data.Downpayment.Expenses)
						if err != nil {
							return err
						}
						downpaymentInt, err = strconv.Atoi(downpayment)
						if err != nil {
							return err
						}
					}
					f.SetCellValue(sheet, pos, downpaymentInt)

					// 중도금
					pos, err = excelize.CoordinatesToCellName(6, i+3)
					if err != nil {
						return err
					}
					f.SetCellValue(sheet, pos, 0)

					// 잔금
					pos, err = excelize.CoordinatesToCellName(7, i+3)
					if err != nil {
						return err
					}
					balanceInt := 0
					if data.Balance.Expenses != "" {
						balance, err := decryptAES256Func(data.Balance.Expenses)
						if err != nil {
							return err
						}
						balanceInt, err = strconv.Atoi(balance)
						if err != nil {
							return err
						}
					}
					f.SetCellValue(sheet, pos, balanceInt)

					// 컷수
					pos, err = excelize.CoordinatesToCellName(8, i+3)
					if err != nil {
						return err
					}
					f.SetCellValue(sheet, pos, data.Cuts)

					// 태스크
					pos, err = excelize.CoordinatesToCellName(9, i+3)
					if err != nil {
						return err
					}
					f.SetCellValue(sheet, pos, listToStringFunc(data.Tasks, true))

					// 컷별단가
					pos, err = excelize.CoordinatesToCellName(10, i+3)
					if err != nil {
						return err
					}
					if data.Cuts == 0 {
						f.SetCellValue(sheet, pos, 0)
					} else {
						f.SetCellValue(sheet, pos, expensesInt/data.Cuts)
					}

					// 정산 체크
					pos, err = excelize.CoordinatesToCellName(11, i+3)
					if err != nil {
						return err
					}
					if data.Downpayment.Status == true { // 계약금 지급일
						f.SetCellValue(sheet, pos, data.Downpayment.PayedDate)
					}
					mpos, err = excelize.CoordinatesToCellName(13, i+3)
					if err != nil {
						return err
					}
					if data.Balance.Status == true { // 잔금
						f.SetCellValue(sheet, pos, data.Balance.PayedDate)
					}

					// 셀 높이 설정
					f.SetRowHeight(sheet, i+3, 25)

					i++
				} else {
					for n, mp := range data.MediumPlating {
						if pnum == 0 && n == 0 {
							// 프로젝트 이름 또는 벤더명
							pos, err = excelize.CoordinatesToCellName(1, i+3)
							if err != nil {
								return err
							}
							mpos, err := excelize.CoordinatesToCellName(1, i+3+plen-1)
							if err != nil {
								return err
							}
							if typ == "project" {
								f.SetCellValue(sheet, pos, data.ProjectName)
							} else {
								f.SetCellValue(sheet, pos, data.Name)
							}
							f.MergeCell(sheet, pos, mpos)
						}
						if vnum == 0 && n == 0 {
							// 프로젝트 이름 또는 벤더명
							pos, err = excelize.CoordinatesToCellName(2, i+3)
							if err != nil {
								return err
							}
							mpos, err := excelize.CoordinatesToCellName(2, i+3+vlen-1)
							if err != nil {
								return err
							}
							if typ == "project" {
								f.SetCellValue(sheet, pos, data.Name)
							} else {
								f.SetCellValue(sheet, pos, data.ProjectName)
							}
							f.MergeCell(sheet, pos, mpos)
						}

						// Total 비용 계산
						expenses, err := decryptAES256Func(data.Expenses)
						if err != nil {
							return err
						}
						expensesInt, err := strconv.Atoi(expenses)
						if err != nil {
							return err
						}

						if n == 0 {
							// 계약일
							pos, err = excelize.CoordinatesToCellName(3, i+3)
							if err != nil {
								return err
							}
							mpos, err = excelize.CoordinatesToCellName(3, i+3+mplen-1)
							if err != nil {
								return err
							}
							f.SetCellValue(sheet, pos, data.Date)
							f.MergeCell(sheet, pos, mpos)

							// Total
							tpos, err = excelize.CoordinatesToCellName(4, i+3)
							if err != nil {
								return err
							}
							mpos, err = excelize.CoordinatesToCellName(4, i+3+mplen-1)
							if err != nil {
								return err
							}
							f.SetCellValue(sheet, tpos, expensesInt)
							f.MergeCell(sheet, tpos, mpos)

							// 계약금
							pos, err = excelize.CoordinatesToCellName(5, i+3)
							if err != nil {
								return err
							}
							mpos, err = excelize.CoordinatesToCellName(5, i+3+mplen-1)
							if err != nil {
								return err
							}
							downpaymentInt := 0
							if data.Downpayment.Expenses != "" {
								downpayment, err := decryptAES256Func(data.Downpayment.Expenses)
								if err != nil {
									return err
								}
								downpaymentInt, err = strconv.Atoi(downpayment)
								if err != nil {
									return err
								}
							}
							f.SetCellValue(sheet, pos, downpaymentInt)
							f.MergeCell(sheet, pos, mpos)
						}

						// 중도금
						pos, err = excelize.CoordinatesToCellName(6, i+3)
						if err != nil {
							return err
						}
						mediumplating, err := decryptAES256Func(mp.Expenses)
						if err != nil {
							return err
						}
						mediumplatingInt, err := strconv.Atoi(mediumplating)
						if err != nil {
							return err
						}
						f.SetCellValue(sheet, pos, mediumplatingInt)

						if n == 0 {
							// 잔금
							pos, err = excelize.CoordinatesToCellName(7, i+3)
							if err != nil {
								return err
							}
							mpos, err := excelize.CoordinatesToCellName(7, i+3+mplen-1)
							if err != nil {
								return err
							}
							balanceInt := 0
							if data.Balance.Expenses != "" {
								balance, err := decryptAES256Func(data.Balance.Expenses)
								if err != nil {
									return err
								}
								balanceInt, err = strconv.Atoi(balance)
								if err != nil {
									return err
								}
							}
							f.SetCellValue(sheet, pos, balanceInt)
							f.MergeCell(sheet, pos, mpos)

							// 컷수
							pos, err = excelize.CoordinatesToCellName(8, i+3)
							if err != nil {
								return err
							}
							mpos, err = excelize.CoordinatesToCellName(8, i+3+mplen-1)
							if err != nil {
								return err
							}
							f.SetCellValue(sheet, pos, data.Cuts)
							f.MergeCell(sheet, pos, mpos)

							// 태스크
							pos, err = excelize.CoordinatesToCellName(9, i+3)
							if err != nil {
								return err
							}
							mpos, err = excelize.CoordinatesToCellName(9, i+3+mplen-1)
							if err != nil {
								return err
							}
							f.SetCellValue(sheet, pos, listToStringFunc(data.Tasks, true))
							f.MergeCell(sheet, pos, mpos)

							// 컷별단가
							pos, err = excelize.CoordinatesToCellName(10, i+3)
							if err != nil {
								return err
							}
							mpos, err = excelize.CoordinatesToCellName(10, i+3+mplen-1)
							if err != nil {
								return err
							}
							if data.Cuts == 0 {
								f.SetCellValue(sheet, pos, 0)
							} else {
								f.SetCellValue(sheet, pos, expensesInt/data.Cuts)
							}
							f.MergeCell(sheet, pos, mpos)

							// 정산 체크
							pos, err = excelize.CoordinatesToCellName(11, i+3)
							if err != nil {
								return err
							}
							mpos, err = excelize.CoordinatesToCellName(11, i+3+mplen-1)
							if err != nil {
								return err
							}
							if data.Downpayment.Status == true { // 계약금
								f.SetCellValue(sheet, pos, data.Downpayment.PayedDate)
							}
							f.MergeCell(sheet, pos, mpos)
						}
						pos, err = excelize.CoordinatesToCellName(12, i+3)
						if err != nil {
							return err
						}
						if mp.Status == true { // 중도금
							f.SetCellValue(sheet, pos, mp.PayedDate)
						}
						if n == 0 {
							pos, err = excelize.CoordinatesToCellName(13, i+3)
							if err != nil {
								return err
							}
							mpos, err = excelize.CoordinatesToCellName(13, i+3+mplen-1)
							if err != nil {
								return err
							}
							if data.Balance.Status == true { // 잔금
								f.SetCellValue(sheet, pos, data.Balance.PayedDate)
							}
							f.MergeCell(sheet, pos, mpos)
						}

						// 셀 높이 설정
						f.SetRowHeight(sheet, i+3, 25)

						i++
					}
				}
				pnum++
				vnum++
			}
		}
	}

	f.SetCellStyle(sheet, "A1", mpos, style)
	f.SetCellStyle(sheet, "D3", strings.ReplaceAll(mpos, "L", "G"), numberStyle)
	f.SetCellStyle(sheet, "J3", strings.ReplaceAll(mpos, "L", "J"), numberStyle)
	f.SetCellStyle(sheet, "D2", "D2", totalStyle)
	if tpos != "" {
		f.SetCellStyle(sheet, "D3", tpos, totalNumStyle)
	}

	// 엑셀 파일 저장
	err = f.SaveAs(path + "/" + excelFileName)
	if err != nil {
		return err
	}

	return nil
}

// handleExportVendorsFunc 함수는 임시 폴더에 저장된 엑셀 파일을 다운로드하는 함수이다,
func handleExportVendorsFunc(w http.ResponseWriter, r *http.Request) {
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

	path := os.TempDir() + "/budget/" + token.ID + "/vendor"

	// path에 있는 파일들을 가져온다.
	fileInfo, err := ioutil.ReadDir(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// path에 파일의 개수가 하나가 아니면 엑셀 파일이 다시 생성되도록 리다이렉트
	if len(fileInfo) != 1 {
		http.Redirect(w, r, "/vendors", http.StatusSeeOther)
		return
	}

	// 파일의 확장자가 xlsx가 아니면 엑셀 파일이 다시 생성되도록 리다이렉트
	ext := filepath.Ext(fileInfo[0].Name())
	if ext != ".xlsx" {
		http.Redirect(w, r, "/vendors", http.StatusSeeOther)
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
		Content:   "벤더 관리 페이지에서 벤더 데이터를 다운로드하였습니다.",
	}

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Disposition", fmt.Sprintf("Attachment; filename=%s", fileInfo[0].Name()))
	http.ServeFile(w, r, path+"/"+fileInfo[0].Name())
}

// handleAddVendorPageFunc 함수는 URL에 objectID를 붙여서 /add-vendor 페이지로 redirect한다.
func handleAddVendorPageFunc(w http.ResponseWriter, r *http.Request) {
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
	objectID := primitive.NewObjectID().Hex()

	q := r.URL.Query()
	project := q.Get("project")
	if project == "" {
		http.Redirect(w, r, fmt.Sprintf("/addvendor?objectid=%s", objectID), http.StatusSeeOther)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/addvendor?objectid=%s&project=%s", objectID, project), http.StatusSeeOther)
	}
}

// handleAddVendorFunc 함수는 벤더관리 페이지에서 벤더 추가 페이지를 띄우는 함수이다.
func handleAddVendorFunc(w http.ResponseWriter, r *http.Request) {
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

	type Recipe struct {
		Token       Token
		ProjectList []Project
		Project     string
	}

	projects, err := getAllProjectsFunc(client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	q := r.URL.Query()
	project := q.Get("project")
	rcp := Recipe{
		Token:       token,
		ProjectList: projects,
		Project:     project,
	}

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "add-vendor", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleAddVendorSubmitFunc 함수는 벤더 추가 페이지에서 Add 버튼을 눌렀을 때 실행되는 함수이다.
func handleAddVendorSubmitFunc(w http.ResponseWriter, r *http.Request) {
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

	v := Vendor{}
	// 벤더 필수 정보 입력
	objectID, err := GetObjectIDfromRequestHeader(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	v.ID, err = primitive.ObjectIDFromHex(objectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	v.Project = r.FormValue("project")
	project, err := getProjectFunc(client, v.Project)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	v.ProjectName = project.Name
	v.Name = strings.TrimSpace(r.FormValue("name"))
	// 총 비용 암호화
	expenses := r.FormValue("expenses")
	if strings.Contains(expenses, ",") {
		expenses = strings.ReplaceAll(expenses, ",", "")
	}
	encryptExpenses, err := encryptAES256Func(expenses)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	v.Expenses = encryptExpenses
	v.Date = r.FormValue("date")

	// 벤더 부가 정보 입력
	if r.FormValue("cuts") != "" { // 컷정보가 있는 경우
		cuts, err := strconv.Atoi(r.FormValue("cuts"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		v.Cuts = cuts
	}
	if r.FormValue("tasks") != "" { // 태스크 정보가 있는 경우
		tasks := r.FormValue("tasks")
		taskList := stringToListFunc(tasks, ",")
		sort.Strings(taskList)
		v.Tasks = taskList
	}

	// 벤더 비용 정보 입력
	if r.FormValue("downpayment") != "" { // 계약금이 적힌 경우
		downpayment := r.FormValue("downpayment")
		if strings.Contains(downpayment, ",") {
			downpayment = strings.ReplaceAll(downpayment, ",", "")
		}
		encryptDownpayment, err := encryptAES256Func(downpayment)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		v.Downpayment.Expenses = encryptDownpayment
		v.Downpayment.Date = r.FormValue("downpaymentdate")           // 계약금 세금 계산서 발행일
		v.Downpayment.PayedDate = r.FormValue("downpaymentpayeddate") // 계약금 지급일
		if r.FormValue("downpaymentstatus") == "true" {               // 계약금 지출 완료인 경우
			v.Downpayment.Status = true
		} else {
			v.Downpayment.Status = false
		}
	}
	// 중도금 입력 칸의 갯수
	mediumplatingNum := r.FormValue("mediumplatingNum")
	mpNum, err := strconv.Atoi(mediumplatingNum)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	v.MediumPlating = []VendorCost{}
	for num := 0; num < mpNum; num++ { // 중도금 칸의 갯수만큼만 입력
		if r.FormValue(fmt.Sprintf("mediumplating%d", num)) != "" { // 중도금이 적힌 경우
			mp := VendorCost{}
			mediumplating := r.FormValue(fmt.Sprintf("mediumplating%d", num))
			if strings.Contains(mediumplating, ",") {
				mediumplating = strings.ReplaceAll(mediumplating, ",", "")
			}
			encryptMediumplatng, err := encryptAES256Func(mediumplating)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			mp.Expenses = encryptMediumplatng
			mp.Date = r.FormValue(fmt.Sprintf("mediumplatingdate%d", num))           // 중도금 세금 계산서 발행일
			mp.PayedDate = r.FormValue(fmt.Sprintf("mediumplatingpayeddate%d", num)) // 중도금 지급일
			if r.FormValue(fmt.Sprintf("mediumplatingstatus%d", num)) == "true" {
				mp.Status = true
			} else {
				mp.Status = false
			}
			v.MediumPlating = append(v.MediumPlating, mp)
		}
	}
	if r.FormValue("balance") != "" { // 잔금이 적힌 경우
		balance := r.FormValue("balance")
		if strings.Contains(balance, ",") {
			balance = strings.ReplaceAll(balance, ",", "")
		}
		encryptBalance, err := encryptAES256Func(balance)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		v.Balance.Expenses = encryptBalance
		v.Balance.Date = r.FormValue("balancedate")             // 잔금 세금 계산서 발행일
		v.Balance.PayedDate = r.FormValue(("balancepayeddate")) // 잔금 지급일
		if r.FormValue("balancestatus") == "true" {             // 잔금 지출 완료인 경우
			v.Balance.Status = true
		} else {
			v.Balance.Status = false
		}
	}

	err = v.CheckErrorFunc()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = addVendorFunc(client, v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log := Log{
		UserID:    token.ID,
		CreatedAt: time.Now(),
		Content:   fmt.Sprintf("프로젝트 %s에 벤더 %s가 추가되었습니다.", v.Project, v.Name),
	}

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/addvendor-success?project=%s", v.Project), http.StatusSeeOther)
}

// handleAddVendorSuccessFunc 함수는 벤더 추가를 성공했다는 페이지를 띄운다.
func handleAddVendorSuccessFunc(w http.ResponseWriter, r *http.Request) {
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
		Token   Token
		Project string
	}
	rcp := Recipe{}
	rcp.Token = token
	q := r.URL.Query()
	project := q.Get("project")
	rcp.Project = project

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "addvendor-success", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleEditVendorFunc 함수는 벤더 정보 수정페이지를 띄우는 함수이다.
func handleEditVendorFunc(w http.ResponseWriter, r *http.Request) {
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
	if id == "" {
		http.Error(w, "URL에 id를 입력해주세요", http.StatusBadRequest)
		return
	}
	isfinished := q.Get("isfinished")

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
		Token       Token
		Vendor      Vendor
		ProjectList []Project
		IsFinished  bool
	}

	rcp := Recipe{}
	rcp.Token = token
	rcp.Vendor, err = getVendorFunc(client, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	projects, err := getAllProjectsFunc(client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rcp.ProjectList = projects
	rcp.IsFinished, err = strconv.ParseBool(isfinished)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "edit-vendor", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleEditVendorSubmitFunc 함수는 벤더 수정 페이지에서 Edit 버튼을 눌러서 벤더를 수정하는 함수이다.
func handleEditVendorSubmitFunc(w http.ResponseWriter, r *http.Request) {
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

	id := r.FormValue("id")
	isfinished := r.FormValue("isfinished")

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

	vendor, err := getVendorFunc(client, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	vendor.Project = r.FormValue("project")
	project, err := getProjectFunc(client, vendor.Project)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	vendor.ProjectName = project.Name
	vendor.Name = strings.TrimSpace(r.FormValue("name"))

	// 총 비용 암호화
	expenses := r.FormValue("expenses")
	if strings.Contains(expenses, ",") {
		expenses = strings.ReplaceAll(expenses, ",", "")
	}
	encryptExpenses, err := encryptAES256Func(expenses)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	vendor.Expenses = encryptExpenses
	vendor.Date = r.FormValue("date")

	// 벤더 부가 정보 입력
	if r.FormValue("cuts") != "" { // 컷정보가 있는 경우
		cuts, err := strconv.Atoi(r.FormValue("cuts"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		vendor.Cuts = cuts
	} else {
		vendor.Cuts = 0
	}
	if r.FormValue("tasks") != "" { // 태스크 정보가 있는 경우
		tasks := r.FormValue("tasks")
		taskList := stringToListFunc(tasks, ",")
		sort.Strings(taskList)
		vendor.Tasks = taskList
	} else {
		vendor.Tasks = nil
	}

	// 벤더 비용 정보 입력
	vendor.Downpayment = VendorCost{}
	if r.FormValue("downpayment") != "" { // 계약금이 적힌 경우
		downpayment := r.FormValue("downpayment")
		if strings.Contains(downpayment, ",") {
			downpayment = strings.ReplaceAll(downpayment, ",", "")
		}
		encryptDownpayment, err := encryptAES256Func(downpayment)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		vendor.Downpayment.Expenses = encryptDownpayment
		vendor.Downpayment.Date = r.FormValue("downpaymentdate")           // 계약금 세금 계산서 발행일
		vendor.Downpayment.PayedDate = r.FormValue("downpaymentpayeddate") // 계약금 지급일
		if r.FormValue("downpaymentstatus") == "true" {                    // 계약금 지출 완료인 경우
			vendor.Downpayment.Status = true
		} else {
			vendor.Downpayment.Status = false
		}
	}
	// 중도금 입력 칸의 갯수
	mediumplatingNum := r.FormValue("mediumplatingNum")
	mpNum, err := strconv.Atoi(mediumplatingNum)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	vendor.MediumPlating = []VendorCost{}
	for num := 0; num < mpNum; num++ { // 중도금 칸의 갯수만큼만 입력
		if r.FormValue(fmt.Sprintf("mediumplating%d", num)) != "" { // 중도금이 적힌 경우
			mp := VendorCost{}
			mediumplating := r.FormValue(fmt.Sprintf("mediumplating%d", num))
			if strings.Contains(mediumplating, ",") {
				mediumplating = strings.ReplaceAll(mediumplating, ",", "")
			}
			encryptMediumplatng, err := encryptAES256Func(mediumplating)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			mp.Expenses = encryptMediumplatng
			mp.Date = r.FormValue(fmt.Sprintf("mediumplatingdate%d", num))           // 중도금 세금 계산서 발행일
			mp.PayedDate = r.FormValue(fmt.Sprintf("mediumplatingpayeddate%d", num)) // 중도금 지급일
			if r.FormValue(fmt.Sprintf("mediumplatingstatus%d", num)) == "true" {    // 중도금 지출 완료인 경우
				mp.Status = true
			} else {
				mp.Status = false
			}
			vendor.MediumPlating = append(vendor.MediumPlating, mp)
		}
	}
	vendor.Balance = VendorCost{}
	if r.FormValue("balance") != "" { // 잔금이 적힌 경우
		balance := r.FormValue("balance")
		if strings.Contains(balance, ",") {
			balance = strings.ReplaceAll(balance, ",", "")
		}
		encryptBalance, err := encryptAES256Func(balance)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		vendor.Balance.Expenses = encryptBalance
		vendor.Balance.Date = r.FormValue("balancedate")           // 잔금 세금 계산서 발행일
		vendor.Balance.PayedDate = r.FormValue("balancepayeddate") // 잔금 지급일
		if r.FormValue("balancestatus") == "true" {                // 잔금 지출 완료인 경우
			vendor.Balance.Status = true
		} else {
			vendor.Balance.Status = false
		}
	}

	err = vendor.CheckErrorFunc()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = setVendorFunc(client, vendor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log := Log{
		UserID:    token.ID,
		CreatedAt: time.Now(),
		Content:   fmt.Sprintf("프로젝트 %s에 벤더 %s가 수정되었습니다.", vendor.Project, vendor.Name),
	}

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/editvendor-success?id=%s&isfinished=%s", id, isfinished), http.StatusSeeOther)
}

// handleEditVendorSuccessFunc 함수는 벤더 수정이 완료됐다는 페이지를 띄운다.
func handleEditVendorSuccessFunc(w http.ResponseWriter, r *http.Request) {
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
	if id == "" {
		http.Error(w, "URL에 id를 입력해주세요", http.StatusBadRequest)
		return
	}
	isfinished := q.Get("isfinished")

	type Recipe struct {
		Token      Token
		ID         string // 벤더 ID
		IsFinished string // 정산 여부 토글
	}
	rcp := Recipe{
		Token:      token,
		ID:         id,
		IsFinished: isfinished,
	}

	w.Header().Set("Content-Type", "text/html")
	err = TEMPLATES.ExecuteTemplate(w, "editvendor-success", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
