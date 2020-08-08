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

const TGAudioMsgTmpl = `{
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
}`

const TGPhotoMsgTmpl = `
{
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
}`

func TestScenarioTGStartValidCode(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
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
	err = testTable.FetchTGAcc(tgid, tgacc)
	if err != nil {
		t.Error(err)
	}
}

func TestScenarioTGStartNotValidCode(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
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

func TestScenarioTGAuthText(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	tgid := 123456789
	bot, _ := NewBot("foobot", "secret", TGBotKind)
	user, _ := NewUser("someuser")
	tgacc, _ := NewTGAcc(tgid, user.PK())
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

//User sent message with valid code only
func TestScenarioTGValidCode(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
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
	bot, _ := NewBot("foobot", "secret", TGBotKind)
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

func TestScenarioTGAudio(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	tgid := 123456789
	bot, _ := NewBot("foobot", "secret", TGBotKind)
	user, _ := NewUser("someuser")
	tgacc, _ := NewTGAcc(tgid, user.PK())
	user.TGID = tgacc.TGID
	errs := testTable.StoreItems(bot, user, tgacc)
	for _, e := range errs {
		if e != nil {
			t.Error(e)
		}
	}
	orig := fmt.Sprintf(TGAudioMsgTmpl, tgid, 1, "sometgfileid")
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
		if msg.Data["orig_duration"] != "1" || msg.Data["orig_file_id"] != "sometgfileid" ||
			msg.Data["orig_mime_type"] != "audio/ogg" || msg.Data["orig_file_size"] != "5070" {
			t.Errorf("expected msg with orig audio data got %+v", msg.Data)
		}
	}
}

func TestScenarioTGPhoto(t *testing.T) {
	defer stopLocalDynamo()
	testTable := startLocalDynamo(t)
	tgid := 123456789
	bot, _ := NewBot("foobot", "secret", TGBotKind)
	user, _ := NewUser("someuser")
	tgacc, _ := NewTGAcc(tgid, user.PK())
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
	err = lm.FetchByUserStatus(testTable, user, 0, "-2d", "now")
	if err != nil {
		t.Error(err)
	}
	if lm.Len() != 1 {
		t.Error("expected 1 Msg in DB")
	}
	data := map[string]string{
		"orig_0_file_id":   "AgADAgADkKsxG8qRwUiqYAcM2WqnNUTauQ8ABAEAAwIAA20AA_NLAwABFgQ",
		"orig_0_file_size": "12635",
		"orig_0_width":     "180",
		"orig_0_height":    "320",
		"orig_1_file_id":   "AgADAgADkKsxG8qRwUiqYAcM2WqnNUTauQ8ABAEAAwIAA3gAA_RLAwABFgQ",
		"orig_1_file_size": "49078",
		"orig_1_width":     "450",
		"orig_1_height":    "800",
		"orig_2_file_id":   "AgADAgADkKsxG8qRwUiqYAcM2WqnNUTauQ8ABAEAAwIAA3kAA_FLAwABFgQ",
		"orig_2_file_size": "73321",
		"orig_2_width":     "720",
		"orig_2_height":    "1280",
	}
	for _, msg := range lm.Items {
		for k, _ := range data {
			if data[k] != msg.Data[k] {
				t.Errorf("%s are not equal, expected %s, got %s", k, data[k], msg.Data[k])
			}
		}
	}
}
