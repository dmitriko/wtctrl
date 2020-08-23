package awsapi

import (
	"context"
	"encoding/json"
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

func extractUserPK(ctx events.APIGatewayWebsocketProxyRequestContext) (string, error) {
	authData, ok := ctx.Authorizer.(map[string]interface{})
	if !ok {
		return "", errors.New("Could not cast Auth data")
	}
	principalId, ok := authData["principalId"]
	if !ok {
		fmt.Printf("%#v", authData)
		return "", errors.New("No Auth data provided")
	}
	return principalId.(string), nil
}

func HandleWSConnReq(table *DTable, ctx events.APIGatewayWebsocketProxyRequestContext) error {
	userPK, err := extractUserPK(ctx)
	if err != nil {
		return err
	}
	if ctx.EventType == "CONNECT" {
		return storeWSConn(table, ctx.DomainName, ctx.Stage, ctx.ConnectionID, userPK)
	} else {
		return clearWSConn(table, ctx.ConnectionID, userPK)
	}
}

type UserCmd struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}

type CmdResp struct {
	Name   string `json:"name"`
	Id     string `json:"id"`
	Body   string `json:"body"`
	Status string `json:"status"`
	SecNum int    `json:"number"`
}

func (cmd *UserCmd) Perform(ctx context.Context, out chan<- []byte, done chan<- error) {
	switch cmd.Name {
	case "ping":
		done <- sendWithContext(ctx, out, &CmdResp{
			Name:   cmd.Name,
			Id:     cmd.Id,
			Status: "done",
			Body:   "pong",
		})
	default:
		done <- errors.New(fmt.Sprintf("Got unknown command %s", cmd.Name))
	}
}

func sendWithContext(ctx context.Context, outCh chan<- []byte, resp *CmdResp) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case outCh <- data:
		return nil
	}
}

// SetCmd, CreateCmd, FetchCmd, DeleteCmd, PingCmd
func handleUserCmd(ctx context.Context, userPK, cmd string, outCh chan<- []byte) error {
	var err error
	userCmd := &UserCmd{}
	err = json.Unmarshal([]byte(cmd), userCmd)
	if err != nil {
		return err
	}
	doneCh := make(chan error)
	go userCmd.Perform(ctx, outCh, doneCh)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-doneCh:
		return err
	}
}
