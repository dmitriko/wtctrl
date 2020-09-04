package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dmitriko/wtctrl/pkg/awsapi"
)

var table *awsapi.DTable
var indexHTML string

func init() {
	table, _ = awsapi.NewDTable(os.Getenv("TABLE_NAME"))
	err := table.Connect()
	if err != nil {
		panic("Could not connect to Dynamo")
	}
	content, err := ioutil.ReadFile("./index.html")
	if err != nil {
		log.Fatal(err)
	}
	indexHTML = string(content)
}

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	fmt.Printf("%#v", req)
	if req.HTTPMethod == "POST" {
		return awsapi.HandleLoginRequest(table, req)
	}
	path := req.Path
	var body string
	var contentType string
	if strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".css") {
		parts := strings.Split(path, "/")
		filePath := fmt.Sprintf("./%s/%s", parts[len(parts)-2], parts[len(parts)-1])
		content, err := ioutil.ReadFile(filePath)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}
		if strings.HasSuffix(path, ".css") {
			contentType = "text/css"
		} else {
			contentType = "text/javascript"
		}
		body = string(content)
	} else {
		body = indexHTML
		contentType = "text/html"
	}
	res := events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": contentType,
		},
		Body: body,
	}
	return res, nil
}

func main() {
	lambda.Start(Handler)
}
