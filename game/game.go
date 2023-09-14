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

	"code.rocket9labs.com/tslocum/bgammon"

	"code.rocketnine.space/tslocum/messeji"

	"code.rocketnine.space/tslocum/kibodo"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/nfnt/resize"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

//go:embed assets
var assetsFS embed.FS

var debugExtra []byte

var debugGame *Game

var mplusNormalFont font.Face

func init() {
	tt, err := opentype.Parse(fonts.MPlus1pRegular_ttf)
	if err != nil {
		log.Fatal(err)
	}

	const dpi = 72
	mplusNormalFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    28,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}

	StatusWriter = messeji.NewTextField(mplusNormalFont)
	GameWriter = messeji.NewTextField(mplusNormalFont)
}

var (
	imgCheckerLight *ebiten.Image
	imgCheckerDark  *ebiten.Image

	smallFont  font.Face
	mediumFont font.Face
	monoFont   font.Face
	largeFont  font.Face

	StatusWriter *messeji.TextField
	GameWriter   *messeji.TextField
)

var (
	lightCheckerColor = color.RGBA{232, 211, 162, 255}
	darkCheckerColor  = color.RGBA{0, 0, 0, 255}
)

const DefaultServerAddress = "ws://localhost:1338" // TODO

const maxStatusWidthRatio = 0.5

const bufferCharacterWidth = 54

const lobbyCharacterWidth = 48

const showGameBufferLines = 4

const (
	minWidth  = 320
	minHeight = 240
)

func init() {
	loadAssets(0)

	initializeFonts()
}

func loadAssets(width int) {
	imgCheckerLight = loadAsset("assets/checker_white.png", width)
	imgCheckerDark = loadAsset("assets/checker_white.png", width)
	//imgCheckerDark = loadAsset("assets/checker_black.png", width)
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
	smallFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    smallFontSize,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}
	mediumFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    mediumFontSize,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}
	largeFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    largeFontSize,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}

	tt, err = opentype.Parse(fonts.PressStart2P_ttf)
	if err != nil {
		log.Fatal(err)
	}
	monoFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    monoFontSize,
		DPI:     dpi,
		Hinting: font.HintingNone,
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

	Client *Client

	Board *board

	lobby        *lobby
	pendingGames []bgammon.GameListing

	runeBuffer  []rune
	inputBuffer string

	Debug int

	debugImg *ebiten.Image

	keyboard      *kibodo.Keyboard
	keyboardInput []*kibodo.Input
	shownKeyboard bool

	statusBuffer *tabbedBuffers
	gameBuffer   *tabbedBuffers

	cpuProfile *os.File

	op *ebiten.DrawImageOptions
}

func NewGame() *Game {
	g := &Game{
		op: &ebiten.DrawImageOptions{
			Filter: ebiten.FilterNearest,
		},
		Board: NewBoard(),

		lobby: NewLobby(),

		runeBuffer: make([]rune, 24),

		keyboard: kibodo.NewKeyboard(),

		statusBuffer: newTabbedBuffers(),
		gameBuffer:   newTabbedBuffers(),

		debugImg: ebiten.NewImage(200, 200),
	}
	g.keyboard.SetKeys(kibodo.KeysQWERTY)

	g.statusBuffer.acceptInput = true

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
	for e := range g.Client.Events {
		switch ev := e.(type) {
		case *bgammon.EventWelcome:
			log.Printf("got welcome message %+v", ev) // TODO
		case *bgammon.EventList:
			if viewBoard || g.lobby.refresh {
				g.lobby.setGameList(ev.Games)

				if g.lobby.refresh {
					ebiten.ScheduleFrame()
					g.lobby.refresh = false
				}
			} else {
				g.pendingGames = ev.Games
			}
		case *bgammon.EventBoard:
			g.Board.gameState = &ev.GameState
			g.Board.ProcessState()
		default:
			log.Printf("Error: Received unknown event: %+v", ev)
		}
	}
}

