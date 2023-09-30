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
	"code.rocketnine.space/tslocum/etk"
	"code.rocketnine.space/tslocum/kibodo"
	"code.rocketnine.space/tslocum/messeji"
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

	imgDice  *ebiten.Image
	imgDice1 *ebiten.Image
	imgDice2 *ebiten.Image
	imgDice3 *ebiten.Image
	imgDice4 *ebiten.Image
	imgDice5 *ebiten.Image
	imgDice6 *ebiten.Image

	smallFont  font.Face
	mediumFont font.Face
	monoFont   font.Face
	largeFont  font.Face
)

var (
	lightCheckerColor = color.RGBA{232, 211, 162, 255}
	darkCheckerColor  = color.RGBA{0, 0, 0, 255}
)

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

	diceSize int

	connectGrid    *etk.Grid
	createGameGrid *etk.Grid
	joinGameGrid   *etk.Grid

	connectFocusPassword bool
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

	resizeDice := func(img image.Image) *ebiten.Image {
		const maxSize = 70
		diceSize = width
		if diceSize > maxSize {
			diceSize = maxSize
		}
		return ebiten.NewImageFromImage(resize.Resize(uint(diceSize), 0, img, resize.Lanczos3))
	}

	const size = 184
	imgDice = ebiten.NewImageFromImage(loadImage("assets/dice.png"))
	imgDice1 = resizeDice(imgDice.SubImage(image.Rect(0, 0, size*1, size*1)))
	imgDice2 = resizeDice(imgDice.SubImage(image.Rect(size*1, 0, size*2, size*1)))
	imgDice3 = resizeDice(imgDice.SubImage(image.Rect(size*2, 0, size*3, size*1)))
	imgDice4 = resizeDice(imgDice.SubImage(image.Rect(0, size*1, size*1, size*2)))
	imgDice5 = resizeDice(imgDice.SubImage(image.Rect(size*1, size*1, size*2, size*2)))
	imgDice6 = resizeDice(imgDice.SubImage(image.Rect(size*2, size*1, size*3, size*2)))
}

func loadImage(assetPath string) image.Image {
	f, err := assetsFS.Open(assetPath)
	if err != nil {
		panic(err)
	}

	img, _, err := image.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	return img
}

