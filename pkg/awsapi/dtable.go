package awsapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	dattr "github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/segmentio/ksuid"
	"github.com/xlzd/gotp"
)

const (
	NO_SUCH_ITEM   = "NoSuchItem"
	ALREADY_EXISTS = "AlreadyExists"

	MsgKeyPrefix          = "msg#"
	EmailKeyPrefix        = "email#"
	UserKeyPrefix         = "user#"
	TelKeyPrefix          = "tel#"
	TGAccKeyPrefix        = "tgacc#"
	BotKeyPrefix          = "bot#"
	InviteKeyPrefix       = "inv#"
	SubscriptionKeyPrefix = "subs#"
	OrgKeyPrefix          = "org#"
	PermKeyPrefix         = "perm#"
	FolderKeyPrefix       = "fldr#"

	TGBotKind               = "tg"
	DummyBotKind            = "dummy"
	RecognizedTextFieldName = "text_recogn"
)

const (
	TGTextMsgKind     = 1
	TGVoiceMsgKind    = 2
	TGPhotoMsgKind    = 3
	TGUnknownMsgKind  = 4
	TGDocMsgKind      = 5
	FolderStreamKind  = 6
	FolderArchiveKind = 7
	FolderTrashKind   = 8
)

type Subscriptions []*Subscription

//Provides access to DynamoDB table
type DTable struct {
	db       *dynamodb.DynamoDB
	Name     string
	Region   string
	Endpoint string
	PKey     string
	Skey     string
}

func NewDTable(name string) (*DTable, error) {
	return &DTable{Name: name, PKey: "PK", Region: "us-west-2"}, nil
}

func (table *DTable) Connect() error {
	conf := &aws.Config{Region: aws.String(table.Region)}
	if table.Endpoint != "" {
		conf.Endpoint = aws.String(table.Endpoint)
	}
	sess, err := session.NewSession(conf)
	if err != nil {
		return err
	}
	table.db = dynamodb.New(sess)
	return nil
}

func (t *DTable) Create() error {
	input := createTableInput // TODO make it copy
	input.TableName = aws.String(t.Name)
	_, err := t.db.CreateTable(input)
	return err
}

func (t *DTable) EnableTTL() error {
	timeToLiveInput.TableName = aws.String(t.Name)
	_, err := t.db.UpdateTimeToLive(timeToLiveInput)
	return err
}

// Option to check uniqueness of stored item by cheking PK
func UniqueOp() func(*dynamodb.PutItemInput) error {
	return func(pii *dynamodb.PutItemInput) error {
		pii.ConditionExpression = aws.String("attribute_not_exists(PK)")
		return nil
	}
}

func (t *DTable) StoreItem(item interface{},
	options ...func(*dynamodb.PutItemInput) error) error {
	av, err := dattr.MarshalMap(item)
	if err != nil {
		return err
	}
	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(t.Name),
	}
	for _, ops := range options {
		err := ops(input)
		if err != nil {
			return err
		}
	}
	_, err = t.db.PutItem(input)
	if err != nil && strings.HasPrefix(err.Error(), "ConditionalCheckFailedException") {
		return errors.New(ALREADY_EXISTS)
	}
	return err
}

//Store bunch of items in transactions, make sure they are new
// by checking PK uniqueness
func (t *DTable) StoreInTransUniq(items ...interface{}) error {
	titems := []*dynamodb.TransactWriteItem{}
	for _, i := range items {
		av, err := dattr.MarshalMap(i)
		if err != nil {
			return err
		}
		titems = append(titems, &dynamodb.TransactWriteItem{
			Put: &dynamodb.Put{
				ConditionExpression: aws.String("attribute_not_exists(PK)"),
				Item:                av,
				TableName:           aws.String(t.Name),
			},
		})
	}
	_, err := t.db.TransactWriteItems(&dynamodb.TransactWriteItemsInput{
		TransactItems: titems})
	return err
}

func (t *DTable) StoreItems(items ...interface{}) []error {
	var output []error
	for _, item := range items {
		err := t.StoreItem(item)
		output = append(output, err)
	}
	return output
}

func (t *DTable) FetchItem(pk string, item interface{}) error {
	return t.FetchSubItem(pk, pk, item)
}

