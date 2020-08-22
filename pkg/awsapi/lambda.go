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
	tokenID, ok := params["token"]
	if !ok {
		return resp, errors.New("Token is not provided.")
	}
	token := &Token{}
	pk := fmt.Sprintf("%s%s", TokenKeyPrefix, tokenID)
	err := table.FetchItem(pk, token)
	if err != nil {
		if err.Error() == NO_SUCH_ITEM {
			resp.PolicyDocument = getAuthPolicy("Deny", arn)
			return resp, nil
		}
		return resp, err
	}
	if !token.IsValid() {
		resp.PolicyDocument = getAuthPolicy("Deny", arn)
		return resp, nil
	}
	resp.PrincipalID = token.UserPK
	resp.PolicyDocument = getAuthPolicy("Allow", arn)
	if token.ONEOFF {
		token.TTL = time.Now().Unix()
		err = table.StoreItem(token)
		if err != nil {
			fmt.Println("ERROR", err.Error())
		}
	}
	return resp, nil
}
