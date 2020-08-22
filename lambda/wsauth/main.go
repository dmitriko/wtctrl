package main

import (
	"context"
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

func handleRequest(ctx context.Context, event events.APIGatewayCustomAuthorizerRequestTypeRequest) (
	events.APIGatewayCustomAuthorizerResponse, error) {
	return awsapi.HandleWSAuthReq(table, event.QueryStringParameters, event.MethodArn)
}

func main() {
	lambda.Start(handleRequest)
}
