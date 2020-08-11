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
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/segmentio/ksuid"
	"github.com/xlzd/gotp"
)

const (
	MsgKeyPrefix   = "msg#"
	NO_SUCH_ITEM   = "NoSuchItem"
	ALREADY_EXISTS = "AlreadyExists"
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

type DMapper interface {
	AsDMap() (map[string]*dynamodb.AttributeValue, error)
	LoadFromD(map[string]*dynamodb.AttributeValue) error
	PK() string
}

// Option to check uniqueness of stored item by cheking PK
func UniqueOp() func(*dynamodb.PutItemInput) error {
	return func(pii *dynamodb.PutItemInput) error {
		pii.ConditionExpression = aws.String("attribute_not_exists(PK)")
		return nil
	}
}

func (t *DTable) StoreItem(item DMapper,
	options ...func(*dynamodb.PutItemInput) error) (*dynamodb.PutItemOutput, error) {
	av, err := item.AsDMap()
	if err != nil {
		return nil, err
	}
	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(t.Name),
	}
	for _, ops := range options {
		err := ops(input)
		if err != nil {
			return nil, err
		}
	}
	out, err := t.db.PutItem(input)
	if err != nil && strings.HasPrefix(err.Error(), "ConditionalCheckFailedException") {
		return out, errors.New(ALREADY_EXISTS)
	}
	return out, err
}