func loadAsset(assetPath string, width int) *ebiten.Image {
	img := loadImage(assetPath)

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

func diceImage(roll int) *ebiten.Image {
	switch roll {
	case 1:
		return imgDice1
	case 2:
		return imgDice2
	case 3:
		return imgDice3
	case 4:
		return imgDice4
	case 5:
		return imgDice5
	case 6:
		return imgDice6
	default:
		log.Panicf("unknown dice roll: %d", roll)
		return nil
	}
}

func setViewBoard(view bool) {
	viewBoard = view
	inputBuffer.SetHandleKeyboard(viewBoard)
	if viewBoard {
		inputBuffer.SetSuffix("_")

		// Exit create game dialog, if open.
		game.lobby.showCreateGame = false
		game.lobby.createGameFocusPassword = false
		game.lobby.createGameName.Field.SetText("")
		game.lobby.createGamePassword.Field.SetText("")
		game.lobby.bufferDirty = true
		game.lobby.bufferButtonsDirty = true
	} else {
		inputBuffer.SetSuffix("")
	}
	game.updateStatusBufferPosition()
	ebiten.ScheduleFrame()
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

	loaded bool

	connectUsername *etk.Input
	connectPassword *etk.Input
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

	etk.Style.TextColorLight = triangleA
	etk.Style.TextColorDark = triangleA
	etk.Style.InputBgColor = color.RGBA{40, 24, 9, 255}

	{
		headerLabel := etk.NewText("Welcome to bgammon.org")
		nameLabel := etk.NewText("Username")
		passwordLabel := etk.NewText("Password")

		infoLabel := etk.NewText("To log in as a guest, enter a username (if you want) and do not enter a password. Press Enter to log in.")

		g.connectUsername = etk.NewInput("", "", func(text string) (handled bool) {
			return false
		})
		g.connectUsername.Field.SetHandleKeyboard(true)

		g.connectPassword = etk.NewInput("", "", func(text string) (handled bool) {
			return false
		})
		g.connectPassword.Field.SetHandleKeyboard(false)

		grid := etk.NewGrid()
		grid.SetColumnPadding(int(g.Board.horizontalBorderSize / 2))
		grid.SetRowPadding(20)
		grid.SetColumnSizes(10, 200)
		grid.SetRowSizes(60, 50, 50)
		grid.AddChildAt(headerLabel, 0, 0, 3, 1)
		grid.AddChildAt(nameLabel, 1, 1, 1, 1)
		grid.AddChildAt(g.connectUsername, 2, 1, 1, 1)
		grid.AddChildAt(passwordLabel, 1, 2, 1, 1)
		grid.AddChildAt(g.connectPassword, 2, 2, 1, 1)
		grid.AddChildAt(infoLabel, 0, 3, 3, 1)
		connectGrid = grid
	}

	{
		headerLabel := etk.NewText("Create match")
		nameLabel := etk.NewText("Name")
		passwordLabel := etk.NewText("Password")

		g.lobby.createGameName = etk.NewInput("", "", func(text string) (handled bool) {
			return false
		})
		g.lobby.createGameName.Field.SetHandleKeyboard(true)

		g.lobby.createGamePassword = etk.NewInput("", "", func(text string) (handled bool) {
			return false
		})
		g.lobby.createGamePassword.Field.SetHandleKeyboard(false)

		grid := etk.NewGrid()
		grid.SetColumnPadding(int(g.Board.horizontalBorderSize / 2))
		grid.SetRowPadding(20)
		grid.SetColumnSizes(10, 200)
		grid.SetRowSizes(60, 50, 50)
		grid.AddChildAt(headerLabel, 0, 0, 3, 1)
		grid.AddChildAt(nameLabel, 1, 1, 1, 1)
		grid.AddChildAt(g.lobby.createGameName, 2, 1, 1, 1)
		grid.AddChildAt(passwordLabel, 1, 2, 1, 1)
		grid.AddChildAt(g.lobby.createGamePassword, 2, 2, 1, 1)
		createGameGrid = grid
	}

	{
		g.lobby.joinGameLabel = etk.NewText("Join match")

		passwordLabel := etk.NewText("Password")

		g.lobby.joinGamePassword = etk.NewInput("", "", func(text string) (handled bool) {
			return false
		})
		g.lobby.joinGamePassword.Field.SetHandleKeyboard(true)

		grid := etk.NewGrid()
		grid.SetColumnPadding(int(g.Board.horizontalBorderSize / 2))
		grid.SetRowPadding(20)
		grid.SetColumnSizes(10, 200)
		grid.SetRowSizes(60, 50, 50)
		grid.AddChildAt(g.lobby.joinGameLabel, 0, 0, 3, 1)
		grid.AddChildAt(passwordLabel, 1, 1, 1, 1)
		grid.AddChildAt(g.lobby.joinGamePassword, 2, 1, 1, 1)
		joinGameGrid = grid
	}

	etk.SetRoot(connectGrid)

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
			ebiten.ScheduleFrame()

			if ev.Player == g.Client.Username {
				gameBuffer.SetText("")
				gameLogged = false
			} else {
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
			if ev.Player == g.Client.Username {
				setViewBoard(false)
			}
			ebiten.ScheduleFrame()

			if ev.Player != g.Client.Username {
				lg(fmt.Sprintf("%s left the match.", ev.Player))
			}
		case *bgammon.EventBoard:
			g.Board.gameState = &ev.GameState
			g.Board.ProcessState()
			setViewBoard(true)
			ebiten.ScheduleFrame()
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
			ebiten.ScheduleFrame()
			lg(fmt.Sprintf("%s rolled %s.", ev.Player, diceFormatted))
		case *bgammon.EventFailedRoll:
			l(fmt.Sprintf("*** Failed to roll: %s", ev.Reason))
		case *bgammon.EventMoved:
			lg(fmt.Sprintf("%s moved %s.", ev.Player, bgammon.FormatMoves(ev.Moves)))
			if ev.Player == g.Client.Username {
				continue
			}
			for _, move := range ev.Moves {
				g.Board.movePiece(move[0], move[1])
			}
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

	g.Username = ""
	g.Password = ""

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

	if !g.loaded {
		g.loaded = true

		// Auto-connect
		if g.Username != "" || g.Password != "" {
			g.Connect()
		}
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
		lastFocus := connectFocusPassword

		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			x, y := ebiten.CursorPosition()
			p := image.Point{x, y}
			if p.In(g.connectUsername.Rect()) {
				connectFocusPassword = false
			} else if p.In(g.connectPassword.Rect()) {
				connectFocusPassword = true
			}
		}

		if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
			connectFocusPassword = !connectFocusPassword
		}

		if connectFocusPassword != lastFocus {
			if connectFocusPassword {
				g.connectUsername.Field.SetHandleKeyboard(false)
				g.connectUsername.Field.SetSuffix("")
				g.connectPassword.Field.SetHandleKeyboard(true)
				g.connectPassword.Field.SetSuffix("_")
			} else {
				g.connectUsername.Field.SetHandleKeyboard(true)
				g.connectUsername.Field.SetSuffix("_")
				g.connectPassword.Field.SetHandleKeyboard(false)
				g.connectPassword.Field.SetSuffix("")
			}
		}

		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyKPEnter) {
			g.Username = g.connectUsername.Text()
			g.Password = g.connectPassword.Text()
			g.Connect()
		}

		err := etk.Update()
		if err != nil {
			log.Fatal(err)
		}
		return nil
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
		ebiten.ScheduleFrame()
	}

	if !viewBoard {
		g.lobby.update()

		if g.lobby.showCreateGame || g.lobby.showJoinGame {
			if g.lobby.showCreateGame {
				if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
					x, y := ebiten.CursorPosition()
					p := image.Point{x, y}
					if p.In(g.lobby.createGameName.Rect()) {
						g.lobby.createGameFocusPassword = false
						g.lobby.createGameName.Field.SetHandleKeyboard(true)
						g.lobby.createGameName.Field.SetSuffix("_")
						g.lobby.createGamePassword.Field.SetHandleKeyboard(false)
						g.lobby.createGamePassword.Field.SetSuffix("")
					} else if p.In(g.lobby.createGamePassword.Rect()) {
						g.lobby.createGameFocusPassword = true
						g.lobby.createGameName.Field.SetHandleKeyboard(false)
						g.lobby.createGameName.Field.SetSuffix("")
						g.lobby.createGamePassword.Field.SetHandleKeyboard(true)
						g.lobby.createGamePassword.Field.SetSuffix("_")
					}
				}

				if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
					g.lobby.createGameFocusPassword = !g.lobby.createGameFocusPassword
					if g.lobby.createGameFocusPassword {
						g.lobby.createGameName.Field.SetHandleKeyboard(false)
						g.lobby.createGameName.Field.SetSuffix("")
						g.lobby.createGamePassword.Field.SetHandleKeyboard(true)
						g.lobby.createGamePassword.Field.SetSuffix("_")
					} else {
						g.lobby.createGameName.Field.SetHandleKeyboard(true)
						g.lobby.createGameName.Field.SetSuffix("_")
						g.lobby.createGamePassword.Field.SetHandleKeyboard(false)
						g.lobby.createGamePassword.Field.SetSuffix("")
					}
				}
			}

			err := etk.Update()
			if err != nil {
				log.Fatal(err)
			}
		}
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
		err := etk.Draw(screen)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	statusBuffer.Draw(screen)
	if !viewBoard { // Lobby
		g.lobby.draw(screen)

		if g.lobby.showCreateGame || g.lobby.showJoinGame {
			err := etk.Draw(screen)
			if err != nil {
				log.Fatal(err)
			}
		}
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

	inputBufferHeight := 50

	g.lobby.buttonBarHeight = inputBufferHeight + int(float64(bufferPadding)*1.5)
	g.lobby.fullscreen = true
	g.lobby.setRect(0, 0, g.screenW, g.screenH-lobbyStatusBufferHeight)

	statusBufferHeight := g.screenH - inputBufferHeight - bufferPadding*3

	x, y, w, h := (g.screenW-bufferWidth)+bufferPadding, bufferPadding, bufferWidth-(bufferPadding*2), statusBufferHeight
	statusBufferRect = image.Rect(x, y, x+w, y+h/2-bufferPadding/2)
	g.updateStatusBufferPosition()

	gameBuffer.SetRect(image.Rect(x, y+h/2+bufferPadding/2, x+w, y+h))

	inputBuffer.SetRect(image.Rect(x, g.screenH-bufferPadding-inputBufferHeight, x+w, g.screenH-bufferPadding))

	displayArea := 200
	g.keyboard.SetRect(0, displayArea, g.screenW, g.screenH-displayArea)

	etk.Layout(g.screenW, g.screenH-lobbyStatusBufferHeight-g.lobby.buttonBarHeight)

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
	os.Exit(0)
}
