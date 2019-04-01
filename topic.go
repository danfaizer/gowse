package gowse

import (
	"sync"
	"time"

	websocket "github.com/gorilla/websocket"
)

// Topic ...
type Topic struct {
	ID           string
	clients      map[string]*Client
	quit         chan bool
	broadcasting chan bool
	mu           sync.Mutex
}

func (t *Topic) run() {
	ticker := time.NewTicker(1 * time.Second)
	defer func() {
		ticker.Stop()
		close(t.broadcasting)
		close(t.quit)
	}()
	for {
		select {
		case <-t.quit:
			return
		case <-ticker.C:
			if len(t.clients) > 0 {
				t.Broadcast(Message{Text: "ping"})
			}
		}
	}
}

func (t *Topic) registerClient(conn *websocket.Conn) *Client {
	t.mu.Lock()
	defer t.mu.Unlock()

	client := &Client{
		ID:           conn.RemoteAddr().String(),
		Connetion:    conn,
		Broadcast:    make(chan interface{}),
		Quit:         make(chan bool),
		Registration: make(chan bool),
	}

	t.clients[conn.RemoteAddr().String()] = client

	return client
}

func (t *Topic) unregisterClient(conn *websocket.Conn) {
	t.mu.Lock()
	defer t.mu.Unlock()

	c, ok := t.clients[conn.RemoteAddr().String()]
	if ok {
		delete(t.clients, c.ID)
		close(c.Broadcast)
		close(c.Quit)
	}
}

// Broadcast ...
func (t *Topic) Broadcast(message interface{}) {
	for _, c := range t.clients {
		c.Broadcast <- message
	}
}
