package awsapi

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	dattr "github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/segmentio/ksuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TItem struct {
	PK        string
	SK        string
	UMS       string                 `dynamodbav:",omitempty"`
	CreatedAt int64                  `dynamodbav:"CRTD"`
	Data      map[string]interface{} `dynamodbav:"D"`
	Amount    int64                  `dynamodbav:"AMNT"`
}

func NewTestItem(id, ums string) (*TItem, error) {
	i := &TItem{PK: id, SK: id, UMS: ums, CreatedAt: time.Now().Unix()}
	i.Data = make(map[string]interface{})

	return i, nil
}

func NewSubItem(pk, sk string) (*TItem, error) {
	i := &TItem{PK: pk, SK: sk, CreatedAt: time.Now().Unix()}
	return i, nil
}

func TestIncrement(t *testing.T) {
	defer stopLocalDynamo()
	table := startLocalDynamo(t)
	item, _ := NewTestItem("foo", "bar")
	assert.Nil(t, table.StoreItem(item))
	assert.Nil(t, table.IncrProp(item.PK, item.SK, "AMNT", 2))
	assert.Nil(t, table.FetchItem(item.PK, item))
	assert.Equal(t, int64(2), item.Amount)
}

func TestSubItemsPrefix(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)

	pk := "someid"
	sk1 := "foo#a"
	sk2 := "bar#a"
	sk3 := "foo#b"
	i1, _ := NewSubItem(pk, sk1)
	i2, _ := NewSubItem(pk, sk2)
	i3, _ := NewSubItem(pk, sk3)
	errs := testTable.StoreItems(i1, i2, i3)
	for _, e := range errs {
		if e != nil {
			t.Error(e)
		}
	}
	fooItems := []TItem{}
	err := testTable.FetchItemsWithPrefix(pk, "foo", &fooItems)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(fooItems))
	assert.Equal(t, i1.PK, fooItems[0].PK)
	assert.Equal(t, i3.PK, fooItems[1].PK)
}

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
	err := testTable.StoreItem(item)
	if err != nil {
		t.Error(err)
	}
	_, err = testTable.UpdateItemData(item.PK, "url", "foobar.com")
	f := &TItem{}
	err = testTable.FetchItem(item.PK, f)
	assert.Equal(t, "foobar.com", f.Data["url"])

}

func TestStoreItems(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)

	msg, _ := NewTestItem("foo", "bar")
	msg.Data["spam"] = "egg"
	err := testTable.StoreItem(msg)
	if err != nil {
		t.Error(err)
	}
	msg_err, _ := NewTestItem("foo", "baz")
	err = testTable.StoreItem(msg_err, UniqueOp())
	if err == nil || err.Error() != ALREADY_EXISTS {
		t.Error("Fail to unique store item")
	}
	fmsg := &TItem{}
	err = testTable.FetchItem(msg.PK, fmsg)
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
	err = dattr.UnmarshalMap(resp.Items[0], item)
	if err != nil {
		t.Error(err)
	}
	if item.PK != "foo" || item.UMS != "bar" {
		t.Errorf("could not query index, got %+v", item)
	}
}

func TestMsgDb(t *testing.T) {
	testTable := startLocalDynamo(t)
	defer stopLocalDynamo()
	msg, err := NewMsg("bot1", "user#user1", TGTextMsgKind, CreatedAtOp("-2d"), UserStatusOp(5),
		DataOp(map[string]interface{}{"url": "https://google.com"}))
	if err != nil {
		t.Error(err)
	}
	if msg.AuthorPK != "user#user1" {
		t.Error("issue with Author")
	}
	err = testTable.StoreItem(msg)
	if err != nil {
		t.Error(err)
	}
	dmsg := &Msg{}
	err = testTable.FetchItem(msg.PK, dmsg)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(dmsg, msg) {
		t.Errorf("%+v != %+v", dmsg, msg)
	}
}

