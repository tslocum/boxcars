package game

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"strings"
	"time"

	"code.rocketnine.space/tslocum/fibs"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/nfnt/resize"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
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

//go:embed assets
var assetsFS embed.FS

var debugExtra []byte

var (
	imgCheckerWhite *ebiten.Image
	imgCheckerBlack *ebiten.Image

	mplusNormalFont font.Face
	mplusBigFont    font.Face
)

const defaultServerAddress = "fibs.com:4321"

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

var spinner = []byte(`-\|/`)

type Game struct {
	touchIDs []ebiten.TouchID
	sprites  Sprites
	op       *ebiten.DrawImageOptions
	Board    *board

	screenW, screenH int

	drawBuffer bytes.Buffer

	spinnerIndex int

	ServerAddress     string
	Username          string
	Password          string
	loggedIn          bool
	usernameConfirmed bool

	Watch bool

	Client *fibs.Client

	runeBuffer  []rune
	inputBuffer string

	Debug int
}

func NewGame() *Game {
	go func() {
		// TODO fetch HTTP request, set debugExtra
	}()

	g := &Game{
		op: &ebiten.DrawImageOptions{
			Filter: ebiten.FilterNearest,
		},
		Board: NewBoard(),

		runeBuffer: make([]rune, 24),
	}

	go func() {
		t := time.NewTicker(time.Second / 4)
		for range t.C {
			_ = g.update()
		}
	}()

	return g
}

func (g *Game) handleEvents() {
	for e := range g.Client.Event {
		switch event := e.(type) {
		case *fibs.EventMove:
			g.Board.movePiece(event.From, event.To)
		}
	}
}

func (g *Game) Connect() {
	g.loggedIn = true

	address := g.ServerAddress
	if address == "" {
		address = defaultServerAddress
	}
	g.Client = fibs.NewClient(address, g.Username, g.Password)

	go g.handleEvents()

	c := g.Client

	if g.Watch {
		go func() {
			time.Sleep(1 * time.Second)
			c.WatchRandomGame()

			go g.renderLoop()
		}()
	}

	go func() {
		err := c.Connect()
		if err != nil {
			log.Fatal(err)
		}
	}()
}

func (g *Game) renderLoop() {
	t := time.NewTicker(time.Second)
	for range t.C {
		gameBoard := g.Board
		v := g.Client.Board.GetIntState()
		gameBoard.SetState(v)
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

// Separate update function for all normal update logic, as Update may only be
// called when there is user input when vsync is disabled.
func (g *Game) update() error {
	return nil
}

func (g *Game) Update() error { // Called by ebiten only when input occurs
	err := g.update()
	if err != nil {
		return err
	}

	if !g.loggedIn {
		f := func() {
			var clearBuffer bool
			defer func() {
				if clearBuffer {
					g.inputBuffer = ""

					if !g.usernameConfirmed {
						g.usernameConfirmed = true
					} else if g.Password != "" {
						g.Connect()
					}
				}
			}()

			if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(g.inputBuffer) > 0 {
				g.inputBuffer = g.inputBuffer[:len(g.inputBuffer)-1]
			}

			if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
				clearBuffer = true
			}

			g.runeBuffer = ebiten.AppendInputChars(g.runeBuffer[:0])
			if len(g.runeBuffer) > 0 {
				g.inputBuffer += string(g.runeBuffer)

				if strings.ContainsRune(g.inputBuffer, '\n') {
					g.inputBuffer = strings.Split(g.inputBuffer, "\n")[0]
					clearBuffer = true
				}
				if !g.usernameConfirmed {
					g.Username = g.inputBuffer
				} else {
					g.Password = g.inputBuffer
				}
				log.Println("INPUT BUFFER IS:" + g.inputBuffer)
			}
		}

		f()
	}

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

	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		g.Debug++
		if g.Debug == 3 {
			g.Debug = 0
		}
		g.Board.debug = g.Debug
	}

	g.Board.update()

	//g.sprites.Update()
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 102, 51, 255})

	// Log in screen
	if !g.loggedIn {
		const welcomeText = `Please enter your FIBS username and password.
If you do not have a FIBS account yet, visit
http://www.fibs.com/help.html#register`
		debugBox := image.NewRGBA(image.Rect(0, 0, g.screenW, g.screenH))
		debugImg := ebiten.NewImageFromImage(debugBox)

		if !g.usernameConfirmed {
			ebitenutil.DebugPrint(debugImg, welcomeText+fmt.Sprintf("\n\nUsername: %s", g.Username))
		} else {
			ebitenutil.DebugPrint(debugImg, welcomeText+fmt.Sprintf("\n\nPassword: %s", strings.Repeat("*", len(g.Password))))
		}

		g.resetImageOptions()
		g.op.GeoM.Scale(2, 2)
		screen.DrawImage(debugImg, g.op)
		return
	}

	// Game screen

	g.Board.draw(screen)

	if g.Debug == 1 {
		debugBox := image.NewRGBA(image.Rect(10, 20, 200, 200))
		debugImg := ebiten.NewImageFromImage(debugBox)

		g.drawBuffer.Reset()

		g.drawBuffer.Write([]byte(fmt.Sprintf("FPS %0.0f\nTPS %0.0f", ebiten.CurrentFPS(), ebiten.CurrentTPS())))

		/* TODO enable when vsync is able to be turned off
		g.spinnerIndex++
		if g.spinnerIndex == 4 {
			g.spinnerIndex = 0
		}*/

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
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	s := ebiten.DeviceScaleFactor()
	outsideWidth, outsideHeight = int(float64(outsideWidth)*s), int(float64(outsideHeight)*s)
	if g.screenW == outsideWidth && g.screenH == outsideHeight {
		return outsideWidth, outsideHeight
	}

	g.screenW, g.screenH = outsideWidth, outsideHeight
	g.Board.setRect(0, 0, g.screenW, g.screenH)
	return outsideWidth, outsideHeight
}

func (g *Game) resetImageOptions() {
	g.op.GeoM.Reset()
}
