//go:build js && wasm

package game

import (
	"strings"
	"syscall/js"

	"github.com/coder/websocket"
)

const AppName = "boxcars-web"

var (
	mobileDevice           = isMobileDevice()
	smallScreen            = mobileDevice
	enableOnScreenKeyboard = mobileDevice
	enableRightClick       = !mobileDevice
)

var dialOptions = &websocket.DialOptions{}

func isMobileDevice() bool {
	navigator := js.Global().Get("navigator")
	agent := navigator.Get("userAgent")
	if agent.IsUndefined() {
		agent = navigator.Get("vendor")
		if agent.IsUndefined() {
			agent = navigator.Get("opera")
			if agent.IsUndefined() {
				return false
			}
		}
	}
	agentString := agent.String()
	return strings.Contains(strings.ToLower(agentString), "android") || strings.Contains(agentString, "iPhone") || strings.Contains(agentString, "iPad") || strings.Contains(agentString, "iPod")
}

func GetLocale() (string, error) {
	return js.Global().Get("navigator").Get("language").String(), nil
}

func DefaultFullscreen() bool {
	return false
}
