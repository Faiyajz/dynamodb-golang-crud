package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

var (
	ErrInternal = errors.New("internal")
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

type Config struct {
	Address string
	Region  string
	Profile string
	ID      string
	Secret  string
}

func New(config Config) (*session.Session, error) {
	return session.NewSessionWithOptions(
		session.Options{
			Config: aws.Config{
				Credentials:      credentials.NewStaticCredentials(config.ID, config.Secret, ""),
				Region:           aws.String(config.Region),
				Endpoint:         aws.String(config.Address),
				S3ForcePathStyle: aws.Bool(true),
			},
			Profile: config.Profile,
		},
	)
}

type Ticket struct {
	UUID   string `json:"uuid"`
	Owner  string `json:"owner"`
	Status string `json:"status"`
}

type TicketStorer interface {
	Insert(ctx context.Context, ticket Ticket) error
}

type Controller struct {
	Storage TicketStorer
}

type TicketStorage struct {
	timeout time.Duration
	client  *dynamodb.DynamoDB
}

var _ TicketStorer = TicketStorage{}

func NewTicketStorage(session *session.Session, timeout time.Duration) TicketStorage {
	return TicketStorage{
		timeout: timeout,
		client:  dynamodb.New(session),
	}
}

func (u TicketStorage) Insert(ctx context.Context, ticket Ticket) error {
	ctx, cancel := context.WithTimeout(ctx, u.timeout)
	defer cancel()

	item, err := dynamodbattribute.MarshalMap(ticket)
	if err != nil {
		log.Println(err)
		return ErrInternal
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String("tickets"),
		Item:      item,
		ExpressionAttributeNames: map[string]*string{
			"#uuid": aws.String("uuid"),
		},
		ConditionExpression: aws.String("attribute_not_exists(#uuid)"),
	}

	if _, err := u.client.PutItemWithContext(ctx, input); err != nil {
		log.Println(err)

		if _, ok := err.(*dynamodb.ConditionalCheckFailedException); ok {
			return ErrConflict
		}

		return ErrInternal
	}

	return nil
}

func (c *Controller) Create(w http.ResponseWriter, r *http.Request) {
	var newTicket Ticket //create an instance of Ticket struct

	//read data from our requests by passing the body of our http request e.g. json.NewDecoder(r.Body)
	//Call .Decode() passing it a pointer to our newTicket Struct which is an instance of Ticket Struct
	//which allows it to match the json to the appropriate properties of the struct
	if err := json.NewDecoder(r.Body).Decode(&newTicket); err != nil {

		//send an internal server error
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Error parsing JSON request")

		log.Fatal(err)
	}

	// tickets = append(tickets, newTicket) //add new new ticket in the tickets slice

	id := uuid.New().String()

	err := c.Storage.Insert(r.Context(), Ticket{
		UUID:   id,
		Owner:  newTicket.Owner,
		Status: newTicket.Status,
	})
	if err != nil {
		switch err {
		case ErrConflict:
			w.WriteHeader(http.StatusConflict)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(id))

	// w.Header().Set("Content-Type", "application/json") //Set the headers and the response
	// w.WriteHeader(http.StatusCreated)

	// json.NewEncoder(w).Encode(newTicket) //ticket created

}

func main() {

	// Create a session instance.
	ses, err := New(Config{
		Address: "http://localhost:4566",
		Region:  "ap-southeast-1",
		Profile: "localstack",
		ID:      "****",
		Secret:  "****",
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Instantiate HTTP app
	controll := Controller{
		Storage: NewTicketStorage(ses, time.Second*15),
	}

	router := mux.NewRouter()
	route := router.PathPrefix("/api/v1").Subrouter()
	route.HandleFunc("/ticket/create", controll.Create)

	log.Fatalln(http.ListenAndServe(":8000", route))

}
