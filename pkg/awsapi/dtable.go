package awsapi

import (
	"errors"
	"fmt"
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

	MsgKeyPrefix    = "msg#"
	EmailKeyPrefix  = "email#"
	UserKeyPrefix   = "user#"
	TelKeyPrefix    = "tel#"
	TGAccKeyPrefix  = "tgacc#"
	BotKeyPrefix    = "bot#"
	InviteKeyPrefix = "inv#"
	FileKeyPrefix   = "file#"

	PicFileKind = "PicFileKind"

	TGBotKind               = "tg"
	RecognizedTextFieldName = "text_recogn"
)

const (
	TGTextMsgKind    = 1
	TGVoiceMsgKind   = 2
	TGPhotoMsgKind   = 3
	TGUnknownMsgKind = 4
)

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
	result, err := t.db.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(t.Name),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(pk),
			},
			"SK": {
				S: aws.String(pk),
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

func (t *DTable) UpdateItemData(pk, key, value string) (*dynamodb.UpdateItemOutput, error) {
	return t.UpdateItemMap(pk, pk, "D", key, value)
}

func (t *DTable) UpdateItemMap(pk, sk, fName, key, value string) (*dynamodb.UpdateItemOutput, error) {
	uii := &dynamodb.UpdateItemInput{
		TableName:    aws.String(t.Name),
		ReturnValues: aws.String("ALL_NEW"),
		ExpressionAttributeNames: map[string]*string{
			"#Data": aws.String(fName),
			"#Key":  aws.String(key),
		},
		UpdateExpression: aws.String("SET #Data.#Key = :v"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":v": {
				S: aws.String(value),
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {S: aws.String(pk)},
			"SK": {S: aws.String(sk)},
		},
	}
	return t.db.UpdateItem(uii)
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

type UMSField struct {
	PK     string
	Status int64
}

func (ums *UMSField) MarshalDynamoDBAttributeValue(av *dynamodb.AttributeValue) error {
	s := fmt.Sprintf("%s#%d", ums.PK, ums.Status)
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

type Msg struct {
	PK        string
	SK        string
	ChannelPK string            `dynamodbav:"Ch"`
	AuthorPK  string            `dynamodbav:"A"`
	Kind      int64             `dynamodbav:"K"`
	UMS       UMSField          `dynamodbav:"UMS"`
	CreatedAt int64             `dynamodbav:"CRTD"`
	Data      map[string]string `dynamodbav:"D"`
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

func DataOp(d map[string]string) func(m *Msg) error {
	return func(m *Msg) error {
		m.Data = d
		return nil
	}
}

//Factory method for Msg
func NewMsg(channel string, pk string, kind int64, options ...func(*Msg) error) (*Msg, error) {
	msg := &Msg{ChannelPK: channel, AuthorPK: pk, Kind: kind, CreatedAt: time.Now().Unix()}
	msg.Data = make(map[string]string)
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

func (lm *ListMsg) FetchByUserStatus(t *DTable, user *User, status int, start, end string) error {
	ums := fmt.Sprintf("%s#%d", user.PK, status)

	start_time, err := StrToTime(start)
	if err != nil {
		return err
	}
	end_time, err := StrToTime(end)
	if err != nil {
		return err
	}
	exprValues := map[string]interface{}{":ums": ums, ":start": start_time.Unix(), ":end": end_time.Unix()}
	resp, err := t.QueryIndex("UMSIndex", "UMS = :ums and CRTD BETWEEN :start AND :end", exprValues)
	if err != nil {
		return err
	}
	// item in db has only PK, K, UMS and CRTD b/c index projection settings
	// so we fetch whole item here
	for _, item := range resp.Items {
		msg := &Msg{}
		pk := *item["PK"].S
		err = t.FetchItem(pk, msg)
		if err != nil {
			if err.Error() == NO_SUCH_ITEM {
				continue
			}
			return err
		}
		lm.Items[msg.PK] = msg
	}
	return nil
}

type User struct {
	PK        string
	SK        string
	Title     string `dynamodbav:"T"`
	Email     string `dynamodbav:"E"`
	Tel       string `dynamodbav:"TL"`
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
	tg.Data[bot.PK] = "ok"
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

const SecretKeyPrefix = "secrt#"

type Secret struct {
	PK     string
	SK     string
	UserPK string `dynamodbav:"U"`
	TTL    int64
}

func NewSecret(u *User, valid int) (*Secret, error) {
	pk := fmt.Sprintf("%s%s", SecretKeyPrefix, ksuid.New())
	s := &Secret{PK: pk, SK: pk, UserPK: u.PK, TTL: time.Now().Unix() + int64(valid)}
	return s, nil
}

func (s *Secret) IsValid() bool {
	return time.Now().Unix() < s.TTL
}