func (t *DTable) FetchSubItem(pk, sk string, item interface{}) error {
	result, err := t.db.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(t.Name),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(pk),
			},
			"SK": {
				S: aws.String(sk),
			},
		},
	})
	if err != nil {
		return err
	}
	if len(result.Item) == 0 {
		return errors.New(NO_SUCH_ITEM)
	}
	err = dattr.UnmarshalMap(result.Item, item)
	if err != nil {
		return err
	}

	return nil
}

func (t *DTable) FetchItemsWithPrefix(pk, prefix string, out interface{}) error {
	cond := "PK = :pk AND begins_with(SK, :prefix)"
	qi := &dynamodb.QueryInput{
		TableName:              aws.String(t.Name),
		KeyConditionExpression: aws.String(cond),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":prefix": {
				S: aws.String(prefix),
			},
			":pk": {
				S: aws.String(pk),
			},
		},
	}

	resp, err := t.db.Query(qi)
	if err != nil {
		return err
	}
	if len(resp.Items) > 0 {
		return dattr.UnmarshalListOfMaps(resp.Items, out)
	}
	return nil
}

func (t *DTable) UpdateItemData(pk, key string, value interface{}) (*dynamodb.UpdateItemOutput, error) {
	return t.UpdateItemMap(pk, pk, "D", key, value)
}

const UpdatedAtField = "updated_at"

func (t *DTable) UpdateItemMap(pk, sk, fName, key string, value interface{}) (*dynamodb.UpdateItemOutput, error) {
	val, err := dattr.Marshal(value)
	if err != nil {
		return nil, err
	}
	uii := &dynamodb.UpdateItemInput{
		TableName:    aws.String(t.Name),
		ReturnValues: aws.String("ALL_NEW"),
		ExpressionAttributeNames: map[string]*string{
			"#Data":      aws.String(fName),
			"#Key":       aws.String(key),
			"#UpdatedAt": aws.String(UpdatedAtField),
		},
		UpdateExpression: aws.String("SET #Data.#Key = :v, #Data.#UpdatedAt = :t"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":v": val,
			":t": {
				N: aws.String(fmt.Sprintf("%d", time.Now().Unix())),
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {S: aws.String(pk)},
			"SK": {S: aws.String(sk)},
		},
	}
	return t.db.UpdateItem(uii)
}

func (t *DTable) IncrProp(pk, sk, propName string, amount int64) error {
	exprVals, err := dattr.MarshalMap(map[string]int64{
		":v": amount},
	)
	if err != nil {
		return err
	}
	uii := &dynamodb.UpdateItemInput{
		TableName: aws.String(t.Name),
		ExpressionAttributeNames: map[string]*string{
			"#PropName": aws.String(propName),
		},
		UpdateExpression:          aws.String("ADD #PropName :v"),
		ExpressionAttributeValues: exprVals,
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {S: aws.String(pk)},
			"SK": {S: aws.String(sk)},
		},
	}
	_, err = t.db.UpdateItem(uii)
	return err
}

func (t *DTable) FetchTGAcc(tgid int, tgacc *TGAcc) error {
	pk := fmt.Sprintf("%s%d", TGAccKeyPrefix, tgid)
	return t.FetchItem(pk, tgacc)
}

func (t *DTable) QueryIndex(
	name string, cond string, exprValues map[string]interface{}) (*dynamodb.QueryOutput, error) {
	av, err := dattr.MarshalMap(exprValues)
	if err != nil {
		return nil, err
	}
	qi := &dynamodb.QueryInput{
		TableName:                 aws.String(t.Name),
		IndexName:                 aws.String(name),
		KeyConditionExpression:    aws.String(cond),
		ExpressionAttributeValues: av,
	}
	return t.db.Query(qi)
}

func (t *DTable) DeleteSubItem(pk, sk string) error {
	input := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(pk),
			},
			"SK": {
				S: aws.String(sk),
			},
		},
		TableName: aws.String(t.Name),
	}
	_, err := t.db.DeleteItem(input)
	return err
}

type UMSField struct {
	PK     string
	Status int64
}

func (ums *UMSField) String() string {
	return fmt.Sprintf("%s#%d", ums.PK, ums.Status)
}

