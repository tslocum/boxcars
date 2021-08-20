package game

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"golang.org/x/image/font/opentype"

	"github.com/nfnt/resize"

	// Asset decoding
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"golang.org/x/image/font"
)

// Copyright 2020 The Ebiten Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

const maxAngle = 256

//go:embed assets
var assetsFS embed.FS

var debugExtra []byte

var (
	imgCheckerWhite *ebiten.Image
	imgCheckerBlack *ebiten.Image

	mplusNormalFont font.Face
	mplusBigFont    font.Face
)

func init() {
	loadAssets(0)

	initializeFonts()
}

func loadAssets(width int) {
	imgCheckerWhite = loadAsset("assets/checker_white.png", width)
	imgCheckerBlack = loadAsset("assets/checker_black.png", width)
}

func loadAsset(assetPath string, width int) *ebiten.Image {
	f, err := assetsFS.Open(assetPath)
	if err != nil {
		panic(err)
	}

	img, _, err := image.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	if width > 0 {
		imgResized := resize.Resize(uint(width), 0, img, resize.Lanczos3)
		return ebiten.NewImageFromImage(imgResized)
	}
	return ebiten.NewImageFromImage(img)

}

func initializeFonts() {
	tt, err := opentype.Parse(fonts.MPlus1pRegular_ttf)
	if err != nil {
		log.Fatal(err)
	}

	const dpi = 72
	mplusNormalFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    24,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}
	mplusBigFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    32,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}
}

// TODO copied

func line(x0, y0, x1, y1 float32, clr color.RGBA) ([]ebiten.Vertex, []uint16) {
	const width = 1

	theta := math.Atan2(float64(y1-y0), float64(x1-x0))
	theta += math.Pi / 2
	dx := float32(math.Cos(theta))
	dy := float32(math.Sin(theta))

	r := float32(clr.R) / 0xff
	g := float32(clr.G) / 0xff
	b := float32(clr.B) / 0xff
	a := float32(clr.A) / 0xff

	return []ebiten.Vertex{
		{
			DstX:   x0 - width*dx/2,
			DstY:   y0 - width*dy/2,
			SrcX:   1,
			SrcY:   1,
			ColorR: r,
			ColorG: g,
			ColorB: b,
			ColorA: a,
		},
		{
			DstX:   x0 + width*dx/2,
			DstY:   y0 + width*dy/2,
			SrcX:   1,
			SrcY:   1,
			ColorR: r,
			ColorG: g,
			ColorB: b,
			ColorA: a,
		},
		{
			DstX:   x1 - width*dx/2,
			DstY:   y1 - width*dy/2,
			SrcX:   1,
			SrcY:   1,
			ColorR: r,
			ColorG: g,
			ColorB: b,
			ColorA: a,
		},
		{
			DstX:   x1 + width*dx/2,
			DstY:   y1 + width*dy/2,
			SrcX:   1,
			SrcY:   1,
			ColorR: r,
			ColorG: g,
			ColorB: b,
			ColorA: a,
		},
	}, []uint16{0, 1, 2, 1, 2, 3}
}

type Sprite struct {
	image      *ebiten.Image
	w          int
	h          int
	x          int
	y          int
	colorWhite bool
}

func (s *Sprite) Update() {
	return // TODO
}

type Sprites struct {
	sprites []*Sprite
	num     int
}

func (s *Sprites) Update() {
	for i := 0; i < s.num; i++ {
		s.sprites[i].Update()
	}
}

const (
	MinSprites = 0
	MaxSprites = 50000
)

type Game struct {
	touchIDs []ebiten.TouchID
	sprites  Sprites
	op       *ebiten.DrawImageOptions
	board    *board

	screenW, screenH int

	drawBuffer bytes.Buffer
}

func NewGame() *Game {
	rand.Seed(time.Now().UnixNano())

	go func() {
		// TODO fetch HTTP request, set debugExtra
	}()

	return &Game{
		op: &ebiten.DrawImageOptions{
			Filter: ebiten.FilterNearest,
		},
		board: NewBoard(),
	}
}

func (g *Game) leftTouched() bool {
	for _, id := range g.touchIDs {
		x, _ := ebiten.TouchPosition(id)
		/*if x < screenWidth/2 {
			return true
		}*/
		_ = x
	}
	return false
}

func (g *Game) rightTouched() bool {
	for _, id := range g.touchIDs {
		x, _ := ebiten.TouchPosition(id)
		/*if x >= screenWidth/2 {
			return true
		}*/
		_ = x
	}
	return false
}

func (g *Game) Update() error {
	g.touchIDs = ebiten.AppendTouchIDs(g.touchIDs[:0])

	// Decrease the number of the sprites.
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || g.leftTouched() {
		g.sprites.num -= 20
		if g.sprites.num < MinSprites {
			g.sprites.num = MinSprites
		}
	}

	// Increase the number of the sprites.
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) || g.rightTouched() {
		g.sprites.num += 20
		if MaxSprites < g.sprites.num {
			g.sprites.num = MaxSprites
		}
	}

	g.board.update()

	//g.sprites.Update()
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 102, 51, 255})

	g.board.draw(screen)

	debugBox := image.NewRGBA(image.Rect(10, 20, 200, 200))
	debugImg := ebiten.NewImageFromImage(debugBox)

	g.drawBuffer.Reset()

	g.drawBuffer.Write([]byte(fmt.Sprintf("FPS %0.0f\nTPS %0.0f", ebiten.CurrentFPS(), ebiten.CurrentTPS())))

	scaleFactor := ebiten.DeviceScaleFactor()
	if scaleFactor != 1.0 {
		g.drawBuffer.WriteRune('\n')
		g.drawBuffer.Write([]byte(fmt.Sprintf("SCA %0.1f", scaleFactor)))
	}

	if debugExtra != nil {
		g.drawBuffer.WriteRune('\n')
		g.drawBuffer.Write(debugExtra)
	}

	ebitenutil.DebugPrint(debugImg, g.drawBuffer.String())

	g.resetImageOptions()
	g.op.GeoM.Translate(3, 0)
	g.op.GeoM.Scale(2, 2)
	screen.DrawImage(debugImg, g.op)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	if g.screenW == outsideWidth && g.screenH == outsideHeight {
		return outsideWidth, outsideHeight
	}

	g.screenW, g.screenH = outsideWidth, outsideHeight

	g.board.setRect(300, 0, g.screenW-300, g.screenH)

	// TODO use scale factor
	return outsideWidth, outsideHeight
}

func (g *Game) resetImageOptions() {
	g.op.GeoM.Reset()
}
