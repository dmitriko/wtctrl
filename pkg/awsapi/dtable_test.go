package awsapi

import (
	"os/exec"
	"reflect"
	"testing"
)

const containerName = "dynamotest"

var dTable *DTable

// Creates container with local DynamodDB, create table
func setUp(t *testing.T) {
	cmd := exec.Command("docker", "run", "--rm", "-d", "--name", containerName,
		"-p", "8000:8000", "amazon/dynamodb-local:latest")
	err := cmd.Run()
	if err != nil {
		t.Fatal(err)
	}
	dTable, err = DTableConnect("MainTest", Endpoint("http://127.0.0.1:8000"), Region("us-west-2"))
	if err != nil {
		tearDown()
		t.Fatal(err)
	}
	err = dTable.Create()
	if err != nil {
		t.Fatal(err)
	}
}

func tearDown() {
	cmd := exec.Command("docker", "kill", containerName)
	cmd.Run()
}

func Messaging(t *testing.T) {
	msg := Msg{"foo", "bar"}
	_, err := dTable.StoreItem(&msg)
	if err != nil {
		t.Error(err)
	}
	fmsg := Msg{}
	err = dTable.FetchItem("foo", &fmsg)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(msg, fmsg) {
		t.Errorf("%+v != %+v", fmsg, msg)
	}
	//	items := []Msg{}
	//	err = dTable.FetchMsgsUMS("bar", &items)
}

func TestDynamo(t *testing.T) {
	setUp(t)
	defer tearDown()
	t.Run("Messages", Messaging)
}
