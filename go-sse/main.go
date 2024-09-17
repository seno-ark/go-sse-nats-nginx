// go-sse/main.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

var (
	appName string
	natsCon *nats.Conn
)

var roomChannels = &RoomChannels{
	mapRoomChannels: make(map[string]map[chan []byte]bool),
}

type RoomChannels struct {
	mu              sync.Mutex
	mapRoomChannels map[string]map[chan []byte]bool
}

func (rc *RoomChannels) Add(key string, ch chan []byte) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if _, ok := rc.mapRoomChannels[key]; !ok {
		rc.mapRoomChannels[key] = make(map[chan []byte]bool)
	}
	rc.mapRoomChannels[key][ch] = true
}
func (rc *RoomChannels) Remove(key string, ch chan []byte) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if chans, ok := rc.mapRoomChannels[key]; ok {
		delete(chans, ch)
		if len(chans) == 0 { // If no channels left, delete the map entry
			delete(rc.mapRoomChannels, key)
		}
	}
}
func (rc *RoomChannels) Broadcast(id string, data []byte) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	for k, v := range rc.mapRoomChannels[id] {
		if v {
			k <- data
		}
	}
}

func init() {
	appName = os.Getenv("NAME")
	natsUri := os.Getenv("NATS_URI")

	var err error
	natsCon, err = nats.Connect(natsUri)
	if err != nil {
		log.Fatal("Error establishing connection to NATS:", err)
	}
}

func main() {
	defer func() {
		natsCon.Close()
	}()
	port := os.Getenv("PORT")

	go natsCon.Subscribe("sse_channel", func(msg *nats.Msg) {
		fmt.Printf("[sse_channel] Received a message: %s\n", string(msg.Data))

		var payload map[string]interface{}
		if err := json.Unmarshal(msg.Data, &payload); err != nil {
			panic(err)
		}

		roomChannels.Broadcast(payload["room_id"].(string), msg.Data)
	})

	http.HandleFunc("GET /events/{id}", sseHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, appName)
	})

	log.Printf("Starting %s at %s\n", appName, port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func sseHandler(w http.ResponseWriter, r *http.Request) {

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	id := r.PathValue("id")

	// Set http headers required for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// You may need this locally for CORS requests
	w.Header().Set("Access-Control-Allow-Origin", "*")

	log.Println(appName, "New Connection", id)
	msgChan := make(chan []byte)
	defer func() {
		close(msgChan)
		log.Println(appName, "Close Connection", id)
	}()

	roomChannels.Add(id, msgChan)

	go func() {
		jsonData, err := json.Marshal(map[string]any{
			"time":   time.Now().UTC(),
			"type":   "info",
			"server": appName,
		})
		if err != nil {
			return
		}
		msgChan <- []byte(jsonData)
	}()

	for {
		select {
		case messageData := <-msgChan:
			fmt.Fprintf(w, "data: %s\n\n", string(messageData))
			flusher.Flush()
		case <-r.Context().Done():
			roomChannels.Remove(id, msgChan)
			return
		}
	}
}
