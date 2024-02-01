//go:build android

package game

import (
	"bytes"
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

func loadUsername() string {
	configDir := userConfigDir()
	buf, err := os.ReadFile(path.Join(configDir, "config"))
	if err != nil {
		return ""
	}
	return string(bytes.TrimSpace(buf))
}

func saveUsername(username string) {
	configDir := userConfigDir()
	_ = os.MkdirAll(filepath.Dir(configDir), 0700)
	_ = os.WriteFile(path.Join(configDir, "config"), []byte(username), 0600)
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
