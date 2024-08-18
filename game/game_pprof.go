//go:build profile

package game

import (
	"log"
	"net/http"
	_ "net/http/pprof"
)

func serveProfile() {
	log.Fatal(http.ListenAndServe("localhost:8880", nil))
}

func init() {
	go serveProfile()
}
