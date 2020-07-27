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
	recover()
	cmd := exec.Command("docker", "kill", containerName)
	cmd.Run()
}

func Messaging(t *testing.T) {
	msg := &Msg{"foo", "bar"}
	_, err := dTable.StoreItem(msg)
	if err != nil {
		t.Error(err)
	}
	fmsg := &Msg{}
	err = dTable.FetchItem("foo", fmsg)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(*msg, *fmsg) {
		t.Errorf("%+v != %+v", fmsg, msg)
	}
	msgs := &ListMsg{}
	err = msgs.FetchByUMS(dTable, "bar")
	if err != nil {
		t.Error(err)
	}
	if msgs.Len() != 1 {
		t.Errorf("Could not fetch messages for UMS bar")
	}

	if !reflect.DeepEqual(msg, msgs.At(0)) {
		t.Errorf("%+v != %+v", msgs.At(0), msg)
	}

	_, err = dTable.StoreItem(&Msg{"baz", "bar"})
	if err != nil {
		t.Error(err)
	}
	msgs = &ListMsg{}
	err = msgs.FetchByUMS(dTable, "baz")
	if err != nil {
		t.Error(err)
	}
	if msgs.Len() != 0 {
		t.Errorf("Expected 0 baz items, got: %v", msgs)
	}

	msgs = &ListMsg{}
	err = msgs.FetchByUMS(dTable, "bar")
	if err != nil {
		t.Error(err)
	}
	if msgs.Len() != 2 {
		t.Errorf("Expected 2 bar items, got: %v", msgs)
	}

}

func TestDynamo(t *testing.T) {
	setUp(t)
	defer tearDown()
	t.Run("Messages", Messaging)
}
