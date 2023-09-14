package mobile

import (
	"code.rocket9labs.com/tslocum/boxcars/game"
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
