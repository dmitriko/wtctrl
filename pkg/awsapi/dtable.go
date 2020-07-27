package awsapi

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
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

func (t *DTable) FetchItem(pk string, item interface{}) error {
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
	err = dynamodbattribute.UnmarshalMap(result.Item, &item)
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
		KeyConditionExpression:    aws.String("UMS = :ums"),
		ExpressionAttributeValues: av,
	}
	return t.db.Query(qi)
}

//Represents list of Msg
type ListMsg struct {
	Items            []*Msg
	LastEvaluatedKey map[string]*dynamodb.AttributeValue
}

func (lm *ListMsg) Len() int {
	return len(lm.Items)
}

func (lm *ListMsg) DAdd(item map[string]*dynamodb.AttributeValue) error {
	m := &Msg{}
	err := dynamodbattribute.UnmarshalMap(item, m)
	if err != nil {
		return err
	}
	lm.Items = append(lm.Items, m)
	return nil
}

func (lm *ListMsg) FetchByUMS(t *DTable, ums string) error {
	exprValues := map[string]interface{}{":ums": ums}
	resp, err := t.QueryIndex("UMSIndex", "UMS = :ums", exprValues)
	if err != nil {
		return err
	}
	for _, item := range resp.Items {
		lm.DAdd(item)
	}
	return nil
	//return dynamodbattribute.UnmarshalListOfMaps(resp.Items, &items)
}

func (lm *ListMsg) At(i int) *Msg {
	if i > lm.Len()-1 {
		return nil
	}
	return lm.Items[i]
}

type Msg struct {
	PK  string
	UMS string
}

func (m *Msg) AsDMap() (map[string]*dynamodb.AttributeValue, error) {
	return dynamodbattribute.MarshalMap(m)
}
