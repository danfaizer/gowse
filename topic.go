package gowse

import (
	"fmt"
	"net/http"
	"sync"

	websocket "github.com/gorilla/websocket"
)

// Subscriber ...
type Subscriber struct {
	id         string
	connection *websocket.Conn
	broadcast  chan interface{}
	done       chan interface{}
}

// Monitor detects when a subscriber closed a connection.
func (s Subscriber) Monitor() {
	// Spawn subscriber reader goroutine. Gowse only allows communication server
	// -> subscriber, thus this goroutine discards all the messages received
	// from a client, but, it disconnects the client if the call NexReader
	// returns and error as that means the client is disconnected. handle
	// subscriber disconnection.
	go func() {
		for {
			if _, _, err := s.connection.NextReader(); err != nil {
				t.unsubscribe(ws)
				return
			}
		}
	}()
}

// Topic ...
type Topic struct {
	ID string
	// TODO the use case for this map could potentially match the one where
	// sync.Map could improve performance, evaluate using it.
	subscriptions map[string]*Subscriber
	mu            sync.RWMutex
	l             Logger
	messages      chan interface{}
	wg            *sync.WaitGroup
}

func (t *Topic) registerSubscriber(conn *websocket.Conn) *Subscriber {
	t.mu.Lock()
	defer t.mu.Unlock()

	subscriber := &Subscriber{
		id:         conn.RemoteAddr().String(),
		connection: conn,
		broadcast:  make(chan interface{}),
	}

	t.subscriptions[conn.RemoteAddr().String()] = subscriber

	return subscriber
}

func (t *Topic) unsubscribe(conn *websocket.Conn) {
	t.mu.Lock()
	defer t.mu.Unlock()
	id := conn.RemoteAddr().String()
	delete(t.subscriptions, id)
}

// Broadcast sends a messages to the all the clients subscribed to the topic.
func (t *Topic) Broadcast(message interface{}) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, c := range t.subscriptions {
		c.broadcast <- message
	}
}

// TopicHandler is called when a new subscriber connects to the topic.
func (t *Topic) TopicHandler(w http.ResponseWriter, r *http.Request) error {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return fmt.Errorf("error upgrading http to websocket connection: ", err)
	}
	// Register subscriber ws connection to the topic.
	subscriber := t.registerSubscriber(ws)
	t.l.Printf("subscriber %s connected\n", subscriber.id)
	return
}
