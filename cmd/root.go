package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/dmitriko/wtctrl/pkg/awsapi"
	"github.com/spf13/cobra"
)

var tgbotName string
var tgbotSecret string
var userEmail string
var userTel string
var userTitle string
var tableName string
var tableRegion string
var tableEndpoint string

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wtctrl",
		Short: "Manage Web Tech Control Application",
	}
	cmd.AddCommand(tgbotRootCmd())
	return cmd
}

func tgbotRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tgbot",
		Short: "Manage Telegram Bot",
	}
	cmd.AddCommand(tgbotRegisterCmd())
	cmd.AddCommand(tgbotInviteUserCmd())
	return cmd
}

func tgbotRegisterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register bot in DynamoDB",
		RunE: func(cmd *cobra.Command, args []string) error {
			return registerBot()
		},
	}
	registerTableFlags(cmd)
	registerTGBotFlags(cmd)
	return cmd
}

func tgbotInviteUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invite",
		Short: "Create User, Invite in DB and print invite url",
		RunE: func(cmd *cobra.Command, args []string) error {
			return inviteUser()
		},
	}
	registerTableFlags(cmd)
	registerTGBotFlags(cmd)
	registerUserFlags(cmd)
	cmd.MarkFlagRequired("title")
	return cmd

}

//Register flags related to DynamoDB
func registerTableFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&tableName, "table-name", os.Getenv("DYNAMO_TABLE"), "DynamoDB table name")
	cmd.Flags().StringVar(&tableRegion, "region", os.Getenv("DYNAMO_REGION"), "AWS region for dynamo db")
	cmd.Flags().StringVar(&tableEndpoint, "endpoint", os.Getenv("DYNAMO_ENDPOINT"), "Endpoint for DynamoDB")
}

func registerTGBotFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&tgbotName, "bot-name", os.Getenv("TGBOT_NAME"), "Telegram Bot name")
	cmd.Flags().StringVar(&tgbotSecret, "secret", os.Getenv("TGBOT_SECRET"), "Telegram Bot secret")
}

func registerUserFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&userTel, "tel", "", "User telephone number")
	cmd.Flags().StringVar(&userEmail, "email", "", "User's email")
	cmd.Flags().StringVar(&userTitle, "title", "", "User's title")
}

func registerBot() error {
	if tgbotSecret == "" {
		return errors.New("--secret must be set")
	}
	if tgbotName == "" {
		return errors.New("--bot-name must be set")
	}
	table, _ := awsapi.NewDTable(tableName)
	if tableRegion != "" {
		table.Region = tableRegion
	}
	if tableEndpoint != "" {
		table.Endpoint = tableEndpoint
	}
	err := table.Connect()
	if err != nil {
		return err
	}

	bot, _ := awsapi.NewBot(awsapi.TGBotKind, tgbotName)
	bot.Secret = tgbotSecret
	_, err = table.StoreItem(bot, awsapi.UniqueOp())
	if err != nil {
		return err
	}
	fmt.Printf("Telegram Bot with name %s is registered.\n", tgbotName)
	return nil
}

func inviteUser() error {
	table, _ := awsapi.NewDTable(tableName)
	if tableRegion != "" {
		table.Region = tableRegion
	}
	if tableEndpoint != "" {
		table.Endpoint = tableEndpoint
	}
	err := table.Connect()
	if err != nil {
		return err
	}

	return nil
}
