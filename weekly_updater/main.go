package main

import (
	"burritobot/model"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/nlopes/slack"
)

func main() {
	// lambda.Start(handler)
	handler(events.APIGatewayProxyRequest{})
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	api := slack.New(model.SlackKey)
	os.Setenv("AWS_PROFILE", "personal")

	svc := dynamodb.New(session.New(&aws.Config{
		Region: aws.String("us-east-1"),
	}))

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
	message += fmt.Sprintf("Forbes 500 Richest Burrito Man: %s\n", topBurritoNetWorth.SlackDisplayName)
	message += fmt.Sprintf("Taco King (most tacos received): %s\n", topTacosReceived.SlackDisplayName)
	message += fmt.Sprintf("Burrito Lord (most burritos received): %s\n", topBurritosReceived.SlackDisplayName)

	_, _, err := api.PostMessage(model.WeeklyUpdateChannel, slack.MsgOptionText(message, false))
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	model.UpdateAllUsers(svc, users)

	dividendMessage := fmt.Sprintf("Everyone has been given their weekly Burrito Dividend of %d burritos.", model.BurritoDividendAmount)

	_, _, err = api.PostMessage(model.WeeklyUpdateChannel, slack.MsgOptionText(dividendMessage, false))
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{}, nil
}
