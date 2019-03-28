package gowse

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

// Server ...
type Server struct {
	Topics map[string]*Topic
	mu     sync.Mutex
}

// Client ...
type Client struct {
	ID        string
	Connetion *websocket.Conn
	Broadcast chan Message
	Quit      chan bool
}

// Message ...
type Message struct {
	Text string `json:"text"`
}

// Topic ...
type Topic struct {
	ID         string
	clients    map[string]*Client
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	broadcast  chan Message
	mu         sync.Mutex
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
		clients:    make(map[string]*Client),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
	go t.run()

	return t
}

// Handler ...
func Handler(ws *websocket.Conn, t *Topic) {
	t.register <- ws

	for {
		var m Message
		err := websocket.JSON.Receive(ws, &m)
		if err != nil {
			t.unregisterClient(ws)
			return
		}
		t.broadcast <- m
	}
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
				t.Broadcast(Message{Text: "ping"})
			}
		}
	}
}

func (c *Client) listen() {
	for {
		select {
		case message := <-c.Broadcast:
			err := websocket.JSON.Send(c.Connetion, message)
			if err != nil {
				fmt.Println("Error publishing message: ", err)
			}
		case <-c.Quit:
			return
		}
	}
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

// Broadcast ...
func (t *Topic) Broadcast(m Message) {
	t1 := time.Now()
	for _, client := range t.clients {
		client.Broadcast <- m
	}
	t2 := time.Now()
	diff := t2.Sub(t1)
	fmt.Printf("Difference: %s\n", diff)
}
