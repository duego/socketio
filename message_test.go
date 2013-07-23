package socketio

import (
	"testing"
)

func TestDisconnect(t *testing.T) {
	m := NewDisconnect()
	if m.String() != "0::" {
		t.Errorf("Disconnect message string")
	}
}

func TestConnect(t *testing.T) {
	m := NewConnect("/mtgox", "Currency=USD")
	if m.String() != "1::/mtgox?Currency=USD" {
		t.Errorf("Invalid connect message string: ", m.String())
	}
}

func TestHeartbeat(t *testing.T) {
	m := NewHeartbeat()
	if m.String() != "2::" {
		t.Errorf("Invalid heartbeat message string")
	}
}

func TestMessage(t *testing.T) {
	m := NewMessage("/mtgox", "Currency=USD", "This is a test message")
	if m.String() != "3::/mtgox?Currency=USD:This is a test message" {
		t.Errorf("Invalid message data string: ", m.String())
	}
}
