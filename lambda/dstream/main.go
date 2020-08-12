package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dmitriko/wtctrl/pkg/awsapi"
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
func updateMsgData(pk, txt string) error {
	table, _ := awsapi.NewDTable(os.Getenv("TABLE_NAME"))
	err := table.Connect()
	if err != nil {
		return err
	}
	_, err = table.UpdateItemData(pk, awsapi.RecognizedTextFieldName, txt)
	return err
}

func replyUser(bot *tb.Bot, tgmsg *tb.Message, txt string) error {
	_, err := bot.Send(tgmsg.Chat, txt)
	return err
}

func runSpeechRecogn(pk, orig string) {
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
		if err = updateMsgData(pk, txt); err != nil {
			fmt.Printf("ERROR updating msg %s with %s", pk, txt)
			return
		}
		if err = replyUser(bot, upd.Message, txt); err != nil {
			fmt.Printf("ERROR responsing to user %s", err.Error())
		}
	}
}

func handleVoiceMsg(pk string, item map[string]events.DynamoDBAttributeValue) {
	if item["D"].DataType() == events.DataTypeMap {
		data := item["D"].Map()
		orig := data["orig"].String()
		_, ok := data[awsapi.RecognizedTextFieldName]
		if !ok {
			runSpeechRecogn(pk, orig)
		}
	}
}

func handleMsg(pk string, item map[string]events.DynamoDBAttributeValue) {
	if item["K"].DataType() == events.DataTypeNumber {
		kind, _ := item["K"].Integer()
		if kind == awsapi.TGVoiceMsgKind {
			handleVoiceMsg(pk, item)
			return
		}
	}
}

func handleRequest(ctx context.Context, e events.DynamoDBEvent) {
	for _, record := range e.Records {
		pk := record.Change.Keys["PK"].String()
		fmt.Printf("Processing %s", pk)
		if strings.HasPrefix(pk, awsapi.MsgKeyPrefix) {
			handleMsg(pk, record.Change.NewImage)
		}

	}
}

func main() {
	lambda.Start(handleRequest)
}
