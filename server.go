package gowse

import (
	"fmt"
	"net/http"
	"sync"

	websocket "github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Server ...
type Server struct {
	Topics map[string]*Topic
	mu     sync.Mutex
}

// New ...
func New() *Server {
	return &Server{
		Topics: make(map[string]*Topic),
	}
}

// CreateTopic ...
func (s *Server) CreateTopic(id string) *Topic {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Topics[id] != nil {
		return s.Topics[id]
	}

	t := &Topic{
		clients: make(map[string]*Client),
		quit:    make(chan bool),
	}
	go t.run()

	return t
}

// TopicHandler ...
func TopicHandler(topic *Topic, w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("error upgrading http to websocket connection: ", err)
		return
	}
	// Register client ws connection to t topic
	client := topic.registerClient(ws)
	fmt.Printf("client %s connected\n", client.ID)

	// Spawn client reader goroutine
	// Communication is server -> client, this goroutine will
	// handle client disconnection
	go func() {
		for {
			if _, _, err := ws.NextReader(); err != nil {
				topic.unregisterClient(ws)
				return
			}
		}
	}()

	for {
		select {
		case message := <-client.Broadcast:
			err := websocket.WriteJSON(ws, message)
			if err != nil {
				return
			}
		case <-client.Quit:
			fmt.Printf("client %s disconnected\n", client.ID)
			return
		}
	}
}
