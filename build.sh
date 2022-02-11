#!/bin/sh

# 사용되는 리소스를 Go파일로 변경하기
go run assets/asset_generate.go

APP="budget"
GOOS=linux GOARCH=amd64 go build -o ./bin/linux/${APP} *.go