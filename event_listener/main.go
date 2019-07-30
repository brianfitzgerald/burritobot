package main

import (
	"burritobot/model"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/nlopes/slack"
)

func main() {
	lambda.Start(handler)
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	api := slack.New(model.SlackKey)

	rtm := api.NewRTM()
	go rtm.ManageConnection()

	os.Setenv("AWS_PROFILE", "personal")

	svc := dynamodb.New(session.New(&aws.Config{
		Region: aws.String("us-east-1"),
	}))

	// initAllUsers(api, svc)

	for msg := range rtm.IncomingEvents {
		fmt.Print("Event Received: ")

		switch ev := msg.Data.(type) {
		case *slack.HelloEvent:

		case *slack.ConnectedEvent:
			fmt.Println("Infos:", ev.Info)
			fmt.Println("Connection counter:", ev.ConnectionCount)

		case *slack.MessageEvent:
			fmt.Printf("Message: %v\n", ev)
			println(ev.Channel)

			foodCountRegex := regexp.MustCompile(":taco:|:burrito:")
			foodCount := foodCountRegex.FindAllStringIndex(ev.Text, -1)

			if strings.Contains(ev.Text, ":burrito:") {
				sendBurritoOrTaco(ev, api, svc, model.Burrito, len(foodCount))
			}

			if strings.Contains(ev.Text, ":taco:") {
				sendBurritoOrTaco(ev, api, svc, model.Taco, len(foodCount))
			}

		case *slack.PresenceChangeEvent:
			fmt.Printf("Presence Change: %v\n", ev)

		case *slack.LatencyReport:
			fmt.Printf("Current latency: %v\n", ev.Value)

		case *slack.RTMError:
			fmt.Printf("Error: %s\n", ev.Error())

		case *slack.InvalidAuthEvent:
			fmt.Printf("Invalid credentials")

		default:

		}
	}

	return events.APIGatewayProxyResponse{}, nil
}

func sendBurritoOrTaco(ev *slack.MessageEvent, api *slack.Client, dynamoSvc *dynamodb.DynamoDB, foodType model.FoodType, count int) error {

	messageText := ev.Text
	mentionedUserID := messageText[strings.Index(ev.Text, "<")+2 : strings.Index(ev.Text, ">")]
	sender, err := api.GetUserInfo(ev.User)
	if err != nil {
		println(err)
	}
	recipient, err := api.GetUserInfo(mentionedUserID)
	if err != nil {
		println(err)
	}

	sendingUser := model.GetUserStats(sender.ID, dynamoSvc)
	receivingUser := model.GetUserStats(recipient.ID, dynamoSvc)

	// make sure can send
	if foodType == model.Burrito && sendingUser.BurritoReserve < 1 {
		_, _, err := api.PostMessage(ev.Channel, slack.MsgOptionText("You do not have enough burritos to do this!", false))
		if err != nil {
			return err
		}
		return nil
	}

	updatedStat := receivingUser.TacosReceived
	if foodType == model.Burrito {
		updatedStat = receivingUser.BurritosReceived
	}
	updatedMessage := fmt.Sprintf("They have now received %d.", updatedStat+count)
	if foodType == model.Taco {
		updatedMessage = ""
	}

	// send burrito / taco to user
	justGot := fmt.Sprintf("a %s", foodType)
	if count > 1 {
		justGot = fmt.Sprintf("%d %ss", count, foodType)
	}
	message := fmt.Sprintf("%s just got %s from %s! %s", recipient.RealName, justGot, sender.RealName, updatedMessage)
	_, _, err = api.PostMessage(ev.Channel, slack.MsgOptionText(message, false))
	if err != nil {
		return err
	}

	if foodType == model.Burrito {
		sendingUser.BurritoReserve -= count
		receivingUser.BurritosReceived += count
		receivingUser.BurritoReserve += count
		message := fmt.Sprintf("%s now has %d burritos left in stock, and %s now has %d.", sender.RealName, sendingUser.BurritoReserve-count, receivingUser.SlackDisplayName, receivingUser.BurritoReserve+count)
		_, _, err = api.PostMessage(ev.Channel, slack.MsgOptionText(message, false))
		if err != nil {
			return err
		}
	} else if foodType == model.Taco {
		receivingUser.TacosReceived += count
	}

	model.UpdateUserStats(sendingUser, dynamoSvc)
	model.UpdateUserStats(receivingUser, dynamoSvc)

	return nil

}
