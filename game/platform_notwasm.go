//go:build !js || !wasm

package game

import (
	"bufio"
	"os"
	"path"
	"path/filepath"

	"github.com/coder/websocket"
)

var dialOptions = &websocket.DialOptions{
	CompressionMode: websocket.CompressionContextTakeover,
}

func focused() bool {
	return true
}

func loadCredentials() (string, string) {
	configDir := userConfigDir()
	if configDir == "" {
		return "", ""
	}
	f, err := os.Open(path.Join(configDir, "config"))
	if err != nil {
		return "", ""
	}

	var i int
	var username, password string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		switch i {
		case 0:
			username = scanner.Text()
		case 1:
			password = scanner.Text()
		}
		i++
	}
	return username, password
}

func saveCredentials(username string, password string) {
	configDir := userConfigDir()
	if configDir == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(configDir), 0700)
	_ = os.WriteFile(path.Join(configDir, "config"), []byte(username+"\n"+password), 0600)
}
