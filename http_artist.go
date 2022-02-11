// 프로젝트 결산 프로그램
//
// Description : http 아티스트 관련 스크립트

package main

import (
	"io/ioutil"
	"net/http"
	"os"
)

// handleUploadArtistsExcelFunc 함수는 dropzone에 엑셀 파일을 업로드하면 임시 폴더로 복사하는 함수이다.
func handleUploadArtistsExcelFunc(w http.ResponseWriter, r *http.Request) {
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

	err = r.ParseMultipartForm(200000)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	path := os.TempDir() + "/budget/" + token.ID + "/artists/" // 엑셀 파일을 저장할 임시 폴더 경로
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
