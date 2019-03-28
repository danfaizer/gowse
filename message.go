package gowse

import "encoding/json"

// Message ...
type Message struct {
	Data interface{}
}

func (m *Message) envelop() (string, error) {
	js, err := json.Marshal(m)
	return string(js), err
}
