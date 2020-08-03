package awsapi

import (
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

func handleTGStartMsg(bot *Bot, table *DTable, text, tgid string) (string, error) {
	var err error
	r := regexp.MustCompile(`\d{6}`)
	code := r.FindString(text)
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
	msg, err := NewTgUserMsg(orig)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(msg.Text, "/start") {
		return handleTGStartMsg(bot, table, msg.Text, msg.UserID())
	}
	return "", err
}
