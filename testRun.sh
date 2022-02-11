#!/bin/sh
go run assets/asset_generate.go
go build
sudo ./budget -http :80