package main

import (
	"context"
	"errors"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

const secretKey = "secret"
const secretVal = "foobar"

func allowResp(event events.APIGatewayCustomAuthorizerRequestTypeRequest) events.APIGatewayCustomAuthorizerResponse {

	resp := events.APIGatewayCustomAuthorizerResponse{PrincipalID: "foo"}

	resp.PolicyDocument = events.APIGatewayCustomAuthorizerPolicy{
		Version: "2012-10-17",
		Statement: []events.IAMPolicyStatement{
			{
				Action:   []string{"execute-api:Invoke"},
				Effect:   "Allow",
				Resource: []string{event.MethodArn},
			},
		},
	}

	resp.Context = map[string]interface{}{
		"foo": "bar",
	}

	return resp
}

func handleRequest(ctx context.Context, event events.APIGatewayCustomAuthorizerRequestTypeRequest) (
	events.APIGatewayCustomAuthorizerResponse, error) {
	if event.QueryStringParameters[secretKey] != secretVal {
		return events.APIGatewayCustomAuthorizerResponse{}, errors.New("Unauthorized")
	}
	return allowResp(event), nil
}

func main() {
	lambda.Start(handleRequest)
}
