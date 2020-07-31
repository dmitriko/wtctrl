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
	ID  string
	UMS string
}

func (item *TItem) PK() string {
	return item.ID
}

func (item *TItem) AsDMap() (map[string]*dynamodb.AttributeValue, error) {
	out := map[string]interface{}{
		"PK":  item.PK(),
		"UMS": item.UMS,
	}
	return dynamodbattribute.MarshalMap(out)
}
func (item *TItem) LoadFromD(av map[string]*dynamodb.AttributeValue) error {
	in := map[string]interface{}{}
	err := dynamodbattribute.UnmarshalMap(av, &in)
	if err != nil {
		return err
	}
	item.ID = in["PK"].(string)
	item.UMS = in["UMS"].(string)
	return nil
}
