package gowse

import (
	"sync"

	websocket "github.com/gorilla/websocket"
)

// Topic ...
type Topic struct {
	ID            string
	subscriptions map[string]*Subscriber
	quit          chan bool
	broadcasting  chan bool
	mu            sync.Mutex
}

func (t *Topic) run() {
	defer func() {
		close(t.broadcasting)
		close(t.quit)
	}()
	for {
		select {
		case <-t.quit:
			return
		}
	}
}

func (t *Topic) registerSubscriber(conn *websocket.Conn) *Subscriber {
	t.mu.Lock()
	defer t.mu.Unlock()

	subscriber := &Subscriber{
		id:           conn.RemoteAddr().String(),
		connection:   conn,
		broadcast:    make(chan interface{}),
		quit:         make(chan bool),
		registration: make(chan bool),
	}

	t.subscriptions[conn.RemoteAddr().String()] = subscriber

	return subscriber
}

func (t *Topic) unsubscribe(conn *websocket.Conn) {
	t.mu.Lock()
	defer t.mu.Unlock()

	s, ok := t.subscriptions[conn.RemoteAddr().String()]
	if ok {
		delete(t.subscriptions, s.id)
		close(s.broadcast)
		close(s.quit)
	}
}

// Broadcast ...
func (t *Topic) Broadcast(message interface{}) {
	for _, c := range t.subscriptions {
		c.broadcast <- message
	}
}
