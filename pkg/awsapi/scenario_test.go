package awsapi

import (
	"fmt"
	"testing"
)

// /start <code> message with valid code

const TGTextMsgTmpl = `{ "update_id": 45554171,
  "message": {
   "message_id": 181,
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
        "text": "%[2]s"}
		}`

const TGVoiceMsgTmpl = `{"update_id":45554177,
 "message": {
  "message_id": 92,
  "from": {
    "id": %[1]d,
    "is_bot": false,
    "first_name": "D",
    "last_name": "K",
    "language_code": "en"
  },
  "chat": {
    "id": %[1]d,
    "first_name": "D",
    "last_name": "K",
    "type": "private"
  },
  "date": 1570375932,
  "voice": {
    "duration": %[2]d,
    "mime_type": "audio/ogg",
    "file_id": "%[3]s",
    "file_size": 5070
  }
}
}`

const TGPhotoMsgTmpl = `
{"update_id":45554176,
 "message": {
  "message_id": 67,
  "from": {
    "id": %[1]d,
    "is_bot": false,
    "first_name": "D",
    "last_name": "K",
    "language_code": "en"
  },
  "chat": {
    "id": %[1]d,
    "first_name": "D",
    "last_name": "K",
    "type": "private"
  },
  "date": 1570263478,
  "photo": [
    {
      "file_id": "AgADAgADkKsxG8qRwUiqYAcM2WqnNUTauQ8ABAEAAwIAA20AA_NLAwABFgQ",
      "file_size": 12635,
      "width": 180,
      "height": 320
    },
    {
      "file_id": "AgADAgADkKsxG8qRwUiqYAcM2WqnNUTauQ8ABAEAAwIAA3gAA_RLAwABFgQ",
      "file_size": 49078,
      "width": 450,
      "height": 800
    },
    {
      "file_id": "AgADAgADkKsxG8qRwUiqYAcM2WqnNUTauQ8ABAEAAwIAA3kAA_FLAwABFgQ",
      "file_size": 73321,
      "width": 720,
      "height": 1280
    }
  ]
 }
}
`

const TGDocMsgTmpl = `
{"update_id":45554177,
 "message": {
    "message_id": 9,
    "from": {
        "id":  %[1]d,
        "is_bot": false,
        "first_name": "D",
        "last_name": "K",
        "language_code": "en"
    },
    "chat": {
        "id":  %[1]d,
        "first_name": "D",
        "last_name": "K",
        "type": "private"
    },
    "date": 1597084525,
    "document": {
        "file_name": "IMG_20200802_195818.jpg",
        "mime_type": "image/jpeg",
        "thumb": {
            "file_id": "AAMCAgADGQEAAwlfMZNtVktT1WdNuj9C_14AASnIEPQAAoAIAAJawYhJucPv_LtVYgNQDWWWLgADAQAHbQADzRIAAhoE",
            "file_unique_id": "AQADUA1lli4AA80SAAI",
            "file_size": 8914,
            "width": 320,
            "height": 240
        },
        "file_id": "BQACAgIAAxkBAAMJXzGTbVZLU9VnTbo_Qv9eAAEpyBD0AAKACAACWsGISbnD7_y7VWIDGgQ",
        "file_unique_id": "AgADgAgAAlrBiEk",
        "file_size": 4398585
    }
}`

func TestScenarioTGStartValidCode(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	tgid := 123456789
	bot, _ := NewBot(TGBotKind, "foobot")
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
	err = testTable.FetchTGAcc(tgid, tgacc)
	if err != nil {
		t.Error(err)
	}
}

