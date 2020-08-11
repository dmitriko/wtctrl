package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dmitriko/wtctrl/pkg/awsapi"
	tb "github.com/dmitriko/wtctrl/pkg/telebot"
)

const TGBOT_NAME = "wtctrlbot"

func handleMessage(body string) error {
	var err error
	table_name := os.Getenv("TABLE_NAME")
	if table_name == "" {
		return errors.New("BOT_SECRET or TABLE_NAME not set")
	}
	table, _ := awsapi.NewDTable(table_name)
	err = table.Connect()
	if err != nil {
		return err
	}
	dbBot, _ := awsapi.NewBot(awsapi.TGBotKind, TGBOT_NAME)
	resp, err := awsapi.HandleTGMsg(dbBot, table, body)
	if err != nil {
		return err
	}

	if resp != "" {
		bot_secret := os.Getenv("TGBOT_SECRET")
		var upd tb.Update
		_ = json.Unmarshal([]byte(body), &upd)
		bot, err := tb.NewBot(tb.Settings{
			Token:       bot_secret,
			Synchronous: true,
		})
		if err != nil {
			log.Printf("Could not init Bot %s", err.Error())
			return nil
		}
		if upd.Message != nil {
			tr, _ := bot.Send(upd.Message.Chat, resp)
			log.Printf("got response from TG %+v", tr)
		}
	}
	return nil
}

func handleRequest(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	res := events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
	}

	err := handleMessage(req.Body)

	if err != nil {
		log.Println("ERROR processing:")
		log.Println(req.Body)
		log.Println(err)
		return res, err
	}

	res = events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    map[string]string{"Content-Type": "text/plan; charset=utf-8"},
		Body:       "ok",
	}
	return res, nil
}

func main() {
	lambda.Start(handleRequest)
}
