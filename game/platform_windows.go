//go:build windows

package game

import "github.com/coder/websocket"

var dialOptions = &websocket.DialOptions{
	CompressionMode: websocket.CompressionContextTakeover,
}

func DefaultFullscreen() bool {
	return false
}
