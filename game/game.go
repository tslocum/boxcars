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

	"code.rocketnine.space/tslocum/messeji"

	"code.rocket9labs.com/tslocum/bgammon"

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

var (
	imgCheckerLight *ebiten.Image
	imgCheckerDark  *ebiten.Image

	smallFont  font.Face
	mediumFont font.Face
	monoFont   font.Face
	largeFont  font.Face
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

const (
	smallFontSize  = 20
	monoFontSize   = 10
	mediumFontSize = 24
	largeFontSize  = 32
)

const (
	monoLineHeight     = 14
	standardLineHeight = 24
)

const lobbyStatusBufferHeight = 75

var (
	bufferTextColor       = triangleALight
	bufferBackgroundColor = color.RGBA{0, 0, 0, 100}
)

var (
	statusBuffer = messeji.NewTextField(defaultFont())
	gameBuffer   = messeji.NewTextField(defaultFont())
	inputBuffer  = messeji.NewInputField(defaultFont())

	statusLogged bool
	gameLogged   bool

	statusBufferRect image.Rectangle // In-game rect of status buffer.

	Debug int

	game *Game
)

func l(s string) {
	m := time.Now().Format("15:04") + " " + s
	if statusBuffer != nil {
		if statusLogged {
			statusBuffer.Write([]byte("\n" + m))
			ebiten.ScheduleFrame()
			return
		}
		statusBuffer.Write([]byte(m))
		statusLogged = true
		ebiten.ScheduleFrame()
		return
	}
	log.Print(m)
}

func lg(s string) {
	m := time.Now().Format("15:04") + " " + s
	if gameBuffer != nil {
		if gameLogged {
			gameBuffer.Write([]byte("\n" + m))
			ebiten.ScheduleFrame()
			return
		}
		gameBuffer.Write([]byte(m))
		gameLogged = true
		ebiten.ScheduleFrame()
		return
	}
	log.Print(m)
}

var defaultFontFace font.Face

func defaultFont() font.Face {
	if defaultFontFace != nil {
		return defaultFontFace
	}

	tt, err := opentype.Parse(fonts.MPlus1pRegular_ttf)
	if err != nil {
		log.Fatal(err)
	}

	const dpi = 72
	defaultFontFace, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    16,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}
	return defaultFontFace
}

func init() {
	//statusBuffer.acceptInput = true
	initializeFonts()

	loadAssets(0)

	statusBuffer.SetForegroundColor(bufferTextColor)
	statusBuffer.SetBackgroundColor(bufferBackgroundColor)

	gameBuffer.SetForegroundColor(bufferTextColor)
	gameBuffer.SetBackgroundColor(bufferBackgroundColor)

	inputBuffer.SetForegroundColor(bufferTextColor)
	inputBuffer.SetBackgroundColor(bufferBackgroundColor)
	inputBuffer.SetSuffix("")
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

func setViewBoard(view bool) {
	viewBoard = view
	inputBuffer.SetHandleKeyboard(viewBoard)
	if viewBoard {
		inputBuffer.SetSuffix("_")

		// Exit create game dialog, if open.
		game.lobby.showCreateGame = 0
		game.lobby.createGameName = ""
		game.lobby.createGamePassword = ""
		game.lobby.bufferDirty = true
		game.lobby.bufferButtonsDirty = true
	} else {
		inputBuffer.SetSuffix("")
	}
	game.updateStatusBufferPosition()
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

	runeBuffer []rune
	userInput  string

	debugImg *ebiten.Image

	keyboard      *kibodo.Keyboard
	keyboardInput []*kibodo.Input
	shownKeyboard bool

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

		debugImg: ebiten.NewImage(200, 200),
	}
	game = g

	g.keyboard.SetKeys(kibodo.KeysQWERTY)

	inputBuffer.SetSelectedFunc(g.acceptInput)

	return g
}

