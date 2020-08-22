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

func storeWSConn(table *DTable, domain, stage, connId, userPK string) error {
	conn, _ := NewWSConn(userPK, connId, domain, stage)
	return table.StoreItem(conn)
}

func clearWSConn(table *DTable, connId, userPK string) error {
	return table.DeletSubItem(userPK, fmt.Sprintf("%s%s", WSConnKeyPrefix, connId))
}

func HandleWSConnReq(table *DTable, ctx events.APIGatewayWebsocketProxyRequestContext) error {
	authData, ok := ctx.Authorizer.(map[string]interface{})
	if !ok {
		return errors.New("Could not cast Auth data")
	}
	principalId, ok := authData["principalId"]
	if !ok {
		fmt.Printf("%#v", authData)
		return errors.New("No Auth data provided")
	}
	userPK := principalId.(string)
	if ctx.EventType == "CONNECT" {
		return storeWSConn(table, ctx.DomainName, ctx.Stage, ctx.ConnectionID, userPK)
	} else {
		return clearWSConn(table, ctx.ConnectionID, userPK)
	}
}
