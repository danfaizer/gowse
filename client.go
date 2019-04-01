package gowse

import (
	websocket "github.com/gorilla/websocket"
)

// Client ...
type Client struct {
	ID           string
	Connetion    *websocket.Conn
	Broadcast    chan interface{}
	Registration chan bool
	Quit         chan bool
}
