package model

import (
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/nlopes/slack"
)

type UserStats struct {
	SlackID          string `json:"slack_id"`
	SlackDisplayName string `json:"slack_display_name"`
	BurritoReserve   int    `json:"burrito_reserve"`
	TacosReceived    int    `json:"tacos_received"`
	BurritosReceived int    `json:"burritos_received"`
}

var (
	GoodResponse = events.APIGatewayProxyResponse{Body: "", StatusCode: 200, Headers: map[string]string{
		"Access-Control-Allow-Origin":      "*",    // Required for CORS support to work
		"Access-Control-Allow-Credentials": "true", // Required for cookies, authorization headers with HTTPS
	}}
)

func InitAllUsers(api *slack.Client, dynamoSvc *dynamodb.DynamoDB) {
	users, err := api.GetUsers()
	if err != nil {
		return
	}
	for _, user := range users {
		fmt.Printf("ID: %s, Name: %s\n", user.ID, user.Name)
		us := &UserStats{
			SlackDisplayName: user.RealName,
			SlackID:          user.ID,
			BurritoReserve:   20,
			BurritosReceived: 0,
			TacosReceived:    0,
		}
		UpdateUserStats(us, dynamoSvc)
	}

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

type FoodType int

const (
	Burrito FoodType = iota
	Taco
)

func (f FoodType) String() string {
	return [...]string{"Burrito", "Taco"}[f]
}

const (
	statsTableName = "burritobot_user_stats"
)

func GetUserStats(senderID string, dynamoSvc *dynamodb.DynamoDB) *UserStats {
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
	user := &UserStats{}
	err = dynamodbattribute.UnmarshalMap(res.Item, user)
	if err != nil {
		fmt.Println(err)
	}

	return user

}

func UpdateUserStats(user *UserStats, svc *dynamodb.DynamoDB) error {

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