func (g *Game) handleEvents() {
	for e := range g.Client.Events {
		switch ev := e.(type) {
		case *bgammon.EventWelcome:
			g.Client.Username = ev.PlayerName

			areIs := "are"
			if ev.Clients == 1 {
				areIs = "is"
			}
			clientsPlural := "s"
			if ev.Clients == 1 {
				clientsPlural = ""
			}
			matchesPlural := "es"
			if ev.Games == 1 {
				matchesPlural = ""
			}
			l(fmt.Sprintf("*** Welcome, %s. There %s %d client%s playing %d match%s.", ev.PlayerName, areIs, ev.Clients, clientsPlural, ev.Games, matchesPlural))
		case *bgammon.EventHelp:
			l(fmt.Sprintf("*** Help: %s", ev.Message))
		case *bgammon.EventNotice:
			l(fmt.Sprintf("*** %s", ev.Message))
		case *bgammon.EventSay:
			l(fmt.Sprintf("<%s> %s", ev.Player, ev.Message))
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

		case *bgammon.EventJoined:
			if ev.PlayerNumber == 1 {
				g.Board.gameState.Player1.Name = ev.Player
			} else if ev.PlayerNumber == 2 {
				g.Board.gameState.Player2.Name = ev.Player
			}
			g.Board.ProcessState()
			setViewBoard(true)

			if ev.Player != g.Client.Username {
				lg(fmt.Sprintf("%s joined the match.", ev.Player))
			}
		case *bgammon.EventFailedJoin:
			l(fmt.Sprintf("*** Failed to join match: %s", ev.Reason))
		case *bgammon.EventLeft:
			if g.Board.gameState.Player1.Name == ev.Player {
				g.Board.gameState.Player1.Name = ""
			} else if g.Board.gameState.Player2.Name == ev.Player {
				g.Board.gameState.Player2.Name = ""
			}
			g.Board.ProcessState()

			if ev.Player != g.Client.Username {
				lg(fmt.Sprintf("%s left the match.", ev.Player))
			}
		case *bgammon.EventBoard:
			g.Board.gameState = &ev.GameState
			g.Board.ProcessState()
			setViewBoard(true)
		case *bgammon.EventRolled:
			g.Board.gameState.Roll1 = ev.Roll1
			g.Board.gameState.Roll2 = ev.Roll2
			var diceFormatted string
			if g.Board.gameState.Turn == 0 {
				if g.Board.gameState.Player1.Name == ev.Player {
					diceFormatted = fmt.Sprintf("%d", g.Board.gameState.Roll1)
				} else {
					diceFormatted = fmt.Sprintf("%d", g.Board.gameState.Roll2)
				}
			} else {
				diceFormatted = fmt.Sprintf("%d-%d", g.Board.gameState.Roll1, g.Board.gameState.Roll2)
			}
			g.Board.ProcessState()
			lg(fmt.Sprintf("%s rolled %s.", ev.Player, diceFormatted))
		case *bgammon.EventFailedRoll:
			l(fmt.Sprintf("*** Failed to roll: %s", ev.Reason))
		case *bgammon.EventMoved:
			lg(fmt.Sprintf("%s moved %s.", ev.Player, bgammon.FormatMoves(ev.Moves)))
		case *bgammon.EventFailedMove:
			g.Client.Out <- []byte("board") // Refresh game state.

			var extra string
			if ev.From != 0 || ev.To != 0 {
				extra = fmt.Sprintf(" from %s to %s", bgammon.FormatSpace(ev.From), bgammon.FormatSpace(ev.To))
			}
			l(fmt.Sprintf("*** Failed to move checker%s: %s", extra, ev.Reason))
			l(fmt.Sprintf("*** Legal moves: %s", bgammon.FormatMoves(g.Board.gameState.Available)))
		case *bgammon.EventFailedOk:
			g.Client.Out <- []byte("board") // Refresh game state.
			l(fmt.Sprintf("*** Failed to submit moves: %s", ev.Reason))
		case *bgammon.EventWin:
			lg(fmt.Sprintf("%s wins!", ev.Player))
		case *bgammon.EventPing:
			g.Client.Out <- []byte(fmt.Sprintf("pong %s", ev.Message))
		default:
			l(fmt.Sprintf("*** Warning: Received unknown event: %+v", ev))
		}
	}
}

