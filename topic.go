package gowse

import (
	"errors"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

// Topic ...
type Topic struct {
	ID         string
	clients    map[string]*Client
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	broadcast  chan Message
	mu         sync.Mutex
}

func (t *Topic) run() {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case conn := <-t.register:
			t.registerClient(conn)
		case conn := <-t.unregister:
			t.unregisterClient(conn)
		case <-ticker.C:
			if len(t.clients) > 0 {

				t.Broadcast(Message{Data: map[string]string{"action": "ping"}})
			}
		}
	}
}

func (t *Topic) registerClient(conn *websocket.Conn) {
	t.mu.Lock()
	defer t.mu.Unlock()

	client := &Client{
		ID:        conn.Request().RemoteAddr,
		Connetion: conn,
		Broadcast: make(chan Message),
		Quit:      make(chan bool),
	}

	t.clients[conn.Request().RemoteAddr] = client

	go client.listen()
}

func (t *Topic) unregisterClient(conn *websocket.Conn) {
	t.mu.Lock()
	defer t.mu.Unlock()

	c, ok := t.clients[conn.Request().RemoteAddr]
	if ok {
		c.Quit <- true
		delete(t.clients, conn.Request().RemoteAddr)
	}
}

// Broadcast ...
func (t *Topic) Broadcast(m Message) error {
	_, err := m.envelop()
	if err != nil {
		return errors.New("unprocessable message")
	}
	for _, client := range t.clients {
		client.Broadcast <- m
	}
	return nil
}
