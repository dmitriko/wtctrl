package awsapi

import (
	"log"

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
	t := &DTable{Name: name, PKey: "PK"}
	for _, option := range options {
		err := option(t)
		if err != nil {
			return t, err
		}
	}
	sess, err := session.NewSession(&aws.Config{
		Endpoint: aws.String(t.Endpoint),
		Region:   aws.String("us-west-2"),
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

func (t *DTable) Store(item interface{}) error {
	av, err := dynamodbattribute.MarshalMap(item)
	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(t.Name),
	}
	resp, err := t.db.PutItem(input)
	if err != nil {
		return err
	}
	log.Printf("resp is: %s", resp.String())
	return nil
}

type Msg struct {
	PK  string
	UMS string
}

//Puts 3 Msg items in table
func Put3Items() error {
	t, _ := DTableConnect("main",
		Endpoint("http://127.0.0.1:8000"))

	msg := Msg{PK: "msg#bar", UMS: "user1#0"}
	err := t.Store(msg)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	err := Put3Items()
	if err != nil {
		log.Fatal(err)
	}
}
