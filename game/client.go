package game

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	"code.rocket9labs.com/tslocum/bgammon"
	"nhooyr.io/websocket"
)

type Client struct {
	Address    string
	Username   string
	Password   string
	Events     chan interface{}
	Out        chan []byte
	connecting bool
}

func newClient(address string, username string, password string) *Client {
	const bufferSize = 10
	return &Client{
		Address:  address,
		Username: username,
		Password: password,
		Events:   make(chan interface{}, bufferSize),
		Out:      make(chan []byte, bufferSize),
	}
}

func (c *Client) Connect() {
	if c.connecting {
		return // TODO reconnect
	}
	c.connecting = true

	if strings.HasPrefix(c.Address, "ws://") || strings.HasPrefix(c.Address, "wss://") {
		c.connectWebSocket()
		return
	}
	c.connectTCP()
}

func (c *Client) logIn() {
	loginInfo := c.Username
	if c.Username != "" && c.Password != "" {
		loginInfo = fmt.Sprintf("%s %s", c.Username, c.Password)
	}
	c.Out <- []byte(fmt.Sprintf("lj %s\nlist\n", loginInfo))
}

func (c *Client) LoggedIn() bool {
	return c.connecting
}

func (c *Client) connectWebSocket() {
	conn, _, err := websocket.Dial(context.Background(), c.Address, nil)
	if err != nil {
		log.Fatalf("failed to connect: %s", err)
	}

	c.logIn()

	go c.handleWebSocketWrite(conn)
	c.handleWebSocketRead(conn)
}

func (c *Client) handleWebSocketWrite(conn *websocket.Conn) {
	for buf := range c.Out {
		split := bytes.Split(buf, []byte("\n"))
		for i := range split {
			if len(split[i]) == 0 {
				continue
			}

			err := conn.Write(context.Background(), websocket.MessageText, split[i])
			if err != nil {
				panic(err)
			}

			if Debug > 0 {
				l(fmt.Sprintf("-> %s", split[i]))
			}
		}
	}
}

func (c *Client) handleWebSocketRead(conn *websocket.Conn) {
	for {
		msgType, msg, err := conn.Read(context.Background())
		if err != nil {
			l("*** Disconnected.")
			return
		} else if msgType != websocket.MessageText {
			panic("received unexpected message type")
		}

		ev, err := bgammon.DecodeEvent(msg)
		if err != nil {
			log.Printf("message: %s", msg)
			panic(err)
		}
		c.Events <- ev

		if Debug > 0 {
			l(fmt.Sprintf("<- %s", msg))
		}
	}
}

func (c *Client) connectTCP() {
	address := c.Address
	if strings.HasPrefix(c.Address, "tcp://") {
		address = c.Address[6:]
	}

	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Fatalf("failed to connect: %s", err)
	}

	// Read a single line of text and parse remaining output as JSON.
	buf := make([]byte, 1)
	var readBytes int
	for {
		conn.Read(buf)

		if buf[0] == '\n' {
			break
		}

		readBytes++
		if readBytes == 512 {
			panic("failed to read server welcome message")
		}
	}

	c.logIn()

	go c.handleTCPWrite(conn.(*net.TCPConn))
	c.handleTCPRead(conn.(*net.TCPConn))
}

func (c *Client) handleTCPWrite(conn *net.TCPConn) {
	for buf := range c.Out {
		_, err := conn.Write(buf)
		if err != nil {
			panic(err)
		}

		_, err = conn.Write([]byte("\n"))
		if err != nil {
			panic(err)
		}

		if Debug > 0 {
			l(fmt.Sprintf("-> %s", buf))
		}
	}
}

func (c *Client) handleTCPRead(conn *net.TCPConn) {
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		if scanner.Err() != nil {
			l("*** Disconnected.")
			return
		}

		ev, err := bgammon.DecodeEvent(scanner.Bytes())
		if err != nil {
			log.Printf("message: %s", scanner.Bytes())
			panic(err)
		}
		c.Events <- ev

		if Debug > 0 {
			l(fmt.Sprintf("<- %s", scanner.Bytes()))
		}
	}
}
