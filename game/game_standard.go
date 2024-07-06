//go:build (!js || !wasm) && !android

package game

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"github.com/leonelquinteros/gotext"
)

const (
	AppName            = "boxcars"
	ShowServerSettings = false
	targetFPS          = 144
)

func DefaultLocale() string {
	return ""
}

func focused() bool {
	return true
}

func userConfigDir() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return path.Join(configDir, "boxcars")
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

func saveReplay(id int, content []byte) error {
	replayDir := ReplayDir()
	if replayDir == "" {
		return fmt.Errorf("failed to determine default download location")
	}

	var (
		timestamp int64
		player1   string
		player2   string
		err       error
	)
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		if bytes.HasPrefix(scanner.Bytes(), []byte("i ")) {
			split := bytes.Split(scanner.Bytes(), []byte(" "))
			if len(split) < 4 {
				return fmt.Errorf("failed to parse replay")
			}

			timestamp, err = strconv.ParseInt(string(split[1]), 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse replay timestamp")
			}

			if bytes.Equal(split[3], []byte(game.Client.Username)) {
				player1, player2 = string(split[3]), string(split[2])
			} else {
				player1, player2 = string(split[2]), string(split[3])
			}
		}
	}

	_ = os.MkdirAll(replayDir, 0700)
	filePath := path.Join(replayDir, fmt.Sprintf("%d_%s_%s.match", timestamp, player1, player2))
	err = os.WriteFile(filePath, content, 0600)
	if err != nil {
		return fmt.Errorf("failed to write replay to %s: %s", filePath, err)
	}
	l(fmt.Sprintf("*** %s: %s", gotext.Get("Downloaded replay"), filePath))
	return nil
}

func showKeyboard() {
	// Do not show the virtual keyboard on desktop platforms.
}

func hideKeyboard() {
	// Do not show the virtual keyboard on desktop platforms.
}
