//go:build js && wasm

package game

import (
	"fmt"
	"net/http"
	"syscall/js"
)

const (
	DefaultServerAddress = "wss://ws.bgammon.org"
	OptimizeDraw         = true
	OptimizeSetRect      = true
	AutoEnableTouchInput = false
	ShowServerSettings   = false
	APPNAME              = "boxcars"
	fieldHeight          = 50
	defaultFontSize      = largeFontSize
)

func DefaultLocale() string {
	return js.Global().Get("navigator").Get("language").String()
}

func focused() bool {
	document := js.Global().Get("document")
	hasFocus := document.Call("hasFocus", nil)
	return hasFocus.Truthy()
}

func loadUsername() string {
	document := js.Global().Get("document")
	header := http.Header{}
	header.Add("Cookie", document.Get("cookie").String())
	request := http.Request{Header: header}
	for _, cookie := range request.Cookies() {
		if cookie.Name == "boxcars_username" {
			return cookie.Value
		}
	}
	return ""
}

func saveUsername(username string) {
	document := js.Global().Get("document")
	document.Set("cookie", fmt.Sprintf("boxcars_username=%s; path=/", username))
}

func saveReplay(id int, content []byte) error {
	l(fmt.Sprintf("*** To download this replay visit https://bgammon.org/match/%d", id))
	return nil
}
