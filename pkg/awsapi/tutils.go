package awsapi

import (
	"os/exec"
	"testing"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

const containerName = "dynamotest"

var testTable *DTable

// Creates container with local DynamodDB, create table
func startLocalDynamo(t *testing.T) {
	cmd := exec.Command("docker", "run", "--rm", "-d", "--name", containerName,
		"-p", "8000:8000", "amazon/dynamodb-local:latest")
	err := cmd.Run()
	if err != nil {
		t.Fatal(err)
	}
	testTable, err = DTableConnect("MainTest", Endpoint("http://127.0.0.1:8000"), Region("us-west-2"))
	if err != nil {
		stopLocalDynamo()
		t.Fatal(err)
	}
	err = testTable.Create()
	if err != nil {
		t.Fatal(err)
	}
}

func stopLocalDynamo() {
	cmd := exec.Command("docker", "kill", containerName)
	cmd.Run()
}

type TItem struct {
	PK  string
	UMS string
}

func (item *TItem) AsDMap() (map[string]*dynamodb.AttributeValue, error) {
	return dynamodbattribute.MarshalMap(item)
}
func (item *TItem) LoadFromD(av map[string]*dynamodb.AttributeValue) error {
	return dynamodbattribute.UnmarshalMap(av, item)
}
