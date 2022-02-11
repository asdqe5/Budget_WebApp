// 프로젝트 결산 프로그램
//
// Description : 경로 관련 스크립트

package main

import (
	"io/ioutil"
	"os"
)

// createFolderFunc 함수는 폴더를 생성하는 함수이다.
func createFolderFunc(path string) error {
	// 폴더 경로가 존재하는지 확인
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) { // 경로가 존재하지 않으면 생성
			os.MkdirAll(path, 0777)
		} else { // 다른 에러 발생시 로그 찍고 리턴
			return err
		}
	}
	return nil
}

// delAllFilesFunc 함수는 입력받은 경로에 있는 모든 파일을 삭제하는 함수이다.
func delAllFilesFunc(path string) error {
	// 경로가 존재하는지 확인
	if _, err := os.Stat(path); err != nil {
		return err
	}

	fileInfo, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	for _, file := range fileInfo {
		os.RemoveAll(path + "/" + file.Name())
	}
	return nil
}

// checkFileExistsFunc 함수는 해당 경로가 존재하는지 확인하는 함수이다.
func checkFileExistsFunc(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil { // 경로가 존재할 때
		return true, nil
	} else if os.IsNotExist(err) { // 경로가 존재하지 않을 때
		return false, nil
	} else { // 다른 에러가 발생했을 때
		return false, err
	}
}
