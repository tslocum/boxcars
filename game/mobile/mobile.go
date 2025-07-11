package mobile

import (
	_ "unsafe" // Import unsafe to enable workaround below.

	"codeberg.org/tslocum/boxcars/game"
	"github.com/hajimehoshi/ebiten/v2/mobile"
)

func init() {
	mobile.SetGame(game.NewGame())
}

// Dummy is a dummy exported function.
//
// gomobile will only compile packages that include at least one exported function.
// Dummy forces gomobile to compile this package.
func Dummy() {}

// Workaround Go compiler bug affecting all Android versions before 12.
// Issue: https://github.com/golang/go/issues/70508

//go:linkname checkPidfdOnce os.checkPidfdOnce
var checkPidfdOnce func() error
