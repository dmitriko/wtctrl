package awsapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	apimngmt "github.com/aws/aws-sdk-go/service/apigatewaymanagementapi"
)

var lambdaDebug bool

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

//Clears WSConn and Subscriptions related to that connection
// Don't worry too much about errors since TTL is there, but anyway
func clearWSConn(table *DTable, connId, userPK string) error {
	var err1, err2, err3, err4 error
	err1 = table.DeleteSubItem(userPK, fmt.Sprintf("%s%s", WSConnKeyPrefix, connId))
	sA := &Subscription{}
	err2 = table.FetchSubItem(userPK, fmt.Sprintf("%s%s", SubscriptionKeyPrefix, connId), sA)
	if err2 == nil {
		err3 = table.DeleteSubItem(sA.UMS, sA.SK) //deletes part B
		err4 = table.DeleteSubItem(sA.PK, sA.SK)
	}
	if err2 != nil && err2.Error() == NO_SUCH_ITEM {
		return err1
	}

	var out []string
	errs := []error{err1, err2, err3, err4}
	for _, e := range errs {
		if e != nil {
			out = append(out, e.Error())
		}
	}
	if len(out) > 0 {
		return errors.New(strings.Join(out, ", "))
	}
	return nil
}

//Exracts User's PK from Authorizer property of ProxyRequestContext
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

//Handles websocket connected/disconnected
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

type CmdResp struct {
	Name   string `json:"name"`
	Id     string `json:"id"`
	Status string `json:"status"`
	SecNum int    `json:"number,omitempty"`
	Error  string `json:"error,omitempty"`
}

type UserCmd interface {
	Perform(context.Context, *DTable, events.APIGatewayWebsocketProxyRequestContext, chan<- []byte, chan<- error)
}

type PingCmd struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}

func (cmd *PingCmd) Perform(
	ctx context.Context, table *DTable, reqCtx events.APIGatewayWebsocketProxyRequestContext, out chan<- []byte, done chan<- error) {
	done <- sendWithContext(ctx, out, &CmdResp{
		Id:     cmd.Id,
		Status: "pong",
	})
}

type MsgFetchByDays struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Days   int    `json:"days"`
	Status int    `json:"status"`
	Desc   bool   `json:"desc"`
}

//It shows PK, CreatedAt, Status and Kind
func MsgIndexView(msg *Msg) ([]byte, error) {
	out := make(map[string]interface{})
	out["PK"] = msg.PK
	out["created"] = msg.CreatedAt
	out["owner"] = msg.UMS.PK
	out["status"] = msg.UMS.Status
	out["kind"] = msg.Kind
	out["name"] = "msg_index"
	return json.Marshal(out)
}

func MsgView(msg *Msg, files []*MsgFile) ([]byte, error) {
	out := make(map[string]interface{})
	out["PK"] = msg.PK
	out["created"] = msg.CreatedAt
	out["owner"] = msg.UMS.PK
	out["status"] = msg.UMS.Status
	out["kind"] = msg.Kind
	out["name"] = "msg_index"
	return json.Marshal(out)
}

func (cmd *MsgFetchByDays) Perform(
	ctx context.Context, table *DTable, reqCtx events.APIGatewayWebsocketProxyRequestContext, out chan<- []byte, done chan<- error) {
	userPK, err := extractUserPK(reqCtx)
	if err != nil {
		done <- err
		return
	}
	start := fmt.Sprintf("-%dd", cmd.Days)
	listMsg := NewListMsg()
	err = listMsg.FetchByUserStatus(table, userPK, cmd.Status, start, "now")
	var sortMeth func() []*Msg
	if cmd.Desc {
		sortMeth = listMsg.Desc
	} else {
		sortMeth = listMsg.Asc
	}
	_ = sendWithContext(ctx, out, &CmdResp{
		Id:     cmd.Id,
		Name:   cmd.Name,
		Status: "started",
	})
	for _, m := range sortMeth() {
		b, err := MsgIndexView(m)
		if err == nil {
			select {
			case <-ctx.Done():
				fmt.Println("ERROR", ctx.Err())
				return
			case out <- b:
			}
		} else {
			fmt.Println("ERROR", err.Error())
		}
	}
	if err != nil {
		done <- err
		return
	}
	done <- sendWithContext(ctx, out, &CmdResp{
		Id:     cmd.Id,
		Name:   cmd.Name,
		Status: "done",
	})
}

type UnsubscribeCmd struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	UMSPK     string `json:"umspk"`
	MsgStatus int64  `json:"status"`
}

func (cmd *UnsubscribeCmd) Perform(
	ctx context.Context, table *DTable, reqCtx events.APIGatewayWebsocketProxyRequestContext,
	out chan<- []byte, done chan<- error) {
	userPK, err := extractUserPK(reqCtx)
	if err != nil {
		done <- err
		return
	}
	connId := reqCtx.ConnectionID
	ums := fmt.Sprintf("%s#%d", cmd.UMSPK, cmd.MsgStatus)
	sk := fmt.Sprintf("%s%s", SubscriptionKeyPrefix, connId)
	_ = table.DeleteSubItem(userPK, sk)
	_ = table.DeleteSubItem(ums, sk)

	done <- sendWithContext(ctx, out, &CmdResp{
		Id:     cmd.Id,
		Status: "ok",
	})
}

