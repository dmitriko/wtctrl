package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

/*
var ginLambda *ginadapter.GinLambda

func _init() {
	// stdout and stderr are sent to AWS CloudWatch Logs
	log.Printf("Gin cold start")
	r := gin.Default()
	r.POST("/", func(c *gin.Context) {
		fmt.Printf("%#v\n", c)
		c.String(http.StatusOK, "foo")
	})
	//	r.StaticFile("/", "./index.html")
	r.GET("/*name", func(c *gin.Context) {
		name := c.Param("name")
		fmt.Println("name is", name)
		if name == "favicon.ico" {
			c.File("./favicon.ico")
		} else {
			c.File("./index.html")
		}
	})
	ginLambda = ginadapter.New(r)
}
*/

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// If no name is provided in the HTTP request body, throw an error
	fmt.Printf("%#v", req)
	//return ginLambda.ProxyWithContext(ctx, req)
	res := events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    map[string]string{"Content-Type": "text/plan; charset=utf-8"},
		Body:       "ok",
	}
	fmt.Printf("%#v", res)
	return res, nil
}

func main() {
	lambda.Start(Handler)
}
