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
	OptimizeDraw         = true
	OptimizeSetRect      = true
	AutoEnableTouchInput = true
	ShowServerSettings   = true
	APPNAME              = "boxcars-android"
	fieldHeight          = 100
)

func init() {
	log.SetOutput(os.Stdout)

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

func loadUsername() string {
	configDir := userConfigDir()
	if configDir == "" {
		return ""
	}
	buf, err := os.ReadFile(path.Join(configDir, "config"))
	if err != nil {
		return ""
	}
	return string(bytes.TrimSpace(buf))
}

func saveUsername(username string) {
	configDir := userConfigDir()
	if configDir == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(configDir), 0700)
	_ = os.WriteFile(path.Join(configDir, "config"), []byte(username), 0600)
}

func userConfigDir() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return path.Join(configDir, "boxcars")
}

func saveReplay(id int, content []byte) error {
	if id <= 0 {
		return nil
	}
	l(fmt.Sprintf("*** %s https://bgammon.org/match/%d", gotext.Get("To download this replay visit"), id))
	return nil
}
