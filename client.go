package socketio

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func socketIOMarshall(v interface{}) (msg []byte, payloadType byte, err error) {
	switch data := v.(type) {
	case string:
		return []byte(data), websocket.TextFrame, nil
	case []byte:
		return data, websocket.BinaryFrame, nil
	}
	return nil, websocket.UnknownFrame, websocket.ErrNotSupported
}

func socketIOUnmarshall(msg []byte, payloadType byte, v interface{}) (err error) {
	switch data := v.(type) {
	case *string:
		str := strings.TrimLeftFunc(string(msg), func(r rune) bool {
			if r == '{' {
				return false
			}
			return true
		})
		*data = string(str)
		return nil
	}
	return websocket.ErrNotSupported
}

type Options struct {
	HbTimeout time.Duration // Override what server says about heartbeat timeouts
}

type SubscribeError struct {
	msg string
}

func (e SubscribeError) Error() string {
	return e.msg
}

func Subscribe(rch chan<- string, wch <-chan string, url, channel string, o Options) error {
	// Handshake Request
	resp, err := http.Get("http://" + url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	defer close(rch)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	bodyParts := strings.Split(string(body), ":")
	if len(bodyParts) != 4 {
		return SubscribeError{fmt.Sprintf("Received invalid body when handshaking: '%s'", body)}
	}
	// Agreed configs
	sessionId := bodyParts[0]

	var heartbeatTimeout time.Duration
	if o.HbTimeout > time.Second {
		heartbeatTimeout = o.HbTimeout
	} else {
		if h, err := strconv.Atoi(bodyParts[1]); err != nil {
			log.Fatal("Invalid timeout specified by server: ", err)
		} else {
			heartbeatTimeout = time.Duration(h) * time.Second
		}
	}
	//connectionTimeout, _ := strconv.Atoi(bodyParts[2])
	supportedProtocols := strings.Split(string(bodyParts[3]), ",")

	// Fail if websocket is not supported
	for i := 0; i < len(supportedProtocols); i++ {
		if supportedProtocols[i] == "websocket" {
			break
		} else if i == len(supportedProtocols)-1 {
			return SubscribeError{"Websocket is not supported"}
		}
	}

	log.Print(bodyParts)

	// Connect
	wsEndpoint := strings.Join(
		[]string{
			strings.TrimSuffix(url, "/"),
			"websocket",
			sessionId},
		"/")
	ws, err := websocket.Dial("ws://"+wsEndpoint, "", "http://localhost/")
	if err != nil {
		return err
	}
	defer ws.Close()

	// Setup channel for both internal packets and packets coming from wch
	out := make(chan string)
	go func() {
		for s := range wch {
			out <- s
		}
	}()

	// Send initial handshake and heartbeat in agreed timeout perios
	go func() {
		out <- "1::" + channel
		for {
			time.Sleep(heartbeatTimeout)
			out <- "2::"
		}
	}()

	errch := make(chan error)
	// Send/Receive loop
	go func() {
		var msg string
		var SocketIOCodec = websocket.Codec{socketIOMarshall, socketIOUnmarshall}
		for {
			// Remove socketio headers
			if err := SocketIOCodec.Receive(ws, &msg); err != nil {
				errch <- err
				return
			}

			log.Print("< ", msg)
			rch <- msg
		}
	}()

	for {
		select {
		case outgoing, ok := <-out:
			if !ok {
				return nil
			}
			log.Print("> ", outgoing)
			if err := websocket.Message.Send(ws, outgoing); err != nil {
				return err
			}
		case err := <-errch:
			return err
		}
	}
}
