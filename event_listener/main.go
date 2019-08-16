package main

import (
	"burritobot/model"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"
)

func main() {
	lambda.Start(handler)
	// local()
}

type verificationEvent struct {
	Token     string `json:"token"`
	Challenge string `json:"challenge"`
	Type      string `json:"type"`
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	println(request.Body)

	body := request.Body

	eventsAPIEvent, e := slackevents.ParseEvent(json.RawMessage([]byte(request.Body)), slackevents.OptionNoVerifyToken())
	if e != nil {
		return events.APIGatewayProxyResponse{}, e
	}

	if eventsAPIEvent.Type == slackevents.URLVerification {
		var r *slackevents.ChallengeResponse
		err := json.Unmarshal([]byte(body), &r)
		if err != nil {
			return events.APIGatewayProxyResponse{}, nil
		}
		return events.APIGatewayProxyResponse{
			StatusCode: 200, Headers: map[string]string{
				"Access-Control-Allow-Origin":      "*",    // Required for CORS support to work
				"Access-Control-Allow-Credentials": "true", // Required for cookies, authorization headers with HTTPS
				"Content-Type":                     "text",
			}, Body: r.Challenge,
		}, nil
	}
	if eventsAPIEvent.Type == slackevents.CallbackEvent {
		innerEvent := eventsAPIEvent.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.MessageEvent:
			foodCountRegex := regexp.MustCompile(":taco:|:burrito:")
			foodCount := foodCountRegex.FindAllStringIndex(ev.Text, -1)

			api := slack.New(model.SlackKey)
			sess := session.New()
			svc := dynamodb.New(sess)

			if strings.Contains(ev.Text, ":burrito:") {
				sendBurritoOrTaco(ev, api, svc, model.Burrito, len(foodCount))
				return model.GoodResponse, nil
			}

			if strings.Contains(ev.Text, ":taco:") {
				sendBurritoOrTaco(ev, api, svc, model.Taco, len(foodCount))
				return model.GoodResponse, nil
			}

		}
	}
	return events.APIGatewayProxyResponse{}, nil
}

func local() {
	api := slack.New(model.SlackKey)

	rtm := api.NewRTM()
	go rtm.ManageConnection()

	os.Setenv("AWS_PROFILE", "personal")

	// svc := dynamodb.New(session.New(&aws.Config{
	// 	Region: aws.String("us-east-1"),
	// }))

	// initAllUsers(api, svc)

	for msg := range rtm.IncomingEvents {
		fmt.Print("Event Received: ")

		switch ev := msg.Data.(type) {
		case *slack.HelloEvent:

		case *slack.ConnectedEvent:
			fmt.Println("Infos:", ev.Info)
			fmt.Println("Connection counter:", ev.ConnectionCount)

		case *slack.MessageEvent:
			/*
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
			*/

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
}

func sendBurritoOrTaco(ev *slackevents.MessageEvent, api *slack.Client, dynamoSvc *dynamodb.DynamoDB, foodType model.FoodType, count int) error {

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

	if sendingUser.SlackID == receivingUser.SlackID {
		_, _, err := api.PostMessage(ev.Channel, slack.MsgOptionText("Currency manipulation is a federal crime!", false))
		if err != nil {
			return err
		}
		return nil
	}

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
	updatedMessage := fmt.Sprintf("They have now received %d total.", updatedStat+count)
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
		message := fmt.Sprintf("%s now has %d burritos left, and %s now has %d.", sender.RealName, sendingUser.BurritoReserve-count, receivingUser.SlackDisplayName, receivingUser.BurritoReserve+count)
		_, _, err = api.PostMessage(ev.Channel, slack.MsgOptionText(message, false))
		if err != nil {
			return err
		}
	} else if foodType == model.Taco {
		receivingUser.TacosReceived += count
		message := fmt.Sprintf("%s has received %d tacos total.", recipient.RealName, receivingUser.TacosReceived+count)
		_, _, err = api.PostMessage(ev.Channel, slack.MsgOptionText(message, false))
		if err != nil {
			return err
		}
	}

	model.UpdateUserStats(sendingUser, dynamoSvc)
	model.UpdateUserStats(receivingUser, dynamoSvc)

	return nil

}
