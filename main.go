package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Controller struct {
	Storage TicketStorerInterface
}

func DatabaseConnection(session *session.Session, timeout time.Duration) TableBasics {
	return TableBasics{
		timeout:        timeout,
		DynamoDbClient: dynamodb.New(session),
	}
}

func (c *Controller) CreateTicket(w http.ResponseWriter, r *http.Request) {
	var newTicket Ticket

	if err := json.NewDecoder(r.Body).Decode(&newTicket); err != nil {

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Error parsing JSON request")
		log.Fatal(err)
	}

	newTicket.UUID = uuid.New().String()

	err := c.Storage.InsertTicket(r.Context(), Ticket{
		UUID:   newTicket.UUID,
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	// _, _ = w.Write([]byte(newTicket.UUID))
	json.NewEncoder(w).Encode(newTicket)

}

func main() {

	// Create a session instance.
	ses, err := CreateNewSession(Config{
		Address: "http://localhost:4566",
		Region:  "ap-southeast-1",
		Profile: "localstack",
		ID:      "AKIAXPTTGDCVODQUK7WF",
		Secret:  "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",
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Instantiate HTTP app
	controllRoute := Controller{
		Storage: DatabaseConnection(ses, time.Second*15),
	}

	router := mux.NewRouter()
	route := router.PathPrefix("/api/v1").Subrouter()

	route.HandleFunc("/ticket/create", controllRoute.CreateTicket)

	log.Fatalln(http.ListenAndServe(":8000", route))

}
