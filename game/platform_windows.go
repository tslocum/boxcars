//go:build windows

package game

import "github.com/coder/websocket"

const AppName = "boxcars-windows"

var (
	smallScreen            = false
	mobileDevice           = false
	enableOnScreenKeyboard = false
	enableRightClick       = true
)

var dialOptions = &websocket.DialOptions{
	CompressionMode: websocket.CompressionContextTakeover,
}

func DefaultFullscreen() bool {
	return false
}
