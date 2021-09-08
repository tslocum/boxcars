// +build js,wasm

package main

import (
	"code.rocketnine.space/tslocum/fibs"
)

func init() {
	fibs.DefaultProxyAddress = "wss://fibsproxy.rocketnine.space"

	AutoWatch = true
}
