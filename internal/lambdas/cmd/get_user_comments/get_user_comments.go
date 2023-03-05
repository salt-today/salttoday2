package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/salt-today/salttoday2/internal/lambdas/api"
)

func main() {
	lambda.Start(api.GetUserCommentsLambdaHandler)
}
