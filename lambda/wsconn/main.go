package main

import (
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handleRequest(req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	fmt.Printf("%+v", req)
	resp := events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       "ok",
	}
	return resp, nil
}

func main() {
	lambda.Start(handleRequest)
}
