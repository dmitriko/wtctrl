package awsapi

import (
	"os/exec"
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

func TestMsgById(t *testing.T) {
	setUp(t)
	defer tearDown()
}
