package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	fmt.Printf("%#v", req)

	res := events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Access-Control-Allow-Origin": "*",
			"Content-Type":                "text/json",
		},
		Body: `{"ok":true}`,
	}
	return res, nil
}

func main() {
	lambda.Start(Handler)
}
