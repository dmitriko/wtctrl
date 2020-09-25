package awsapi

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestCmdGotPing(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	user, _ := NewUser("foo")
	domain := "foobar.com"
	connId := "someid="
	stage := "prod"
	reqCtx := getProxyContext("MESSAGE", domain, stage, connId, user.PK)
	outCh := make(chan []byte)
	doneCh := make(chan bool)
	cmd := `{"name":"ping", "id":"somerandom"}`
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	var output []string
	go collectOutput(ctx, &output, outCh, doneCh)
	err := handleUserCmd(ctx, testTable, reqCtx, cmd, outCh)
	assert.Nil(t, err)
	doneCh <- true
	if assert.Equal(t, 1, len(output)) {
		r := make(map[string]string)
		assert.Nil(t, json.Unmarshal([]byte(output[0]), &r))
		assert.Equal(t, "ping", r["name"])
	}
}

func TestCmdUnmarshal(t *testing.T) {
	input := `{"name":"msgfetchbydays", "subs": true, "days":20, "status":0}`
	cmd, err := UnmarshalCmd([]byte(input))
	assert.Nil(t, err)
	if assert.NotNil(t, cmd) {
		assert.Equal(t, "msgfetchbydays", cmd.(*MsgFetchByDays).Name)
	}
}

func TestCmdMsgFetch(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	domain := "foobar.com"
	connId := "someid="
	stage := "prod"
	updated := time.Now().Unix() + 5
	user1, _ := NewUser("user1")
	msg1, _ := NewMsg("bot1", user1.PK, TGPhotoMsgKind)
	msg1.Data["text"] = "foobar"
	msg1.Data[UpdatedAtField] = updated
	pic1, _ := NewMsgFile(msg1.PK, FileKindTgThumb, "image/jpeg", "wtctrl-udatab", "bar")
	pic2, _ := NewMsgFile(msg1.PK, FileKindTgMediumPic, "image/jpeg", "wtctrl-udatab", "bar")
	pic3, _ := NewMsgFile(msg1.PK, FileKindTgBigPic, "image/jpeg", "wtctrl-udatab", "bar")
	assert.Nil(t, testTable.StoreItem(pic1))
	assert.Nil(t, testTable.StoreItem(pic2))
	assert.Nil(t, testTable.StoreItem(pic3))
	assert.Nil(t, testTable.StoreItem(msg1))
	reqCtx := getProxyContext("MESSAGE", domain, stage, connId, user1.PK)
	outCh := make(chan []byte)
	doneCh := make(chan bool)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	output := make([]string, 0)
	go collectOutput(ctx, &output, outCh, doneCh)
	input := fmt.Sprintf(`{"name":"fetchmsg", "pk":"%s"}`, msg1.PK)
	err := handleUserCmd(ctx, testTable, reqCtx, input, outCh)
	if assert.Nil(t, err) {
		doneCh <- true
	}
	assert.Equal(t, 1, len(output))
	resp := make(map[string]interface{})
	assert.Nil(t, json.Unmarshal([]byte(output[0]), &resp))
	assert.Equal(t, msg1.PK, resp["pk"].(string))

	fdata := resp["files"].(map[string]interface{})
	thumb, ok := fdata["thumb"].(map[string]interface{})
	assert.True(t, ok)
	url, ok := thumb["url"].(string)
	assert.True(t, ok)
	assert.True(t, len(url) > 0)
	text, _ := resp["text"].(string)
	assert.Equal(t, "foobar", text)

	assert.Equal(t, updated, int64(resp["updated"].(float64)))
}

