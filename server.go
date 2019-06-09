package gowse

import (
	"sync"
)

// Logger defines the shape of the component needed by the gowse server and
// topics to log info.
type Logger interface {
	Printf(format string, v ...interface{})
}

// Server ...
type Server struct {
	Topics   map[string]*Topic
	mu       sync.Mutex
	l        Logger
	topicsWG sync.WaitGroup
}

// NewServer creates and initializes a Gowse Server. If the logger argument is
// nil the logs written by the server will be discarded.
func NewServer(l Logger) *Server {
	if l == nil {
		l = voidLogger{}
	}
	return &Server{
		Topics: make(map[string]*Topic),
		l:      l,
	}
}

// CreateTopic creates a topic where clients can connect to receive messages.
func (s *Server) CreateTopic(id string) *Topic {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Topics[id] != nil {
		return s.Topics[id]
	}
	t := &Topic{
		subscriptions: make(map[string]*Subscriber),
		messages:      make(chan interface{}, defaultTopicMsgChannelSize),

		wg: &s.topicsWG,
	}
	s.topicsWG.Add(1)
	s.Topics[id] = t
	return t
}

// Stops ...
func (s *Server) Stop() {

	s.topicsWG.Wait()
}

// voidLogger that does not send the logs to any output.
type voidLogger struct {
}

func (l voidLogger) Printf(format string, v ...interface{}) {
	return
}
