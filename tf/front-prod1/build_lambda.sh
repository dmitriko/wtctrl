#!/bin/sh
set -ex
curr=$(pwd)

cd "../../lambda/webapp"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -v
cd $curr