func (g *Game) Connect() {
	g.loggedIn = true

	address := g.ServerAddress
	if address == "" {
		address = DefaultServerAddress
	}
	g.Client = newClient(address, g.Username, g.Password)
	g.lobby.c = g.Client
	g.Board.Client = g.Client
	g.statusBuffer.client = g.Client

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

	go c.Connect()
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
	if g.pendingGames != nil && viewBoard {
		g.lobby.setGameList(g.pendingGames)
		g.pendingGames = nil
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
				g.keyboard.Show()
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

			// Process on-screen keyboard input.
			g.keyboardInput = g.keyboard.AppendInput(g.keyboardInput[:0])
			for _, input := range g.keyboardInput {
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
		}

		f()
	}

	if ebiten.IsKeyPressed(ebiten.KeyControl) && inpututil.IsKeyJustPressed(ebiten.KeyD) {
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

		g.statusBuffer.update()
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

	screen.Fill(tableColor)

	// Log in screen
	if !g.loggedIn {
		g.keyboard.Draw(screen)

		const welcomeText = `Please enter your FIBS username and password.
If you do not have a FIBS account yet, visit
http://www.com/help.html#register`
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

	g.gameBuffer.draw(screen)
	g.statusBuffer.draw(screen)
	if !viewBoard {
		// Lobby screen
		g.lobby.draw(screen)
	} else {
		// Game board screen
		g.Board.draw(screen)
	}

	if g.Debug > 0 {
		g.drawBuffer.Reset()

		g.drawBuffer.Write([]byte(fmt.Sprintf("FPS %c %0.0f", spinner[g.spinnerIndex], ebiten.CurrentFPS())))

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

		g.debugImg.Clear()

		ebitenutil.DebugPrint(g.debugImg, g.drawBuffer.String())

		g.resetImageOptions()
		g.op.GeoM.Translate(3, 0)
		g.op.GeoM.Scale(2, 2)
		screen.DrawImage(g.debugImg, g.op)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	s := ebiten.DeviceScaleFactor()
	outsideWidth, outsideHeight = int(float64(outsideWidth)*s), int(float64(outsideHeight)*s)
	if outsideWidth < minWidth {
		outsideWidth = minWidth
	}
	if outsideHeight < minHeight {
		outsideHeight = minHeight
	}
	if g.screenW == outsideWidth && g.screenH == outsideHeight {
		return outsideWidth, outsideHeight
	}

	g.screenW, g.screenH = outsideWidth, outsideHeight

	statusBufferWidth := text.BoundString(g.statusBuffer.chatFont, strings.Repeat("A", bufferCharacterWidth)).Dx()
	if statusBufferWidth > int(float64(g.screenW)*maxStatusWidthRatio) {
		statusBufferWidth = int(float64(g.screenW) * maxStatusWidthRatio)
	}

	g.Board.fullHeight = true
	g.Board.setRect(0, 0, g.screenW-statusBufferWidth, g.screenH)

	availableWidth := g.screenW - (g.Board.innerW + int(g.Board.horizontalBorderSize*2))
	if availableWidth > statusBufferWidth {
		statusBufferWidth = availableWidth
		g.Board.setRect(0, 0, g.screenW-statusBufferWidth, g.screenH)
	}

	if g.Board.h > g.Board.w {
		g.Board.fullHeight = false
		g.Board.setRect(0, 0, g.Board.w, g.Board.w)
	}

	if g.screenW > 200 {
		g.statusBuffer.padding = 2
		g.gameBuffer.padding = 2
	} else if g.screenW > 100 {
		g.statusBuffer.padding = 1
		g.gameBuffer.padding = 1
	} else {
		g.statusBuffer.padding = 0
		g.gameBuffer.padding = 0
	}

	bufferPadding := int(g.Board.horizontalBorderSize / 2)

	gameBufferHeight := (g.gameBuffer.chatLineHeight * showGameBufferLines) + (g.gameBuffer.padding * 4)

	g.lobby.buttonBarHeight = gameBufferHeight + int(float64(bufferPadding)*1.5)
	minLobbyWidth := text.BoundString(mediumFont, strings.Repeat("A", lobbyCharacterWidth)).Dx()
	if g.Board.w >= minLobbyWidth {
		g.lobby.fullscreen = false
		g.lobby.setRect(0, 0, g.Board.w, g.screenH)
	} else {
		g.lobby.fullscreen = true
		g.lobby.setRect(0, 0, g.screenW, g.screenH)
	}

	if true || availableWidth >= 150 { // TODO allow chat window to be repositioned
		statusBufferHeight := g.screenH - gameBufferHeight - bufferPadding*3

		g.statusBuffer.docked = true
		g.statusBuffer.setRect((g.screenW-statusBufferWidth)+bufferPadding, bufferPadding, statusBufferWidth-(bufferPadding*2), statusBufferHeight)

		g.gameBuffer.docked = true
		g.gameBuffer.setRect((g.screenW-statusBufferWidth)+bufferPadding, (g.screenH-(gameBufferHeight))-bufferPadding, statusBufferWidth-(bufferPadding*2), gameBufferHeight)
	} else {
		// Clamp buffer position.
		bx, by := g.statusBuffer.x, g.statusBuffer.y
		var bw, bh int
		if g.statusBuffer.w == 0 && g.statusBuffer.h == 0 {
			// Set initial buffer position.
			bx = 0
			by = g.screenH / 2
			// Set initial buffer size.
			bw = g.screenW
			bh = g.screenH / 2
		} else {
			// Scale existing buffer size
			bx, by = bx*(outsideWidth/g.screenW), by*(outsideHeight/g.screenH)
			bw, bh = g.statusBuffer.w*(outsideWidth/g.screenW), g.statusBuffer.h*(outsideHeight/g.screenH)
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

		g.statusBuffer.docked = false
		g.statusBuffer.setRect(bx, by, bw, bh)
	}

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

type messageHandler struct {
	t *textBuffer
}

func NewMessageHandler(t *textBuffer) *messageHandler {
	return &messageHandler{
		t: t,
	}
}

func (m *messageHandler) Write(p []byte) (n int, err error) {
	fmt.Print(string(p))

	m.t.Write(p)
	return len(p), nil
}

// TODO

type WhoInfo struct {
	Username   string
	Opponent   string
	Watching   string
	Ready      bool
	Away       bool
	Rating     int
	Experience int
	Idle       int
	LoginTime  int
	ClientName string
}

func (w *WhoInfo) String() string {
	opponent := "In the lobby"
	if w.Opponent != "" && w.Opponent != "-" {
		opponent = "playing against " + w.Opponent
	}
	clientName := ""
	if w.ClientName != "" && w.ClientName != "-" {
		clientName = " using " + w.ClientName
	}
	return fmt.Sprintf("%s (rated %d with %d exp) is %s%s", w.Username, w.Rating, w.Experience, opponent, clientName)
}
