package awsapi

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/segmentio/ksuid"
)

func TestStoreInTrans(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	item1, _ := NewTestItem("foo1", "bar1")
	item2, _ := NewTestItem("foo2", "bar2")
	err := testTable.StoreInTransUniq(item1, item2)
	if err != nil {
		t.Error(err)
	}
	item3, _ := NewTestItem("foo3", "bar3")
	item4, _ := NewTestItem("foo1", "bar1") // shoud be failing
	err = testTable.StoreInTransUniq(item3, item4)
	if err == nil || strings.HasPrefix("TransactionCanceledException", err.Error()) {
		t.Error("this should be failing")
	}
}

func TestDBUpdateItem(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	item, _ := NewTestItem("foo", "bar")
	item.Data["spam"] = "egg"
	item.Data["url"] = "example.com"
	_, err := testTable.StoreItem(item)
	if err != nil {
		t.Error(err)
	}
	_, err = testTable.UpdateItemData(item.PK(), "url", "foobar.com")
	f := &TItem{}
	err = testTable.FetchItem(item.PK(), f)
	if f.Data["url"] != "foobar.com" {
		t.Errorf("update did not work %+v", f)
	}
}

func TestStoreItems(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)

	msg, _ := NewTestItem("foo", "bar")
	msg.Data["spam"] = "egg"
	_, err := testTable.StoreItem(msg)
	if err != nil {
		t.Error(err)
	}
	msg_err, _ := NewTestItem("foo", "baz")
	_, err = testTable.StoreItem(msg_err, UniqueOp())
	if err == nil || err.Error() != ALREADY_EXISTS {
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
		t.FailNow()
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

func TestMsgDb(t *testing.T) {
	testTable := startLocalDynamo(t)
	defer stopLocalDynamo()
	msg, err := NewMsg("bot1", "user#user1", TGTextMsgKind, CreatedAtOp("-2d"), UserStatusOp(5),
		DataOp(map[string]string{"url": "https://google.com"}))
	if err != nil {
		t.Error(err)
	}
	if msg.AuthorPK != "user#user1" {
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

	if dmsg.AuthorPK != msg.AuthorPK {
		t.Error("Could not fetch Msg.Author")
	}

	if !reflect.DeepEqual(dmsg, msg) {
		t.Errorf("%+v is not eq to %+v", dmsg, msg)
	}
}

func TestMsgSimple(t *testing.T) {
	msg, err := NewMsg("bot1", "user#user1", TGPhotoMsgKind, CreatedAtOp("-2d"), UserStatusOp(5),
		DataOp(map[string]string{"url": "https://google.com"}))
	if err != nil {
		t.Error(err)
	}
	id, err := ksuid.Parse(msg.ID)
	if err != nil {
		t.Error(err)
	}
	var dur time.Duration
	dur = id.Time().Sub(time.Unix(msg.CreatedAt, 0))
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
	if msg.AuthorPK != "user#user1" {
		t.Errorf("AuthorPK is not correct, expected %s got %s", "user#user1", msg.AuthorPK)
	}
	if msg.ChannelPK != "bot1" {
		t.Error("ChannelPK is not correct")
	}
	if msg.Kind != TGPhotoMsgKind {
		t.Error("Kind is not correct")
	}
}

func TestMsgList(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	var err error
	user1, _ := NewUser("user1")
	user2, _ := NewUser("user2")
	msg1, err := NewMsg("bot1", user1.PK(), TGTextMsgKind, CreatedAtOp("-10d"), UserStatusOp(5),
		DataOp(map[string]string{"url": "https://example1.com"}))

	msg2, err := NewMsg("bot1", user1.PK(), TGTextMsgKind, CreatedAtOp("-2d"), UserStatusOp(5),
		DataOp(map[string]string{"url": "https://example2.com"}))

	msg3, err := NewMsg("bot1", user2.PK(), TGTextMsgKind, CreatedAtOp("-2d"), UserStatusOp(5),

		DataOp(map[string]string{"url": "https://example3.com"}))

	errs := testTable.StoreItems(msg1, msg2, msg3)
	for _, e := range errs {
		if e != nil {
			t.Error(e)
		}
	}
	lm := NewListMsg()
	err = lm.FetchByUserStatus(testTable, user1, 5, "-3d", "now")
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
	testTable := startLocalDynamo(t)
	var err error
	usr1, _ := NewUser("Someone")
	e1, _ := NewEmail("foo@bar", usr1.PK())
	_, err = testTable.StoreItem(e1)
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
}
func TestNewUser(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	usr1, _ := NewUser("Someone")
	err := usr1.SetTel("555555")
	if err != nil {
		t.Error(err)
	}
	err = usr1.SetEmail("foo@bar")
	if err != nil {
		t.Error(err)
	}
	err = testTable.StoreNewUser(usr1)
	if err != nil {
		t.Error(err)
	}
	usr1f := &User{}
	err = testTable.FetchItem(usr1.PK(), usr1f)
	if err != nil {
		t.Error(err)
	}
	if usr1f.Email != "foo@bar" || usr1f.Tel != "555555" {
		t.Errorf("%+v", usr1f)
	}
	email := &Email{}
	err = testTable.FetchItem(EmailKeyPrefix+"foo@bar", email)
	if err != nil {
		t.Error(err)
	}
	if email.OwnerPK != usr1.PK() {
		t.Error("could not fetch email")
	}
	tel := &Tel{}
	err = testTable.FetchItem(TelKeyPrefix+"555555", tel)
	if err != nil {
		t.Error(err)
	}
	if tel.OwnerPK != usr1.PK() {
		t.Error("could not fetch telephone")
	}
	usr2, _ := NewUser("Somebodyelse")
	usr2.SetTel("555555")
	err = testTable.StoreNewUser(usr2)
	if err == nil || strings.HasPrefix("TransactionCanceledException", err.Error()) {
		t.Error("expect error here")
	}
}

func TestSetTG(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	tgid := "sometgid"
	usr, _ := NewUser("Foo")
	bot, _ := NewBot(TGBotKind, "somebot", "somesecret")
	err := testTable.StoreUserTG(usr, tgid, bot)
	if err != nil {
		t.Error(err)
	}
	u := &User{}
	err = testTable.FetchItem(usr.PK(), u)
	if err != nil {
		t.Error(err)
	}
	if u.TGID != tgid {
		t.Error("could not save TGID")
	}
	tg := &TGAcc{}
	err = testTable.FetchItem(TGAccKeyPrefix+tgid, tg)
	if err != nil {
		t.Error(err)
	}
	if tg.OwnerPK != usr.PK() || tg.TGID != tgid {
		t.Error("Could not fetch TG data")
	}
	if _, ok := tg.Data[bot.PK()]; !ok {
		t.Error("tg account does not have associated bot")
	}
}

func TestBot(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	bot, _ := NewBot(TGBotKind, "foo", "somesecret")
	bot.Data["foo"] = "bar"
	_, err := testTable.StoreItem(bot)
	if err != nil {
		t.Error(err)
	}
	bf := &Bot{}
	err = testTable.FetchItem(bot.PK(), bf)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(bf, bot) {
		t.Errorf("%+v != %+v", bot, bf)
	}
	if bot.InviteUrl("111111") != "https://t.me/foo?start=111111" {
		t.Errorf("Invite URL is not correct, got %s expected %s", bot.InviteUrl("111111"),
			"http://t.me/foo?start=111111")
	}
}

func TestInvite(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	user, _ := NewUser("foo")
	bot, _ := NewBot(TGBotKind, "somebot", "somesecret")
	valid := 24 //hours
	inv, _ := NewInvite(user, bot, valid)
	if !inv.IsValid() {
		t.Errorf("%+v expected to be valid", inv)
	}
	inv.TTL = time.Now().Unix()
	if inv.IsValid() {
		t.Errorf("%+v expected to be invalid", inv)
	}
	if !regexp.MustCompile(`^[0-9]{6}$`).MatchString(inv.OTP) {
		t.Errorf("OTP is not properly set, %+v", inv)
	}
	_, err := testTable.StoreItem(inv, UniqueOp())
	if err != nil {
		t.Error(err)
	}
	invf := &Invite{}
	err = testTable.FetchItem(inv.PK(), invf)
	if err != nil {
		t.Error(err)
	}
	if invf.OTP != inv.OTP || invf.UserPK != inv.UserPK || invf.BotID != inv.BotID || invf.TTL != inv.TTL {
		t.Errorf("%+v !+ %+v", invf, inv)
	}
	url := inv.Url
	expected_url := "https://t.me/somebot?start=" + inv.OTP
	if url != expected_url {
		t.Errorf("expected %s got %s", expected_url, url)
	}
	inv, _ = NewInvite(user, bot, valid)
	_, err = testTable.StoreItem(inv)
	if err != nil {
		t.Error(err)
	}
	invf2 := &Invite{}
	err = testTable.FetchInvite(bot, inv.OTP, invf2)
	if err != nil {
		t.Error(err)
	}
	invf2 = &Invite{}
	err = testTable.FetchInvite(bot, "000000", invf2)
	if err == nil || err.Error() != NO_SUCH_ITEM {
		t.Error("should not be happening")
	}
}

func TestFile(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	user, _ := NewUser("Foobar")
	f, _ := NewFile(user.PK(), PicFileKind)
	f.Data["tgid"] = "spamegg"
	_, err := testTable.StoreItem(f)
	if err != nil {
		t.Error(err)
	}
}