func TestCmdFetchByDays(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	domain := "foobar.com"
	connId := "someid="
	stage := "prod"
	user1, _ := NewUser("user1")
	user2, _ := NewUser("user2")
	msg1, err := NewMsg("bot1", user1.PK, TGTextMsgKind, CreatedAtOp("-10d"), UserStatusOp(0),
		DataOp(map[string]interface{}{"text": "msg1"}))
	msg2, err := NewMsg("bot1", user1.PK, TGTextMsgKind, CreatedAtOp("-3d"), UserStatusOp(5),
		DataOp(map[string]interface{}{"text": "msg2"}))
	msg3, err := NewMsg("bot1", user1.PK, TGTextMsgKind, CreatedAtOp("-2d"), UserStatusOp(5),
		DataOp(map[string]interface{}{"text": "msg3"}))
	msg4, err := NewMsg("bot1", user2.PK, TGTextMsgKind, CreatedAtOp("-2d"), UserStatusOp(0),
		DataOp(map[string]interface{}{"text": "msg4"}))
	errs := testTable.StoreItems(msg1, msg2, msg3, msg4)
	for _, e := range errs {
		if e != nil {
			t.Error(e)
		}
	}
	reqCtx := getProxyContext("MESSAGE", domain, stage, connId, user1.PK)
	outCh := make(chan []byte)
	doneCh := make(chan bool)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	var output []string

	output = make([]string, 0)
	go collectOutput(ctx, &output, outCh, doneCh)
	input := `{"name":"msgfetchbydays", "id":"somerandom", "days":20, "status":5, "desc":true}`
	err = handleUserCmd(ctx, testTable, reqCtx, input, outCh)
	if assert.Nil(t, err) {
		doneCh <- true
	}
	assert.Equal(t, 4, len(output))

	resp1 := make(map[string]interface{})
	err = json.Unmarshal([]byte(output[0]), &resp1)
	assert.Nil(t, err)

	resp2 := make(map[string]interface{})
	err = json.Unmarshal([]byte(output[1]), &resp2)
	assert.Nil(t, err)

	resp3 := make(map[string]interface{})
	err = json.Unmarshal([]byte(output[2]), &resp3)
	assert.Nil(t, err)

	resp4 := make(map[string]interface{})
	err = json.Unmarshal([]byte(output[3]), &resp4)
	assert.Nil(t, err)

	assert.Equal(t, msg3.PK, resp2["pk"].(string))
	assert.Equal(t, msg2.PK, resp3["pk"].(string))
}

func TestCmdStartStopSubscr(t *testing.T) {
	defer stopLocalDynamo()
	table := startLocalDynamo(t)
	domain := "foobar.com"
	connId := "someid="
	stage := "prod"
	userPK := "user#user1"
	reqCtx := getProxyContext("MESSAGE", domain, stage, connId, userPK)
	outCh := make(chan []byte)
	doneCh := make(chan bool)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	output := make([]string, 0)
	go collectOutput(ctx, &output, outCh, doneCh)
	input := `{"name":"subscr", "status":0, "umspk":"user#user1", "id":"foo"}`
	err := handleUserCmd(ctx, table, reqCtx, input, outCh)
	if assert.Nil(t, err) {
		doneCh <- true
	}
	assert.Equal(t, 1, len(output))
	sA := &Subscription{}
	sB := &Subscription{}
	assert.Nil(t, table.FetchSubItem(userPK, fmt.Sprintf("%s%s", SubscriptionKeyPrefix, connId), sA))
	assert.Nil(t, table.FetchSubItem("user#user1#0", fmt.Sprintf("%s%s", SubscriptionKeyPrefix, connId), sB))

	output = make([]string, 0)
	go collectOutput(ctx, &output, outCh, doneCh)
	input = `{"name":"unsubscr", "status":0, "umspk":"user#user1", "id":"foo"}`
	err = handleUserCmd(ctx, table, reqCtx, input, outCh)
	if assert.Nil(t, err) {
		doneCh <- true
	}
	assert.Equal(t, 1, len(output))
	s2A := &Subscription{}
	err = table.FetchSubItem(userPK, fmt.Sprintf("%s%s", SubscriptionKeyPrefix, connId), s2A)
	if assert.NotNil(t, err) {
		assert.Equal(t, NO_SUCH_ITEM, err.Error())
	}
	s2B := &Subscription{}
	err = table.FetchSubItem("user#user1#0", fmt.Sprintf("%s%s", SubscriptionKeyPrefix, connId), s2B)
	if assert.NotNil(t, err) {
		assert.Equal(t, NO_SUCH_ITEM, err.Error())
	}
}
func TestHandleRequestOTP(t *testing.T) {
	defer stopLocalDynamo()
	table := startLocalDynamo(t)
	user, _ := NewUser("foo")
	user.TGID = "123456789"
	bot, _ := NewBot(DummyBotKind, "foo")
	user.Bots = []string{bot.PK}
	user.Tel = "55555"
	assert.Nil(t, table.StoreItem(bot))
	assert.Nil(t, table.StoreNewUser(user))

	reqOTP := `{"key": "55555"}`
	req := events.APIGatewayProxyRequest{
		Path:       "/prod1/reqotp",
		HTTPMethod: "POST",
		RequestContext: events.APIGatewayProxyRequestContext{
			Stage: "prod1",
		},
		Body: reqOTP,
	}

	resp, err := HandleLoginRequest(table, req)
	assert.Nil(t, err)
	r := &OTPReqRespBody{}
	assert.Nil(t, json.Unmarshal([]byte(resp.Body), r))
	assert.True(t, r.OK)
	assert.True(t, r.RequestPK != "")
	lReq := &LoginRequest{}
	assert.Nil(t, table.FetchItem(r.RequestPK, lReq))

}

