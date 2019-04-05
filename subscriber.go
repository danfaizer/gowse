package gowse

import (
	websocket "github.com/gorilla/websocket"
)

// Subscriber ...
type Subscriber struct {
	id           string
	connection   *websocket.Conn
	broadcast    chan interface{}
	registration chan bool
	quit         chan bool
}
