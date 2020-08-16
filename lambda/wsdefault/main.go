package main

import (
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handleRequest(req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	fmt.Printf("%+v", req)
	resp := events.APIGatewayProxyResponse{}
	return resp, nil
}

func main() {
	lambda.Start(handleRequest)
}