func TestHandleLogin(t *testing.T) {
	defer stopLocalDynamo()
	table := startLocalDynamo(t)
	user, _ := NewUser("foo")
	lreq, _ := NewLoginRequest(user.PK)
	assert.Nil(t, table.StoreItem(lreq))
	assert.Nil(t, table.StoreItem(user))
	reqLogin := fmt.Sprintf(`{"request_pk": "%s", "otp":"%s"}`, lreq.PK, lreq.OTP)
	req := events.APIGatewayProxyRequest{
		Path:       "/prod1/login",
		HTTPMethod: "POST",
		RequestContext: events.APIGatewayProxyRequestContext{
			Stage: "prod1",
		},
		Body: reqLogin,
	}
	resp, err := HandleLoginRequest(table, req)
	assert.Nil(t, err)
	loginResp := &LoginResp{}
	err = json.Unmarshal([]byte(resp.Body), loginResp)
	assert.Nil(t, err)
	tokenPK := fmt.Sprintf("%s%s", TokenKeyPrefix, loginResp.Token)
	token := &Token{}
	assert.Nil(t, table.FetchItem(tokenPK, token))
	assert.Equal(t, 4, len(loginResp.Folders))

	wrongLogin := fmt.Sprintf(`{"request_pk": "%s", "otp":"000000"}`, lreq.PK)
	req = events.APIGatewayProxyRequest{
		Path:       "/prod1/login",
		HTTPMethod: "POST",
		RequestContext: events.APIGatewayProxyRequestContext{
			Stage: "prod1",
		},
		Body: wrongLogin,
	}
	resp, err = HandleLoginRequest(table, req)
	assert.Nil(t, err)
	loginResp = &LoginResp{}
	err = json.Unmarshal([]byte(resp.Body), loginResp)
	assert.Nil(t, err)
	assert.False(t, loginResp.Ok)

	// too many login attempts
	lreq, _ = NewLoginRequest(user.PK)
	assert.Nil(t, table.StoreItem(lreq))
	assert.Nil(t, table.StoreItem(user))
	reqLogin = fmt.Sprintf(`{"request_pk": "%s", "otp":"%s"}`, lreq.PK, lreq.OTP)
	req = events.APIGatewayProxyRequest{
		Path:       "/prod1/login",
		HTTPMethod: "POST",
		RequestContext: events.APIGatewayProxyRequestContext{
			Stage: "prod1",
		},
		Body: reqLogin,
	}
	_, err = HandleLoginRequest(table, req)
	_, err = HandleLoginRequest(table, req)
	_, err = HandleLoginRequest(table, req)
	_, err = HandleLoginRequest(table, req)
	_, err = HandleLoginRequest(table, req)
	resp, err = HandleLoginRequest(table, req)
	assert.Nil(t, err)
	loginResp = &LoginResp{}
	err = json.Unmarshal([]byte(resp.Body), loginResp)
	assert.Nil(t, err)
	assert.False(t, loginResp.Ok)
}

func TestCmdMsgUpdateData(t *testing.T) {
	defer stopLocalDynamo()
	table := startLocalDynamo(t)
	domain := "foobar.com"
	connId := "someid="
	stage := "prod"
	user1, _ := NewUser("user1")
	msg1, _ := NewMsg("bot1", user1.PK, TGPhotoMsgKind)
	msg1.Data["text"] = "foobar"
	assert.Nil(t, table.StoreItem(msg1))
	reqCtx := getProxyContext("MESSAGE", domain, stage, connId, user1.PK)
	outCh := make(chan []byte)
	doneCh := make(chan bool)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	output := make([]string, 0)
	go collectOutput(ctx, &output, outCh, doneCh)
	input := fmt.Sprintf(`{"name":"msgupdate", "id":"fooid", "key":"text", "value":"spamegg", "pk":"%s"}`, msg1.PK)
	err := handleUserCmd(ctx, table, reqCtx, input, outCh)
	if assert.Nil(t, err) {
		doneCh <- true
	}
	if assert.Equal(t, 1, len(output)) {
		resp := make(map[string]interface{})
		assert.Nil(t, json.Unmarshal([]byte(output[0]), &resp))
		assert.Equal(t, "fooid", resp["id"].(string))
		assert.Equal(t, msg1.PK, resp["pk"].(string))
	}
	m := &Msg{}
	require.Nil(t, table.FetchItem(msg1.PK, m))
	assert.Equal(t, m.Data["text"].(string), "spamegg")
}

