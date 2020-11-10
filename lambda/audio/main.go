package main

import (
	"os"

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

func main() {
	lambda.Start(Handler)
}
