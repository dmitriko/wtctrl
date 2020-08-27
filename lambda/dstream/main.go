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

func handleRequest(ctx context.Context, e events.DynamoDBEvent) {
	awsapi.HandleDBEvent(ctx, table, e)
}

func main() {
	lambda.Start(handleRequest)
}