func TestCmdMsgUpdateUMS(t *testing.T) {
	defer stopLocalDynamo()
	table := startLocalDynamo(t)
	domain := "foobar.com"
	connId := "someid="
	stage := "prod"
	user1, _ := NewUser("user1")
	msg1, _ := NewMsg("bot1", user1.PK, TGPhotoMsgKind)
	msg1.Data["text"] = "foobar"
	assert.Nil(t, table.StoreItem(msg1))
	reqCtx := getProxyContext("MESSAGE", domain, stage, connId, user1.PK)
	outCh := make(chan []byte)
	doneCh := make(chan bool)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	output := make([]string, 0)
	go collectOutput(ctx, &output, outCh, doneCh)
	input := fmt.Sprintf(`{"name":"msgupdate", "id":"fooid", "key":"ums", "value":"user1#foo#2", "pk":"%s"}`, msg1.PK)
	err := handleUserCmd(ctx, table, reqCtx, input, outCh)
	if assert.Nil(t, err) {
		doneCh <- true
	}
	if assert.Equal(t, 1, len(output)) {
		resp := make(map[string]interface{})
		assert.Nil(t, json.Unmarshal([]byte(output[0]), &resp))
		assert.Equal(t, "fooid", resp["id"].(string))
		assert.Equal(t, msg1.PK, resp["pk"].(string))
	}
	m := &Msg{}
	require.Nil(t, table.FetchItem(msg1.PK, m))
	assert.Equal(t, m.UMS.String(), "user1#foo#2")
}

func TestCmdFetchTimeStamps(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	domain := "foobar.com"
	connId := "someid="
	stage := "prod"
	user1, _ := NewUser("user1")
	user2, _ := NewUser("user2")
	msg1, err := NewMsg("bot1", user1.PK, TGTextMsgKind, CreatedAtOp("-10d"), UserStatusOp(0),
		DataOp(map[string]interface{}{"text": "msg1"}))
	msg2, err := NewMsg("bot1", user1.PK, TGTextMsgKind, CreatedAtOp("-3d"), UserStatusOp(5),
		DataOp(map[string]interface{}{"text": "msg2"}))
	msg3, err := NewMsg("bot1", user1.PK, TGTextMsgKind, CreatedAtOp("-2d"), UserStatusOp(5),
		DataOp(map[string]interface{}{"text": "msg3"}))
	msg4, err := NewMsg("bot1", user2.PK, TGTextMsgKind, CreatedAtOp("-2d"), UserStatusOp(0),
		DataOp(map[string]interface{}{"text": "msg4"}))
	errs := testTable.StoreItems(msg1, msg2, msg3, msg4)
	for _, e := range errs {
		if e != nil {
			t.Error(e)
		}
	}
	reqCtx := getProxyContext("MESSAGE", domain, stage, connId, user1.PK)
	outCh := make(chan []byte)
	doneCh := make(chan bool)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	var output []string

	output = make([]string, 0)
	go collectOutput(ctx, &output, outCh, doneCh)
	start := time.Now().Unix() - 4*24*60*60
	end := time.Now().Unix() - 2.5*24*60*60
	input := fmt.Sprintf(
		`{"name":"msgfetchbytstamp", "id":"foo", "start":%d, "end": %d, "ums":"%s"}`,
		start, end, msg2.UMS.String())
	err = handleUserCmd(ctx, testTable, reqCtx, input, outCh)
	if assert.Nil(t, err) {
		doneCh <- true
	}
	require.Equal(t, 3, len(output))

	resp1 := make(map[string]interface{})
	err = json.Unmarshal([]byte(output[0]), &resp1)
	assert.Nil(t, err)

	resp2 := make(map[string]interface{})
	err = json.Unmarshal([]byte(output[1]), &resp2)
	assert.Nil(t, err)

	resp3 := make(map[string]interface{})
	err = json.Unmarshal([]byte(output[2]), &resp3)
	assert.Nil(t, err)

	assert.Equal(t, msg2.PK, resp2["pk"].(string))
}
