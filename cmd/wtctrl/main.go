package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/dmitriko/wtctrl/pkg/awsapi"
	"github.com/docopt/docopt-go"
)

func main() {
	usage := `Web Tech Control CLI

Usage:
  wtctrl tgbot register [--table=<table>] [--region=<region>] [--endpoint=<url>] [--bot-name=<name>] [--secret=<secret>]
  wtctrl tgbot invite  [--table=<table>] [--region=<region>] [--endpoint=<url>] [--bot-name=<name>] --title=<title>  [--email=<email>] [--tel=<telephone>]
  wtctrl user create-token [--table=<table>] [--region=<region>] [--endpoint=<url>] [--tel=<telephone>] [--email=<email>]
  wtctrl user send-ws [--table=<table>] [--region=<region>] [--endpoint=<url>] [--tel=<telephone>] [--email=<email>] -m=<message>
  wtctrl -h | --help

Options:
  -h --help           Show this screen.
  --table=<table>     DynamoDB table name, default to $DYNAMO_TABLE
  --region=<region>   DynamoDB region, default to $DYNAMO_REGION
  --endpoint=<url>    DynamoDB endpoint for local testing, no default
  --bot-name=<name>   Name of Telegram bot, default to $TGBOT_NAME
  --secret=<secret>   Secret code of the bot, defaut to $TGBOT_SECRET
  --title=<title>     Title the only required flag to create new user via invite
  -m=<message>        Text send to user
`

	args, _ := docopt.ParseDoc(usage)
	//fmt.Printf("%#v", args)
	var err error
	if isTgbot, ok := args["tgbot"]; ok && isTgbot.(bool) {
		err = tgbot(args)
	}
	if args["user"].(bool) {
		err = user(args)
	}
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Done.")
}

var REQUIRED_DEFAULTS = map[string]string{
	"--bot-name": "TGBOT_NAME",
	"--secret":   "TGBOT_SECRET",
}
var TABLE_ARG_DEFAULTS = map[string]string{
	"--table":  "DYNAMO_TABLE",
	"--region": "DYNAMO_REGION",
}

func tableFromArgs(args map[string]interface{}) (*awsapi.DTable, error) {
	for k, v := range TABLE_ARG_DEFAULTS {
		if args[k] == nil {
			args[k] = os.Getenv(v)
		}
	}
	tableName := args["--table"].(string)
	region := args["--region"].(string)
	var endpoint string
	if args["--endpoint"] != nil {
		endpoint = args["--endpoint"].(string)
	}
	if tableName == "" || region == "" {
		return nil, errors.New("--table and --region must be set")
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

func userFromArgs(table *awsapi.DTable, args map[string]interface{}) (*awsapi.User, error) {
	user := &awsapi.User{}
	var email, tel string
	if args["--email"] != nil {
		email = args["--email"].(string)
	}
	if args["--tel"] != nil {
		tel = args["--tel"].(string)
	}
	if tel != "" {
		item := &awsapi.Tel{}
		pk := fmt.Sprintf("%s%s", awsapi.TelKeyPrefix, tel)
		//		fmt.Println("Fetching ", pk)
		err := table.FetchItem(pk, item)
		if err != nil {
			return nil, err
		}
		err = table.FetchItem(item.OwnerPK, user)
		if err != nil {
			return nil, err
		}
		return user, nil
	}
	if email != "" {
		item := &awsapi.Email{}
		pk := fmt.Sprintf("%s%s", awsapi.EmailKeyPrefix, email)
		//		fmt.Println("Fetching ", pk)
		err := table.FetchItem(pk, item)
		if err != nil {
			return nil, err
		}
		err = table.FetchItem(item.OwnerPK, user)
		if err != nil {
			return nil, err
		}
		return user, nil
	}
	return nil, errors.New("--tel OR --email must be set")
}

func user(args map[string]interface{}) error {
	table, err := tableFromArgs(args)
	if err != nil {
		return err
	}
	user, err := userFromArgs(table, args)
	if err != nil {
		return err
	}
	if args["create-token"].(bool) {
		return userCreateToken(table, user)
	}
	if args["send-ws"].(bool) {
		m := args["-m"].(string)
		return userSendWS(table, user, m)
	}
	return errors.New("No proper command was given.")
}

func userSendWS(table *awsapi.DTable, user *awsapi.User, msg string) error {
	conns := []*awsapi.WSConn{}
	sess := session.New(&aws.Config{
		Region: aws.String("us-west-2"),
	})
	err := user.FetchWSConns(table, &conns)
	if err != nil {
		return err
	}
	if len(conns) == 0 {
		return errors.New("User has no open connections")
	}
	for _, conn := range conns {
		err = conn.Send(sess, []byte(msg))
		if err != nil {
			fmt.Println(err.Error())
		}
	}
	return nil
}

func userCreateToken(table *awsapi.DTable, user *awsapi.User) error {
	token, _ := awsapi.NewToken(user, 24)
	err := table.StoreItem(token)
	if err != nil {
		return err
	}
	fmt.Println(awsapi.PK2ID(awsapi.TokenKeyPrefix, token.PK))
	return nil
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
	err := table.StoreItem(bot, awsapi.UniqueOp())
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
	err = table.StoreItem(inv, awsapi.UniqueOp())
	if err != nil {
		return err
	}
	fmt.Printf("Please, use this url to start messaging: %s \n", inv.Url)
	return nil
}
