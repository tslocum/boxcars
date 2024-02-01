//go:build android

package game

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/leonelquinteros/gotext"
	"golang.org/x/text/language"
)

const (
	AppName            = "boxcars-android"
	ShowServerSettings = true
)

func init() {
	log.SetOutput(os.Stdout)

	AutoEnableTouchInput = true

	// Detect timezone.
	out, err := exec.Command("/system/bin/getprop", "persist.sys.timezone").Output()
	if err != nil {
		return
	}
	tz, err := time.LoadLocation(strings.TrimSpace(string(out)))
	if err != nil {
		return
	}
	time.Local = tz

	// Detect locale.
	out, err = exec.Command("/system/bin/getprop", "persist.sys.locale").Output()
	if err != nil {
		return
	}
	tag, err := language.Parse(strings.TrimSpace(string(out)))
	if err != nil {
		return
	}
	LoadLocale(&tag)
}

func DefaultLocale() string {
	return ""
}

func focused() bool {
	return true
}

func userConfigDir() string {
	return "/data/data/com.rocket9labs.boxcars"
}

func loadCredentials() (string, string) {
	configDir := userConfigDir()
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
	_ = os.MkdirAll(filepath.Dir(configDir), 0700)
	_ = os.WriteFile(path.Join(configDir, "config"), []byte(username+"\n"+password), 0600)
}

func saveReplay(id int, content []byte) error {
	if id <= 0 {
		return nil
	}
	l(fmt.Sprintf("*** %s https://bgammon.org/match/%d", gotext.Get("To download this replay visit"), id))
	return nil
}

func showKeyboard() {
	game.keyboardHintVisible = true
}
