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
)

const MsgKeyPrefix = "msg#"

//Provides access to DynamoDB table
type DTable struct {
	db       *dynamodb.DynamoDB
	Name     string
	Region   string
	Endpoint string
	PKey     string
	Skey     string
}

//DTableConnect option
func Endpoint(endpoint string) func(*DTable) error {
	return func(t *DTable) error {
		t.Endpoint = endpoint
		return nil
	}
}

//DTableConnect option
func Region(region string) func(*DTable) error {
	return func(t *DTable) error {
		t.Region = region
		return nil
	}
}

//Connects to DynomoDB table
func DTableConnect(name string, options ...func(*DTable) error) (*DTable, error) {
	t := &DTable{Name: name, PKey: "PK", Region: "us-west-2"}
	for _, option := range options {
		err := option(t)
		if err != nil {
			return t, err
		}
	}
	sess, err := session.NewSession(&aws.Config{
		Endpoint: aws.String(t.Endpoint),
		Region:   aws.String(t.Region),
	})
	if err != nil {
		return t, err
	}
	t.db = dynamodb.New(sess)
	return t, nil
}

func (t *DTable) Create() error {
	input := createTableInput // TODO make it copy
	input.TableName = aws.String(t.Name)
	_, err := t.db.CreateTable(input)
	return err
}

type DMapper interface {
	AsDMap() (map[string]*dynamodb.AttributeValue, error)
	LoadFromD(map[string]*dynamodb.AttributeValue) error
}

func (t *DTable) StoreItem(item DMapper) (*dynamodb.PutItemOutput, error) {
	av, err := item.AsDMap()
	if err != nil {
		return nil, err
	}
	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(t.Name),
	}
	return t.db.PutItem(input)
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
	err = item.LoadFromD(result.Item)
	if err != nil {
		return err
	}
	return nil
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

//Common fields for any struct stored in DynamoDB
type DItem struct {
	ID        string
	CreatedAt time.Time
	Data      map[string]string
	Orig      map[string]interface{}
}

type Msg struct {
	DItem
	Channel    string
	Author     string
	Kind       string
	ID         string
	UserStatus int
	CreatedAt  time.Time
	Data       map[string]string
}

//Option for new msg
func CreatedAtOp(ts string) func(*Msg) error {
	return func(m *Msg) error {
		t, err := StrToTime(ts)
		if err != nil {
			return err
		}
		m.CreatedAt = t
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
func NewMsg(channel string, author string, kind string, options ...func(*Msg) error) (*Msg, error) {
	msg := &Msg{Channel: channel, Author: author, Kind: kind, CreatedAt: time.Now()}
	for _, opt := range options {
		err := opt(msg)
		if err != nil {
			return nil, err
		}
	}
	if msg.ID == "" {
		id, err := ksuid.NewRandomWithTime(msg.CreatedAt)
		if err != nil {
			return nil, err
		}
		msg.ID = id.String()
		msg.CreatedAt = id.Time() //there is a bit difference from origin, we need this to be stored
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
	ums := fmt.Sprintf("%s#%d", m.Author, m.UserStatus)
	item := map[string]interface{}{
		"PK":  m.PK(),
		"UMS": ums,
		"C":   m.Channel,
		"K":   m.Kind,
	}
	if len(m.Data) > 0 {
		item["D"] = m.Data
	}
	return dynamodbattribute.MarshalMap(item)
}

// Set .Author and .UserStatus from UMS string that is <user>#<status> stored in DB
func (m *Msg) SetUserStatus(ums string) error {
	s := strings.Split(ums, "#")
	if len(s) != 2 {
		return errors.New("Could not parse " + ums)
	}
	m.Author = s[0]
	i, err := strconv.Atoi(s[1])
	if err != nil {
		return err
	}
	m.UserStatus = i
	return nil
}

func loadFromDynamo(key_prefix string, av map[string]*dynamodb.AttributeValue) (*DItem, error) {
	ditem := &DItem{}
	item := map[string]interface{}{}
	err := dynamodbattribute.UnmarshalMap(av, &item)
	if err != nil {
		return nil, err
	}
	ditem.Orig = item
	ditem.Data = make(map[string]string)
	ditem.ID = strings.Replace(item["PK"].(string), key_prefix, "", -1)
	kid, err := ksuid.Parse(ditem.ID)
	if err != nil {
		return nil, err
	}
	ditem.CreatedAt = kid.Time()
	d, ok := item["D"].(map[string]interface{})
	if ok {
		for k, v := range d {
			ditem.Data[k] = v.(string)
		}
	}
	return ditem, nil
}

func (m *Msg) LoadFromD(av map[string]*dynamodb.AttributeValue) error {
	ditem, err := loadFromDynamo(MsgKeyPrefix, av)
	if err != nil {
		return err
	}
	m.ID = ditem.ID
	m.CreatedAt = ditem.CreatedAt
	m.Data = ditem.Data
	err = m.SetUserStatus(ditem.Orig["UMS"].(string))
	if err != nil {
		return err
	}
	ch, ok := ditem.Orig["C"].(string)
	if ok {
		m.Channel = ch
	}
	k, ok := ditem.Orig["K"].(string)
	if ok {
		m.Kind = k
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

func (lm *ListMsg) FetchByUserStatus(t *DTable, uid string, status int, start, end string) error {
	ums := fmt.Sprintf("%s#%d", uid, status)
	start_pk, err := GetMsgPK(start)
	if err != nil {
		return err
	}
	end_pk, err := GetMsgPK(end)
	if err != nil {
		return err
	}
	exprValues := map[string]interface{}{":ums": ums, ":start": start_pk, ":end": end_pk}
	resp, err := t.QueryIndex("UMSIndex", "UMS = :ums and PK BETWEEN :start AND :end", exprValues)
	if err != nil {
		return err
	}
	for _, item := range resp.Items {
		msg := &Msg{}
		err = msg.LoadFromD(item)
		if err != nil {
			return err
		}
		lm.Items[msg.ID] = msg
	}
	return nil
}

type User struct {
	ID    string
	Title string
	Email string
	Tel   string
	Tgid  string
	Data  map[string]string
}

//func NewUser(title, email, tel string)
