#!/bin/sh
CDIR=$(pwd)
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd $DIR
./build_lambda.sh && terraform apply -auto-approve
rm -rf ../../lambda/dstream/dstream
rm -rf ../../lambda/tgwebhook/tgwebhook
cd $CDIR
