package awsapi

import (
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-lambda-go/events"
)

func getAuthPolicy(effect, arn string) events.APIGatewayCustomAuthorizerPolicy {
	return events.APIGatewayCustomAuthorizerPolicy{
		Version: "2012-10-17",
		Statement: []events.IAMPolicyStatement{
			{
				Action:   []string{"execute-api:Invoke"},
				Effect:   effect,
				Resource: []string{arn},
			},
		},
	}
}

func HandleWSAuthReq(table *DTable, params map[string]string, arn string) (
	events.APIGatewayCustomAuthorizerResponse, error) {
	resp := events.APIGatewayCustomAuthorizerResponse{}
	secretPK, ok := params["secret"]
	if !ok {
		return resp, errors.New("Secret is not provided.")
	}
	secret := &Secret{}
	err := table.FetchItem(secretPK, secret)
	if err != nil {
		if err.Error() == NO_SUCH_ITEM {
			resp.PolicyDocument = getAuthPolicy("Deny", arn)
			return resp, nil
		}
		return resp, err
	}
	if !secret.IsValid() {
		resp.PolicyDocument = getAuthPolicy("Deny", arn)
		return resp, nil
	}
	resp.PrincipalID = secret.UserPK
	resp.PolicyDocument = getAuthPolicy("Allow", arn)
	if secret.ONEOFF {
		secret.TTL = time.Now().Unix()
		err = table.StoreItem(secret)
		if err != nil {
			fmt.Println("ERROR", err.Error())
		}
	}
	return resp, nil
}