func TestScenarioTGStartNotValidCode(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	tgid := 123456789
	bot, _ := NewBot(TGBotKind, "fooboot")
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

func TestScenarioTGAuthText(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	tgid := 123456789
	bot, _ := NewBot(TGBotKind, "foobot")
	user, _ := NewUser("someuser")
	tgacc, _ := NewTGAcc(tgid, user.PK)
	user.TGID = tgacc.TGID
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
	err = lm.FetchByUserStatus(testTable, user.PK, 0, "-2d", "now")
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

//User sent message with valid code only
func TestScenarioTGValidCode(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	tgid := 123456789
	bot, _ := NewBot(TGBotKind, "foobot")
	user, _ := NewUser("someuser")
	inv, _ := NewInvite(user, bot, 24)
	errs := testTable.StoreItems(bot, user, inv)
	for _, e := range errs {
		if e != nil {
			t.Error(e)
		}
	}
	orig := fmt.Sprintf(TGTextMsgTmpl, tgid, inv.OTP)

	resp, err := HandleTGMsg(bot, testTable, orig)
	if err != nil {
		t.Error(err)
	}
	tgacc := &TGAcc{}
	err = testTable.FetchTGAcc(tgid, tgacc)
	if err != nil {
		t.Error(err)
	}
	if resp != WELCOME {
		t.Error("Expected welcome message")
	}
}

// User sent no code no /start message but has no account
func TestScenarioTGNonAuth(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	tgid := 123456789
	bot, _ := NewBot(TGBotKind, "foobot")
	errs := testTable.StoreItems(bot)
	for _, e := range errs {
		if e != nil {
			t.Error(e)
		}
	}
	text := "65 euro gas station"
	orig := fmt.Sprintf(TGTextMsgTmpl, tgid, text)
	resp, err := HandleTGMsg(bot, testTable, orig)
	if err != nil {
		t.Error(err)
	}
	if resp != NEED_CODE {
		t.Error("expected need code response")
	}
}

func TestScenarioTGVoice(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	tgid := 123456789
	bot, _ := NewBot(TGBotKind, "foobot")
	user, _ := NewUser("someuser")
	tgacc, _ := NewTGAcc(tgid, user.PK)
	user.TGID = tgacc.TGID
	errs := testTable.StoreItems(bot, user, tgacc)
	for _, e := range errs {
		if e != nil {
			t.Error(e)
		}
	}
	orig := fmt.Sprintf(TGVoiceMsgTmpl, tgid, 1, "sometgfileid")
	_, err := HandleTGMsg(bot, testTable, orig)
	if err != nil {
		t.Error(err)
	}
	lm := NewListMsg()
	err = lm.FetchByUserStatus(testTable, user.PK, 0, "-2d", "now")
	if err != nil {
		t.Error(err)
	}
	if lm.Len() != 1 {
		t.Error("expected 1 Msg in DB")
	}
	for _, msg := range lm.Items {
		if msg.Data["orig"] != orig {
			t.Errorf("expected %s, got %s", orig, msg.Data["orig"])
		}
		if msg.Kind != TGVoiceMsgKind {
			t.Errorf("msg.Kind is not correct, got %v expected %v", msg.Kind, TGVoiceMsgKind)
		}
	}
}

func TestScenarioTGPhoto(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	tgid := 123456789
	bot, _ := NewBot(TGBotKind, "foobot")
	user, _ := NewUser("someuser")
	tgacc, _ := NewTGAcc(tgid, user.PK)
	user.TGID = tgacc.TGID
	errs := testTable.StoreItems(bot, user, tgacc)
	for _, e := range errs {
		if e != nil {
			t.Error(e)
		}
	}
	orig := fmt.Sprintf(TGPhotoMsgTmpl, tgid)
	_, err := HandleTGMsg(bot, testTable, orig)
	if err != nil {
		t.Error(err)
	}
	lm := NewListMsg()
	err = lm.FetchByUserStatus(testTable, user.PK, 0, "-2d", "now")
	if err != nil {
		t.Error(err)
	}
	if lm.Len() != 1 {
		t.Error("expected 1 Msg in DB")
	}
	for _, msg := range lm.Items {
		if msg.Kind != TGPhotoMsgKind {
			t.Errorf("msg.Kind is not correct, got %v expected %v", msg.Kind, TGPhotoMsgKind)
		}
		if msg.Data["orig"] != orig {
			t.Errorf("expected %s, got %s", orig, msg.Data["orig"])
		}

	}
}
