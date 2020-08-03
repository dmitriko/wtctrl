package awsapi

import (
	"fmt"
	"testing"
)

// /start <code> message with valid code

const TGTextMsgTmpl = `{"message_id": 181,
        "from": {"id": %[1]d,
         "is_bot": false,
         "first_name": "D",
         "last_name": "K",
         "language_code": "en"},
        "chat": {"id": %[1]d,
         "first_name": "D",
         "last_name": "K",
         "type": "private"},
        "date": 1571403733,
        "text": "%[2]s"}`

func TestScenarioTGStartValidCode(t *testing.T) {
	defer stopLocalDynamo()
	startLocalDynamo(t)
	tgid := 123456789
	bot, _ := NewBot("foobot", "secret", TGBotKind)
	user, _ := NewUser("someuser")
	inv, _ := NewInvite(user, bot, 24)
	errs := testTable.StoreItems(bot, user, inv)
	for _, e := range errs {
		if e != nil {
			t.Error(e)
		}
	}
	orig := fmt.Sprintf(TGTextMsgTmpl, tgid, "/start "+inv.OTP)

	_, err := HandleTGMsg(bot, testTable, orig)
	if err != nil {
		t.Error(err)
	}
	tgacc := &TGAcc{}
	err = testTable.FetchItem(TGAccKeyPrefix+string(tgid), tgacc)
	if err != nil {
		t.Error(err)
	}
}

func TestScenarioTGStartNotValidCode(t *testing.T) {
	defer stopLocalDynamo()
	startLocalDynamo(t)
	tgid := 123456789
	bot, _ := NewBot("foobot", "secret", TGBotKind)
	user, _ := NewUser("someuser")
	inv, _ := NewInvite(user, bot, 24)
	errs := testTable.StoreItems(bot, user, inv)
	for _, e := range errs {
		if e != nil {
			t.Error(e)
		}
	}
	orig := fmt.Sprintf(TGTextMsgTmpl, tgid, "/start "+"000000")

	resp, err := HandleTGMsg(bot, testTable, orig)
	if err != nil {
		t.Error(err)
	}
	if resp != WRONG_CODE {
		t.Errorf("Expected wrong code response, got %s", resp)
	}
}

func _TestScenarioTGText(t *testing.T) {
	defer stopLocalDynamo()
	startLocalDynamo(t)
	tgid := 123456789
	bot, _ := NewBot("foobot", "secret", TGBotKind)
	user, _ := NewUser("someuser")
	tgacc, _ := NewTGAcc(string(tgid), user.PK())
	user.TGID = string(tgid)
	errs := testTable.StoreItems(bot, user, tgacc)
	for _, e := range errs {
		if e != nil {
			t.Error(e)
		}
	}
	text := "65 euro gas station"
	orig := fmt.Sprintf(TGTextMsgTmpl, tgid, text)
	_, err := HandleTGMsg(bot, testTable, orig)
	if err != nil {
		t.Error(err)
	}
	lm := NewListMsg()
	err = lm.FetchByUserStatus(testTable, user, 0, "-2d", "now")
	if err != nil {
		t.Error(err)
	}
	if lm.Len() != 1 {
		t.Error("expected 1 Msg in DB")
	}
	for _, msg := range lm.Items {
		txt, ok := msg.Data["text"]
		if !ok && txt != text {
			t.Errorf("expected msg with text %s got %+v", text, msg)
		}
	}
}
