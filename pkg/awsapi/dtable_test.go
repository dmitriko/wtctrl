package awsapi

import (
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/segmentio/ksuid"
)

func TestStoreItems(t *testing.T) {
	startLocalDynamo(t)
	defer stopLocalDynamo()

	msg := &TItem{"foo", "bar"}
	_, err := testTable.StoreItem(msg)
	if err != nil {
		t.Error(err)
	}
	fmsg := &TItem{}
	err = testTable.FetchItem("foo", fmsg)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(*msg, *fmsg) {
		t.Errorf("%+v != %+v", fmsg, msg)
	}
	exprValues := map[string]interface{}{":ums": "bar"}
	resp, err := testTable.QueryIndex("UMSIndex", "UMS = :ums", exprValues)
	if err != nil {
		t.Error(err)
	}
	if len(resp.Items) != 1 {
		t.Error("Could not fetch item from index")
	}
	item := &TItem{}
	dynamodbattribute.UnmarshalMap(resp.Items[0], item)
	if item.ID != "foo" || item.UMS != "bar" {
		t.Errorf("could not query index, got %+v", item)
	}
}

func TestDBMsg(t *testing.T) {
	startLocalDynamo(t)
	defer stopLocalDynamo()
	msg, err := NewMsg("bot1", "user1", "tgtext", CreatedAtOp("-2d"), UserStatusOp(5),
		DataOp(map[string]string{"url": "https://google.com"}))
	if err != nil {
		t.Error(err)
	}
	if msg.Author != "user1" {
		t.Error("issue with Author")
	}
	_, err = testTable.StoreItem(msg)
	if err != nil {
		t.Error(err)
	}
	dmsg := &Msg{}
	err = testTable.FetchItem(msg.PK(), dmsg)
	if err != nil {
		t.Error(err)
	}
	if dmsg.ID != msg.ID {
		t.Error("Could not fetch msg from dynamo")
	}
	if dmsg.Data["url"] != "https://google.com" {
		t.Error("Could not store/fetch msg.Data[url]")
	}

	if dmsg.Author != msg.Author {
		t.Error("Could not fetch Msg.Author")
	}

	if !reflect.DeepEqual(dmsg, msg) {
		t.Errorf("%+v is not eq to %+v", dmsg, msg)
	}
}

func TestSimpleMessaging(t *testing.T) {
	msg, err := NewMsg("bot1", "user1", "tgtext", CreatedAtOp("-2d"), UserStatusOp(5),
		DataOp(map[string]string{"url": "https://google.com"}))
	if err != nil {
		t.Error(err)
	}
	id, err := ksuid.Parse(msg.ID)
	if err != nil {
		t.Error(err)
	}
	var dur time.Duration
	dur = id.Time().Sub(msg.CreatedAt)
	if dur.Milliseconds() > 1000.0 {
		t.Errorf("time from id != CreatedAt, but has diff %s", dur.String())
	}
	dur = time.Now().Sub(id.Time())
	if int(dur.Hours()) != 48 {
		t.Errorf("time from msg id is not as expected, but has diff %s", dur.String())
	}
	if msg.UserStatus != 5 {
		t.Error("UserStatus is not correct")
	}
	if msg.Author != "user1" {
		t.Error("Author is not correct")
	}
	if msg.Channel != "bot1" {
		t.Error("Channel is not correct")
	}
}

func TestListMsg(t *testing.T) {
	defer stopLocalDynamo()
	startLocalDynamo(t)
	var err error
	msg1, err := NewMsg("bot1", "user1", "tgtext", CreatedAtOp("-10d"), UserStatusOp(5),
		DataOp(map[string]string{"url": "https://example1.com"}))

	msg2, err := NewMsg("bot1", "user1", "tgtext", CreatedAtOp("-2d"), UserStatusOp(5),
		DataOp(map[string]string{"url": "https://example2.com"}))

	msg3, err := NewMsg("bot1", "user2", "tgtext", CreatedAtOp("-2d"), UserStatusOp(5),
		DataOp(map[string]string{"url": "https://example3.com"}))

	errs := testTable.StoreItems(msg1, msg2, msg3)
	for _, e := range errs {
		if e != nil {
			t.Error(e)
		}
	}
	lm := NewListMsg()
	err = lm.FetchByUserStatus(testTable, "user1", 5, "-3d", "now")
	if err != nil {
		t.Error(err)
	}
	if lm.Len() != 1 {
		t.Errorf("Fetch wrong amount of Msgs %d, expected 1", lm.Len())
	}
	if _, ok := lm.Items[msg2.ID]; !ok {
		t.Error("expected msg2 is fetched")
	}
}

func TestUser(t *testing.T) {
	defer stopLocalDynamo()
	startLocalDynamo(t)
	usr1 := NewUser("Someone", "5555555")
	err := testTable.StoreNewUser(usr1)
	if err != nil {
		t.Error(err)
	}
	usr1_fetched := &User{}
	err = testTable.FetchItem(usr1.PK(), usr1_fetched)
	if err != nil {
		t.Error(err)
	}
}
