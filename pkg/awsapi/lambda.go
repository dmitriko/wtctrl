package awsapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
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

func clearWSConn(table *DTable, connId, userPK string) error {
	return table.DeletSubItem(userPK, fmt.Sprintf("%s%s", WSConnKeyPrefix, connId))
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
	Id     string `json:"id,omitempty"`
	Body   string `json:"body"`
	Status string `json:"status"`
	SecNum int    `json:"number,omitempty"`
}

type UserCmd interface {
	Perform(context.Context, *DTable, events.APIGatewayWebsocketProxyRequestContext, chan<- []byte, chan<- error)
}

type PingCmd struct {
	Name string `json:"name"`
}

func (cmd *PingCmd) Perform(
	ctx context.Context, table *DTable, reqCtx events.APIGatewayWebsocketProxyRequestContext, out chan<- []byte, done chan<- error) {
	done <- sendWithContext(ctx, out, &CmdResp{
		Name:   cmd.Name,
		Status: "done",
		Body:   "pong",
	})
}

type MsgFetchByDays struct {
	Name   string `json:"name"`
	Days   int    `json:"days"`
	Status int    `json:"status"`
	Desc   bool   `json:"desc"`
}

func (cmd *MsgFetchByDays) GetName() string {
	return cmd.Name
}

func MsgView(msg *Msg) ([]byte, error) {
	out := make(map[string]interface{})
	out["PK"] = msg.PK
	out["owner"] = msg.AuthorPK
	out["created"] = msg.CreatedAt
	out["data"] = msg.Data
	out["status"] = msg.UMS.Status
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
	for _, m := range sortMeth() {
		b, err := MsgView(m)
		if err == nil {
			out <- b
		} else {
			fmt.Println("ERROR", err.Error())
		}
	}
	if err != nil {
		done <- err
		return
	}
	done <- nil
}

func UnmarshalCmd(data []byte) (UserCmd, error) {
	cmds := map[string]UserCmd{
		"ping":           &PingCmd{},
		"msgfetchbydays": &MsgFetchByDays{},
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
		return nil, fmt.Errorf("Uknown command, %s", string(data))
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
