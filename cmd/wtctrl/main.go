package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/dmitriko/wtctrl/pkg/awsapi"
	"github.com/docopt/docopt-go"
)

func main() {
	usage := `Web Tech Control CLI

Usage:
  wtctrl tgbot register [--table=<table>] [--region=<region>] [--endpoint=<url>] [--bot-name=<name>] [--secret=<secret>]
  wtctrl tgbot invite  [--table==<table>] [--region=<region>] [--endpoint=<url>] [--bot-name=<name>] --title=<title>  [--email=<email>] [--tel=<telephone>]
  wtctrl user create-token (--email|--tel)
  wtctrl -h | --help

Options:
  -h --help           Show this screen.
  --table=<table>     DynamoDB table name, default to $DYNAMO_TABLE
  --region=<region>   DynamoDB region, default to $DYNAMO_REGION
  --endpoint=<url>    DynamoDB endpoint for local testing, no default
  --bot-name=<name>   Name of Telegram bot, default to $TGBOT_NAME
  --secret=<secret>   Secret code of the bot, defaut to $TGBOT_SECRET
  --title=<title>     Title the only required flag to create new user via invite
`

	args, _ := docopt.ParseDoc(usage)
	var err error
	if args["tgbot"].(bool) {
		err = tgbot(args)
	}
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Done.")
}

var REQUIRED_DEFAULTS = map[string]string{
	"--bot-name": "TGBOT_NAME",
	"--secret":   "TGBOT_SECRET",
	"--table":    "DYNAMO_TABLE",
	"--region":   "DYNAMO_REGION",
}

func tableFromArgs(args map[string]interface{}) (*awsapi.DTable, error) {
	tableName := args["--table"].(string)
	region := args["--region"].(string)
	var endpoint string
	if args["--endpoint"] != nil {
		endpoint = args["--endpoint"].(string)
	}
	table, _ := awsapi.NewDTable(tableName)
	table.Region = region
	if endpoint != "" {
		table.Endpoint = endpoint
	}
	err := table.Connect()
	if err != nil {
		return nil, err
	}
	return table, nil
}

func tgbot(args map[string]interface{}) error {
	for k, v := range REQUIRED_DEFAULTS {
		if args[k] == nil {
			args[k] = os.Getenv(v)
		}
	}
	var errs []string
	for k, _ := range REQUIRED_DEFAULTS {
		if args[k].(string) == "" || args[k] == nil {
			errs = append(errs, fmt.Sprintf("%s must be set", k))
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, " "))
	}

	botName := args["--bot-name"].(string)
	secret := args["--secret"].(string)
	table, err := tableFromArgs(args)
	if err != nil {
		return err
	}
	if args["register"].(bool) {
		return tgbotRegister(table, botName, secret)
	}
	if args["invite"].(bool) {
		var title, tel, email string
		tel, _ = args["--tel"].(string)
		email, _ = args["--email"].(string)
		title, _ = args["--title"].(string)
		return tgbotInvite(table, botName, title, email, tel)
	}
	return nil
}

func tgbotRegister(table *awsapi.DTable, botName, secret string) error {
	fmt.Println("Registering ", botName)
	bot, _ := awsapi.NewBot(awsapi.TGBotKind, botName)
	bot.Secret = secret
	_, err := table.StoreItem(bot, awsapi.UniqueOp())
	if err != nil {
		return err
	}
	return nil
}

func tgbotInvite(table *awsapi.DTable, botName, title, email, tel string) error {
	var err error
	if title == "" {
		return errors.New("--title must be provided")
	}
	user, _ := awsapi.NewUser(title)
	if tel != "" {
		if err = user.SetTel(tel); err != nil {
			return err
		}
	}
	if email != "" {
		if err = user.SetEmail(email); err != nil {
			return err
		}
	}
	bot, _ := awsapi.NewBot(awsapi.TGBotKind, botName)
	inv, _ := awsapi.NewInvite(user, bot, 24)
	err = table.StoreNewUser(user)
	if err != nil {
		return err
	}
	_, err = table.StoreItem(inv, awsapi.UniqueOp())
	if err != nil {
		return err
	}
	fmt.Printf("Please, use this url to start messaging: %s \n", inv.Url)
	return nil
}
