// go-api/main.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/nats-io/nats.go"
)

var (
	appName string
	natsCon *nats.Conn
)

func init() {
	appName = os.Getenv("NAME")
	natsUri := os.Getenv("NATS_URI")
	var err error

	for i := 0; i < 5; i++ {
		natsCon, err = nats.Connect(natsUri)
		if err == nil {
			break
		}

		fmt.Println("Waiting before connecting to NATS at:", natsUri)
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatal("Error establishing connection to NATS:", err)
	}
}

func main() {
	defer func() {
		natsCon.Close()
	}()
	port := os.Getenv("PORT")

	http.HandleFunc("POST /publish", publishHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, appName)
	})

	log.Printf("Starting %s at %s\n", appName, port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

type PublishMessage struct {
	Type    string    `json:"type"`
	Time    time.Time `json:"time"`
	RoomID  string    `json:"room_id"`
	Message string    `json:"message"`
}

func publishHandler(w http.ResponseWriter, r *http.Request) {
	var message PublishMessage
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if message.RoomID == "" || message.Message == "" {
		http.Error(w, "invalid data", http.StatusBadRequest)
		return
	}

	message.Type = "update"
	message.Time = time.Now().UTC()
	log.Println(appName, "POST /publish", message)

	jsonData, err := json.Marshal(message)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := natsCon.Publish("sse_channel", jsonData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Success")
}
