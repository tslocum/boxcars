//go:build linux && !android

package game

import (
	"os"
)

func GetLocale() (string, error) {
	return os.Getenv("LANG"), nil
}
