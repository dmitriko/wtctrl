package awsapi

import (
	"os/exec"
	"testing"
	"time"

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
	ID        string
	UMS       string
	CreatedAt int64
	Data      map[string]string
}

func NewTestItem(id, ums string) (*TItem, error) {
	i := &TItem{ID: id, UMS: ums, CreatedAt: time.Now().Unix()}
	i.Data = make(map[string]string)

	return i, nil
}

func (item *TItem) PK() string {
	return item.ID
}

func (item *TItem) AsDMap() (map[string]*dynamodb.AttributeValue, error) {
	out := map[string]interface{}{
		"PK":   item.PK(),
		"UMS":  item.UMS,
		"CRTD": item.CreatedAt,
	}
	if len(item.Data) > 0 {
		out["D"] = item.Data
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
	item.CreatedAt = int64(in["CRTD"].(float64))
	item.Data = make(map[string]string)
	d, ok := in["D"].(map[string]interface{})
	if ok {
		for k, v := range d {
			item.Data[k] = v.(string)
		}
	}
	return nil
}
