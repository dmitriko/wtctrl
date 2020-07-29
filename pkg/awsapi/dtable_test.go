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
	if item.PK != "foo" || item.UMS != "bar" {
		t.Errorf("could not query index, got %+v", item)
	}
}

func TestDBMessaging(t *testing.T) {
	startLocalDynamo(t)
	defer stopLocalDynamo()
	msg, err := NewMsg("bot1", "user1", CreatedAtOp("-2d"), UserStatusOp(5),
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
	msg, err := NewMsg("bot1", "user1", CreatedAtOp("-2d"), UserStatusOp(5),
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