func (ums *UMSField) MarshalDynamoDBAttributeValue(av *dynamodb.AttributeValue) error {
	s := ums.String()
	av.S = &s
	return nil
}

func (ums *UMSField) UnmarshalDynamoDBAttributeValue(av *dynamodb.AttributeValue) error {
	parts := strings.Split(*av.S, "#")
	if len(parts) != 3 {
		return errors.New(fmt.Sprintf("Could not parse %s", *av.S))
	}
	ums.PK = fmt.Sprintf("%s#%s", parts[0], parts[1])
	n, err := strconv.ParseInt(parts[2], 10, 0)
	if err != nil {
		return err
	}
	ums.Status = n
	return nil
}

func (ums *UMSField) Parse(s string) error {
	parts := strings.Split(s, "#")
	if len(parts) != 3 {
		return fmt.Errorf("Could not parse %s", s)

	}
	n, err := strconv.Atoi(parts[2])
	if err != nil {
		return err
	}
	ums.PK = fmt.Sprintf("%s#%s", parts[0], parts[1])
	ums.Status = int64(n)
	return nil
}

type Msg struct {
	PK        string
	SK        string
	ChannelPK string                 `dynamodbav:"Ch"`
	AuthorPK  string                 `dynamodbav:"A"`
	Kind      int64                  `dynamodbav:"K"`
	UMS       UMSField               `dynamodbav:"UMS"`
	CreatedAt int64                  `dynamodbav:"CRTD"`
	Data      map[string]interface{} `dynamodbav:"D"`
}

//Option for new msg
func CreatedAtOp(ts string) func(*Msg) error {
	return func(m *Msg) error {
		t, err := StrToTime(ts)
		if err != nil {
			return err
		}
		m.CreatedAt = t.Unix()
		return nil
	}
}

func UserStatusOp(s int) func(*Msg) error {
	return func(m *Msg) error {
		m.UMS.Status = int64(s)
		return nil
	}
}

func DataOp(d map[string]interface{}) func(m *Msg) error {
	return func(m *Msg) error {
		m.Data = d
		return nil
	}
}

//Factory method for Msg
func NewMsg(channel string, pk string, kind int64, options ...func(*Msg) error) (*Msg, error) {
	msg := &Msg{ChannelPK: channel, AuthorPK: pk, Kind: kind, CreatedAt: time.Now().Unix()}
	msg.Data = make(map[string]interface{})
	for _, opt := range options {
		err := opt(msg)
		if err != nil {
			return nil, err
		}
	}
	if msg.PK == "" {
		id, err := ksuid.NewRandomWithTime(time.Unix(msg.CreatedAt, 0))
		if err != nil {
			return nil, err
		}
		msg.PK = fmt.Sprintf("%s%s", MsgKeyPrefix, id.String())
		msg.CreatedAt = id.Time().Unix() //there is a bit difference from origin, we need this to be stored
	}

	msg.UMS.PK = msg.AuthorPK
	msg.SK = msg.PK
	return msg, nil
}

func (m *Msg) Reload(table *DTable) error {
	return table.FetchItem(m.PK, m)
}

func (m *Msg) UpdatedAt() int64 {
	var updated float64
	if m.Data != nil {
		updated, _ = m.Data[UpdatedAtField].(float64)
	}
	if updated != 0 {
		return int64(updated)
	}
	return m.CreatedAt
}

//Represents list of Msg
type ListMsg struct {
	Items            map[string]*Msg
	LastEvaluatedKey map[string]*dynamodb.AttributeValue
}

func NewListMsg() *ListMsg {
	return &ListMsg{map[string]*Msg{}, map[string]*dynamodb.AttributeValue{}}
}

func (lm *ListMsg) Len() int {
	return len(lm.Items)
}

func (lm *ListMsg) FetchByUserStatus(t *DTable, userPK string, status int, start, end string) error {
	ums := fmt.Sprintf("%s#%d", userPK, status)

	start_time, err := StrToTime(start)
	if err != nil {
		return err
	}
	end_time, err := StrToTime(end)
	if err != nil {
		return err
	}
	return lm.FetchByUMS(t, userPK, ums, start_time.Unix(), end_time.Unix())
}

