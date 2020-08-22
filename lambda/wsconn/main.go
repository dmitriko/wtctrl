package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dmitriko/wtctrl/pkg/awsapi"
)

var table *awsapi.DTable

func init() {
	table, _ = awsapi.NewDTable(os.Getenv("TABLE_NAME"))
	err := table.Connect()
	if err != nil {
		panic("Could not connect to Dynamo")
	}
}

func handleRequest(req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	err := awsapi.HandleWSConnReq(table, req.RequestContext)
	if err != nil {
		fmt.Println("ERROR", err.Error())
	}
	resp := events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       "ok",
	}
	return resp, err
}

func main() {
	lambda.Start(handleRequest)
}
