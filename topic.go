package gowse

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	websocket "github.com/gorilla/websocket"
)

const (
	defaultTopicMsgChannelSize         = 256
	defaultClientOperationsChannelSize = 100
	subscriberMaxReceiveMessageSeconds = 2
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Subscriber represents a client subscribed to a topic.
type Subscriber struct {
	ID         string
	connection *websocket.Conn
}

// NewSubscriber creates a subscriber by upgrading the http request to a
// websocket.
func NewSubscriber(w http.ResponseWriter, r *http.Request, t *Topic) (*Subscriber, error) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, fmt.Errorf("error upgrading http to websocket connection: ", err)
	}
	subscriber := &Subscriber{
		ID:         ws.RemoteAddr().String(),
		connection: ws,
	}
	return subscriber, nil
}

// SendMessage sends a message to the subscriber.
func (s *Subscriber) SendMessage(msg interface{}) error {
	return s.connection.WriteJSON(msg)
}

// Close closes the underlaying connection with the subscriber.
func (s *Subscriber) Close() {
	s.connection.Close()
}

// Monitor detects when a subscriber closed a connection.
func (s *Subscriber) Monitor(closed chan<- *Subscriber) {
	// Spawn subscriber reader goroutine. Gowse only allows communication server
	// -> subscriber, thus this goroutine discards all the messages received
	// from a client, but, it disconnects the client if a call to NexReader
	// returns and error as that means the client is disconnected.
	go func() {
		var err error
		for {
			_, _, err = s.connection.NextReader()
			if err != nil {
				break
			}
		}
		closed <- s
	}()
}

// Topic represents and endpoint where multiple clients can subscribe to receive
// the broadcasted messages.
type Topic struct {
	ID               string
	subscriptions    map[string]*Subscriber
	l                Logger
	messages         chan interface{}
	addSubscriber    chan *Subscriber
	removeSubscriber chan *Subscriber
	ctx              context.Context
}

// NewTopic creates a new topic given and ID, a logger and a context.
func NewTopic(ctx context.Context, ID string, l Logger) *Topic {
	t := &Topic{
		ID:            ID,
		subscriptions: make(map[string]*Subscriber),
		l:             l,
		messages:      make(chan interface{}, defaultTopicMsgChannelSize),
		addSubscriber: make(chan *Subscriber, defaultClientOperationsChannelSize),
	}
	return t
}

// Process starts the topic to accept new subscribers and to broadcast messages.
func (t *Topic) Process(wg *sync.WaitGroup) {
	go t.process(wg)
}

func (t *Topic) process(wg *sync.WaitGroup) {
	sendMsgWG := new(sync.WaitGroup)
LOOP:
	for {
		select {
		case m := <-t.messages:
			// Ensure there are no goroutines sending last message.
			sendMsgWG.Wait()
			subscribers := t.subscribers()
			t.sendMsg(subscribers, m, sendMsgWG)
			break
		case s := <-t.addSubscriber:
			// We only add a subscriber if it does not exist.
			if _, ok := t.subscriptions[s.ID]; !ok {
				t.subscriptions[s.ID] = s
				s.Monitor(t.removeSubscriber)
			}
			break
		case s := <-t.removeSubscriber:
			delete(t.subscriptions, s.ID)
			break
		case <-t.ctx.Done():
			break LOOP
		}
	}
	// Wait possible messages to be sent.
	sendMsgWG.Wait()
	// Before quitting we will try to send the remaining messages to the existing clients.
	for m := range t.messages {
		subscribers := t.subscribers()
		t.sendMsg(subscribers, m, sendMsgWG)
		sendMsgWG.Wait()
	}
	// Close all the connections to force quite all the the remaining routines monitoring subscribers.
	for _, s := range t.subscriptions {
		s.Close()
	}
	// Remove all the remaining subscribers.
	for len(t.subscriptions) > 0 {
		s := <-t.removeSubscriber
		delete(t.subscriptions, s.ID)
	}
	// Signal the wg the go routine is done.
	wg.Done()
}

func (t *Topic) sendMsg(subscribers []*Subscriber, msg interface{}, wg *sync.WaitGroup) {
	for _, s := range subscribers {
		s := s
		wg.Add(1)
		go t.sendMsgWithTimeout(s, msg, wg)
	}
}

func (t *Topic) sendMsgWithTimeout(s *Subscriber, msg interface{}, wg *sync.WaitGroup) {
	send := func(done chan<- struct{}) {
		err := s.SendMessage(msg)
		if err != nil {
			t.l.Printf("error sending message to the client %s:%+v", s.ID, err)
			// If we were unable to send a message we disconnect the client, to
			// avoid the possibility of sending out of order messages.
			s.Close()
		}
		done <- struct{}{}
	}
	done := make(chan struct{})
	send(done)
	select {
	case <-done:
		break
	case <-time.After(time.Second * subscriberMaxReceiveMessageSeconds):
		s.Close()
		break
	}
	wg.Done()
}

// TopicHandler is called when a new subscriber connects to the topic.
func (t *Topic) TopicHandler(w http.ResponseWriter, r *http.Request) error {
	s, err := NewSubscriber(w, r, t)
	t.addSubscriber <- s
	return err
}

func (t *Topic) subscribers() []*Subscriber {
	var subscribers []*Subscriber
	for _, s := range t.subscriptions {
		s := s
		subscribers = append(subscribers, s)
	}
	return subscribers
}

// Broadcast sends a messages to the all the clients subscribed to the topic.
// Calling broadcast after canceling the context passed to the server that
// created the topic caused a panic.
func (t *Topic) Broadcast(message interface{}) {
	t.messages <- message
}
