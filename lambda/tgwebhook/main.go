package main

import (
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

func putInSQS(text string) error {

	queue_url := os.Getenv("QUEUE_URL")
	if queue_url == "" {
		return errors.New("QUEUE_URL env var is not set")
	}

	sess := session.Must(session.NewSession())
	svc := sqs.New(sess)
	send_input := &sqs.SendMessageInput{
		MessageGroupId: aws.String("tgwebhook"),
		MessageBody:    aws.String(text),
		QueueUrl:       aws.String(queue_url),
	}
	_, err := svc.SendMessage(send_input)
	if err != nil {
		return err
	}
	return nil
}

func handleRequest(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	log.Println(req.Body)

	err := putInSQS(req.Body)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
		}, err
	}

	res := events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    map[string]string{"Content-Type": "text/plan; charset=utf-8"},
		Body:       "ok",
	}
	return res, nil
}

func main() {
	lambda.Start(handleRequest)
}
