package awsapi

import (
	"os"
	"os/exec"
	"testing"
)

const containerName = "dynamotest"

func createDynamoWithTF(t *testing.T) {
	cmd := exec.Command("terraform", "apply", "-refresh=false", "-auto-approve",
		"-state=/dev/null", "-lock=false", "-var", "table_name=MainTest", "dynamodb-testing")
	cmd.Dir = "../../tf/"
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		t.Fatal(err)
	}
}

func createDynamoFromGO(t *testing.T, testTable *DTable) {
	err := testTable.Create()
	if err != nil {
		t.Fatal(err)
	}
	err = testTable.EnableTTL()
	if err != nil {
		t.Fatal(err)
	}
}

// Creates container with local DynamodDB, create table
func startLocalDynamo(t *testing.T) *DTable {
	cmd := exec.Command("docker", "run", "--rm", "-d", "--name", containerName,
		"-p", "8000:8000", "amazon/dynamodb-local:latest")
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		t.Fatal(err)
	}
	testTable, _ := NewDTable("MainTest")
	testTable.Endpoint = "http://127.0.0.1:8000"
	err = testTable.Connect()
	if err != nil {
		stopLocalDynamo()
		t.Fatal(err)
	}
	//createDynamoWithTF(t)
	createDynamoFromGO(t, testTable)
	return testTable
}

func stopLocalDynamo() {
	cmd := exec.Command("docker", "kill", containerName)
	cmd.Run()
}
