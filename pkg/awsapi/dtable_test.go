package awsapi

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/segmentio/ksuid"
)

func TestStoreInTrans(t *testing.T) {
	defer stopLocalDynamo()
	startLocalDynamo(t)
	item1 := &TItem{"foo1", "bar1"}
	item2 := &TItem{"foo2", "bar2"}
	err := testTable.StoreInTransUniq(item1, item2)
	if err != nil {
		t.Error(err)
	}
	item3 := &TItem{"foo3", "bar3"}
	item4 := &TItem{"foo1", "bar1"} // shoud be failing
	err = testTable.StoreInTransUniq(item3, item4)
	if err == nil || strings.HasPrefix("TransactionCanceledException", err.Error()) {
		t.Error("this should be failing")
	}
}

func TestStoreItems(t *testing.T) {
	defer stopLocalDynamo()
	startLocalDynamo(t)

	msg := &TItem{"foo", "bar"}
	_, err := testTable.StoreItem(msg)
	if err != nil {
		t.Error(err)
	}
	msg_err := &TItem{"foo", "baz"}
	_, err = testTable.StoreItem(msg_err, UniqueOp())
	if err == nil || !strings.HasPrefix(err.Error(), "ConditionalCheckFailedException") {
		t.Error("Fail to unique store item")
	}
	fmsg := &TItem{}
	err = testTable.FetchItem(msg.PK(), fmsg)
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
	err = item.LoadFromD(resp.Items[0])
	if err != nil {
		t.Error(err)
	}
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
	usr1, _ := NewUser("Someone", "5555555")
	e1, _ := NewEmail("foo@bar", usr1.PK())
	_, err := testTable.StoreItem(e1)
	if err != nil {
		t.Error(err)
	}
	ef := &Email{}
	err = testTable.FetchItem(e1.PK(), ef)
	if err != nil {
		t.Error(err)
	}
	if e1.Email != ef.Email || e1.OwnerPK != ef.OwnerPK {
		t.Errorf("%+v != %+v", e1, ef)
	}
	t1, _ := NewTel("5555555", usr1.PK())
	_, err = testTable.StoreItem(t1)
	if err != nil {
		t.Error(err)
	}
	t_stored := &Tel{}
	err = testTable.FetchItem(t1.PK(), t_stored)
	if err != nil {
		t.Error(err)
	}
	if t1.Number != t_stored.Number || t1.OwnerPK != t_stored.OwnerPK {
		t.Errorf("%+v != %+v", t1, t_stored)
	}
	tg1, err := NewTGAcc("tgid1", usr1.PK())
	_, err = testTable.StoreItem(tg1)
	if err != nil {
		t.Error(err)
	}
	tgf := &TGAcc{}
	err = testTable.FetchItem(tg1.PK(), tgf)
	if err != nil {
		t.Error(err)
	}
	if tg1.TGID != tgf.TGID || tg1.OwnerPK != tgf.OwnerPK {
		t.Errorf("%+v != %+v", tg1, tgf)
	}
	/*
		err := testTable.StoreNewUser(usr1)
		if err != nil {
			t.Error(err)
		}
		usr1_fetched := &User{}
		err = testTable.FetchItem(usr1.PK(), usr1_fetched)
		if err != nil {
			t.Error(err)
		}*/
}
