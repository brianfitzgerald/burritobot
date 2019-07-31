package main

import (
	"burritobot/model"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/nlopes/slack"
)

func main() {
	lambda.Start(handler)
	// handler(events.APIGatewayProxyRequest{})
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	sess := session.New()
	svc := dynamodb.New(sess)
	api := slack.New(model.SlackKey)

	users := model.GetAllUsers(svc)

	topBurritosReceived := users[0]
	topTacosReceived := users[0]
	topBurritoNetWorth := users[0]

	for _, user := range users {
		if user.BurritosReceived > topBurritosReceived.BurritosReceived {
			topBurritosReceived = user
		}
		if user.BurritoReserve > topBurritoNetWorth.BurritoReserve {
			topBurritoNetWorth = user
		}
		if user.TacosReceived > topTacosReceived.TacosReceived {
			topTacosReceived = user
		}
	}

	message := fmt.Sprintf("*Your Weekly Burrito Market News*\n")
	message += fmt.Sprintf("Forbes 500 Richest Burrito Man: %s", topBurritoNetWorth.SlackDisplayName)
	message += fmt.Sprintf("Burrito King: %s", topTacosReceived.SlackDisplayName)
	message += fmt.Sprintf("Taco Lord: %s", topTacosReceived.SlackDisplayName)

	_, _, err := api.PostMessage("bot-testing", slack.MsgOptionText(message, false))
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{}, nil
}
