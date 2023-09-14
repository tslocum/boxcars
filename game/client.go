package game

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"

	"github.com/gobwas/ws/wsutil"

	"code.rocket9labs.com/tslocum/bgammon"

	"github.com/gobwas/ws"
)

type Client struct {
	Address  string
	Username string
	Password string
	Events   chan interface{}
	Out      chan []byte
	conn     *net.TCPConn
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

	conn, br, _, err := ws.Dial(context.Background(), c.Address)
	if err != nil {
		panic(err)
	}
	c.conn = conn.(*net.TCPConn)

	if br != nil {
		ws.PutReader(br)
	}

	// Log in.
	loginInfo := c.Username
	if c.Username != "" && c.Password != "" {
		loginInfo = fmt.Sprintf("%s %s", c.Username, c.Password)
	}
	c.Out <- []byte(fmt.Sprintf("lj %s\nlist\ncreate public\n", loginInfo))

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

			err := wsutil.WriteClientMessage(c.conn, ws.OpText, split[i])
			if err != nil {
				panic(err)
			}

			//if debug > 0 {
			log.Println(fmt.Sprintf("-> %s", split[i]))
			//}
		}
	}
}

func (c *Client) handleRead() {
	if c.conn == nil {
		panic("nil con")
	}

	var messages []wsutil.Message
	var err error

	var i int
	for {
		log.Printf("READ FRAME %d", i)
		i++

		messages, err = wsutil.ReadServerMessage(c.conn, messages[:0])
		if err != nil {
			panic(err)
		}

		for _, msg := range messages {
			ev, err := bgammon.DecodeEvent(msg.Payload)
			if err != nil {
				log.Printf("message: %s", msg.Payload)
				panic(err)
			}
			c.Events <- ev

			//if debug > 0 {
			log.Println(fmt.Sprintf("<- %s", msg.Payload))
			//}
		}
	}
}

func (c *Client) LoggedIn() bool {
	return c.conn != nil
}
