package awsapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	tb "github.com/dmitriko/wtctrl/pkg/telebot"
)

const (
	NEED_CODE  = "Please, provide an invitation code."
	WRONG_CODE = "This code is wrong or expired."
	WELCOME    = "Welcome!"
)

var CODE_REGEXP = regexp.MustCompile(`\d{6}`)

func handleTGStartMsg(bot *Bot, table *DTable, tgmsg *tb.Message) (string, error) {
	var err error
	code := CODE_REGEXP.FindString(tgmsg.Text)
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
	err = table.StoreUserTG(user, tgmsg.Sender.ID, bot)
	if err != nil {
		return "", err
	}
	inv.Data["accepted"] = fmt.Sprintf("%d", time.Now().Unix())
	err = table.StoreItem(inv)
	if err != nil {
		//ignoring it
		log.Printf("could not store Invite %+v, reason: %s", inv, err.Error())
	}
	return WELCOME, nil
}

// Handles message got via webhook from Telegram
func HandleTGMsg(bot *Bot, table *DTable, orig string) (string, error) {
	var upd tb.Update
	err := json.Unmarshal([]byte(orig), &upd)
	if err != nil {
		return "", err
	}

	tgmsg := upd.Message
	if tgmsg == nil {
		return "", errors.New("Message is not supported")
	}

	// Handle message form non auth user with /start <code>, just <code> or just /start
	if strings.HasPrefix(tgmsg.Text, "/start") {
		return handleTGStartMsg(bot, table, tgmsg)
	}
	tgacc := &TGAcc{}
	err = table.FetchTGAcc(tgmsg.Sender.ID, tgacc)
	if err != nil {
		if err.Error() == NO_SUCH_ITEM {
			if len(tgmsg.Text) == 6 && CODE_REGEXP.MatchString(tgmsg.Text) {
				return handleTGStartMsg(bot, table, tgmsg)
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

	msg, err := NewMsg(bot.PK, user.PK, TGUnknownMsgKind)
	if err != nil {
		return "", err
	}

	msg.Data["orig"] = orig

	if tgmsg.Text != "" {
		msg.Data["text"] = tgmsg.Text
		msg.Kind = TGTextMsgKind
	}

	if tgmsg.Photo != nil {
		msg.Kind = TGPhotoMsgKind
		if tgmsg.Photo.Caption != "" {
			msg.Data["text"] = tgmsg.Photo.Caption
		}
	}

	if tgmsg.Voice != nil {
		msg.Kind = TGVoiceMsgKind
	}

	err = table.StoreItem(msg)
	return "", err
}

type DummyTGBot struct {
	ChatID string
	Sent   string
}

func (b *DummyTGBot) Send(to tb.Recipient, what interface{}, options ...interface{}) (*tb.Message, error) {
	b.ChatID = to.Recipient()
	b.Sent = what.(string)
	return &tb.Message{}, nil
}

var dummyTGBot *DummyTGBot

func BotSendText(table *DTable, bot *Bot, user *User, text string) error {
	var sendFunc func(to tb.Recipient, what interface{}, options ...interface{}) (*tb.Message, error)
	if bot.Kind == TGBotKind {
		tgbot, err := tb.NewBot(tb.Settings{
			Token:       bot.Secret,
			Synchronous: true,
		})
		if err != nil {
			return err
		}
		sendFunc = tgbot.Send
	} else {
		dummyTGBot = &DummyTGBot{}
		sendFunc = dummyTGBot.Send
	}
	_, err := sendFunc(user, text)
	if err != nil {
		return err
	}
	return nil

}

func SendOtp(table *DTable, userPK, otp string) error {
	var err error
	user := &User{}
	err = table.FetchItem(userPK, user)
	if err != nil {
		return err
	}
	var bot *Bot
	if len(user.Bots) > 0 {
		botPK := user.Bots[0]
		err := table.FetchItem(botPK, &bot)
		if err != nil {
			return err
		}
	} else {
		errors.New("User does not use any bots")
	}
	return BotSendText(table, bot, user, otp)
}
