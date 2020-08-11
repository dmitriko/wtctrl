package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	tb "github.com/dmitriko/wtctrl/pkg/telebot"
)

func handleMessage(body string) error {
	//bot_name := os.Genenv("TGBOT_NAME")
	bot_secret := os.Getenv("TGBOT_SECRET")

	if bot_secret == "" {
		return errors.New("BOT_NAME or BOT_SECRET not set")
	}
	_, err := tb.NewBot(tb.Settings{
		Token:       bot_secret,
		Synchronous: true,
	})
	if err != nil {
		return err
	}
	var upd tb.Update
	if err = json.Unmarshal([]byte(body), &upd); err == nil {
		fmt.Printf("%+v", upd.Message)
	}
	return nil
}

func handleRequest(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	log.Println(req.Body)

	res := events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
	}

	if err := handleMessage(req.Body); err != nil {
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
