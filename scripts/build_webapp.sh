#!/bin/bash
CDIR=$(pwd)
BUCKET_NAME="app-wtctrl-com"
BUCKET_URL="https://${BUCKET_NAME}.s3.us-west-2.amazonaws.com"
JS_URL="$BUCKET_URL/js/"
CSS_URL="$BUCKET_URL/css/"
ICONS_URL="$BUCKET_URL/icons/"
FONTS_URL="$BUCKET_URL/fonts/"

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd $DIR/../frontend/qwtctrl
quasar build
cd dist/spa
sed -i"bk" "s^=js/^=${JS_URL}^g" index.html 
sed -i"bk" "s^=icons/^=${ICONS_URL}^g" index.html
sed -i"bk" "s^=css/^=${CSS_URL}^g" index.html 
sed -i"bk" "s^=fonts/^=${FONTS_URL}^g" index.html 
rm -rf index.htmlbk
cd ..
aws s3 sync spa/ s3://${BUCKET_NAME}/
cp spa/index.html $DIR/../lambda/webapp
cd $CDIR
