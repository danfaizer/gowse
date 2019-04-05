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
		subscriptions: make(map[string]*Subscriber),
		quit:          make(chan bool),
	}
	// Start goroutine that handles Topic connections and message broadcasting
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
	// Register subscriber ws connection to t topic
	subscriber := topic.registerSubscriber(ws)
	fmt.Printf("subscriber %s connected\n", subscriber.id)

	// Spawn subscriber reader goroutine
	// Communication is server -> subscriber, this goroutine will
	// handle subscriber disconnection.
	go func() {
		for {
			subscriber.registration <- true
			if _, _, err := ws.NextReader(); err != nil {
				topic.unsubscribe(ws)
				return
			}
		}
	}()
	// lock handler initialitaziton until subscriber unregister
	// goroutine has started.
	<-subscriber.registration

	for {
		select {
		case message := <-subscriber.broadcast:
			err := websocket.WriteJSON(ws, message)
			if err != nil {
				fmt.Printf("error broadcasting message: %s\n", err)
			}
		case <-subscriber.quit:
			fmt.Printf("subscriber %s disconnected\n", subscriber.id)
			return
		}
	}
}
