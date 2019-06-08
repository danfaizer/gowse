package gowse

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	websocket "github.com/gorilla/websocket"
)

const (
	defaultTopicMsgChannelSize  = 256
	clientOperationsChannelSize = 256
)

// Subscriber ...
type Subscriber struct {
	id         string
	connection *websocket.Conn
	wg         *sync.WaitGroup
	done       chan struct{}
	cancel     context.CancelFunc
}

// SendMessage sends a message to the subscriber.
func (s *Subscriber) SendMessage(msg interface{}) error {
	return s.connection.WriteJSON(msg)
}

// Monitor detects when a subscriber closed a connection.
func (s *Subscriber) Monitor() {
	// Spawn subscriber reader goroutine. Gowse only allows communication server
	// -> subscriber, thus this goroutine discards all the messages received
	// from a client, but, it disconnects the client if the call NexReader
	// returns and error as that means the client is disconnected.
	go func() {
		read := make(chan error)
		reader := func() {
			_, _, err := s.connection.NextReader()
			read <- err
		}
		go reader()
		var err error
	LOOP:
		for {
			select {
			case err = <-read:
				if err != nil {
					break LOOP
				}
				go reader()
				break
			case _ = <-s.done:
				// connection.close will force the next call to the reader function
				// to return an error so the loop will break ensuring no goroutines are still running.
				s.connection.Close()
			}
		}
		s.wg.Done()
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
	ctx           context.Context
	wg            *sync.WaitGroup
}

func (t *Topic) run() {
	go t.process()
	go t.monitor()
}

func (t *Topic) process() {
	for msg := range t.messages {

	}
}

func (t *Topic) monitor() {
 for {
	 select {
		 <- t.
	 }
 }
}

func (t *Topic) unsubscribe(conn *websocket.Conn) {
	t.mu.Lock()
	defer t.mu.Unlock()
	id := conn.RemoteAddr().String()
	delete(t.subscriptions, id)
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

// sendMessage sends a message to all the current subscribed clients.
func (t *Topic) sendMessage(m interface{}) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, c := range t.subscriptions {

	}
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