func TestMsgSimple(t *testing.T) {
	msg, err := NewMsg("bot1", "user#user1", TGPhotoMsgKind, CreatedAtOp("-2d"), UserStatusOp(5),
		DataOp(map[string]interface{}{"url": "https://google.com"}))
	if err != nil {
		t.Error(err)
	}
	idstr := PK2ID(MsgKeyPrefix, msg.PK)
	id, err := ksuid.Parse(idstr)
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
	if msg.UMS.Status != 5 {
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
	upd := float64(time.Now().Unix() + 5)
	msg.Data[UpdatedAtField] = upd
	assert.Equal(t, int64(upd), msg.UpdatedAt())

}

func TestMsgList(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	var err error
	user1, _ := NewUser("user1")
	user2, _ := NewUser("user2")
	msg1, err := NewMsg("bot1", user1.PK, TGTextMsgKind, CreatedAtOp("-10d"), UserStatusOp(5),
		DataOp(map[string]interface{}{"url": "https://example1.com"}))

	msg2, err := NewMsg("bot1", user1.PK, TGTextMsgKind, CreatedAtOp("-2d"), UserStatusOp(5),
		DataOp(map[string]interface{}{"url": "https://example2.com"}))

	msg3, err := NewMsg("bot1", user2.PK, TGTextMsgKind, CreatedAtOp("-2d"), UserStatusOp(5),
		DataOp(map[string]interface{}{"url": "https://example3.com"}))

	errs := testTable.StoreItems(msg1, msg2, msg3)
	for _, e := range errs {
		if e != nil {
			t.Error(e)
		}
	}
	lm := NewListMsg()
	err = lm.FetchByUserStatus(testTable, user1.PK, 5, "-3d", "now")
	if err != nil {
		t.Error(err)
	}
	if lm.Len() != 1 {
		t.Errorf("Fetch wrong amount of Msgs %d, expected 1", lm.Len())
	}
	if _, ok := lm.Items[msg2.PK]; !ok {
		t.Error("expected msg2 is fetched")
	}
}

func TestUser(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	var err error
	usr1, _ := NewUser("Someone")
	usr1.Data["lang"] = "en-US"
	e1, _ := NewEmail("foo@bar", usr1.PK)
	err = testTable.StoreItem(e1)
	if err != nil {
		t.Error(err)
	}
	ef := &Email{}
	err = testTable.FetchItem(e1.PK, ef)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(e1, ef) {
		t.Errorf("%#v != %#v", e1, ef)
		t.FailNow()
	}

	t1, _ := NewTel("5555555", usr1.PK)
	err = testTable.StoreItem(t1)
	if err != nil {
		t.Error(err)
	}
	t_stored := &Tel{}
	err = testTable.FetchItem(t1.PK, t_stored)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(t1, t_stored) {
		t.Errorf("%#v != %#v", t1, t_stored)
	}

	tg1, err := NewTGAcc(99999999, usr1.PK)
	tg1.Data["bot1"] = "ok"
	err = testTable.StoreItem(tg1)
	if err != nil {
		t.Error(err)
	}
	tgf := &TGAcc{}
	err = testTable.FetchItem(tg1.PK, tgf)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(tg1, tgf) {
		t.Errorf("%#v != %#v", tg1, tgf)
	}
}

func TestUserNew(t *testing.T) {
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
	err = testTable.FetchItem(usr1.PK, usr1f)
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
	if email.OwnerPK != usr1.PK {
		t.Error("could not fetch email")
	}
	tel := &Tel{}
	err = testTable.FetchItem(TelKeyPrefix+"555555", tel)
	if err != nil {
		t.Error(err)
	}
	if tel.OwnerPK != usr1.PK {
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
	tgid := 999999999
	usr, _ := NewUser("Foo")
	bot, _ := NewBot(TGBotKind, "somebot")
	err := testTable.StoreUserTG(usr, tgid, bot)
	if err != nil {
		t.Error(err)
	}
	u := &User{}
	err = testTable.FetchItem(usr.PK, u)
	if err != nil {
		t.Error(err)
	}
	if u.TGID != fmt.Sprintf("%d", tgid) {
		t.Error("could not save TGID")
	}
	tg := &TGAcc{}
	pk := fmt.Sprintf("%s%d", TGAccKeyPrefix, tgid)
	err = testTable.FetchItem(pk, tg)
	if err != nil {
		t.Error(err)
	}
	if tg.OwnerPK != usr.PK || tg.TGID != "999999999" {
		t.Errorf("Could not fetch TG data, %#v", tg)
	}
	var found bool
	for _, botPK := range usr.Bots {
		if botPK == bot.PK {
			found = true
			break
		}
	}
	if !found {
		t.Error("tg account does not have associated bot")
	}
}

func TestBot(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	bot, _ := NewBot(TGBotKind, "foo")
	bot.Data["foo"] = "bar"
	err := testTable.StoreItem(bot)
	if err != nil {
		t.Error(err)
	}
	bf := &Bot{}
	err = testTable.FetchItem(bot.PK, bf)
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
	bot, _ := NewBot(TGBotKind, "somebot")
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
	err := testTable.StoreItem(inv, UniqueOp())
	if err != nil {
		t.Error(err)
	}
	invf := &Invite{}
	err = testTable.FetchItem(inv.PK, invf)
	if err != nil {
		t.Error(err)
	}
	if invf.OTP != inv.OTP || invf.UserPK != inv.UserPK || invf.BotPK != inv.BotPK || invf.TTL != inv.TTL {
		t.Errorf("%+v !+ %+v", invf, inv)
	}
	url := inv.Url
	expected_url := "https://t.me/somebot?start=" + inv.OTP
	if url != expected_url {
		t.Errorf("expected %s got %s", expected_url, url)
	}
	inv, _ = NewInvite(user, bot, valid)
	err = testTable.StoreItem(inv)
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

func TestUserToken(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	user, _ := NewUser("foo")
	assert.Nil(t, testTable.StoreItem(user))
	s, _ := NewToken(user, 24)
	assert.True(t, s.IsValid())
	s.TTL = time.Now().Unix()
	assert.False(t, s.IsValid())
}

func TestWSConn(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	user, _ := NewUser("foo")
	conn1, _ := NewWSConn(user.PK, "someidA=", "foobar.com", "prod")
	conn2, _ := NewWSConn(user.PK, "someidB=", "eggspam.com", "prod")
	assert.Nil(t, testTable.StoreItem(conn1))
	assert.Nil(t, testTable.StoreItem(conn2))
	conns := []WSConn{}
	assert.Nil(t, user.FetchWSConns(testTable, &conns))
	assert.Equal(t, 2, len(conns))
	assert.Equal(t, conns[0].SK, fmt.Sprintf("%s%s", WSConnKeyPrefix, "someidA="))
	assert.Equal(t, conns[0].Endpoint(), "https://foobar.com/prod")
}

func TestSubscriptionItem(t *testing.T) {
	defer stopLocalDynamo()
	table := startLocalDynamo(t)
	domain := "https://foobar.com"
	connId := "someid="
	stage := "prod"
	status := 0
	user, _ := NewUser("foo")
	s1, s2, _ := NewSubscription(user.PK, user.PK, status, domain, stage, connId)
	assert.Nil(t, table.StoreItem(s1))
	assert.Nil(t, table.StoreItem(s2))
	assert.Equal(t, "https://foobar.com/prod", s1.Endpoint())
	assert.Equal(t, connId, s1.ConnectionId())
	assert.Equal(t, s1.SK, s2.SK)
	assert.NotEqual(t, s1.PK, s2.PK)
	s3 := &Subscription{}
	err := table.FetchSubItem(user.PK, s1.SK, s3)
	assert.Nil(t, err)
	if !reflect.DeepEqual(s1, s3) {
		t.Errorf("%#v != %#v", s1, s3)
	}

}

func TestSubscriptionByUMS(t *testing.T) {
	defer stopLocalDynamo()
	table := startLocalDynamo(t)
	domain := "https://foobar.com"
	connId1 := "someid="
	connId2 := "otherid="
	stage := "prod"
	status1 := 0
	status2 := 1
	user, _ := NewUser("foo")
	s1, _, _ := NewSubscription(user.PK, user.PK, status1, domain, stage, connId1)
	s2, _, _ := NewSubscription(user.PK, user.PK, status2, domain, stage, connId2)
	assert.Nil(t, table.StoreItem(s1))
	assert.Nil(t, table.StoreItem(s2))
	//	var subs Subscriptions
	//	err := table.FetchSubsWithUMS(user.PK, status2, &subs)
	//	assert.Nil(t, err)
}

func TestMsgFile(t *testing.T) {
	defer stopLocalDynamo()
	table := startLocalDynamo(t)
	msg, _ := NewMsg("bot1", "user#user1", TGPhotoMsgKind)
	f1, _ := NewMsgFile(msg.PK, FileKindTgThumb, "image/jpeg", "somebucket", "somekey")
	f1.Data["foo"] = "bar"
	assert.Nil(t, table.StoreItem(f1))
	f2 := &MsgFile{}
	assert.Nil(t, table.FetchSubItem(msg.PK, f1.SK, f2))
	if !reflect.DeepEqual(f1, f2) {
		t.Errorf("%#v != %#v", f1, f2)
	}
}

func TestLoginReq(t *testing.T) {
	defer stopLocalDynamo()
	table := startLocalDynamo(t)
	user, _ := NewUser("foo")
	req, _ := NewLoginRequest(user.PK)
	assert.Equal(t, 6, len(req.OTP))
	assert.Nil(t, table.StoreItem(req))
	r2 := &LoginRequest{}
	assert.Nil(t, table.FetchItem(req.PK, r2))
	if !reflect.DeepEqual(req, r2) {
		t.Errorf("%#v != %#v", req, r2)
	}
	ok, res := req.IsOTPValid("000000")
	assert.False(t, ok)
	assert.Equal(t, OTP_WRONG, res)
	ok, res = req.IsOTPValid(req.OTP)
	if !assert.True(t, ok) {
		t.Error(res)
	}
	req.Attempts = 5
	ok, res = req.IsOTPValid(req.OTP)
	assert.False(t, ok)
	assert.Equal(t, TOO_MANY_ATTEMPTS, res)

}

func TestOrgNew(t *testing.T) {
	defer stopLocalDynamo()
	table := startLocalDynamo(t)
	user, _ := NewUser("foo")
	org, _ := NewOrg("Foo", "Foo bar", []*User{user})
	org.Data["foo"] = "bar"
	require.Nil(t, table.StoreItem(org))
	o := &Org{}
	require.Nil(t, table.FetchItem(org.PK, o))
	assert.Equal(t, "Foo", o.Title)
}

func TestPermValue(t *testing.T) {
	//	defer stopLocalDynamo()
	//	table := startLocalDynamo(t)
	var err error
	err = checkPermValue("foo")
	require.NotNil(t, err)
	err = checkPermValue("")
	require.NotNil(t, err)
	err = checkPermValue("f")
	require.NotNil(t, err)
	err = checkPermValue("tf")
	require.NotNil(t, err)
	err = checkPermValue("trf")
	require.NotNil(t, err)
	err = checkPermValue("trwf")
	require.NotNil(t, err)
	err = checkPermValue("trwa")
	require.Nil(t, err)
}

func TestFolderNew(t *testing.T) {
	defer stopLocalDynamo()
	table := startLocalDynamo(t)
	user, _ := NewUser("foo")
	f, _ := NewFolder(user.PK, "foo", 0, FolderStreamKind)
	assert.Equal(t, fmt.Sprintf("%s%d", FolderKeyPrefix, 0), f.SK)
	assert.Equal(t, user.PK, f.PK)
	require.Nil(t, table.StoreItem(f))
	f_ := &Folder{}
	require.Nil(t, table.FetchSubItem(f.PK, f.SK, f_))
	if !reflect.DeepEqual(f, f_) {
		t.Errorf("%#v != %#v", f, f_)
	}
}

func TestUserFolderPerm(t *testing.T) {
	defer stopLocalDynamo()
	table := startLocalDynamo(t)
	user1, _ := NewUser("foo")
	folder, _ := NewFolder(user1.PK, "foo", 5, FolderStreamKind)
	user2, _ := NewUser("bar")
	perm, _ := NewUserPerm(user2.PK, folder, "tr")
	require.Nil(t, table.StoreItem(perm))
	assert.Equal(t, user2.PK, perm.PK)
	assert.Equal(t, fmt.Sprintf("%s%s#%s#tr", PermKeyPrefix, folder.PK, folder.SK), perm.SK)
	if can, _ := folder.UserCanRead(table, user1); !can {
		t.Error("Owner should read")
	}
	if can, _ := folder.UserCanRead(table, user2); !can {
		t.Error("User with perms should be able to read")
	}
	if can, _ := folder.UserCanWrite(table, user2); can {
		t.Error("User should not be able to write")
	}
}

func TestUserEnsureDefaultFolders(t *testing.T) {
	defer stopLocalDynamo()
	table := startLocalDynamo(t)
	user, _ := NewUser("foo")
	require.Nil(t, user.EnsureDefaultFolders(table))
	var folders []*Folder
	require.Nil(t, table.FetchItemsWithPrefix(user.PK, FolderKeyPrefix, &folders))
	require.Equal(t, 4, len(folders))

	assert.Equal(t, fmt.Sprintf("%s0", FolderKeyPrefix), folders[0].SK)
	assert.Equal(t, fmt.Sprintf("%s1", FolderKeyPrefix), folders[1].SK)
	assert.Equal(t, fmt.Sprintf("%s2", FolderKeyPrefix), folders[2].SK)
	assert.Equal(t, fmt.Sprintf("%s3", FolderKeyPrefix), folders[3].SK)

}
