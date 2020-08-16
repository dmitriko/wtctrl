#!/bin/sh
set -ex
curr=$(pwd)

cd "../../lambda/wsauth"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -v
cd $curr

cd "../../lambda/wsconn"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -v
cd $curr

cd "../../lambda/wsdefault"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -v
cd $curr
