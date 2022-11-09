package main

var TableName = "tickets"

type Ticket struct {
	UUID   string `json:"uuid"`
	Owner  string `json:"owner"`
	Status string `json:"status"`
}
