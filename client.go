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
	pulse := time.NewTicker(heartbeatTimeout)
	defer pulse.Stop()

	out := make(chan string)
	defer close(out)
	// Send heartbeat in agreed timeout period
	go func() {
		for _ = range pulse.C {
			out <- "2::"
		}
	}()

	errch := make(chan error)
	// Receive loop
	go func() {
		defer close(rch)
		var msg string
		var SocketIOCodec = websocket.Codec{socketIOMarshall, socketIOUnmarshall}
		for {
			// Remove socketio headers
			if err := SocketIOCodec.Receive(ws, &msg); err != nil {
				errch <- err
				return
			}

			log.Println("< ", msg)
			rch <- msg
		}
	}()

	// Send loop
	go func() {
		for outgoing := range out {
			log.Println("> ", outgoing)
			if err := websocket.Message.Send(ws, outgoing); err != nil {
				errch <- err
				return
			}
		}
	}()

	// Initial connect
	if err := websocket.Message.Send(ws, NewConnect(channel, "").String()); err != nil {
		return err
	}

	for {
		select {
		case outgoing, ok := <-wch:
			// Forward incoming strings to our outgoing channel
			// Closing this channel stops the flow
			if !ok {
				return nil
			}
			out <- outgoing
		case err := <-errch:
			return err
		}
	}
}
