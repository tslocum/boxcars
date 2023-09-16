package game

import (
	"bytes"
	"context"
	"fmt"
	"log"

	"code.rocket9labs.com/tslocum/bgammon"
	"nhooyr.io/websocket"
)

type Client struct {
	Address  string
	Username string
	Password string
	Events   chan interface{}
	Out      chan []byte
	conn     *websocket.Conn
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
	if c.conn != nil {
		return // TODO reconnect
	}

	conn, _, err := websocket.Dial(context.Background(), c.Address, nil)
	if err != nil {
		log.Fatalf("failed to connect: %s", err)
	}
	c.conn = conn

	// Log in.
	loginInfo := c.Username
	if c.Username != "" && c.Password != "" {
		loginInfo = fmt.Sprintf("%s %s", c.Username, c.Password)
	}
	c.Out <- []byte(fmt.Sprintf("lj %s\nlist\n", loginInfo))

	go c.handleWrite()
	c.handleRead()
}
func (c *Client) handleWrite() {
	for buf := range c.Out {
		split := bytes.Split(buf, []byte("\n"))
		for i := range split {
			if len(split[i]) == 0 {
				continue
			}

			err := c.conn.Write(context.Background(), websocket.MessageText, split[i])
			if err != nil {
				panic(err)
			}

			//if debug > 0 {
			l(fmt.Sprintf("-> %s", split[i]))
			//}
		}
	}
}

func (c *Client) handleRead() {
	if c.conn == nil {
		panic("nil con")
	}

	for {
		msgType, msg, err := c.conn.Read(context.Background())
		if err != nil {
			panic(err)
		} else if msgType != websocket.MessageText {
			panic("received unexpected message type")
		}

		ev, err := bgammon.DecodeEvent(msg)
		if err != nil {
			log.Printf("message: %s", msg)
			panic(err)
		}
		c.Events <- ev

		//if debug > 0 {
		l(fmt.Sprintf("<- %s", msg))
		//}
	}
}

func (c *Client) LoggedIn() bool {
	return c.conn != nil
}
