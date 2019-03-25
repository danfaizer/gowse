package gowse

import (
	"fmt"
	"time"

	"golang.org/x/net/websocket"
)

// Message ...
type Message struct {
	Text string `json:"text"`
}

// Topic ...
type Topic struct {
	clients          map[string]*websocket.Conn
	addClientChan    chan *websocket.Conn
	removeClientChan chan *websocket.Conn
	broadcastChan    chan Message
}

// NewServer ...
func NewServer(ws *websocket.Conn, t *Topic) {
	go t.run()

	t.addClientChan <- ws

	for {
		var m Message

		err := websocket.JSON.Receive(ws, &m)
		if err != nil {
			t.broadcastChan <- Message{err.Error()}
			t.removeClient(ws)
			return
		}

	}
}

// CreateTopic returns a new Topic object
func CreateTopic() *Topic {
	return &Topic{
		clients:          make(map[string]*websocket.Conn),
		addClientChan:    make(chan *websocket.Conn),
		removeClientChan: make(chan *websocket.Conn),
		broadcastChan:    make(chan Message),
	}
}

// run receives from the Topic channels and calls the appropriate Topic method
func (t *Topic) run() {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case conn := <-t.addClientChan:
			t.addClient(conn)
		case conn := <-t.removeClientChan:
			t.removeClient(conn)
		case m := <-t.broadcastChan:
			t.Publish(m)
		case <-ticker.C:
			if len(t.clients) > 0 {
				t.Publish(Message{Text: "ping"})
			}
		}
	}
}

// removeClient removes a conn from the pool
func (t *Topic) removeClient(conn *websocket.Conn) {
	_, ok := t.clients[conn.Request().RemoteAddr]
	if ok {
		delete(t.clients, conn.Request().RemoteAddr)
	}
}

// addClient adds a conn to the pool
func (t *Topic) addClient(conn *websocket.Conn) {
	t.clients[conn.Request().RemoteAddr] = conn
}

// Publish ...
func (t *Topic) Publish(m Message) {
	for _, conn := range t.clients {
		go func(m Message, conn *websocket.Conn) {
			err := websocket.JSON.Send(conn, m)
			if err != nil {
				fmt.Println("Error publishing message: ", err)
				return
			}
		}(m, conn)
	}
}
