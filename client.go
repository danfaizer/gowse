package gowse

import (
	"fmt"

	"golang.org/x/net/websocket"
)

// Client ...
type Client struct {
	ID        string
	Connetion *websocket.Conn
	Broadcast chan Message
	Quit      chan bool
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
