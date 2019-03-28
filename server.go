package gowse

import (
	"sync"

	"golang.org/x/net/websocket"
)

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
		clients:    make(map[string]*Client),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
	go t.run()

	return t
}

// TopicHandler ...
func TopicHandler(ws *websocket.Conn, t *Topic) {
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
