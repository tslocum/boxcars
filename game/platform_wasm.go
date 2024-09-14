//go:build js && wasm

package game

import (
	"fmt"
	"net/http"
	"strings"
	"syscall/js"

	"github.com/coder/websocket"
)

const (
	AppName            = "boxcars-web"
	ShowServerSettings = false
	ShowQuitDialog     = false
	targetFPS          = 144
)

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

func focused() bool {
	document := js.Global().Get("document")
	hasFocus := document.Call("hasFocus", nil)
	return hasFocus.Truthy()
}

func loadCredentials() (string, string) {
	document := js.Global().Get("document")
	header := http.Header{}
	header.Add("Cookie", document.Get("cookie").String())
	request := http.Request{Header: header}

	var username, password string
	for _, cookie := range request.Cookies() {
		switch cookie.Name {
		case "boxcars_username":
			username = cookie.Value
		case "boxcars_password":
			password = cookie.Value
		}
	}
	return username, password
}

func saveCredentials(username string, password string) {
	document := js.Global().Get("document")
	document.Set("cookie", fmt.Sprintf("boxcars_username=%s; path=/", username))
	document.Set("cookie", fmt.Sprintf("boxcars_password=%s; path=/", password))
}

func GetLocale() (string, error) {
	return js.Global().Get("navigator").Get("language").String(), nil
}

func DefaultFullscreen() bool {
	return false
}

func ReplayDir() string {
	return ""
}
