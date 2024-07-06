//go:build js && wasm

package game

import (
	"fmt"
	"net/http"
	"strings"
	"syscall/js"

	"github.com/leonelquinteros/gotext"
)

const (
	AppName            = "boxcars"
	ShowServerSettings = false
	targetFPS          = 144
)

func init() {
	userAgentData := js.Global().Get("navigator").Get("userAgentData")
	if !userAgentData.IsUndefined() {
		mobile := userAgentData.Get("mobile")
		if !mobile.IsUndefined() && mobile.Bool() {
			AutoEnableTouchInput = true
			return
		}
	}
	userAgent := js.Global().Get("navigator").Get("userAgent").String()
	for _, system := range []string{"Android", "iPhone", "iPad", "iPod"} {
		if strings.Contains(userAgent, system) {
			AutoEnableTouchInput = true
			return
		}
	}
}

func DefaultLocale() string {
	return js.Global().Get("navigator").Get("language").String()
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

func saveReplay(id int, content []byte) error {
	if id <= 0 {
		return nil
	}
	l(fmt.Sprintf("*** %s https://bgammon.org/match/%d", gotext.Get("To download this replay visit"), id))
	return nil
}

func mobileDevice() bool {
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

func showKeyboard() {
	if !mobileDevice() {
		return
	}
	game.keyboard.SetVisible(true)
	scheduleFrame()
}

func hideKeyboard() {
	if !mobileDevice() {
		return
	}
	game.keyboard.SetVisible(false)
	scheduleFrame()
}
