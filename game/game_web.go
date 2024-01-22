//go:build js && wasm

package game

import (
	"fmt"
	"net/http"
	"syscall/js"

	"github.com/leonelquinteros/gotext"
)

const (
	OptimizeDraw         = true
	OptimizeSetRect      = true
	AutoEnableTouchInput = false
	ShowServerSettings   = false
	APPNAME              = "boxcars"
	fieldHeight          = 50
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
	if id <= 0 {
		return nil
	}
	l(fmt.Sprintf("*** %s https://bgammon.org/match/%d", gotext.Get("To download this replay visit"), id))
	return nil
}
