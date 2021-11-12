//go:build js && wasm
// +build js,wasm

package main

import (
	"net/http"
	"syscall/js"

	"code.rocketnine.space/tslocum/boxcars/game"
)

func parseFlags(g *game.Game) {
	v := js.Global().Get("document").Get("cookie").String()

	header := http.Header{}
	header.Add("Cookie", v)
	request := http.Request{Header: header}

	cookieV, err := request.Cookie("BoxcarsUsername")
	if err == nil {
		g.Username = cookieV.Value
	}
	cookieV, err = request.Cookie("BoxcarsPassword")
	if err == nil {
		g.Password = cookieV.Value
	}

	//alert := js.Global().Get("alert")
	//alert.Invoke(fmt.Sprintf("%+v", request.Cookies()))
}
