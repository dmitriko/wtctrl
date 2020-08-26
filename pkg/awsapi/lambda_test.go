package awsapi

import (
	"context"
	"fmt"
	"testing"
	"time"

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

	disconnReq := getProxyContext("DISCONNECT", domain, stage, connId, user.PK)
	err = HandleWSConnReq(testTable, disconnReq)
	assert.Nil(t, err)
	dconn := &WSConn{}
	err = testTable.FetchSubItem(user.PK, fmt.Sprintf("%s%s", WSConnKeyPrefix, connId), dconn)
	if assert.NotNil(t, err) {
		assert.Equal(t, NO_SUCH_ITEM, err.Error())
	}
}

func collectOutput(ctx context.Context, out *[]string, inCh <-chan []byte, doneCh <-chan bool) {
	for {
		select {
		case <-doneCh:
			return
		case <-ctx.Done():
			return
		case s := <-inCh:
			*out = append(*out, string(s))
		}
	}
}

func TestWSNewSender(t *testing.T) {
	connId := "someid="
	toUserCh := make(chan []byte)
	endpoint := "https://foobar.com/stage"
	_, err := NewWSSender(endpoint, connId, toUserCh)
	assert.Nil(t, err)
}

func TestWSGotCmdPing(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	user, _ := NewUser("foo")
	domain := "foobar.com"
	connId := "someid="
	stage := "prod"
	_ = getProxyContext("MESSAGE", domain, stage, connId, user.PK)
	outCh := make(chan []byte)
	doneCh := make(chan bool)
	cmd := `{"name":"ping", "id":"somerandom"}`
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	var output []string
	go collectOutput(ctx, &output, outCh, doneCh)
	err := handleUserCmd(ctx, testTable, user.PK, cmd, outCh)
	assert.Nil(t, err)
	doneCh <- true
	assert.Equal(t, 1, len(output))
}

func TestCmdUnmarshal(t *testing.T) {
	input := `{"name":"msgfetchbydays", "subs": true, "days":20, "status":0}`
	cmd, err := UnmarshalCmd([]byte(input))
	assert.Nil(t, err)
	if assert.NotNil(t, cmd) {
		assert.Equal(t, "msgfetchbydays", cmd.GetName())
	}
}

func TestCmdFetchByDays(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	input := `{"name":"msgfetchbydays", "subs": true, "days":20, "status":0}`
	domain := "foobar.com"
	connId := "someid="
	stage := "prod"
	user1, _ := NewUser("user1")
	user2, _ := NewUser("user2")
	msg1, err := NewMsg("bot1", user1.PK, TGTextMsgKind, CreatedAtOp("-10d"), UserStatusOp(0),
		DataOp(map[string]string{"text": "msg1"}))
	msg2, err := NewMsg("bot1", user1.PK, TGTextMsgKind, CreatedAtOp("-2d"), UserStatusOp(5),
		DataOp(map[string]string{"text": "msg2"}))
	msg3, err := NewMsg("bot1", user2.PK, TGTextMsgKind, CreatedAtOp("-2d"), UserStatusOp(5),
		DataOp(map[string]string{"text": "msg3"}))
	msg4, err := NewMsg("bot1", user1.PK, TGTextMsgKind, CreatedAtOp("-2d"), UserStatusOp(0),
		DataOp(map[string]string{"text": "msg4"}))
	errs := testTable.StoreItems(msg1, msg2, msg3, msg4)
	for _, e := range errs {
		if e != nil {
			t.Error(e)
		}
	}
	_ = getProxyContext("MESSAGE", domain, stage, connId, user1.PK)
	outCh := make(chan []byte)
	doneCh := make(chan bool)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	var output []string
	go collectOutput(ctx, &output, outCh, doneCh)
	err = handleUserCmd(ctx, testTable, user1.PK, input, outCh)
	if assert.Nil(t, err) {
		doneCh <- true
	}
	assert.Equal(t, 2, len(output))
}
