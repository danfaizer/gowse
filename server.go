package gowse

import (
	"context"
	"sync"
)

// Logger defines the shape of the component needed by the gowse server and
// topics to log info and errors.
type Logger interface {
	Infof(format string, v ...interface{})
	Errorf(format string, v ...interface{})
}

// Server allows to create topics that clients can subscribe to.
type Server struct {
	Topics    map[string]*Topic
	ctx       context.Context
	cancelCtx context.CancelFunc
	mu        sync.Mutex
	l         Logger
	topicsWG  sync.WaitGroup
}

// NewServer creates and initializes a gowse Server. If the logger argument is
// nil the logs written by the server will be discarded.
func NewServer(l Logger) *Server {
	if l == nil {
		l = voidLogger{}
	}
	ctx, cancelCtx := context.WithCancel(context.Background())
	return &Server{
		Topics:    make(map[string]*Topic),
		l:         l,
		ctx:       ctx,
		cancelCtx: cancelCtx,
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

// Stop signals all the topics created by the server to finish and blocks the
// calling goroutine until all the Topics have finished. The caller must ensure
// all the calls to topic.TopicHandler had completed and no futher calls to that
// method will be made. That usually means the http server receiving the client
// subscritions has already been stopped.
func (s *Server) Stop() {
	s.cancelCtx()
	s.topicsWG.Wait()
}

// voidLogger that does not send the logs to any output.
type voidLogger struct {
}

func (l voidLogger) Infof(format string, v ...interface{}) {}

func (l voidLogger) Errorf(format string, v ...interface{}) {}
