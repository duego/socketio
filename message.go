package socketio

import (
	"fmt"
	"strconv"
)

type IOMessage struct {
	Type     int
	Id       int
	Endpoint *IOEndpoint
	Data     string
}

type IOEndpoint struct {
	Path  string
	Query string
}

func (e IOEndpoint) String() string {
	if e.Query != "" {
		return e.Path + "?" + e.Query
	}
	return e.Path
}

type Disconnect IOMessage
type Connect IOMessage
type Heartbeat IOMessage
type Message IOMessage
type JSON IOMessage
type Event IOMessage
type ACK IOMessage
type Error IOMessage
type Noop IOMessage

func (m IOMessage) String() string {
	var t,
		id,
		endpoint string

	t = strconv.Itoa(m.Type)
	if m.Id != 0 {
		id = strconv.Itoa(m.Id)
	}
	if m.Endpoint != nil {
		endpoint = m.Endpoint.String()
	}
	raw := fmt.Sprintf("%s:%s:%s", t, id, endpoint)

	if m.Data != "" {
		raw += ":" + m.Data
	}
	return raw
}

func NewEndpoint(path, query string) *IOEndpoint {
	return &IOEndpoint{Path: path, Query: query}
}

func NewDisconnect() *IOMessage {
	return &IOMessage{Type: 0}
}

func NewConnect(path, query string) *IOMessage {
	return &IOMessage{Type: 1, Endpoint: NewEndpoint(path, query)}
}

func NewHeartbeat() *IOMessage {
	return &IOMessage{Type: 2}
}

func NewMessage(path, query, data string) *IOMessage {
	return &IOMessage{Type: 3, Endpoint: NewEndpoint(path, query), Data: data}
}

func Parse(rawMessage string) *IOMessage {
	return &IOMessage{}
}