func (lm *ListMsg) FetchByUMS(t *DTable, userPK, ums string, start, end int64) error {
	exprValues := map[string]interface{}{":ums": ums, ":start": start, ":end": end}
	resp, err := t.QueryIndex("UMSIndex", "UMS = :ums and CRTD BETWEEN :start AND :end", exprValues)
	if err != nil {
		return err
	}
	for _, item := range resp.Items {
		msg := &Msg{}
		err = dattr.UnmarshalMap(item, msg)
		if err != nil {
			continue
		}
		lm.Items[msg.PK] = msg
	}
	return nil
}

func (lm *ListMsg) GetPKs() []string {
	keys := make([]string, 0, len(lm.Items))
	for k := range lm.Items {
		keys = append(keys, k)
	}
	return keys
}

func (lm *ListMsg) Asc() []*Msg {
	result := make([]*Msg, 0, len(lm.Items))
	ids := lm.GetPKs()
	sort.Strings(ids)
	for _, k := range ids {
		result = append(result, lm.Items[k])
	}
	return result
}

func (lm *ListMsg) Desc() []*Msg {
	result := make([]*Msg, 0, len(lm.Items))
	ids := lm.GetPKs()
	sort.Sort(sort.Reverse(sort.StringSlice(ids)))
	for _, k := range ids {
		result = append(result, lm.Items[k])
	}
	return result
}

type User struct {
	PK        string
	SK        string
	Title     string   `dynamodbav:"T"`
	Email     string   `dynamodbav:"E"`
	Tel       string   `dynamodbav:"TL"`
	Bots      []string `dynamodbav:"B,set"`
	TGID      string
	CreatedAt int64                  `dynamodbav:"CRTD"`
	Data      map[string]interface{} `dynamodbav:"D"`
}

func NewUser(title string) (*User, error) {
	user := &User{Title: title}
	user.Data = make(map[string]interface{})
	kid := ksuid.New()
	user.CreatedAt = int64(time.Now().Unix())
	user.PK = fmt.Sprintf("%s%s", UserKeyPrefix, kid.String())
	user.SK = user.PK
	return user, nil
}

func (u *User) SetTel(t string) error {
	tel, err := NewTel(t, u.PK)
	if err != nil {
		return err
	}
	u.Tel = tel.String()
	return nil
}

func (u *User) SetEmail(e string) error {
	email, err := NewEmail(e, u.PK)
	if err != nil {
		return err
	}
	u.Email = email.String()
	return nil
}

// interface for telebot
func (u *User) Recipient() string {
	return u.TGID
}

func (t *Tel) String() string {
	return t.Number
}

func (e *Email) String() string {
	return strings.Replace(e.PK, EmailKeyPrefix, "", 1)
}

func (t *TGAcc) String() string {
	return t.TGID
}

func (t *DTable) StoreUserTG(user *User, tgid int, bot *Bot) error {
	tg, err := NewTGAcc(tgid, user.PK)
	if err != nil {
		return err
	}
	if user.Bots == nil {
		user.Bots = []string{bot.PK}
	} else {
		user.Bots = append(user.Bots, bot.PK)

	}

	err = t.StoreItem(tg, UniqueOp())
	if err != nil {
		return err
	}
	user.TGID = tg.TGID
	err = t.StoreItem(user)
	return err
}

//Store user, telephon number, email in one transaction
//it fails if number or email already exist
func (t *DTable) StoreNewUser(user *User) error {
	items := []interface{}{user}
	if user.Tel != "" {
		tel, err := NewTel(user.Tel, user.PK)
		if err != nil {
			return err
		}
		items = append(items, tel)
	}
	if user.Email != "" {
		email, err := NewEmail(user.Email, user.PK)
		if err != nil {
			return err
		}
		items = append(items, email)
	}
	return t.StoreInTransUniq(items...)
}

func (u *User) FetchWSConns(table *DTable, out interface{}) error {
	return table.FetchItemsWithPrefix(u.PK, WSConnKeyPrefix, out)
}

type Email struct {
	PK        string //email#foo@bar.com
	SK        string
	OwnerPK   string `dynamodbav:"O"`
	CreatedAt int64  `dynamodbav:"CRTD"`
}

func NewEmail(email, owner_pk string) (*Email, error) {
	e := &Email{PK: fmt.Sprintf("%s%s", EmailKeyPrefix, email), OwnerPK: owner_pk,
		CreatedAt: time.Now().Unix()}
	e.SK = e.PK
	return e, nil
}

