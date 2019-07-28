package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/nlopes/slack"
)

type userStats struct {
	SlackID          string `json:"slack_id"`
	SlackDisplayName string `json:"slack_display_name"`
	BurritoReserve   int    `json:"burrito_reserve"`
	TacosReceived    int    `json:"tacos_received"`
	BurritosReceived int    `json:"burritos_received"`
}

var (
	channels = []string{"CB59S6C1H", "CLVV2NQJK"}
)

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

type foodType int

const (
	burrito foodType = iota
	taco
)

func (f foodType) String() string {
	return [...]string{"Burrito", "Taco"}[f]
}

const (
	statsTableName = "burritobot_user_stats"
)

func main() {
	api := slack.New(SlackKey)

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

			if strings.Contains(ev.Text, ":burrito:") && contains(channels, ev.Channel) {
				sendBurritoOrTaco(ev, api, svc, burrito)
			}

			if strings.Contains(ev.Text, ":taco:") && contains(channels, ev.Channel) {
				sendBurritoOrTaco(ev, api, svc, taco)
			}

		case *slack.PresenceChangeEvent:
			fmt.Printf("Presence Change: %v\n", ev)

		case *slack.LatencyReport:
			fmt.Printf("Current latency: %v\n", ev.Value)

		case *slack.RTMError:
			fmt.Printf("Error: %s\n", ev.Error())

		case *slack.InvalidAuthEvent:
			fmt.Printf("Invalid credentials")
			return

		default:

		}
	}
}

func initAllUsers(api *slack.Client, dynamoSvc *dynamodb.DynamoDB) {
	users, err := api.GetUsers()
	if err != nil {
		return
	}
	for _, user := range users {
		fmt.Printf("ID: %s, Name: %s\n", user.ID, user.Name)
		us := &userStats{
			SlackDisplayName: user.RealName,
			SlackID:          user.ID,
			BurritoReserve:   20,
			BurritosReceived: 0,
			TacosReceived:    0,
		}
		updateUserStats(us, dynamoSvc)
	}

}

func getUserStats(senderID string, dynamoSvc *dynamodb.DynamoDB) *userStats {
	senderDynamoKey := map[string]*dynamodb.AttributeValue{
		"slack_id": {
			S: aws.String(senderID),
		},
	}

	// get sender
	getUserInput := &dynamodb.GetItemInput{
		Key:       senderDynamoKey,
		TableName: aws.String(statsTableName),
	}
	res, err := dynamoSvc.GetItem(getUserInput)
	if err != nil {
		fmt.Println(err)
	}
	user := &userStats{}
	err = dynamodbattribute.UnmarshalMap(res.Item, user)
	if err != nil {
		fmt.Println(err)
	}

	return user

}

func updateUserStats(user *userStats, svc *dynamodb.DynamoDB) error {

	item, err := dynamodbattribute.MarshalMap(user)
	if err != nil {
		return err
	}

	putUserInput := &dynamodb.PutItemInput{
		Item:      item,
		TableName: aws.String(statsTableName),
	}
	_, err = svc.PutItem(putUserInput)
	if err != nil {
		return err
	}

	return nil

}

func sendBurritoOrTaco(ev *slack.MessageEvent, api *slack.Client, dynamoSvc *dynamodb.DynamoDB, foodType foodType) error {

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

	sendingUser := getUserStats(sender.ID, dynamoSvc)
	receivingUser := getUserStats(recipient.ID, dynamoSvc)

	// make sure can send
	if foodType == burrito && sendingUser.BurritoReserve < 1 {
		_, _, err := api.PostMessage(ev.Channel, slack.MsgOptionText("You do not have enough burritos to do this!", false))
		if err != nil {
			return err
		}
		return nil
	}

	updatedStat := receivingUser.TacosReceived
	if foodType == burrito {
		updatedStat = receivingUser.BurritosReceived
	}
	updatedMessage := fmt.Sprintf("They have now received %d.", updatedStat+1)
	if foodType == taco {
		updatedMessage = ""
	}

	// send burrito / taco to user
	message := fmt.Sprintf("User %s just got a %s from %s! %s", recipient.RealName, foodType.String(), sender.RealName, updatedMessage)
	_, _, err = api.PostMessage(ev.Channel, slack.MsgOptionText(message, false))
	if err != nil {
		return err
	}

	if foodType == burrito {
		sendingUser.BurritoReserve--
		receivingUser.BurritosReceived++
		message := fmt.Sprintf("%s now has %d burritos left in stock.", sender.RealName, sendingUser.BurritoReserve-1)
		_, _, err = api.PostMessage(ev.Channel, slack.MsgOptionText(message, false))
		if err != nil {
			return err
		}
	} else if foodType == taco {
		receivingUser.TacosReceived++
	}

	updateUserStats(sendingUser, dynamoSvc)
	updateUserStats(receivingUser, dynamoSvc)

	return nil

}
