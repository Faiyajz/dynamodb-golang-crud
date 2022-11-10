package main

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type TicketStorerInterface interface {
	InsertTicket(ctx context.Context, ticket Ticket) error
}

func (basics TableBasics) InsertTicket(ctx context.Context, ticket Ticket) error {
	ctx, cancel := context.WithTimeout(ctx, basics.timeout)
	defer cancel()

	item, err := dynamodbattribute.MarshalMap(ticket)
	if err != nil {
		panic(err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(TableName),
		Item:      item,
		ExpressionAttributeNames: map[string]*string{
			"#uuid": aws.String("uuid"),
		},
		ConditionExpression: aws.String("attribute_not_exists(#uuid)"),
	}

	if _, err := basics.DynamoDbClient.PutItemWithContext(ctx, input); err != nil {
		log.Printf("Couldn't add item to table. Here's why: %v\n", err)

		if _, errorFound := err.(*dynamodb.ConditionalCheckFailedException); errorFound {
			return ErrConflict
		}

		return ErrInternal
	}

	return nil
}
