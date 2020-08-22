package awsapi

import (
	"fmt"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

func TestWSAuthRequest(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	user, _ := NewUser("foo")
	token, _ := NewToken(user, 24)
	token.ONEOFF = true
	assert.Nil(t, testTable.StoreItem(token))
	arn := "arn:::somename"
	assert.True(t, token.IsValid())
	resp, err := HandleWSAuthReq(testTable, map[string]string{"token": token.Id()}, arn)
	assert.Nil(t, err)
	assert.Equal(t, user.PK, resp.PrincipalID)
	assert.Equal(t, "Allow", resp.PolicyDocument.Statement[0].Effect)
	assert.Equal(t, arn, resp.PolicyDocument.Statement[0].Resource[0])
	// second time it shoudl fail once we have ONEOFF true
	resp, err = HandleWSAuthReq(testTable, map[string]string{"token": token.Id()}, arn)
	assert.Equal(t, "Deny", resp.PolicyDocument.Statement[0].Effect)

	resp, err = HandleWSAuthReq(testTable, map[string]string{"token": "foo"}, arn)
	assert.Equal(t, "Deny", resp.PolicyDocument.Statement[0].Effect)

	token2, _ := NewToken(user, 0)
	assert.Nil(t, testTable.StoreItem(token2))
	resp, err = HandleWSAuthReq(testTable, map[string]string{"token": token2.Id()}, arn)
	assert.Equal(t, "Deny", resp.PolicyDocument.Statement[0].Effect)
}

func getProxyContext(eType, domain, stage, connId, principalId string) events.APIGatewayWebsocketProxyRequestContext {
	return events.APIGatewayWebsocketProxyRequestContext{
		EventType:    eType,
		Authorizer:   map[string]interface{}{"principalId": principalId},
		ConnectionID: connId,
		DomainName:   domain,
		Stage:        stage}
}

func TestWSConnDiscon(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	user, _ := NewUser("foo")
	domain := "foobar.com"
	connId := "someid="
	stage := "prod"
	connReq := getProxyContext("CONNECT", domain, stage, connId, user.PK)
	err := HandleWSConnReq(testTable, connReq)
	assert.Nil(t, err)
	conn := &WSConn{}
	err = testTable.FetchSubItem(user.PK, fmt.Sprintf("%s%s", WSConnKeyPrefix, connId), conn)
	assert.Nil(t, err)
	assert.Equal(t, "https://foobar.com/prod", conn.Endpoint())
}
