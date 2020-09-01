package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-gonic/gin"
)

var ginLambda *ginadapter.GinLambda

func init() {
	log.Printf("Gin cold start")
	r := gin.Default()
	r.POST("/", func(c *gin.Context) {
		fmt.Printf("%#v\n", c)
		c.String(http.StatusOK, "foo")
	})

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

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	fmt.Printf("%#v", req)

	stage := req.RequestContext.Stage

	if ginLambda == nil {

		log.Printf("Gin cold start")
		r := gin.Default()
		r.GET(fmt.Sprintf("/%s/*filepath", stage), func(c *gin.Context) {
			filepath := c.Param("filepath")
			if _, err := os.Stat(filepath); os.IsNotExist(err) {
				c.File("./index.html")
			} else {
				c.File(filepath)
			}
		})
		ginLambda = ginadapter.New(r)
	}

	//	return ginLambda.ProxyWithContext(ctx, req)

	res := events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    map[string]string{"Content-Type": "text/plan; charset=utf-8"},
		Body:       "ok",
	}
	return res, nil

}

func main() {
	lambda.Start(Handler)
}