func (g *Game) Connect() {
	g.loggedIn = true
	l(fmt.Sprintf("*** Connecting..."))

	address := g.ServerAddress
	if address == "" {
		address = DefaultServerAddress
	}
	g.Client = newClient(address, g.Username, g.Password)
	g.lobby.c = g.Client
	g.Board.Client = g.Client
	//statusBuffer.client = g.Client

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

func (g *Game) Update() error { // Called by ebiten only when input occurs
	if ebiten.IsWindowBeingClosed() {
		g.Exit()
		return nil
	}
	if g.pendingGames != nil && viewBoard {
		g.lobby.setGameList(g.pendingGames)
		g.pendingGames = nil
	}

	if ebiten.IsKeyPressed(ebiten.KeyControl) && inpututil.IsKeyJustPressed(ebiten.KeyP) {
		err := g.toggleProfiling()
		if err != nil {
			return err
		}
	}

	if !g.loggedIn {
		err := g.keyboard.Update()
		if err != nil {
			return fmt.Errorf("failed to update virtual keyboard: %s", err)
		}

		f := func() {
			var clearBuffer bool
			defer func() {
				if strings.ContainsRune(g.userInput, '\n') {
					g.userInput = strings.Split(g.userInput, "\n")[0]
					clearBuffer = true
				}
				if !g.usernameConfirmed {
					g.Username = g.userInput
				} else {
					g.Password = g.userInput
				}

				if clearBuffer {
					g.userInput = ""

					if !g.usernameConfirmed {
						g.usernameConfirmed = true
					} else if g.Password != "" {
						g.Connect()
					}
				}

				inputBuffer.SetText(g.userInput)
			}()

			if !g.shownKeyboard {
				g.keyboard.Show()
				g.shownKeyboard = true
			}

			if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(g.userInput) > 0 {
				g.userInput = g.userInput[:len(g.userInput)-1]
			}

			if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
				clearBuffer = true
			}

			g.runeBuffer = ebiten.AppendInputChars(g.runeBuffer[:0])
			if len(g.runeBuffer) > 0 {
				g.userInput += string(g.runeBuffer)
			}

			// Process on-screen keyboard input.
			g.keyboardInput = g.keyboard.AppendInput(g.keyboardInput[:0])
			for _, input := range g.keyboardInput {
				if input.Rune > 0 {
					g.userInput += string(input.Rune)
					continue
				}
				if input.Key == ebiten.KeyBackspace {
					if len(g.userInput) > 0 {
						g.userInput = g.userInput[:len(g.userInput)-1]
					}
				} else if input.Key == ebiten.KeyEnter {
					g.userInput += "\n"
				}
			}
		}

		f()
	}

	if ebiten.IsKeyPressed(ebiten.KeyControl) && inpututil.IsKeyJustPressed(ebiten.KeyD) {
		Debug++
		if Debug == 3 {
			Debug = 0
		}
		g.Board.debug = Debug
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		setViewBoard(!viewBoard)
	}

	if !viewBoard {
		g.lobby.update()
	} else {
		g.Board.update()

		err := inputBuffer.Update()
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(tableColor)

	// Log in screen
	if !g.loggedIn {
		g.keyboard.Draw(screen)

		const welcomeText = `Connect to bgammon.org
  To log in as a guest, enter a username (if you want) and
  do not enter a password.`
		debugBox := image.NewRGBA(image.Rect(0, 0, g.screenW, g.screenH))
		debugImg := ebiten.NewImageFromImage(debugBox)

		if !g.usernameConfirmed {
			ebitenutil.DebugPrint(debugImg, welcomeText+fmt.Sprintf("\n\nUsername: %s_", g.Username))
		} else {
			ebitenutil.DebugPrint(debugImg, welcomeText+fmt.Sprintf("\n\nPassword: %s_", strings.Repeat("*", len(g.Password))))
		}

		g.resetImageOptions()
		g.op.GeoM.Scale(2, 2)
		screen.DrawImage(debugImg, g.op)
		return
	}

	statusBuffer.Draw(screen)
	if !viewBoard { // Lobby
		g.lobby.draw(screen)
	} else { // Game board
		gameBuffer.Draw(screen)
		inputBuffer.Draw(screen)
		g.Board.draw(screen)
	}

	if Debug > 0 {
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

func (g *Game) updateStatusBufferPosition() {
	if viewBoard {
		statusBuffer.SetRect(statusBufferRect)
	} else {
		statusBuffer.SetRect(image.Rect(0, g.screenH-lobbyStatusBufferHeight, g.screenW, g.screenH))
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

	bufferWidth := text.BoundString(defaultFont(), strings.Repeat("A", bufferCharacterWidth)).Dx()
	if bufferWidth > int(float64(g.screenW)*maxStatusWidthRatio) {
		bufferWidth = int(float64(g.screenW) * maxStatusWidthRatio)
	}

	g.Board.fullHeight = true
	g.Board.setRect(0, 0, g.screenW-bufferWidth, g.screenH)

	availableWidth := g.screenW - (g.Board.innerW + int(g.Board.horizontalBorderSize*2))
	if availableWidth > bufferWidth {
		bufferWidth = availableWidth
		g.Board.setRect(0, 0, g.screenW-bufferWidth, g.screenH)
	}

	if g.Board.h > g.Board.w {
		g.Board.fullHeight = false
		g.Board.setRect(0, 0, g.Board.w, g.Board.w)
	}

	if g.screenW > 200 {
		statusBuffer.SetPadding(4)
		gameBuffer.SetPadding(4)
		inputBuffer.SetPadding(4)
	} else if g.screenW > 100 {
		statusBuffer.SetPadding(2)
		gameBuffer.SetPadding(2)
		inputBuffer.SetPadding(2)
	} else {
		statusBuffer.SetPadding(0)
		gameBuffer.SetPadding(0)
		inputBuffer.SetPadding(0)
	}

	bufferPadding := int(g.Board.horizontalBorderSize / 2)

	//inputBufferHeight := (gameBuffer.chatLineHeight * showGameBufferLines) + (gameBuffer.padding * 4)
	inputBufferHeight := 50

	g.lobby.buttonBarHeight = inputBufferHeight + int(float64(bufferPadding)*1.5)
	/*minLobbyWidth := text.BoundString(mediumFont, strings.Repeat("A", lobbyCharacterWidth)).Dx()
	if g.Board.w >= minLobbyWidth {
		g.lobby.fullscreen = false
		g.lobby.setRect(0, 0, g.Board.w, g.screenH-lobbyStatusBufferHeight)
	} else {*/
	g.lobby.fullscreen = true
	g.lobby.setRect(0, 0, g.screenW, g.screenH-lobbyStatusBufferHeight)
	/*}*/

	if true || availableWidth >= 150 { // TODO allow buffers to be repositioned
		statusBufferHeight := g.screenH - inputBufferHeight - bufferPadding*3

		x, y, w, h := (g.screenW-bufferWidth)+bufferPadding, bufferPadding, bufferWidth-(bufferPadding*2), statusBufferHeight
		statusBufferRect = image.Rect(x, y, x+w, y+h/2-bufferPadding/2)
		g.updateStatusBufferPosition()

		gameBuffer.SetRect(image.Rect(x, y+h/2+bufferPadding/2, x+w, y+h))

		inputBuffer.SetRect(image.Rect(x, g.screenH-bufferPadding-inputBufferHeight, x+w, g.screenH-bufferPadding))
	} /* else {
		// Clamp buffer position.
		bx, by := statusBuffer.x, statusBuffer.y
		var bw, bh int
		if statusBuffer.w == 0 && statusBuffer.h == 0 {
			// Set initial buffer position.
			bx = 0
			by = g.screenH / 2
			// Set initial buffer size.
			bw = g.screenW
			bh = g.screenH / 2
		} else {
			// Scale existing buffer size
			bx, by = bx*(outsideWidth/g.screenW), by*(outsideHeight/g.screenH)
			bw, bh = statusBuffer.w*(outsideWidth/g.screenW), statusBuffer.h*(outsideHeight/g.screenH)
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

		statusBuffer.docked = false
		statusBuffer.setRect(bx, by, bw, bh)
	}*/

	displayArea := 200
	g.keyboard.SetRect(0, displayArea, g.screenW, g.screenH-displayArea)
	return outsideWidth, outsideHeight
}

func (g *Game) resetImageOptions() {
	g.op.GeoM.Reset()
}

func (g *Game) acceptInput() bool {
	input := inputBuffer.Text()
	if len(input) == 0 {
		return true
	}

	if input[0] == '/' {
		input = input[1:]
	} else {
		l(fmt.Sprintf("<%s> %s", g.Client.Username, input))
		input = "say " + input
	}

	g.Client.Out <- []byte(input)
	return true
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