//For telephone number
type Tel struct {
	PK        string
	SK        string
	Number    string `dynamodbav:"NMBR"`
	OwnerPK   string `dynamodbav:"O"`
	CreatedAt int64  `dynamodbav:"CRTD"`
}

func NewTel(number, owner_pk string) (*Tel, error) {
	tel := &Tel{PK: fmt.Sprintf("%s%s", TelKeyPrefix, number),
		Number: number, OwnerPK: owner_pk, CreatedAt: time.Now().Unix()}
	tel.SK = tel.PK
	return tel, nil
}

//For Telegram account
type TGAcc struct {
	PK        string
	SK        string
	TGID      string                 `dynamodbav:"ID"`
	OwnerPK   string                 `dynamodbav:"O"`
	CreatedAt int64                  `dynamodbav:"CRTD"`
	Data      map[string]interface{} `dynamodbav:"D"`
}

func NewTGAcc(tgid int, owner_pk string) (*TGAcc, error) {
	tacc := &TGAcc{PK: fmt.Sprintf("%s%d", TGAccKeyPrefix, tgid),
		TGID:    fmt.Sprintf("%d", tgid),
		OwnerPK: owner_pk, CreatedAt: time.Now().Unix(),
		Data: make(map[string]interface{})}
	tacc.SK = tacc.PK
	return tacc, nil
}

type Bot struct {
	PK        string
	SK        string
	Name      string                 `dynamodbav:"N"`
	Kind      string                 `dynamodbav:"K"`
	Secret    string                 `dynamodbav:"S"`
	CreatedAt int64                  `dynamodbav:"CRTD"`
	Data      map[string]interface{} `dynamodbav:"D"`
}

func GetBotPK(kind, name string) string {
	return fmt.Sprintf("%s%s#%s", BotKeyPrefix, name, kind)
}

func NewBot(kind, name string) (*Bot, error) {
	bot := &Bot{PK: GetBotPK(kind, name),
		Kind: kind, Name: name, Data: make(map[string]interface{})}
	bot.CreatedAt = int64(time.Now().Unix())
	bot.SK = bot.PK
	return bot, nil
}

func (b *Bot) InviteUrl(otp string) string {
	return fmt.Sprintf("%s/%s?start=%s", "https://t.me", b.Name, otp)
}

type Invite struct {
	PK        string
	SK        string
	BotPK     string `dynamodbav:"B"`
	UserPK    string `dynamodbav:"U"`
	OTP       string `dynamodbav:"OTP"`
	CreatedAt int64  `dynamodbav:"CRTD"`
	TTL       int64
	Url       string
	Data      map[string]interface{}
}

func NewInvite(u *User, b *Bot, valid int) (*Invite, error) {
	inv := &Invite{
		UserPK:    u.PK,
		BotPK:     b.PK,
		CreatedAt: time.Now().Unix(),
		TTL:       int64(valid)*60*60 + time.Now().Unix(),
	}
	inv.OTP = gotp.NewDefaultTOTP(gotp.RandomSecret(16)).Now()
	inv.Data = make(map[string]interface{})
	inv.Url = b.InviteUrl(inv.OTP)
	pk, err := MakeInvPK(b, inv.OTP)
	if err != nil {
		return nil, err
	}
	inv.PK = pk
	inv.SK = pk
	return inv, nil
}

func (inv *Invite) IsValid() bool {
	if inv.TTL > time.Now().Unix() {
		return true
	}
	return false
}

func MakeInvPK(bot *Bot, code string) (string, error) {
	if bot.PK == "" {
		return "", errors.New("Bot.PK is empty")
	}
	return fmt.Sprintf("%s%s#%s", InviteKeyPrefix, bot.PK, code), nil
}

func (t *DTable) FetchInvite(bot *Bot, code string, inv *Invite) error {
	pk, err := MakeInvPK(bot, code)
	if err != nil {
		return err
	}
	err = t.FetchItem(pk, inv)
	if err != nil {
		//	fmt.Printf("Could not fetch invite for PK %s", pk)
		return err
	}
	if !inv.IsValid() {
		fmt.Printf("%#v is expired \n", inv)
		return errors.New(NO_SUCH_ITEM)
	}
	if inv.Data == nil {
		inv.Data = make(map[string]interface{})
	}
	return nil
}

