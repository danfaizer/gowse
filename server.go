package gowse

import (
	"context"
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
	ctx      context.Context
	mu       sync.Mutex
	l        Logger
	topicsWG sync.WaitGroup
}

// NewServer creates and initializes a Gowse Server. If the logger argument is
// nil the logs written by the server will be discarded.
func NewServer(ctx context.Context, l Logger) *Server {
	if l == nil {
		l = voidLogger{}
	}
	return &Server{
		Topics: make(map[string]*Topic),
		l:      l,
	}
}

// CreateTopic creates a topic where clients can connect to receive messages.
func (s *Server) CreateTopic(ID string) *Topic {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Topics[ID] != nil {
		return s.Topics[ID]
	}
	ctx, _ := context.WithCancel(s.ctx)
	t := NewTopic(ctx, ID, s.l)
	s.Topics[ID] = t
	s.topicsWG.Add(1)
	t.Process(&s.topicsWG)
	return t
}

// WaitFinished blocks the calling go routine until all the topics are finished Before
// calling Finish the server must be signaled to finish by caneling the the
// context used to create it.
func (s *Server) WaitFinished() {
	s.topicsWG.Wait()
}

// voidLogger that does not send the logs to any output.
type voidLogger struct {
}

func (l voidLogger) Printf(format string, v ...interface{}) {
	return
}
