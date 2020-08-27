package awsapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/dmitriko/wtctrl/pkg/azr"
	tb "github.com/dmitriko/wtctrl/pkg/telebot"
)

func voiceToText(bot *tb.Bot, voice *tb.Voice) (string, error) {
	if voice.Duration > 59 {
		return "", errors.New(fmt.Sprintf("Record should be less then 60 sec, got %d", voice.Duration))
	}
	file, err := bot.GetFile(&voice.File)
	if err != nil {
		return "", err
	}
	txt, err := azr.DoSpeechRecogn(file)
	if err != nil {
		return "", err
	}
	return txt, nil
}

// Updates Msg.Data with recognized text
func updateMsgData(pk string, table *DTable, txt string) error {
	_, err := table.UpdateItemData(pk, RecognizedTextFieldName, txt)
	return err
}

func replyUser(bot *tb.Bot, tgmsg *tb.Message, txt string) error {
	_, err := bot.Send(tgmsg.Chat, txt)
	return err
}

func runSpeechRecogn(pk string, table *DTable, orig string) {
	var upd tb.Update
	err := json.Unmarshal([]byte(orig), &upd)
	if err != nil {
		fmt.Printf("ERROR: %s", err.Error())
		return
	}
	if upd.Message == nil || upd.Message.Voice == nil {
		fmt.Printf("ERROR: msg is not well formed %s", orig)
		return
	}

	bot, err := tb.NewBot(tb.Settings{
		Token:       os.Getenv("TGBOT_SECRET"),
		Synchronous: true,
	})
	if err != nil {
		fmt.Printf("ERROR creating bot %s", err.Error())
		return
	}

	if upd.Message.Voice.Duration > 59 {
		_ = replyUser(bot, upd.Message, "it's too long")
		return
	}

	txt, err := voiceToText(bot, upd.Message.Voice)
	if err != nil {
		fmt.Printf("ERROR speech recognition: %s", err.Error())
		return
	}
	if txt != "" {
		if err = updateMsgData(pk, table, txt); err != nil {
			fmt.Printf("ERROR updating msg %s with %s", pk, txt)
			return
		}
		if err = replyUser(bot, upd.Message, txt); err != nil {
			fmt.Printf("ERROR responding to user %s", err.Error())
		}
	}
}

func handleVoiceMsg(pk string, table *DTable, item map[string]events.DynamoDBAttributeValue) {
	if item["D"].DataType() == events.DataTypeMap {
		data := item["D"].Map()
		orig := data["orig"].String()
		_, ok := data[RecognizedTextFieldName]
		if !ok {
			runSpeechRecogn(pk, table, orig)
		}
	}
}

func handleNewMsg(pk string, table *DTable, item map[string]events.DynamoDBAttributeValue) {
	if item["K"].DataType() == events.DataTypeNumber {
		kind, _ := item["K"].Integer()
		if kind == TGVoiceMsgKind {
			handleVoiceMsg(pk, table, item)
			return
		}
	}
}

func notifySubsciptions(table *DTable, pk, eventName string, item map[string]events.DynamoDBAttributeValue) {
	var subs Subscriptions
	if item["UMS"].DataType() == events.DataTypeString {
		ums := item["UMS"].String()
		err := table.FetchItemsWithPrefix(ums, SubscriptionKeyPrefix, &subs)
		if err != nil {
			fmt.Println("ERROR", err.Error())
			return
		}
		for _, s := range subs {
			err = s.SendDBEvent(pk, eventName, ums)
			if err != nil {
				fmt.Println("ERROR", err.Error())
			}
		}
	}
}

func HandleDBEvent(ctx context.Context, table *DTable, e events.DynamoDBEvent) {
	/*	outputJson, err := json.Marshal(e)
		if err != nil {
			fmt.Printf("could not marshal event. details: %v", err)
			return
		}
		fmt.Printf("\n%s\n", outputJson)
	*/

	for _, record := range e.Records {
		pk := record.Change.Keys["PK"].String()
		if strings.HasPrefix(pk, MsgKeyPrefix) {
			notifySubsciptions(table, pk, record.EventName, record.Change.NewImage)
			if record.EventName == "INSERT" {
				handleNewMsg(pk, table, record.Change.NewImage)
			}
		}

	}
}
