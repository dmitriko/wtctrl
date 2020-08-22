package awsapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWSAuthRequest(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	user, _ := NewUser("foo")
	secret, _ := NewSecret(user, 24)
	assert.Nil(t, testTable.StoreItem(secret))
	arn := "arn:::somename"
	resp, err := handleWSAuthReq(testTable, map[string]string{"secret": secret.PK}, arn)
	assert.Nil(t, err)
	assert.Equal(t, user.PK, resp.PrincipalID)
	assert.Equal(t, "Allow", resp.PolicyDocument.Statement[0].Effect)
	assert.Equal(t, arn, resp.PolicyDocument.Statement[0].Resource[0])

	resp, err = handleWSAuthReq(testTable, map[string]string{"secret": "foo"}, arn)
	assert.Equal(t, "Deny", resp.PolicyDocument.Statement[0].Effect)

	secret2, _ := NewSecret(user, 0)
	assert.Nil(t, testTable.StoreItem(secret2))
	resp, err = handleWSAuthReq(testTable, map[string]string{"secret": secret2.PK}, arn)
	assert.Equal(t, "Deny", resp.PolicyDocument.Statement[0].Effect)
}
