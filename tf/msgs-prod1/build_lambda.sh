#!/bin/sh
set -ex
curr=$(pwd)

cd "../../lambda/tgwebhook"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -v
cd $curr

cd "../../lambda/dstream"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -v

cd $curr