func PK2ID(prefix, pk string) string {
	return strings.Replace(pk, prefix, "", 1)
}

const TokenKeyPrefix = "token#"

type Token struct {
	PK     string
	SK     string
	UserPK string `dynamodbav:"U"`
	TTL    int64
	ONEOFF bool `dynamodbav:"OF"`
}

func NewToken(u *User, valid int) (*Token, error) {
	pk := fmt.Sprintf("%s%s", TokenKeyPrefix, ksuid.New())
	s := &Token{PK: pk, SK: pk, UserPK: u.PK, TTL: time.Now().Unix() + int64(valid*60*60)}
	return s, nil
}

func (s *Token) IsValid() bool {
	return time.Now().Unix() < s.TTL
}

func (s *Token) Id() string {
	return PK2ID(TokenKeyPrefix, s.PK)
}

const WSConnKeyPrefix = "wsconn#"

type WSConn struct {
	PK         string
	SK         string
	TTL        int64
	DomainName string `dynamodbav":"D"`
	Stage      string `dynamodbav":"S"`
	CreatedAt  int64  `dynamodbav":"CRTD"`
}

func NewWSConn(userPK, id, domain, stage string) (*WSConn, error) {
	created := time.Now().Unix()
	c := &WSConn{PK: userPK,
		SK:         fmt.Sprintf("%s%s", WSConnKeyPrefix, id),
		TTL:        (created + 24*60*60),
		DomainName: domain,
		Stage:      stage,
		CreatedAt:  created}
	return c, nil
}

func (c *WSConn) Endpoint() string {
	return fmt.Sprintf("https://%s/%s", c.DomainName, c.Stage)
}

func (c *WSConn) Id() string {
	return PK2ID(WSConnKeyPrefix, c.SK)
}

func (c *WSConn) Send(data []byte) error {
	sender, _ := NewWSSender(c.Endpoint(), c.Id(), nil)

	return sender.Send(data)
}

type Subscription struct {
	PK        string
	SK        string
	OwnerPK   string            `dynamodbav:"O"`
	Domain    string            `dynamodbav:"DN"`
	Stage     string            `dynamodbav:"S"`
	UMS       string            `dynamodbav:"U"` // we don't need it in index
	CreatedAt int64             `dynamodbav:"CRTD"`
	TTL       int64             `dynamodbav:"TTL"`
	Data      map[string]string `dynamodbav:"D,omitempty"`
}

// we have to store 2 objects in db for one subcription
// A) PK UserPK, SK Prefix+ConnId
// B) PK UMS, SK Prefix+ConnId
func NewSubscription(ownerPK, umsPK string, umsStatus int, domain, stage, connId string) (
	*Subscription, *Subscription, error) {
	ums := fmt.Sprintf("%s#%d", umsPK, umsStatus)
	sA := &Subscription{}
	sA.UMS = ums
	sA.Domain = domain
	sA.Stage = stage
	sA.TTL = time.Now().Unix() + 24*60*60
	sA.CreatedAt = time.Now().Unix()
	sA.PK = ownerPK
	sA.SK = fmt.Sprintf("%s%s", SubscriptionKeyPrefix, connId)
	sB := &Subscription{UMS: ums, Domain: domain, Stage: stage, TTL: sA.TTL, CreatedAt: sA.CreatedAt}
	sB.PK = ums
	sB.SK = sA.SK
	sB.OwnerPK = ownerPK
	return sA, sB, nil
}

func (s *Subscription) Endpoint() string {
	return fmt.Sprintf("%s/%s", s.Domain, s.Stage)
}

func (s *Subscription) ConnectionId() string {
	return PK2ID(SubscriptionKeyPrefix, s.SK)
}

type SubscrEvent struct {
	EventName string `json:"event_name"`
	Name      string `json:"name"`
	PK        string `json:"pk"`
	UMS       string `json:"ums"`
	MsgKind   int64  `json:"kind"`
}

