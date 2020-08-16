#!/bin/sh
CDIR=$(pwd)
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd $DIR
./build_lambda.sh && terraform apply -auto-approve
rm -rf ../../lambda/wsauth/wsauth
rm -rf ../../lambda/wsdefault/wsdefault
cd $CDIR