type SubscribeCmd struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	UMSPK     string `json:"umspk"`
	MsgStatus int    `json:"status"`
}

func (cmd *SubscribeCmd) Perform(
	ctx context.Context, table *DTable, reqCtx events.APIGatewayWebsocketProxyRequestContext,
	out chan<- []byte, done chan<- error) {
	userPK, err := extractUserPK(reqCtx)
	if err != nil {
		done <- err
		return
	}
	if userPK != cmd.UMSPK {
		done <- sendWithContext(ctx, out, &CmdResp{
			Id:     cmd.Id,
			Status: "error",
			Name:   cmd.Name,
			Error:  "no permissions",
		})
		return
	}
	sa, sb, _ := NewSubscription(userPK, cmd.UMSPK, cmd.MsgStatus, reqCtx.DomainName, reqCtx.Stage, reqCtx.ConnectionID)
	err = table.StoreItem(sa)
	if err != nil {
		done <- err
		return
	}
	err = table.StoreItem(sb)
	if err != nil {
		done <- err
		return
	}
	done <- sendWithContext(ctx, out, &CmdResp{
		Id:     cmd.Id,
		Status: "ok",
		Name:   cmd.Name,
	})
}

func UnmarshalCmd(data []byte) (UserCmd, error) {
	cmds := map[string]UserCmd{
		"ping":           &PingCmd{},
		"msgfetchbydays": &MsgFetchByDays{},
		"subscr":         &SubscribeCmd{},
		"unsubscr":       &UnsubscribeCmd{},
	}
	var s struct {
		Name string `json:"name"`
	}
	err := json.Unmarshal(data, &s)
	if err != nil {
		return nil, err
	}
	c, ok := cmds[s.Name]
	if !ok {
		return nil, fmt.Errorf("Unkown command, %s", string(data))
	}
	err = json.Unmarshal(data, c)
	if err != nil {
		return nil, err
	}
	return c, nil
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
func handleUserCmd(ctx context.Context, table *DTable,
	reqCtx events.APIGatewayWebsocketProxyRequestContext, cmd string, outCh chan<- []byte) error {
	var err error
	if lambdaDebug {
		fmt.Println("Handling user wire", cmd)
	}
	userCmd, err := UnmarshalCmd([]byte(cmd))
	if err != nil {
		return err
	}
	doneCh := make(chan error)
	go userCmd.Perform(ctx, table, reqCtx, outCh, doneCh)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-doneCh:
		return err
	}
}

type WSSender struct {
	Endpoint string
	ConnId   string
	ToUserCh <-chan []byte
	Sess     *session.Session
}

func NewWSSender(endpoint, connId string, toUserCh <-chan []byte) (*WSSender, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	return &WSSender{Endpoint: endpoint, ConnId: connId, ToUserCh: toUserCh, Sess: sess}, nil
}

func (s *WSSender) Start(ctx context.Context, doneCh <-chan bool) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-doneCh:
			return
		case data := <-s.ToUserCh:
			err := s.Send(data)
			if err != nil {
				fmt.Println("ERROR", err.Error())
				return
			}
		}
	}
}

func (s *WSSender) Send(data []byte) error {
	conf := &aws.Config{Endpoint: aws.String(s.Endpoint)}
	if s.Sess.Config.Region != nil {
		conf.Region = s.Sess.Config.Region
	} else {
		conf.Region = aws.String(os.Getenv("AWS_REGION"))
	}
	api := apimngmt.New(s.Sess, conf)
	_, err := api.PostToConnection(&apimngmt.PostToConnectionInput{
		ConnectionId: aws.String(s.ConnId),
		Data:         data,
	})
	return err
}

func HandleWSDefaultReq(req events.APIGatewayWebsocketProxyRequest, table *DTable) (
	events.APIGatewayProxyResponse, error) {
	resp := events.APIGatewayProxyResponse{StatusCode: 400}
	ctx, cancel := context.WithTimeout(context.Background(), 28*time.Second)
	defer cancel()
	toUserCh := make(chan []byte)
	stopSendingCh := make(chan bool)
	connId := req.RequestContext.ConnectionID
	endpoint := fmt.Sprintf("https://%s/%s", req.RequestContext.DomainName, req.RequestContext.Stage)
	sender, err := NewWSSender(endpoint, connId, toUserCh)
	if err != nil {
		return resp, err
	}
	go sender.Start(ctx, stopSendingCh)
	err = handleUserCmd(ctx, table, req.RequestContext, req.Body, toUserCh)
	stopSendingCh <- true
	if err != nil {
		fmt.Println("ERROR", err.Error())
		return resp, err
	}

	resp.StatusCode = http.StatusOK
	resp.Body = "ok"
	return resp, nil
}
