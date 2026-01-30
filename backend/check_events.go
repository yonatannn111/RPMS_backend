package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type Event struct {
	ID    string    `json:"id"`
	Title string    `json:"title"`
	Date  time.Time `json:"date"`
}

func main() {
	resp, err := http.Get("http://localhost:8080/api/v1/events")
	if err != nil {
		fmt.Println("Error fetching events:", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading body:", err)
		return
	}

	var events []Event
	err = json.Unmarshal(body, &events)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		fmt.Println("Body:", string(body))
		return
	}

	fmt.Println("Current Time:", time.Now())
	fmt.Println("Events found:", len(events))
	for _, e := range events {
		isUpcoming := e.Date.After(time.Now())
		fmt.Printf("Event: %s, Date: %s, IsUpcoming: %v\n", e.Title, e.Date, isUpcoming)
	}
}
