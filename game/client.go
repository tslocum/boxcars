package game

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"code.rocket9labs.com/tslocum/bgammon"
	"github.com/leonelquinteros/gotext"
	"nhooyr.io/websocket"
)

type Client struct {
	Address       string
	Username      string
	Password      string
	Events        chan interface{}
	Out           chan []byte
	connecting    bool
	loggedIn      bool
	resetPassword bool
}

func newClient(address string, username string, password string, resetPassword bool) *Client {
	const bufferSize = 64
	return &Client{
		Address:       address,
		Username:      username,
		Password:      password,
		Events:        make(chan interface{}, bufferSize),
		Out:           make(chan []byte, bufferSize),
		resetPassword: resetPassword,
	}
}

func (c *Client) Connect() {
	if c.connecting {
		return
	}
	c.connecting = true

	if c.Address == "" {
		c.Address = DefaultServerAddress
	}

	if strings.HasPrefix(c.Address, "ws://") || strings.HasPrefix(c.Address, "wss://") {
		c.connectWebSocket()
		return
	}
	c.connectTCP(nil)
}

func (c *Client) logIn() []byte {
	if c.resetPassword {
		return []byte(fmt.Sprintf("resetpassword %s\n", c.Username))
	} else if game.register {
		c.Username = game.Username
		c.Password = game.Password
		return []byte(fmt.Sprintf("rj %s/%s %s %s %s\nlist\n", AppName, AppLanguage, game.Email, game.Username, game.Password))
	}
	loginInfo := strings.ReplaceAll(c.Username, " ", "_")
	if c.Username != "" && c.Password != "" {
		loginInfo = fmt.Sprintf("%s %s", strings.ReplaceAll(c.Username, " ", "_"), strings.ReplaceAll(c.Password, " ", "_"))
	}
	return []byte(fmt.Sprintf("lj %s/%s %s\nlist\n", AppName, AppLanguage, loginInfo))
}

func (c *Client) LoggedIn() bool {
	return c.connecting
}

func (c *Client) connectWebSocket() {
	connectTime := time.Now()
	reconnect := func() {
		if c.resetPassword || time.Since(connectTime) < 20*time.Second {
			c.connecting = false
			return
		}
		for {
			if !focused() {
				time.Sleep(2 * time.Second)
				continue
			}
			l(fmt.Sprintf("*** %s...", gotext.Get("Reconnecting")))
			time.Sleep(2 * time.Second)
			go c.connectWebSocket()
			break
		}
	}

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	conn, _, err := websocket.Dial(ctx, c.Address, nil)
	if err != nil {
		reconnect()
		return
	}

	for _, msg := range bytes.Split(c.logIn(), []byte("\n")) {
		if len(msg) == 0 {
			continue
		}

		ctx, _ = context.WithTimeout(context.Background(), 10*time.Second)

		err = conn.Write(ctx, websocket.MessageText, msg)
		if err != nil {
			reconnect()
			return
		}
	}

	go c.handleWebSocketWrite(conn)
	c.handleWebSocketRead(conn)

	reconnect()
}

func (c *Client) handleWebSocketWrite(conn *websocket.Conn) {
	var ctx context.Context
	for buf := range c.Out {
		split := bytes.Split(buf, []byte("\n"))
		for i := range split {
			if len(split[i]) == 0 {
				continue
			}

			ctx, _ = context.WithTimeout(context.Background(), 10*time.Second)

			err := conn.Write(ctx, websocket.MessageText, split[i])
			if err != nil {
				conn.Close(websocket.StatusNormalClosure, gotext.Get("Write error"))
				return
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
		if err != nil || msgType != websocket.MessageText {
			conn.Close(websocket.StatusNormalClosure, gotext.Get("Read error"))
			return
		}

		ev, err := bgammon.DecodeEvent(msg)
		if err != nil {
			log.Printf("warning: failed to parse message: %s", msg)
			l("*** " + gotext.Get("Warning: Received unrecognized event from server."))
			l("*** " + gotext.Get("You may need to upgrade your client."))
			continue
		}
		if !c.loggedIn {
			if _, ok := ev.(*bgammon.EventWelcome); ok {
				c.loggedIn = true
			}
		}
		c.Events <- ev

		if Debug > 0 {
			l(fmt.Sprintf("<- %s", msg))
		}
	}
}

func (c *Client) connectTCP(conn net.Conn) {
	address := c.Address
	if strings.HasPrefix(c.Address, "tcp://") {
		address = c.Address[6:]
	}

	connectTime := time.Now()
	reconnect := func() {
		if c.resetPassword || time.Since(connectTime) < 20*time.Second {
			c.connecting = false
			return
		}
		for {
			if !focused() {
				time.Sleep(2 * time.Second)
				continue
			}
			l(fmt.Sprintf("*** %s...", gotext.Get("Reconnecting")))
			time.Sleep(2 * time.Second)
			go c.connectTCP(nil)
			break
		}
	}

	if conn == nil {
		var err error
		conn, err = net.DialTimeout("tcp", address, 10*time.Second)
		if err != nil {
			reconnect()
			return
		}
	}

	// Read a single line of text and parse remaining output as JSON.
	buf := make([]byte, 1)
	var readBytes int
	for {
		_, err := conn.Read(buf)
		if err != nil {
			reconnect()
			return
		}

		if buf[0] == '\n' {
			break
		}

		readBytes++
		if readBytes == 512 {
			reconnect()
			return
		}
	}

	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	_, err := conn.Write(c.logIn())
	if err != nil {
		reconnect()
		return
	}

	go c.handleTCPWrite(conn)
	c.handleTCPRead(conn)

	reconnect()
}

func (c *Client) handleTCPWrite(conn net.Conn) {
	for buf := range c.Out {
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

		_, err := conn.Write(buf)
		if err != nil {
			conn.Close()
			return
		}

		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

		_, err = conn.Write([]byte("\n"))
		if err != nil {
			conn.Close()
			return
		}

		if Debug > 0 {
			l(fmt.Sprintf("-> %s", buf))
		}
	}
}

func (c *Client) handleTCPRead(conn net.Conn) {
	conn.SetReadDeadline(time.Now().Add(40 * time.Second))

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		if scanner.Err() != nil {
			conn.Close()
			return
		}

		ev, err := bgammon.DecodeEvent(scanner.Bytes())
		if err != nil {
			log.Printf("warning: failed to parse message: %s", scanner.Bytes())
			l("*** " + gotext.Get("Warning: Received unrecognized event from server."))
			l("*** " + gotext.Get("You may need to upgrade your client."))
			continue
		}
		if !c.loggedIn {
			if _, ok := ev.(*bgammon.EventWelcome); ok {
				c.loggedIn = true
			}
		}
		c.Events <- ev

		if Debug > 0 {
			l(fmt.Sprintf("<- %s", scanner.Bytes()))
		}

		conn.SetReadDeadline(time.Now().Add(40 * time.Second))
	}
}