func (s *Subscription) SendDBEvent(pk, name, ums string, kind int64) error {
	sender, _ := NewWSSender(s.Endpoint(), s.ConnectionId(), nil)
	event := SubscrEvent{PK: pk, EventName: name, Name: "dbevent", UMS: ums, MsgKind: kind}
	data, err := json.Marshal(event)
	fmt.Printf("Sending %s to %s", data, s.ConnectionId())
	if err != nil {
		return err
	}
	return sender.Send(data)
}

const (
	MsgFileKeyPrefix    = "file#"
	FileKindTgThumb     = "thumb"
	FileKindTgMediumPic = "mediumpic"
	FileKindTgBigPic    = "bigpic"
	FileKindTgVoice     = "voice"
)

type MsgFile struct {
	PK        string
	SK        string
	FileKind  string                 `dynamodbav:"FK"`
	CreatedAt int64                  `dynamodbav:"CRTD"`
	Data      map[string]interface{} `dynamodbav:"D,omitempty"`
	Mime      string                 `dynamodbav:"M"`
	Bucket    string                 `dynamodbav:"B"`
	Key       string                 `dynamodbav:"K"`
}

func NewMsgFile(pk, kind, mime, bucket, key string) (*MsgFile, error) {
	f := &MsgFile{}
	f.PK = pk
	f.SK = fmt.Sprintf("%s%s", MsgFileKeyPrefix, kind)
	f.Mime = mime
	f.FileKind = kind
	f.Bucket = bucket
	f.Key = key
	f.CreatedAt = time.Now().Unix()
	f.Data = make(map[string]interface{})
	return f, nil
}

const LoginRequestKeyPrefix = "inreq#"

type LoginRequest struct {
	PK        string
	SK        string
	UserPK    string
	CreatedAt int64 `dynamodbav:"CRTD"`
	OTP       string
	Status    int64 // 0 just created, 1 accepted, 2 too many attemts
	TTL       int64
	Attempts  int64 `dynamodbav:"A"`
}

func NewLoginRequest(userPK string) (*LoginRequest, error) {
	req := &LoginRequest{}
	req.PK = fmt.Sprintf("%s%s", LoginRequestKeyPrefix, ksuid.New().String())
	req.SK = req.PK
	req.UserPK = userPK
	req.CreatedAt = time.Now().Unix()
	req.TTL = req.CreatedAt + 20*60
	req.OTP = gotp.NewDefaultTOTP(gotp.RandomSecret(16)).Now()
	return req, nil
}

const (
	OTP_EXPIRED       = "OTP_EXPIRED"
	TOO_MANY_ATTEMPTS = "TOO_MANY_ATTEMPTS"
	OTP_WRONG         = "OTP_WRONG"
)

func (req *LoginRequest) IsOTPValid(otp string) (bool, string) {
	if req.Attempts >= 5 {
		return false, TOO_MANY_ATTEMPTS
	}

	if req.TTL <= time.Now().Unix() {
		return false, OTP_EXPIRED
	}

	if req.OTP == otp {
		return true, "ok"
	}
	return false, OTP_WRONG

}

type Org struct {
	PK        string
	SK        string
	Title     string                 `dynamodbav:"T"`
	Motto     string                 `dynamodbav:"M"`
	Admins    []string               `dynamodbav:"ADM"`
	CreatedAt int64                  `dynamodbav:"CRTD"`
	Data      map[string]interface{} `dynamodbav:"D,omitempty"`
}

func NewOrg(title, motto string, admins []*User) (*Org, error) {
	org := &Org{}
	org.Title = title
	org.Motto = motto
	org.PK = fmt.Sprintf("%s%s", OrgKeyPrefix, ksuid.New())
	org.SK = org.PK
	org.CreatedAt = time.Now().Unix()
	for _, u := range admins {
		org.Admins = append(org.Admins, u.PK)
	}
	org.Data = make(map[string]interface{})
	return org, nil
}

type UserPerm struct {
	PK        string
	SK        string
	CreatedAt int64  `dynamodbav:"CRTD"`
	Value     string `dynamodbav:"V"`
}

