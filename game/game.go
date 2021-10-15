package game

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"os"
	"path"
	"runtime/pprof"
	"strings"
	"time"

	"code.rocketnine.space/tslocum/fibs"
	"code.rocketnine.space/tslocum/kibodo"
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

var debugGame *Game

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
	toStart    time.Time
	toTime     time.Duration
	toX        int
	toY        int
	colorWhite bool
	premove    bool
}

type Sprites struct {
	sprites []*Sprite
	num     int
}

var spinner = []byte(`-\|/`)

var viewBoard bool // View board or lobby

type Game struct {
	screenW, screenH int

	drawBuffer bytes.Buffer
	lastDraw   time.Time

	spinnerIndex int

	ServerAddress     string
	Username          string
	Password          string
	loggedIn          bool
	usernameConfirmed bool

	Watch bool
	TV    bool

	Client *fibs.Client

	Board *board

	lobby      *lobby
	pendingWho []*fibs.WhoInfo

	runeBuffer  []rune
	inputBuffer string

	Debug int

	keyboard      *kibodo.Keyboard
	shownKeyboard bool

	buffers *tabbedBuffers

	cpuProfile *os.File

	op *ebiten.DrawImageOptions
}

func NewGame() *Game {
	fibs.Debug = 1 // TODO

	g := &Game{
		op: &ebiten.DrawImageOptions{
			Filter: ebiten.FilterNearest,
		},
		Board: NewBoard(),

		lobby: NewLobby(),

		runeBuffer: make([]rune, 24),

		keyboard: kibodo.NewKeyboard(),

		buffers: newTabbedBuffers(),
	}
	g.keyboard.SetKeys(kibodo.KeysQWERTY)

	// TODO
	go func() {
		/*
			time.Sleep(5 * time.Second)
			g.lobby.offset += 10
			g.lobby.bufferDirty = true
			g.toggleProfiling()
			g.lobby.drawBuffer()
			g.toggleProfiling()
			os.Exit(0)
		*/

		t := time.NewTicker(time.Second / 4)
		for range t.C {

			_ = g.update()
		}
	}()

	debugGame = g // TODO
	return g
}

