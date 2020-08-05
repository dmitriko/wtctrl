package awsapi

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
)

const (
	NEED_CODE  = "Please, provide invitation code."
	WRONG_CODE = "This code is wrong or expired"
	WELCOME    = "Welcome!"
)

var CODE_REGEXP = regexp.MustCompile(`\d{6}`)

func handleTGAuthTextMsg(bot *Bot, table *DTable, user *User, tgmsg *TGUserMsg) (string, error) {
	msg, err := NewMsg(bot.PK(), user.PK(), TGTextMsgKind)
	if err != nil {
		return "", err
	}
	msg.Data["text"] = tgmsg.Text
	_, err = table.StoreItem(msg)
	return "", err
}

func handleTGStartMsg(bot *Bot, table *DTable, text, tgid string) (string, error) {
	var err error
	code := CODE_REGEXP.FindString(text)
	if code == "" {
		return NEED_CODE, nil
	}
	inv := &Invite{}
	err = table.FetchInvite(bot, code, inv)
	if err != nil && err.Error() == NO_SUCH_ITEM {
		return WRONG_CODE, nil
	}
	user := &User{}
	err = table.FetchItem(inv.UserPK, user)
	if err != nil {
		log.Printf("ERROR: Could not find user for invite %+v", inv)
		return "", err
	}
	err = table.StoreUserTG(user, tgid, bot)
	if err != nil {
		return "", err
	}
	inv.Data["accepted"] = string(time.Now().Unix())
	_, err = table.StoreItem(inv)
	if err != nil {
		//ignoring it
		log.Printf("could not store Invite %+v, reason: %s", inv, err.Error())
	}
	return WELCOME, nil
}

// Handles message got via webhook from Telegram
func HandleTGMsg(bot *Bot, table *DTable, orig string) (string, error) {
	tgmsg, err := NewTgUserMsg(orig)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(tgmsg.Text, "/start") {
		return handleTGStartMsg(bot, table, tgmsg.Text, tgmsg.TGID())
	}
	tgacc := &TGAcc{}
	err = table.FetchItem(TGAccKeyPrefix+tgmsg.TGID(), tgacc)
	if err != nil {
		if err.Error() == NO_SUCH_ITEM {
			if len(tgmsg.Text) == 6 && CODE_REGEXP.MatchString(tgmsg.Text) {
				return handleTGStartMsg(bot, table, tgmsg.Text, tgmsg.TGID())
			}
			return NEED_CODE, nil
		}
		return "", err
	}
	user := &User{}
	err = table.FetchItem(tgacc.OwnerPK, user)
	if err != nil {
		return "", err
	}
	if tgmsg.IsAudio() {
		return handleTGAuthAudioMsg(bot, table, user, tgmsg)
	}
	return handleTGAuthTextMsg(bot, table, user, tgmsg)
}

func handleTGAuthAudioMsg(bot *Bot, table *DTable, user *User, tgmsg *TGUserMsg) (string, error) {
	msg, err := NewMsg(bot.PK(), user.PK(), TGVoiceMsgKind)
	if err != nil {
		return "", err
	}
	msg.Data["orig_duration"] = fmt.Sprintf("%d", tgmsg.Voice.Duration)
	msg.Data["orig_mime_type"] = tgmsg.Voice.MimeType
	msg.Data["orig_file_id"] = tgmsg.Voice.FileId
	msg.Data["orig_file_size"] = fmt.Sprintf("%d", tgmsg.Voice.FileSize)
	_, err = table.StoreItem(msg)
	return "", err
}