func checkPermValue(value string) error {
	if len(value) < 1 || len(value) > 4 {
		return errors.New("Perm value must get lenght between 0 and 5")
	}
	if value[0] != 't' {
		return errors.New("First letter of value must be t")
	}
	if len(value) > 1 && value[1] != 'r' {
		return errors.New("Second letter of value must be r")
	}
	if len(value) > 2 && value[2] != 'w' {
		return errors.New("Third letter of value must be w")
	}
	if len(value) > 3 && value[3] != 'a' {
		return errors.New("Forth letter of value must be a")
	}
	return nil
}

func NewUserPerm(userPK string, f *Folder, value string) (*UserPerm, error) {
	if err := checkPermValue(value); err != nil {
		return nil, err
	}
	perm := &UserPerm{}
	perm.PK = userPK
	perm.SK = fmt.Sprintf("%s%s#%s#%s", PermKeyPrefix, f.PK, f.SK, value)
	return perm, nil
}

type Folder struct {
	PK        string
	SK        string
	Title     string `dynamodbav:"T"`
	Kind      int64  `dynamodbav:"K"`
	CreatedAt int64  `dynamodbav:"CRTD"`
}

func NewFolder(ownerPK, title string, id, kind int64) (*Folder, error) {
	sk := fmt.Sprintf("%s%d", FolderKeyPrefix, id)
	f := &Folder{}
	f.PK = ownerPK
	f.SK = sk
	f.Title = title
	f.Kind = kind
	f.CreatedAt = time.Now().Unix()
	return f, nil
}

func (u *User) HasPerm(table *DTable, folderPK, folderSK, value string) (bool, error) {
	if err := checkPermValue(value); err != nil {
		return false, err
	}
	var perms []*UserPerm
	if folderPK == u.PK {
		return true, nil
	}
	prefix := fmt.Sprintf("%s%s#%s#%s", PermKeyPrefix, folderPK, folderSK, value)
	if err := table.FetchItemsWithPrefix(u.PK, prefix, &perms); err != nil {
		return false, err
	}
	if len(perms) == 1 {
		return true, nil
	}
	return false, nil
}

func (f *Folder) UserCanRead(table *DTable, u *User) (bool, error) {
	return u.HasPerm(table, f.PK, f.SK, "tr")
}

func (f *Folder) UserCanTarget(table *DTable, u *User) (bool, error) {
	return u.HasPerm(table, f.PK, f.SK, "t")
}

func (f *Folder) UserCanWrite(table *DTable, u *User) (bool, error) {
	return u.HasPerm(table, f.PK, f.SK, "trw")
}

func (f *Folder) UserCanAdmin(table *DTable, u *User) (bool, error) {
	return u.HasPerm(table, f.PK, f.SK, "trwa")
}

// Ensure User account has 4 default folders:
// INBOX, Archive, Selected, Trash
func (u *User) EnsureDefaultFolders(table *DTable) error {

	inbox, _ := NewFolder(u.PK, "INBOX", 0, FolderStreamKind)
	archive, _ := NewFolder(u.PK, "Archive", 1, FolderArchiveKind)
	selected, _ := NewFolder(u.PK, "Selected", 2, FolderStreamKind)
	trash, _ := NewFolder(u.PK, "Trash", 3, FolderTrashKind)

	folderMap := make(map[string]*Folder)
	folderMap[inbox.SK] = inbox
	folderMap[archive.SK] = archive
	folderMap[selected.SK] = selected
	folderMap[trash.SK] = trash

	var existed []*Folder
	if err := table.FetchItemsWithPrefix(u.PK, FolderKeyPrefix, &existed); err != nil {
		return err
	}
	existedMap := make(map[string]bool)
	for _, f := range existed {
		existedMap[f.SK] = true
	}

	for sk, folder := range folderMap {
		if _, exists := existedMap[sk]; !exists {
			if err := table.StoreItem(folder); err != nil {
				return err
			}
		}
	}
	return nil
}

func (table *DTable) FetchFolderViews(userPK string, folderView *[]FolderView) error {
	var folders []*Folder
	if err := table.FetchItemsWithPrefix(userPK, FolderKeyPrefix, &folders); err != nil {
		return err
	}
	for _, folder := range folders {
		folderID := PK2ID(FolderKeyPrefix, folder.SK)
		*folderView = append(*folderView, FolderView{
			Title: folder.Title,
			UMS:   fmt.Sprintf("%s#%s", folder.PK, folderID),
			Kind:  folder.Kind,
		})
	}
	return nil
}