//Store bunch of items in transactions, make sure they are new
// by checking PK uniqueness
func (t *DTable) StoreInTransUniq(items ...DMapper) error {
	titems := []*dynamodb.TransactWriteItem{}
	for _, i := range items {
		av, err := i.AsDMap()
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

func (t *DTable) StoreItems(items ...DMapper) []error {
	var output []error
	for _, item := range items {
		_, err := t.StoreItem(item)
		output = append(output, err)
	}
	return output
}

func (t *DTable) FetchItem(pk string, item DMapper) error {
	result, err := t.db.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(t.Name),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {
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
	err = item.LoadFromD(result.Item)
	if err != nil {
		return err
	}
	return nil
}

func (t *DTable) UpdateItemData(pk, key, value string) (*dynamodb.UpdateItemOutput, error) {
	uii := &dynamodb.UpdateItemInput{
		TableName:    aws.String(t.Name),
		ReturnValues: aws.String("ALL_NEW"),
		ExpressionAttributeNames: map[string]*string{
			"#Data": aws.String("D"),
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
	av, err := dynamodbattribute.MarshalMap(exprValues)
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

type Msg struct {
	ChannelPK  string
	AuthorPK   string
	Kind       int64
	ID         string
	UserStatus int
	CreatedAt  int64
	Data       map[string]string
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
		m.UserStatus = s
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
	if msg.ID == "" {
		id, err := ksuid.NewRandomWithTime(time.Unix(msg.CreatedAt, 0))
		if err != nil {
			return nil, err
		}
		msg.ID = id.String()
		msg.CreatedAt = id.Time().Unix() //there is a bit difference from origin, we need this to be stored
	}
	return msg, nil
}

func (m *Msg) PK() string {
	if m.ID == "" {
		return ""
	}
	return MsgKeyPrefix + m.ID
}

func (m *Msg) AsDMap() (map[string]*dynamodb.AttributeValue, error) {
	ums := fmt.Sprintf("%s#%d", m.AuthorPK, m.UserStatus)
	item := map[string]interface{}{
		"PK":   m.PK(),
		"UMS":  ums,
		"Ch":   m.ChannelPK,
		"CRTD": m.CreatedAt,
		"K":    m.Kind,
	}
	if len(m.Data) > 0 {
		item["D"] = m.Data
	}
	return dynamodbattribute.MarshalMap(item)
}

// Set .Author and .UserStatus from UMS string that is <prefix>#<author>#<status> stored in DB
func (m *Msg) SetUserStatus(ums string) error {
	s := strings.Split(ums, "#")
	if len(s) != 3 {
		return errors.New("Could not parse " + ums)
	}
	m.AuthorPK = s[0] + "#" + s[1]
	i, err := strconv.Atoi(s[2])
	if err != nil {
		return err
	}
	m.UserStatus = i
	return nil
}

func IdFromPk(pk interface{}, prefix string) string {
	p := pk.(string)
	return strings.Replace(p, prefix, "", -1)
}

func PK2ID(pkin interface{}, prefix string) (string, error) {
	pk, ok := pkin.(string)
	if !ok {
		return "", errors.New(fmt.Sprintf("Could not cast string from %+v", pkin))
	}
	return strings.Replace(pk, prefix, "", -1), nil
}

func (m *Msg) Update(table *DTable) error {
	return table.FetchItem(m.PK(), m)
}

func (m *Msg) LoadFromD(av map[string]*dynamodb.AttributeValue) error {
	item := map[string]interface{}{}
	err := dynamodbattribute.UnmarshalMap(av, &item)
	if err != nil {
		return err
	}
	id, err := PK2ID(item["PK"], MsgKeyPrefix)
	if err != nil {
		return err
	}
	m.ID = id
	m.CreatedAt = int64(item["CRTD"].(float64))
	m.Data = UnmarshalDataProp(item["D"])
	err = m.SetUserStatus(item["UMS"].(string))
	if err != nil {
		return err
	}
	ch, ok := item["Ch"].(string)
	if ok {
		m.ChannelPK = ch
	}
	k, ok := item["K"].(float64)
	if ok {
		m.Kind = int64(k)
	}
	return nil
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

func GetMsgPK(strtime string) (string, error) {
	t, err := StrToTime(strtime)
	if err != nil {
		return "", err
	}
	id, err := ksuid.NewRandomWithTime(t)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%s", MsgKeyPrefix, id.String()), nil
}

func (lm *ListMsg) FetchByUserStatus(t *DTable, user *User, status int, start, end string) error {
	ums := fmt.Sprintf("%s#%d", user.PK(), status)

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
		lm.Items[msg.ID] = msg
	}
	return nil
}

const UserKeyPrefix = "user#"

type User struct {
	ID        string
	Title     string
	Email     string
	Tel       string
	TGID      string
	CreatedAt int64
	Data      map[string]string
}

func (u *User) PK() string {
	if u.ID == "" {
		return ""
	}
	return fmt.Sprintf("%s%s", UserKeyPrefix, u.ID)
}

func (u *User) AsDMap() (map[string]*dynamodb.AttributeValue, error) {
	item := map[string]interface{}{
		"PK":   u.PK(),
		"E":    u.Email,
		"T":    u.Tel,
		"TG":   u.TGID,
		"CRTD": u.CreatedAt,
	}
	if len(u.Data) > 0 {
		item["D"] = u.Data
	}
	return dynamodbattribute.MarshalMap(item)
}

func (u *User) LoadFromD(av map[string]*dynamodb.AttributeValue) error {
	item := map[string]interface{}{}
	err := dynamodbattribute.UnmarshalMap(av, &item)
	if err != nil {
		return err
	}

	id, err := PK2ID(item["PK"], UserKeyPrefix)
	if err != nil {
		return err
	}
	u.ID = id

	created_at, ok := item["CRTD"].(float64)
	if ok {
		u.CreatedAt = int64(created_at)
	}

	u.Data = make(map[string]string)
	data, ok := item["D"].(map[string]string)
	if ok {
		for k, v := range data {
			u.Data[k] = v
		}
	}
	email, ok := item["E"].(string)
	if ok {
		u.Email = email
	}
	t, ok := item["T"].(string)
	if ok {
		u.Tel = t
	}
	tg, ok := item["TG"].(string)
	if ok {
		u.TGID = tg
	}
	return nil
}

func NewUser(title string) (*User, error) {
	user := &User{Title: title}
	user.Data = make(map[string]string)
	kid := ksuid.New()
	user.CreatedAt = int64(time.Now().Unix())
	user.ID = kid.String()
	return user, nil
}

func (u *User) SetTel(t string) error {
	tel, err := NewTel(t, u.PK())
	if err != nil {
		return err
	}
	u.Tel = tel.String()
	return nil
}

func (u *User) SetEmail(e string) error {
	email, err := NewEmail(e, u.PK())
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
	return e.Email
}

func (t *TGAcc) String() string {
	return t.TGID
}

func (t *DTable) StoreUserTG(user *User, tgid int, bot *Bot) error {
	tg, err := NewTGAcc(tgid, user.PK())
	if err != nil {
		return err
	}
	tg.Data[bot.PK()] = "ok"
	_, err = t.StoreItem(tg, UniqueOp())
	if err != nil {
		return err
	}
	user.TGID = tg.TGID
	_, err = t.StoreItem(user)
	return err
}

//Store user, telephon number, email in one transaction
//it fails if number, email or tg id already exist
func (t *DTable) StoreNewUser(user *User) error {
	items := []DMapper{user}
	if user.Tel != "" {
		tel, err := NewTel(user.Tel, user.PK())
		if err != nil {
			return err
		}
		items = append(items, tel)
	}
	if user.Email != "" {
		email, err := NewEmail(user.Email, user.PK())
		if err != nil {
			return err
		}
		items = append(items, email)
	}
	return t.StoreInTransUniq(items...)
}

const EmailKeyPrefix = "email"

type Email struct {
	Email     string
	OwnerPK   string
	CreatedAt time.Time
}

func (e *Email) PK() string {
	return fmt.Sprintf("%s%s", EmailKeyPrefix, e.Email)
}

func (e *Email) LoadFromD(av map[string]*dynamodb.AttributeValue) error {
	item := map[string]interface{}{}
	err := dynamodbattribute.UnmarshalMap(av, &item)
	if err != nil {
		return err
	}
	created, err := time.Parse(time.RFC3339, item["C"].(string))
	if err != nil {
		return err
	}
	e.Email = IdFromPk(item["PK"], EmailKeyPrefix)
	e.OwnerPK = item["O"].(string)
	e.CreatedAt = created
	return nil
}

func (e *Email) AsDMap() (map[string]*dynamodb.AttributeValue, error) {
	item := map[string]interface{}{
		"PK": e.PK(),
		"O":  e.OwnerPK,
		"C":  e.CreatedAt.Format(time.RFC3339),
	}

	return dynamodbattribute.MarshalMap(item)
}

func NewEmail(email, owner_pk string) (*Email, error) {
	return &Email{Email: email, OwnerPK: owner_pk, CreatedAt: time.Now()}, nil
}

const TelKeyPrefix = "tel"

//For telephone number
type Tel struct {
	Number    string
	OwnerPK   string
	CreatedAt time.Time
}

func (t *Tel) PK() string {
	return fmt.Sprintf("%s%s", TelKeyPrefix, t.Number)
}

func (t *Tel) LoadFromD(av map[string]*dynamodb.AttributeValue) error {
	item := map[string]interface{}{}
	err := dynamodbattribute.UnmarshalMap(av, &item)
	if err != nil {
		return err
	}
	created, err := time.Parse(time.RFC3339, item["C"].(string))
	if err != nil {
		return err
	}
	t.Number = IdFromPk(item["PK"], TelKeyPrefix)
	t.OwnerPK = item["O"].(string)
	t.CreatedAt = created
	return nil
}

func (t *Tel) AsDMap() (map[string]*dynamodb.AttributeValue, error) {
	item := map[string]interface{}{
		"PK": t.PK(),
		"O":  t.OwnerPK,
		"C":  t.CreatedAt.Format(time.RFC3339),
	}

	return dynamodbattribute.MarshalMap(item)
}

func NewTel(number, owner_pk string) (*Tel, error) {
	return &Tel{Number: number, OwnerPK: owner_pk, CreatedAt: time.Now()}, nil
}

const TGAccKeyPrefix = "tgacc"

//For Telegram account
type TGAcc struct {
	TGID      string
	OwnerPK   string
	CreatedAt int64
	Data      map[string]string
}

func (t *TGAcc) PK() string {
	return fmt.Sprintf("%s%s", TGAccKeyPrefix, t.TGID)
}

func (t *TGAcc) LoadFromD(av map[string]*dynamodb.AttributeValue) error {
	item := map[string]interface{}{}
	err := dynamodbattribute.UnmarshalMap(av, &item)
	if err != nil {
		return err
	}
	t.TGID = IdFromPk(item["PK"], TGAccKeyPrefix)
	t.OwnerPK = item["O"].(string)
	t.CreatedAt = UnmarshalCreated(item["CRTD"])
	t.Data = UnmarshalDataProp(item["D"])

	return nil
}

func UnmarshalCreated(c interface{}) int64 {
	crtd, ok := c.(float64)
	if ok {
		return int64(crtd)
	}
	return 0
}

func (t *TGAcc) AsDMap() (map[string]*dynamodb.AttributeValue, error) {
	item := map[string]interface{}{
		"PK":   t.PK(),
		"O":    t.OwnerPK,
		"CRTD": t.CreatedAt,
	}
	if len(t.Data) > 0 {
		item["D"] = t.Data
	}
	return dynamodbattribute.MarshalMap(item)
}

func NewTGAcc(tgid int, owner_pk string) (*TGAcc, error) {
	return &TGAcc{TGID: fmt.Sprintf("%d", tgid), OwnerPK: owner_pk, CreatedAt: time.Now().Unix(),
		Data: make(map[string]string)}, nil
}

const TGBotKind = "tg"
const BotKeyPrefix = "bot#"

type Bot struct {
	Name      string
	Kind      string
	Secret    string
	CreatedAt int64
	Data      map[string]string
}

func NewBot(kind, name string) (*Bot, error) {
	bot := &Bot{Kind: kind, Name: name, Data: make(map[string]string)}
	bot.CreatedAt = int64(time.Now().Unix())
	return bot, nil
}

func (b *Bot) InviteUrl(otp string) string {
	return fmt.Sprintf("%s/%s?start=%s", "https://t.me", b.Name, otp)
}

func (b *Bot) PK() string {
	return fmt.Sprintf("%s%s#%s", BotKeyPrefix, b.Name, b.Kind)
}

func (b *Bot) AsDMap() (map[string]*dynamodb.AttributeValue, error) {
	item := map[string]interface{}{
		"PK":   b.PK(),
		"S":    b.Secret,
		"K":    b.Kind,
		"CRTD": b.CreatedAt,
		"N":    b.Name,
	}
	if len(b.Data) > 0 {
		item["D"] = b.Data
	}
	return dynamodbattribute.MarshalMap(item)
}

func (b *Bot) LoadFromD(av map[string]*dynamodb.AttributeValue) error {
	item := map[string]interface{}{}
	err := dynamodbattribute.UnmarshalMap(av, &item)
	if err != nil {
		return err
	}
	created_at, ok := item["CRTD"].(float64)
	if ok {
		b.CreatedAt = int64(created_at)
	}
	b.Data = UnmarshalDataProp(item["D"])
	b.Kind, ok = item["K"].(string)
	if !ok {
		return errors.New("Kind is not set for bot in db")
	}

	s, ok := item["S"].(string)
	if ok {
		b.Secret = s
	}
	b.Name, ok = item["N"].(string)
	if !ok {
		return errors.New("Name is not set for bot in db")
	}
	return nil
}

func UnmarshalDataProp(d interface{}) map[string]string {
	r := make(map[string]string)
	s, ok := d.(map[string]interface{})
	if ok {
		for k, v := range s {
			val, ok := v.(string)
			if ok {
				r[k] = val
			} else {
				r[k] = ""
			}
		}
	}
	return r
}

const InviteKeyPrefix = "inv"

type Invite struct {
	BotPK     string
	UserPK    string
	OTP       string
	CreatedAt time.Time
	TTL       int64
	Url       string
	Data      map[string]string
}

func NewInvite(u *User, b *Bot, valid int) (*Invite, error) {
	inv := &Invite{
		UserPK:    u.PK(),
		BotPK:     b.PK(),
		CreatedAt: time.Now(),
		TTL:       int64(valid)*60*60 + time.Now().Unix(),
	}
	inv.OTP = gotp.NewDefaultTOTP(gotp.RandomSecret(16)).Now()
	inv.Data = make(map[string]string)
	inv.Url = b.InviteUrl(inv.OTP)
	return inv, nil
}

func (inv *Invite) PK() string {
	return fmt.Sprintf("%s#%s#%s", InviteKeyPrefix, inv.BotPK, inv.OTP)
}

func (inv *Invite) IsValid() bool {
	if inv.TTL > time.Now().Unix() {
		return true
	}
	return false
}

func (inv *Invite) LoadFromD(av map[string]*dynamodb.AttributeValue) error {
	item := map[string]interface{}{}
	err := dynamodbattribute.UnmarshalMap(av, &item)
	if err != nil {
		return err
	}
	created, err := time.Parse(time.RFC3339, item["C"].(string))
	if err != nil {
		return err
	}
	inv.UserPK, _ = item["U"].(string)
	inv.BotPK = item["B"].(string)
	inv.OTP = item["O"].(string)
	inv.Url = item["Url"].(string)
	ttl, ok := item["T"].(float64)
	if ok {
		inv.TTL = int64(ttl)
	}
	inv.CreatedAt = created
	d, ok := item["D"].(map[string]interface{})
	inv.Data = make(map[string]string)
	if ok {
		for k, v := range d {
			inv.Data[k] = v.(string)
		}
	}
	return nil
}

func (inv *Invite) AsDMap() (map[string]*dynamodb.AttributeValue, error) {
	item := map[string]interface{}{
		"PK":  inv.PK(),
		"U":   inv.UserPK,
		"B":   inv.BotPK,
		"O":   inv.OTP,
		"T":   inv.TTL,
		"Url": inv.Url,
		"C":   inv.CreatedAt.Format(time.RFC3339),
	}
	if len(inv.Data) > 0 {
		item["D"] = inv.Data
	}
	return dynamodbattribute.MarshalMap(item)
}

func (t *DTable) FetchInvite(bot *Bot, code string, inv *Invite) error {
	inv.OTP = code
	inv.BotPK = bot.PK()
	err := t.FetchItem(inv.PK(), inv)
	if err != nil {
		return err
	}
	if !inv.IsValid() {
		return errors.New(NO_SUCH_ITEM)
	}
	return nil
}

const PicFileKind = "PicFileKind"
const FileKeyPrefix = "file#"

type File struct {
	ID        string
	OwnerPK   string
	Kind      string
	Data      map[string]string
	CreatedAt int64
}

func NewFile(pk, kind string) (*File, error) {
	f := &File{}
	f.ID = ksuid.New().String()
	f.OwnerPK = pk
	f.Kind = kind
	f.Data = make(map[string]string)
	f.CreatedAt = time.Now().Unix()
	return f, nil
}

func (f *File) PK() string {
	return FileKeyPrefix + f.ID
}

func (f *File) AsDMap() (map[string]*dynamodb.AttributeValue, error) {
	item := map[string]interface{}{
		"PK":   f.PK(),
		"O":    f.OwnerPK,
		"K":    f.Kind,
		"CRTD": f.CreatedAt,
	}
	if len(f.Data) > 0 {
		item["D"] = f.Data
	}
	return dynamodbattribute.MarshalMap(item)
}

func (f *File) LoadFromD(av map[string]*dynamodb.AttributeValue) error {
	item := map[string]interface{}{}
	err := dynamodbattribute.UnmarshalMap(av, &item)
	if err != nil {
		return err
	}
	f.ID = IdFromPk(item["PK"], FileKeyPrefix)
	f.OwnerPK = item["O"].(string)
	f.Kind = item["K"].(string)
	f.CreatedAt = UnmarshalCreated(item["CRTD"])
	f.Data = UnmarshalDataProp(item["D"])

	return nil
}