func (g *Game) handleEvents() {
	for e := range g.Client.Event {
		switch event := e.(type) {
		case *fibs.EventWho:
			if viewBoard || g.lobby.refresh {
				g.lobby.setWhoInfo(event.Who)

				if g.lobby.refresh {
					ebiten.ScheduleFrame()
					g.lobby.refresh = false
				}
			} else {
				g.pendingWho = event.Who
			}
		case *fibs.EventBoardState:
			g.Board.SetState(event.S, event.V)
		case *fibs.EventMove:
			g.Board.movePiece(event.From, event.To)
		case *fibs.EventDraw:
			g.Board.ProcessState()
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
	g.lobby.c = g.Client
	g.Board.Client = g.Client

	go g.handleEvents()

	c := g.Client

	if g.TV {
		go func() {
			time.Sleep(time.Second)
			c.Out <- []byte("tv")
		}()
	} else if g.Watch {
		go func() {
			time.Sleep(time.Second)
			c.Out <- []byte("watch")
		}()
	}

	go func() {
		err := c.Connect()
		if err != nil {
			log.Fatal(err)
		}
	}()
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

	if ebiten.IsWindowBeingClosed() {
		g.Exit()
		return nil
	}
	if g.pendingWho != nil && viewBoard {
		g.lobby.setWhoInfo(g.pendingWho)
		g.pendingWho = nil
	}

	if ebiten.IsKeyPressed(ebiten.KeyControl) && inpututil.IsKeyJustPressed(ebiten.KeyP) {
		err = g.toggleProfiling()
		if err != nil {
			return err
		}
	}

	err = g.keyboard.Update()
	if err != nil {
		return fmt.Errorf("failed to update virtual keyboard: %s", err)
	}

	if !g.loggedIn {
		f := func() {
			var clearBuffer bool
			defer func() {
				if strings.ContainsRune(g.inputBuffer, '\n') {
					g.inputBuffer = strings.Split(g.inputBuffer, "\n")[0]
					clearBuffer = true
				}
				if !g.usernameConfirmed {
					g.Username = g.inputBuffer
				} else {
					g.Password = g.inputBuffer
				}

				if clearBuffer {
					g.inputBuffer = ""

					if !g.usernameConfirmed {
						g.usernameConfirmed = true
					} else if g.Password != "" {
						g.Connect()
					}
				}
			}()

			if !g.shownKeyboard {
				ch := make(chan *kibodo.Input, 10)
				go func() {
					for input := range ch {
						if input.Rune > 0 {
							g.inputBuffer += string(input.Rune)
							continue
						}
						if input.Key == ebiten.KeyBackspace {
							if len(g.inputBuffer) > 0 {
								g.inputBuffer = g.inputBuffer[:len(g.inputBuffer)-1]
							}
						} else if input.Key == ebiten.KeyEnter {
							g.inputBuffer += "\n"
						}
					}
				}()
				g.keyboard.Show(ch)
				g.shownKeyboard = true
			}

			if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(g.inputBuffer) > 0 {
				g.inputBuffer = g.inputBuffer[:len(g.inputBuffer)-1]
			}

			if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
				clearBuffer = true
			}

			g.runeBuffer = ebiten.AppendInputChars(g.runeBuffer[:0])
			if len(g.runeBuffer) > 0 {
				g.inputBuffer += string(g.runeBuffer)
			}
		}

		f()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		g.Debug++
		if g.Debug == 3 {
			g.Debug = 0
		}
		g.Board.debug = g.Debug
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		viewBoard = !viewBoard
	}

	if !viewBoard {
		g.lobby.update()
	} else {
		g.Board.update()
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	frameTime := time.Second / 175
	if time.Since(g.lastDraw) < frameTime {
		//time.Sleep(time.Until(g.lastDraw.Add(frameTime)))
		// TODO causes panics on WASM
		// draw offscreen and cache, redraw cached image instead of sleeping?
	}
	g.lastDraw = time.Now()

	screen.Fill(color.RGBA{0, 102, 51, 255})

	// Log in screen
	if !g.loggedIn {
		g.keyboard.Draw(screen)

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

	if !viewBoard {
		// Lobby screen
		g.lobby.draw(screen)
	} else {
		// Game board screen
		g.Board.draw(screen)

		g.buffers.draw(screen)
	}

	if g.Debug > 0 {
		debugBox := image.NewRGBA(image.Rect(10, 20, 200, 200))
		debugImg := ebiten.NewImageFromImage(debugBox)

		g.drawBuffer.Reset()

		g.drawBuffer.Write([]byte(fmt.Sprintf("FPS %0.0f %c\nTPS %0.0f", ebiten.CurrentFPS(), spinner[g.spinnerIndex], ebiten.CurrentTPS())))

		g.spinnerIndex++
		if g.spinnerIndex == 4 {
			g.spinnerIndex = 0
		}

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

	g.lobby.setRect(0, 0, g.screenW, g.screenH)
	g.Board.setRect(0, 0, g.screenW, g.screenH)

	// Clamp buffer position.
	bx, by := g.buffers.x, g.buffers.y
	var bw, bh int
	if g.buffers.w == 0 && g.buffers.h == 0 {
		// Set initial buffer position.
		bx = g.screenW / 2
		by = g.screenH / 2
		// Set initial buffer size.
		bw = g.screenW / 2
		bh = g.screenH / 4
	} else {
		// Scale existing buffer size
		bx, by = bx*(outsideWidth/g.screenW), by*(outsideHeight/g.screenH)
		bw, bh = g.buffers.w*(outsideWidth/g.screenW), g.buffers.h*(outsideHeight/g.screenH)
		if bw < 200 {
			bw = 200
		}
		if bh < 100 {
			bh = 100
		}
	}
	padding := 7
	if bx > g.screenW-padding {
		bx = g.screenW - padding
	}
	if by > g.screenH-padding {
		by = g.screenH - padding
	}
	g.buffers.setRect(bx, by, bw, bh)

	displayArea := 200
	g.keyboard.SetRect(0, displayArea, g.screenW, g.screenH-displayArea)
	return outsideWidth, outsideHeight
}

func (g *Game) resetImageOptions() {
	g.op.GeoM.Reset()
}

func (g *Game) toggleProfiling() error {
	if g.cpuProfile == nil {
		log.Println("Profiling started...")

		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		g.cpuProfile, err = os.Create(path.Join(homeDir, "cpu.prof")) // TODO add flag
		if err != nil {
			return err
		}

		if err := pprof.StartCPUProfile(g.cpuProfile); err != nil {
			return err
		}

		return nil
	}

	pprof.StopCPUProfile()
	g.cpuProfile.Close()
	g.cpuProfile = nil

	log.Println("Profiling stopped")
	return nil
}

func (g *Game) Exit() {
	g.Board.drawFrame <- false
	os.Exit(0)
}
