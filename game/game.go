package game

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"path"
	"regexp"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"time"

	"code.rocket9labs.com/tslocum/bgammon"
	"code.rocket9labs.com/tslocum/bgammon-bei-bot/bot"
	"code.rocket9labs.com/tslocum/bgammon/pkg/server"
	"code.rocket9labs.com/tslocum/etk"
	"code.rocket9labs.com/tslocum/etk/kibodo"
	"code.rocket9labs.com/tslocum/tabula"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/leonelquinteros/gotext"
	"github.com/nfnt/resize"
	"golang.org/x/image/font/opentype"
	"golang.org/x/text/language"
)

const (
	version              = "v1.3.6"
	baseButtonHeight     = 54
	MaxDebug             = 2
	DefaultServerAddress = "wss://ws.bgammon.org"
)

var AutoEnableTouchInput bool

var AppLanguage = "en"

var (
	anyNumbers  = regexp.MustCompile(`[0-9]+`)
	onlyNumbers = regexp.MustCompile(`^[0-9]+$`)
)

//go:embed asset locales
var assetFS embed.FS

var (
	imgCheckerTop  *ebiten.Image
	imgCheckerSide *ebiten.Image

	imgDice  *ebiten.Image
	imgDice1 *ebiten.Image
	imgDice2 *ebiten.Image
	imgDice3 *ebiten.Image
	imgDice4 *ebiten.Image
	imgDice5 *ebiten.Image
	imgDice6 *ebiten.Image

	imgCubes   *ebiten.Image
	imgCubes2  *ebiten.Image
	imgCubes4  *ebiten.Image
	imgCubes8  *ebiten.Image
	imgCubes16 *ebiten.Image
	imgCubes32 *ebiten.Image
	imgCubes64 *ebiten.Image

	fontMutex = &sync.Mutex{}
)

var (
	lightCheckerColor = color.RGBA{232, 211, 162, 255}
	darkCheckerColor  = color.RGBA{0, 0, 0, 255}
)

const maxStatusWidthRatio = 0.5

const bufferCharacterWidth = 21

const (
	minWidth  = 320
	minHeight = 240
)

var (
	extraSmallFontSize  = 14
	smallFontSize       = 20
	mediumFontSize      = 24
	mediumLargeFontSize = 32
	largeFontSize       = 36
)

var (
	bufferTextColor       = triangleALight
	bufferBackgroundColor = color.RGBA{40, 24, 9, 255}
)

var (
	statusBuffer *etk.Text
	gameBuffer   *etk.Text
	inputBuffer  *Input

	statusLogged bool
	gameLogged   bool

	lobbyStatusBufferHeight = 75

	Debug int8

	game *Game

	diceSize int

	connectGrid    *etk.Grid
	registerGrid   *etk.Grid
	resetGrid      *etk.Grid
	createGameGrid *etk.Grid
	joinGameGrid   *etk.Grid

	createGameContainer *etk.Grid
	joinGameContainer   *etk.Grid
	historyContainer    *etk.Grid
	listGamesContainer  *etk.Grid

	displayFrame    *etk.Frame
	connectFrame    *etk.Frame
	createGameFrame *etk.Frame
	joinGameFrame   *etk.Frame
	historyFrame    *etk.Frame
	listGamesFrame  *etk.Frame
)

const sampleRate = 44100

var (
	audioContext *audio.Context

	SoundDie1, SoundDie2, SoundDie3                []byte
	SoundDice1, SoundDice2, SoundDice3, SoundDice4 []byte
	SoundMove1, SoundMove2, SoundMove3             []byte
	SoundJoinLeave                                 []byte
	SoundSay                                       []byte
)

func init() {
	gotext.SetDomain("boxcars")
}

func l(s string) {
	m := time.Now().Format("3:04") + " " + s
	if statusLogged {
		_, _ = statusBuffer.Write([]byte("\n" + m))
		scheduleFrame()
		return
	}
	_, _ = statusBuffer.Write([]byte(m))
	statusLogged = true
	scheduleFrame()
}

func lg(s string) {
	m := time.Now().Format("3:04") + " " + s
	if gameLogged {
		_, _ = gameBuffer.Write([]byte("\n" + m))
		scheduleFrame()
		return
	}
	_, _ = gameBuffer.Write([]byte(m))
	gameLogged = true
	scheduleFrame()
}

var (
	loadedCheckerWidth = -1
	diceImageSize      = 0.0
	cubesImageSize     = 0.0
)

func loadImageAssets(width int) {
	if width == loadedCheckerWidth {
		return
	}
	loadedCheckerWidth = width

	imgCheckerTop = loadAsset("asset/image/checker_top.png", width)
	imgCheckerSide = loadAsset("asset/image/checker_side.png", width)

	resizeDice := func(img image.Image, scale float64) *ebiten.Image {
		if game == nil {
			panic("nil game")
		}

		maxSize := etk.Scale(100)
		if maxSize > game.screenW/10 {
			maxSize = game.screenW / 10
		}
		if maxSize > game.screenH/10 {
			maxSize = game.screenH / 10
		}

		diceSize = etk.Scale(width)
		if diceSize > maxSize {
			diceSize = maxSize
		}
		if scale == 1 {
			diceImageSize = float64(diceSize)
		} else {
			cubesImageSize = float64(diceSize) * scale
		}
		return ebiten.NewImageFromImage(resize.Resize(uint(float64(diceSize)*scale), 0, img, resize.Lanczos3))
	}

	const size = 184
	imgDice = ebiten.NewImageFromImage(loadImage("asset/image/dice.png"))
	imgDice1 = resizeDice(imgDice.SubImage(image.Rect(0, 0, size*1, size*1)), 1)
	imgDice2 = resizeDice(imgDice.SubImage(image.Rect(size*1, 0, size*2, size*1)), 1)
	imgDice3 = resizeDice(imgDice.SubImage(image.Rect(size*2, 0, size*3, size*1)), 1)
	imgDice4 = resizeDice(imgDice.SubImage(image.Rect(0, size*1, size*1, size*2)), 1)
	imgDice5 = resizeDice(imgDice.SubImage(image.Rect(size*1, size*1, size*2, size*2)), 1)
	imgDice6 = resizeDice(imgDice.SubImage(image.Rect(size*2, size*1, size*3, size*2)), 1)
	imgCubes = ebiten.NewImageFromImage(loadImage("asset/image/cubes.png"))
	imgCubes2 = resizeDice(imgCubes.SubImage(image.Rect(0, 0, size*1, size*1)), 0.6)
	imgCubes4 = resizeDice(imgCubes.SubImage(image.Rect(size*1, 0, size*2, size*1)), 0.6)
	imgCubes8 = resizeDice(imgCubes.SubImage(image.Rect(size*2, 0, size*3, size*1)), 0.6)
	imgCubes16 = resizeDice(imgCubes.SubImage(image.Rect(0, size*1, size*1, size*2)), 0.6)
	imgCubes32 = resizeDice(imgCubes.SubImage(image.Rect(size*1, size*1, size*2, size*2)), 0.6)
	imgCubes64 = resizeDice(imgCubes.SubImage(image.Rect(size*2, size*1, size*3, size*2)), 0.6)
}

func loadAudioAssets() {
	audioContext = audio.NewContext(sampleRate)
	p := "asset/audio/"

	SoundDie1 = LoadBytes(p + "die1.ogg")
	SoundDie2 = LoadBytes(p + "die2.ogg")
	SoundDie3 = LoadBytes(p + "die3.ogg")

	SoundDice1 = LoadBytes(p + "dice1.ogg")
	SoundDice2 = LoadBytes(p + "dice2.ogg")
	SoundDice3 = LoadBytes(p + "dice3.ogg")
	SoundDice4 = LoadBytes(p + "dice4.ogg")

	SoundMove1 = LoadBytes(p + "move1.ogg")
	SoundMove2 = LoadBytes(p + "move2.ogg")
	SoundMove3 = LoadBytes(p + "move3.ogg")

	SoundJoinLeave = LoadBytes(p + "joinleave.ogg")
	SoundSay = LoadBytes(p + "say.ogg")

	dieSounds = [][]byte{
		SoundDie1,
		SoundDie2,
		SoundDie3,
	}
	randomizeByteSlice(dieSounds)

	diceSounds = [][]byte{
		SoundDice1,
		SoundDice2,
		SoundDice3,
		SoundDice4,
	}
	randomizeByteSlice(diceSounds)

	moveSounds = [][]byte{
		SoundMove1,
		SoundMove2,
		SoundMove3,
	}
	randomizeByteSlice(moveSounds)
}

func loadImage(assetPath string) image.Image {
	f, err := assetFS.Open(assetPath)
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

func LoadBytes(p string) []byte {
	b, err := assetFS.ReadFile(p)
	if err != nil {
		panic(err)
	}
	return b
}

func LoadWAV(context *audio.Context, p string) *audio.Player {
	f, err := assetFS.Open(p)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	stream, err := wav.DecodeWithSampleRate(sampleRate, f)
	if err != nil {
		panic(err)
	}

	player, err := audioContext.NewPlayer(io.NopCloser(stream))
	if err != nil {
		panic(err)
	}

	// Workaround to prevent delays when playing for the first time.
	player.SetVolume(0)
	player.Play()
	player.Pause()
	player.Rewind()
	player.SetVolume(1)

	return player
}

type oggStream struct {
	*vorbis.Stream
}

func (s *oggStream) Close() error {
	return nil
}

func LoadOGG(context *audio.Context, p string) *audio.Player {
	b := LoadBytes(p)

	stream, err := vorbis.DecodeWithSampleRate(sampleRate, bytes.NewReader(b))
	if err != nil {
		panic(err)
	}

	player, err := audioContext.NewPlayer(&oggStream{Stream: stream})
	if err != nil {
		panic(err)
	}

	return player
}

func diceImage(roll int8) *ebiten.Image {
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

func cubeImage(value int8) *ebiten.Image {
	switch value {
	case 2:
		return imgCubes2
	case 4:
		return imgCubes4
	case 8:
		return imgCubes8
	case 16:
		return imgCubes16
	case 32:
		return imgCubes32
	default:
		return imgCubes64
	}
}

func setViewBoard(view bool) {
	if view != viewBoard {
		go hideKeyboard()
		inputBuffer.SetText("")
	}

	var refreshLobby bool
	if !view && viewBoard != view {
		refreshLobby = true
	}
	viewBoard = view

	switch {
	case game.needLayoutConnect && !game.loggedIn:
		game.layoutConnect()
	case game.needLayoutLobby && game.loggedIn && !viewBoard:
		game.layoutLobby()
	case game.needLayoutBoard && game.loggedIn && viewBoard:
		game.layoutBoard()
	}

	game.Board.selectRollGrid.SetVisible(false)

	if viewBoard {
		// Exit dialogs.
		game.lobby.showJoinGame = false
		game.lobby.showCreateGame = false
		game.lobby.createGameName.SetText("")
		game.lobby.createGamePassword.SetText("")
		game.lobby.rebuildButtonsGrid()

		game.setRoot(game.Board.frame)
		etk.SetFocus(inputBuffer)

		game.Board.uiGrid.SetRect(game.Board.uiGrid.Rect())
	} else {
		if !game.loggedIn {
			game.setRoot(connectFrame)
		} else if game.lobby.showCreateGame {
			game.setRoot(createGameFrame)
		} else if game.lobby.showJoinGame {
			game.setRoot(joinGameFrame)
		} else if game.lobby.showHistory {
			game.setRoot(historyFrame)
		} else {
			game.setRoot(listGamesFrame)
		}

		game.Board.menuGrid.SetVisible(false)
		game.Board.settingsGrid.SetVisible(false)
		game.Board.selectSpeed.SetMenuVisible(false)
		game.Board.leaveGameGrid.SetVisible(false)

		statusBuffer.SetRect(statusBuffer.Rect())

		game.Board.playerRoll1, game.Board.playerRoll2, game.Board.playerRoll3 = 0, 0, 0
		game.Board.playerRollStale = false
		game.Board.opponentRoll1, game.Board.opponentRoll2, game.Board.opponentRoll3 = 0, 0, 0
		game.Board.opponentRollStale = false
	}

	if refreshLobby && game.Client != nil {
		game.Client.Out <- []byte("list")
	}

	scheduleFrame()
}

type Sprite struct {
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

var fieldHeight int

var spinner = []byte(`-\|/`)

var viewBoard bool // View board or lobby

var (
	drawScreen     int
	updatedGame    bool
	lastDraw       time.Time
	gameUpdateLock = &sync.Mutex{}
)

func scheduleFrame() {
	gameUpdateLock.Lock()
	defer gameUpdateLock.Unlock()

	updatedGame = false
	drawScreen = 2
}

type replayFrame struct {
	Game  *bgammon.Game
	Event []byte
}

type Game struct {
	screenW, screenH int

	drawBuffer bytes.Buffer
	drawTick   int

	spinnerIndex int

	ServerAddress string
	Email         string
	Username      string
	Password      string
	register      bool
	loggedIn      bool

	TV bool

	Client *Client

	Board *board

	lobby *lobby

	keyboard      *etk.Keyboard
	keyboardFrame *etk.Frame

	volume float64 // Volume range is 0-1.

	runeBuffer []rune

	debugImg *ebiten.Image

	cpuProfile *os.File

	connectUsername *Input
	connectPassword *Input
	connectServer   *Input

	registerEmail    *Input
	registerUsername *Input
	registerPassword *Input

	resetEmail      *Input
	resetInfo       *etk.Text
	resetInProgress bool

	tutorialFrame *etk.Frame

	pressedKeys  []ebiten.Key
	pressedRunes []rune

	cursorX, cursorY int

	rootWidget etk.Widget

	touchIDs []ebiten.TouchID

	lastRefresh time.Time

	forceLayout bool

	bufferWidth int

	connectGridY            int
	showConnectStatusBuffer bool

	needLayoutConnect bool
	needLayoutLobby   bool
	needLayoutBoard   bool

	LoadReplay []byte

	savedUsername string
	savedPassword string

	initialized bool
	loaded      bool

	showRegister bool
	showReset    bool

	downloadReplay int

	replay         bool
	replayData     []byte
	replayFrame    int
	replayFrames   []*replayFrame
	replaySummary1 []byte
	replaySummary2 []byte

	*sync.Mutex
}

func NewGame() *Game {
	ebiten.SetVsyncEnabled(false)
	ebiten.SetScreenClearedEveryFrame(false)
	ebiten.SetTPS(targetFPS)
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetWindowClosingHandled(true)

	g := &Game{
		keyboard:      etk.NewKeyboard(),
		keyboardFrame: etk.NewFrame(),

		runeBuffer: make([]rune, 24),

		tutorialFrame: etk.NewFrame(),

		debugImg: ebiten.NewImage(200, 200),
		volume:   1,

		Mutex: &sync.Mutex{},
	}
	g.keyboard.SetScheduleFrameFunc(scheduleFrame)
	g.keyboard.SetKeys(kibodo.KeysMobileQWERTY)
	g.keyboard.SetExtendedKeys(kibodo.KeysMobileSymbols)
	if !AutoEnableTouchInput {
		g.keyboard.SetVisible(false)
	}
	g.keyboardFrame.AddChild(g.keyboard)
	g.savedUsername, g.savedPassword = loadCredentials()
	g.tutorialFrame.SetPositionChildren(true)
	game = g

	return g
}

func (g *Game) initialize() {
	loadAudioAssets()
	loadImageAssets(0)

	if AutoEnableTouchInput {
		etk.Bindings.ConfirmRune = 199
		etk.Bindings.BackRune = 231

		etk.Style.BorderSize /= 2

		extraSmallFontSize /= 2
		smallFontSize /= 2
		mediumFontSize /= 2
		mediumLargeFontSize /= 2
		largeFontSize /= 2
	}

	fnt, err := opentype.Parse(fonts.MPlus1pRegular_ttf)
	if err != nil {
		log.Fatal(err)
	}

	etk.Style.TextFont = fnt
	etk.Style.TextSize = largeFontSize

	etk.Style.TextColorLight = triangleA
	etk.Style.TextColorDark = triangleA
	etk.Style.InputBgColor = color.RGBA{40, 24, 9, 255}

	etk.Style.ScrollAreaColor = color.RGBA{26, 15, 6, 255}
	etk.Style.ScrollHandleColor = color.RGBA{180, 154, 108, 255}

	etk.Style.ButtonTextColor = color.RGBA{0, 0, 0, 255}
	etk.Style.ButtonBgColor = color.RGBA{225, 188, 125, 255}

	etk.Style.BorderColorLeft = color.RGBA{233, 207, 170, 255}
	etk.Style.BorderColorTop = color.RGBA{233, 207, 170, 255}

	etk.Style.ScrollBorderColorLeft = color.RGBA{210, 182, 135, 255}
	etk.Style.ScrollBorderColorTop = color.RGBA{210, 182, 135, 255}

	statusBuffer = etk.NewText("")
	gameBuffer = etk.NewText("")
	inputBuffer = &Input{etk.NewInput("", acceptInput)}

	statusBuffer.SetForeground(bufferTextColor)
	statusBuffer.SetBackground(bufferBackgroundColor)

	gameBuffer.SetForeground(bufferTextColor)
	gameBuffer.SetBackground(bufferBackgroundColor)

	inputBuffer.SetForeground(bufferTextColor)
	inputBuffer.SetBackground(bufferBackgroundColor)
	inputBuffer.SetSuffix("")

	fieldHeight = etk.Scale(50)
	if AutoEnableTouchInput {
		fieldHeight = etk.Scale(32)
	}

	displayFrame = etk.NewFrame()
	displayFrame.SetPositionChildren(true)

	g.Board = NewBoard()
	g.lobby = NewLobby()

	xPadding := etk.Scale(10)
	yPadding := etk.Scale(20)
	labelWidth := etk.Scale(200)
	if AutoEnableTouchInput {
		xPadding = 0
		yPadding /= 2
		labelWidth /= 2
	}

	connectAddress := game.ServerAddress
	if connectAddress == "" {
		connectAddress = DefaultServerAddress
	}
	g.connectServer = &Input{etk.NewInput(connectAddress, func(text string) (handled bool) {
		g.selectConnect()
		return false
	})}

	{
		headerLabel := newCenteredText(gotext.Get("Register"))
		emailLabel := newCenteredText(gotext.Get("Email"))
		nameLabel := newCenteredText(gotext.Get("Username"))
		passwordLabel := newCenteredText(gotext.Get("Password"))
		serverLabel := newCenteredText(gotext.Get("Server"))

		g.registerEmail = &Input{etk.NewInput("", func(text string) (handled bool) {
			g.selectConfirmRegister()
			return false
		})}
		centerInput(g.registerEmail)

		g.registerUsername = &Input{etk.NewInput("", func(text string) (handled bool) {
			g.selectConfirmRegister()
			return false
		})}
		centerInput(g.registerUsername)

		g.registerPassword = &Input{etk.NewInput("", func(text string) (handled bool) {
			g.selectConfirmRegister()
			return false
		})}
		centerInput(g.registerPassword)
		g.registerPassword.SetMask('*')

		cancelButton := etk.NewButton(gotext.Get("Cancel"), func() error {
			g.selectCancel()
			return nil
		})

		submitButton := etk.NewButton(gotext.Get("Submit"), func() error {
			g.selectConfirmRegister()
			return nil
		})

		infoLabel := etk.NewText(gotext.Get("Please enter a valid email address, or it will not be possible to reset your password."))

		footerLabel := etk.NewText("Boxcars " + version)
		footerLabel.SetHorizontal(etk.AlignEnd)
		footerLabel.SetVertical(etk.AlignEnd)

		grid := etk.NewGrid()
		grid.SetColumnPadding(int(g.Board.horizontalBorderSize / 2))
		grid.SetRowPadding(yPadding)
		grid.SetColumnSizes(xPadding, labelWidth, -1, -1, xPadding)
		grid.AddChildAt(headerLabel, 0, 0, 4, 1)
		grid.AddChildAt(etk.NewBox(), 4, 0, 1, 1)
		grid.AddChildAt(emailLabel, 1, 1, 2, 1)
		grid.AddChildAt(g.registerEmail, 2, 1, 2, 1)
		grid.AddChildAt(nameLabel, 1, 2, 2, 1)
		grid.AddChildAt(g.registerUsername, 2, 2, 2, 1)
		grid.AddChildAt(passwordLabel, 1, 3, 2, 1)
		grid.AddChildAt(g.registerPassword, 2, 3, 2, 1)
		y := 4
		if ShowServerSettings {
			centerInput(g.connectServer)
			grid.AddChildAt(serverLabel, 1, y, 2, 1)
			grid.AddChildAt(g.connectServer, 2, y, 2, 1)
			y++
		}
		{
			subGrid := etk.NewGrid()
			subGrid.SetColumnSizes(-1, yPadding, -1)
			subGrid.AddChildAt(cancelButton, 0, 0, 1, 1)
			subGrid.AddChildAt(submitButton, 2, 0, 1, 1)
			grid.AddChildAt(subGrid, 1, y, 3, 1)
		}
		grid.AddChildAt(infoLabel, 1, y+1, 3, 1)
		grid.AddChildAt(footerLabel, 1, y+2, 3, 1)
		registerGrid = grid
	}

	{
		headerLabel := newCenteredText(gotext.Get("Reset Password"))
		emailLabel := newCenteredText(gotext.Get("Email"))
		serverLabel := newCenteredText(gotext.Get("Server"))

		g.resetEmail = &Input{etk.NewInput("", func(text string) (handled bool) {
			g.selectConfirmReset()
			return false
		})}
		centerInput(g.resetEmail)

		cancelButton := etk.NewButton(gotext.Get("Cancel"), func() error {
			g.selectCancel()
			return nil
		})

		submitButton := etk.NewButton(gotext.Get("Submit"), func() error {
			g.selectConfirmReset()
			return nil
		})

		g.resetInfo = etk.NewText("")

		footerLabel := etk.NewText("Boxcars " + version)
		footerLabel.SetHorizontal(etk.AlignEnd)
		footerLabel.SetVertical(etk.AlignEnd)

		grid := etk.NewGrid()
		grid.SetColumnPadding(int(g.Board.horizontalBorderSize / 2))
		grid.SetRowPadding(yPadding)
		grid.SetColumnSizes(xPadding, labelWidth, -1, -1, xPadding)
		grid.AddChildAt(headerLabel, 0, 0, 4, 1)
		grid.AddChildAt(etk.NewBox(), 4, 0, 1, 1)
		grid.AddChildAt(emailLabel, 1, 1, 2, 1)
		grid.AddChildAt(g.resetEmail, 2, 1, 2, 1)
		y := 2
		if ShowServerSettings {
			centerInput(g.connectServer)
			grid.AddChildAt(serverLabel, 1, y, 2, 1)
			grid.AddChildAt(g.connectServer, 2, y, 2, 1)
			y++
		}
		{
			subGrid := etk.NewGrid()
			subGrid.SetColumnSizes(-1, yPadding, -1)
			subGrid.AddChildAt(cancelButton, 0, 0, 1, 1)
			subGrid.AddChildAt(submitButton, 2, 0, 1, 1)
			grid.AddChildAt(subGrid, 1, y, 3, 1)
		}
		grid.AddChildAt(g.resetInfo, 1, y+1, 3, 1)
		grid.AddChildAt(footerLabel, 1, y+2, 3, 1)
		resetGrid = grid
	}

	{
		headerLabel := newCenteredText(gotext.Get("%s - Free Online Backgammon", "bgammon.org"))
		nameLabel := newCenteredText(gotext.Get("Username"))
		passwordLabel := newCenteredText(gotext.Get("Password"))
		serverLabel := newCenteredText(gotext.Get("Server"))
		if AutoEnableTouchInput {
			headerLabel.SetFont(etk.Style.TextFont, etk.Scale(mediumLargeFontSize))
			headerLabel.SetHorizontal(etk.AlignCenter)
		}

		infoLabel := etk.NewText(gotext.Get("To log in as a guest, enter a username (if you want) and do not enter a password."))

		footerLabel := etk.NewText("Boxcars " + version)
		footerLabel.SetHorizontal(etk.AlignEnd)
		footerLabel.SetVertical(etk.AlignEnd)

		g.connectUsername = &Input{etk.NewInput("", func(text string) (handled bool) {
			g.selectConnect()
			return false
		})}
		centerInput(g.connectUsername)

		g.connectPassword = &Input{etk.NewInput("", func(text string) (handled bool) {
			g.selectConnect()
			return false
		})}
		centerInput(g.connectPassword)
		g.connectPassword.SetMask('*')

		connectButton := etk.NewButton(gotext.Get("Connect"), func() error {
			g.selectConnect()
			return nil
		})

		registerButton := etk.NewButton(gotext.Get("Register"), g.selectRegister)

		resetButton := etk.NewButton(gotext.Get("Reset Password"), g.selectReset)

		offlineButton := etk.NewButton(gotext.Get("Play Offline"), func() error {
			g.playOffline()
			return nil
		})

		grid := etk.NewGrid()
		grid.SetColumnPadding(int(g.Board.horizontalBorderSize / 2))
		grid.SetRowPadding(yPadding)
		grid.SetColumnSizes(xPadding, labelWidth, -1, -1, xPadding)
		grid.AddChildAt(headerLabel, 0, 0, 4, 1)
		grid.AddChildAt(etk.NewBox(), 4, 0, 1, 1)
		grid.AddChildAt(nameLabel, 1, 1, 2, 1)
		grid.AddChildAt(g.connectUsername, 2, 1, 2, 1)
		grid.AddChildAt(passwordLabel, 1, 2, 2, 1)
		grid.AddChildAt(g.connectPassword, 2, 2, 2, 1)
		g.connectGridY = 3
		if ShowServerSettings {
			centerInput(g.connectServer)
			grid.AddChildAt(serverLabel, 1, g.connectGridY, 2, 1)
			grid.AddChildAt(g.connectServer, 2, g.connectGridY, 2, 1)
			g.connectGridY++
		}
		{
			subGrid := etk.NewGrid()
			subGrid.SetColumnSizes(-1, yPadding, -1)
			subGrid.SetRowSizes(-1, yPadding, -1)
			subGrid.AddChildAt(connectButton, 0, 0, 1, 1)
			subGrid.AddChildAt(registerButton, 2, 0, 1, 1)
			grid.AddChildAt(subGrid, 1, g.connectGridY, 3, 1)
		}
		{
			subGrid := etk.NewGrid()
			subGrid.SetColumnSizes(-1, yPadding, -1)
			subGrid.SetRowSizes(-1, yPadding, -1)
			subGrid.AddChildAt(resetButton, 0, 0, 1, 1)
			subGrid.AddChildAt(offlineButton, 2, 0, 1, 1)
			grid.AddChildAt(subGrid, 1, g.connectGridY+1, 3, 1)
		}
		grid.AddChildAt(infoLabel, 1, g.connectGridY+2, 3, 1)
		grid.AddChildAt(footerLabel, 1, g.connectGridY+3, 3, 1)
		connectGrid = grid

		connectFrame = etk.NewFrame(connectGrid)
		connectFrame.SetPositionChildren(true)
	}

	{
		headerLabel := newCenteredText(gotext.Get("Create new match"))
		nameLabel := newCenteredText(gotext.Get("Name"))
		pointsLabel := newCenteredText(gotext.Get("Points"))
		passwordLabel := newCenteredText(gotext.Get("Password"))
		variantLabel := newCenteredText(gotext.Get("Variant"))

		g.lobby.createGameName = &Input{etk.NewInput("", func(text string) (handled bool) {
			g.lobby.confirmCreateGame()
			return false
		})}
		centerInput(g.lobby.createGameName)

		g.lobby.createGamePoints = &Input{etk.NewInput("", func(text string) (handled bool) {
			g.lobby.confirmCreateGame()
			return false
		})}
		centerInput(g.lobby.createGamePoints)

		g.lobby.createGamePassword = &Input{etk.NewInput("", func(text string) (handled bool) {
			g.lobby.confirmCreateGame()
			return false
		})}
		centerInput(g.lobby.createGamePassword)
		g.lobby.createGamePassword.SetMask('*')

		g.lobby.createGameAceyCheckbox = etk.NewCheckbox(g.lobby.toggleVariantAcey)
		g.lobby.createGameAceyCheckbox.SetBorderColor(triangleA)
		g.lobby.createGameAceyCheckbox.SetCheckColor(triangleA)
		g.lobby.createGameAceyCheckbox.SetSelected(false)

		aceyDeuceyLabel := &ClickableText{
			Text: newCenteredText(gotext.Get("Acey-deucey")),
			onSelected: func() {
				g.lobby.createGameAceyCheckbox.SetSelected(!g.lobby.createGameAceyCheckbox.Selected())
				g.lobby.toggleVariantAcey()
			},
		}
		aceyDeuceyLabel.SetVertical(etk.AlignCenter)

		aceyDeuceyGrid := etk.NewGrid()
		aceyDeuceyGrid.SetColumnSizes(fieldHeight, xPadding, -1)
		aceyDeuceyGrid.SetRowSizes(fieldHeight, -1)
		aceyDeuceyGrid.AddChildAt(g.lobby.createGameAceyCheckbox, 0, 0, 1, 1)
		aceyDeuceyGrid.AddChildAt(aceyDeuceyLabel, 2, 0, 1, 1)

		g.lobby.createGameTabulaCheckbox = etk.NewCheckbox(g.lobby.toggleVariantTabula)
		g.lobby.createGameTabulaCheckbox.SetBorderColor(triangleA)
		g.lobby.createGameTabulaCheckbox.SetCheckColor(triangleA)
		g.lobby.createGameTabulaCheckbox.SetSelected(false)

		tabulaLabel := &ClickableText{
			Text: newCenteredText(gotext.Get("Tabula")),
			onSelected: func() {
				g.lobby.createGameTabulaCheckbox.SetSelected(!g.lobby.createGameTabulaCheckbox.Selected())
				g.lobby.toggleVariantTabula()
			},
		}
		tabulaLabel.SetVertical(etk.AlignCenter)

		tabulaGrid := etk.NewGrid()
		tabulaGrid.SetColumnSizes(fieldHeight, xPadding, -1)
		tabulaGrid.SetRowSizes(fieldHeight, -1)
		tabulaGrid.AddChildAt(g.lobby.createGameTabulaCheckbox, 0, 0, 1, 1)
		tabulaGrid.AddChildAt(tabulaLabel, 2, 0, 1, 1)

		variantPadding := 20
		variantWidth := 400
		if AutoEnableTouchInput {
			variantPadding = 30
			variantWidth = 500
		}
		variantFlex := etk.NewFlex()
		variantFlex.SetGaps(0, variantPadding)
		variantFlex.SetChildSize(variantWidth, fieldHeight)
		variantFlex.AddChild(aceyDeuceyGrid)
		variantFlex.AddChild(tabulaGrid)

		grid := etk.NewGrid()
		grid.SetColumnPadding(int(g.Board.horizontalBorderSize / 2))
		grid.SetRowPadding(yPadding)
		grid.SetColumnSizes(xPadding, labelWidth, -1, xPadding)
		grid.SetRowSizes(60, fieldHeight, fieldHeight, fieldHeight, fieldHeight)
		grid.AddChildAt(headerLabel, 0, 0, 3, 1)
		grid.AddChildAt(etk.NewBox(), 3, 0, 1, 1)
		grid.AddChildAt(nameLabel, 1, 1, 1, 1)
		grid.AddChildAt(g.lobby.createGameName, 2, 1, 1, 1)
		grid.AddChildAt(pointsLabel, 1, 2, 1, 1)
		grid.AddChildAt(g.lobby.createGamePoints, 2, 2, 1, 1)
		grid.AddChildAt(passwordLabel, 1, 3, 1, 1)
		grid.AddChildAt(g.lobby.createGamePassword, 2, 3, 1, 1)
		grid.AddChildAt(variantLabel, 1, 4, 1, 1)
		grid.AddChildAt(variantFlex, 2, 4, 1, 1)
		grid.AddChildAt(etk.NewBox(), 0, 5, 1, 1)
		createGameGrid = grid

		createGameContainer = etk.NewGrid()
		createGameContainer.AddChildAt(createGameGrid, 0, 0, 1, 1)
		createGameContainer.AddChildAt(statusBuffer, 0, 1, 1, 1)
		createGameContainer.AddChildAt(g.lobby.buttonsGrid, 0, 2, 1, 1)

		createGameFrame = etk.NewFrame()
		createGameFrame.SetPositionChildren(true)
		createGameFrame.AddChild(createGameContainer)
		createGameFrame.AddChild(g.tutorialFrame)
	}

	{
		g.lobby.joinGameLabel = newCenteredText(gotext.Get("Join match"))

		passwordLabel := newCenteredText(gotext.Get("Password"))

		g.lobby.joinGamePassword = &Input{etk.NewInput("", func(text string) (handled bool) {
			g.lobby.confirmJoinGame()
			return false
		})}
		centerInput(g.lobby.joinGamePassword)
		g.lobby.joinGamePassword.SetMask('*')

		grid := etk.NewGrid()
		grid.SetColumnPadding(int(g.Board.horizontalBorderSize / 2))
		grid.SetRowPadding(yPadding)
		grid.SetColumnSizes(xPadding, labelWidth, -1, xPadding)
		grid.SetRowSizes(60, fieldHeight, fieldHeight)
		grid.AddChildAt(g.lobby.joinGameLabel, 0, 0, 3, 1)
		grid.AddChildAt(etk.NewBox(), 3, 0, 1, 1)
		grid.AddChildAt(passwordLabel, 1, 1, 1, 1)
		grid.AddChildAt(g.lobby.joinGamePassword, 2, 1, 1, 1)
		joinGameGrid = grid

		joinGameContainer = etk.NewGrid()
		joinGameContainer.AddChildAt(joinGameGrid, 0, 0, 1, 1)
		joinGameContainer.AddChildAt(statusBuffer, 0, 1, 1, 1)
		joinGameContainer.AddChildAt(g.lobby.buttonsGrid, 0, 2, 1, 1)

		joinGameFrame = etk.NewFrame()
		joinGameFrame.SetPositionChildren(true)
		joinGameFrame.AddChild(joinGameContainer)
		joinGameFrame.AddChild(g.tutorialFrame)
	}

	{
		historyFrame = etk.NewFrame()

		g.lobby.rebuildButtonsGrid()

		dividerLine := etk.NewBox()
		dividerLine.SetBackground(bufferTextColor)

		dateLabel := newCenteredText(gotext.Get("Date"))
		dateLabel.SetFollow(false)
		dateLabel.SetScrollBarVisible(false)
		resultLabel := newCenteredText(gotext.Get("Result"))
		resultLabel.SetFollow(false)
		resultLabel.SetScrollBarVisible(false)
		opponentLabel := newCenteredText(gotext.Get("Opponent"))
		opponentLabel.SetFollow(false)
		opponentLabel.SetScrollBarVisible(false)
		if AutoEnableTouchInput {
			dateLabel.SetFont(etk.Style.TextFont, etk.Scale(mediumFontSize))
			resultLabel.SetFont(etk.Style.TextFont, etk.Scale(mediumFontSize))
			opponentLabel.SetFont(etk.Style.TextFont, etk.Scale(mediumFontSize))
		}

		g.lobby.historyUsername = &Input{etk.NewInput("", func(text string) (handled bool) {
			g.selectHistorySearch()
			return false
		})}
		centerInput(g.lobby.historyUsername)
		g.lobby.historyUsername.SetScrollBarVisible(false)

		searchButton := etk.NewButton(gotext.Get("Search"), g.selectHistorySearch)

		indentA, indentB := etk.Scale(lobbyIndentA), etk.Scale(lobbyIndentB)

		historyItemHeight := game.itemHeight()
		if AutoEnableTouchInput {
			historyItemHeight /= 2
		}
		g.lobby.historyList = etk.NewList(historyItemHeight, g.lobby.selectHistory)
		g.lobby.historyList.SetColumnSizes(int(float64(indentA)*1.25), int(float64(indentB)*1.25)-int(float64(indentA)*1.25), -1)
		g.lobby.historyList.SetHighlightColor(color.RGBA{79, 55, 30, 255})

		headerGrid := etk.NewGrid()
		headerGrid.SetColumnSizes(int(float64(indentA)*1.25), int(float64(indentB)*1.25)-int(float64(indentA)*1.25), -1, 400, 200)
		headerGrid.AddChildAt(dateLabel, 0, 0, 1, 1)
		headerGrid.AddChildAt(resultLabel, 1, 0, 1, 1)
		headerGrid.AddChildAt(opponentLabel, 2, 0, 1, 1)
		headerGrid.AddChildAt(g.lobby.historyUsername, 3, 0, 1, 1)
		headerGrid.AddChildAt(searchButton, 4, 0, 1, 1)

		newLabel := func(text string, horizontal etk.Alignment) *etk.Text {
			t := etk.NewText(text)
			t.SetVertical(etk.AlignCenter)
			t.SetHorizontal(horizontal)
			return t
		}

		g.lobby.historyRatingCasualBackgammonSingle = newLabel("...", etk.AlignStart)
		g.lobby.historyRatingCasualBackgammonMulti = newLabel("...", etk.AlignStart)
		g.lobby.historyRatingCasualAceySingle = newLabel("...", etk.AlignStart)
		g.lobby.historyRatingCasualAceyMulti = newLabel("...", etk.AlignStart)
		g.lobby.historyRatingCasualTabulaSingle = newLabel("...", etk.AlignStart)
		g.lobby.historyRatingCasualTabulaMulti = newLabel("...", etk.AlignStart)

		ratingGrid := func(singleLabel *etk.Text, multiLabel *etk.Text) *etk.Grid {
			const dividerSize = 10
			g := etk.NewGrid()
			g.SetColumnSizes(-1, dividerSize, -1)
			g.AddChildAt(newLabel(gotext.Get("Single"), etk.AlignEnd), 0, 0, 1, 1)
			g.AddChildAt(singleLabel, 2, 0, 1, 1)
			g.AddChildAt(newLabel(gotext.Get("Multi"), etk.AlignEnd), 0, 1, 1, 1)
			g.AddChildAt(multiLabel, 2, 1, 1, 1)
			return g
		}

		historyDividerLine := etk.NewBox()
		historyDividerLine.SetBackground(bufferTextColor)

		g.lobby.historyPageLabel = newLabel("...", etk.AlignCenter)

		pageControlGrid := etk.NewGrid()
		pageControlGrid.AddChildAt(etk.NewButton("<", g.selectHistoryPrevious), 0, 0, 1, 1)
		pageControlGrid.AddChildAt(g.lobby.historyPageLabel, 1, 0, 1, 1)
		pageControlGrid.AddChildAt(etk.NewButton(">", g.selectHistoryNext), 2, 0, 1, 1)

		historyRatingGrid := etk.NewGrid()
		historyRatingGrid.SetRowSizes(2, -1, -1, -1)
		historyRatingGrid.AddChildAt(historyDividerLine, 0, 0, 3, 1)
		historyRatingGrid.AddChildAt(newLabel(gotext.Get("Backgammon"), etk.AlignCenter), 0, 1, 1, 1)
		historyRatingGrid.AddChildAt(ratingGrid(g.lobby.historyRatingCasualBackgammonSingle, g.lobby.historyRatingCasualBackgammonMulti), 0, 2, 1, 2)
		historyRatingGrid.AddChildAt(newLabel(gotext.Get("Acey-deucey"), etk.AlignCenter), 1, 1, 1, 1)
		historyRatingGrid.AddChildAt(ratingGrid(g.lobby.historyRatingCasualAceySingle, g.lobby.historyRatingCasualAceyMulti), 1, 2, 1, 2)
		historyRatingGrid.AddChildAt(newLabel(gotext.Get("Tabula"), etk.AlignCenter), 2, 1, 1, 1)
		historyRatingGrid.AddChildAt(ratingGrid(g.lobby.historyRatingCasualTabulaSingle, g.lobby.historyRatingCasualTabulaMulti), 2, 2, 1, 2)

		historyContainer = etk.NewGrid()
		historyContainer.AddChildAt(headerGrid, 0, 0, 1, 1)
		historyContainer.AddChildAt(dividerLine, 0, 1, 1, 1)
		historyContainer.AddChildAt(g.lobby.historyList, 0, 2, 1, 1)
		historyContainer.AddChildAt(pageControlGrid, 0, 3, 1, 1)
		historyContainer.AddChildAt(historyRatingGrid, 0, 4, 1, 1)
		historyContainer.AddChildAt(statusBuffer, 0, 5, 1, 1)
		historyContainer.AddChildAt(g.lobby.buttonsGrid, 0, 6, 1, 1)

		historyFrame.SetPositionChildren(true)
		historyFrame.AddChild(historyContainer)
	}

	{
		listGamesFrame = etk.NewFrame()

		g.lobby.rebuildButtonsGrid()

		dividerLine := etk.NewBox()
		dividerLine.SetBackground(triangleA)

		statusLabel := newCenteredText(gotext.Get("Status"))
		statusLabel.SetFollow(false)
		statusLabel.SetScrollBarVisible(false)
		ratingLabel := newCenteredText(gotext.Get("Rating"))
		ratingLabel.SetFollow(false)
		ratingLabel.SetScrollBarVisible(false)
		pointsLabel := newCenteredText(gotext.Get("Points"))
		pointsLabel.SetFollow(false)
		pointsLabel.SetScrollBarVisible(false)
		nameLabel := newCenteredText(gotext.Get("Match Name"))
		nameLabel.SetFollow(false)
		nameLabel.SetScrollBarVisible(false)
		if AutoEnableTouchInput {
			statusLabel.SetFont(etk.Style.TextFont, etk.Scale(mediumFontSize))
			ratingLabel.SetFont(etk.Style.TextFont, etk.Scale(mediumFontSize))
			pointsLabel.SetFont(etk.Style.TextFont, etk.Scale(mediumFontSize))
			nameLabel.SetFont(etk.Style.TextFont, etk.Scale(mediumFontSize))
		}

		g.lobby.historyButton = etk.NewButton(gotext.Get("History"), game.selectHistory)
		g.lobby.historyButton.SetVisible(false)

		indentA, indentB := etk.Scale(lobbyIndentA), etk.Scale(lobbyIndentB)

		headerGrid := etk.NewGrid()
		headerGrid.SetColumnSizes(indentA, indentB-indentA, indentB-indentA, -1, 300)
		headerGrid.AddChildAt(statusLabel, 0, 0, 1, 1)
		headerGrid.AddChildAt(ratingLabel, 1, 0, 1, 1)
		headerGrid.AddChildAt(pointsLabel, 2, 0, 1, 1)
		headerGrid.AddChildAt(nameLabel, 3, 0, 1, 1)
		headerGrid.AddChildAt(g.lobby.historyButton, 4, 0, 1, 1)

		listGamesContainer = etk.NewGrid()
		listGamesContainer.AddChildAt(headerGrid, 0, 0, 1, 1)
		listGamesContainer.AddChildAt(dividerLine, 0, 1, 1, 1)
		listGamesContainer.AddChildAt(g.lobby.availableMatchesList, 0, 2, 1, 1)
		listGamesContainer.AddChildAt(statusBuffer, 0, 3, 1, 1)
		listGamesContainer.AddChildAt(g.lobby.buttonsGrid, 0, 4, 1, 1)

		listGamesFrame.SetPositionChildren(true)
		listGamesFrame.AddChild(listGamesContainer)
		listGamesFrame.AddChild(g.tutorialFrame)
	}

	statusBuffer.SetScrollBarColors(etk.Style.ScrollAreaColor, etk.Style.ScrollHandleColor)
	gameBuffer.SetScrollBarColors(etk.Style.ScrollAreaColor, etk.Style.ScrollHandleColor)
	inputBuffer.SetScrollBarColors(etk.Style.ScrollAreaColor, etk.Style.ScrollHandleColor)
	g.lobby.availableMatchesList.SetScrollBarColors(etk.Style.ScrollAreaColor, etk.Style.ScrollHandleColor)
	g.lobby.historyList.SetScrollBarColors(etk.Style.ScrollAreaColor, etk.Style.ScrollHandleColor)

	{
		scrollBarWidth := etk.Scale(32)
		statusBuffer.SetScrollBarWidth(scrollBarWidth)
		gameBuffer.SetScrollBarWidth(scrollBarWidth)
		inputBuffer.SetScrollBarWidth(scrollBarWidth)
		g.lobby.availableMatchesList.SetScrollBarWidth(scrollBarWidth)
		g.lobby.historyList.SetScrollBarWidth(scrollBarWidth)
	}

	g.needLayoutConnect = true
	g.needLayoutLobby = true
	g.needLayoutBoard = true

	g.setRoot(connectFrame)

	if g.savedUsername != "" {
		g.connectUsername.SetText(g.savedUsername)
		g.connectPassword.SetText(g.savedPassword)
		etk.SetFocus(g.connectPassword)
	} else {
		etk.SetFocus(g.connectUsername)
	}

	go g.handleAutoRefresh()
	go g.handleUpdateTimeLabels()

	etk.SetRoot(displayFrame)
	scheduleFrame()
}

func (g *Game) playOffline() {
	go hideKeyboard()
	if g.loggedIn {
		return
	}

	// Start the local BEI server.
	beiServer := &tabula.BEIServer{}
	beiConns := beiServer.ListenLocal()

	// Connect to the local BEI server.
	beiClient := bot.NewLocalBEIClient(<-beiConns, false)

	// Start the local bgammon server.
	server := server.NewServer("", "", "", "", "", false, true, false)
	serverConns := server.ListenLocal()

	// Connect the bots.
	go bot.NewLocalClient(<-serverConns, "", "BOT_tabula", "", 1, bgammon.VariantBackgammon, beiClient)
	go bot.NewLocalClient(<-serverConns, "", "BOT_tabula_acey", "", 1, bgammon.VariantAceyDeucey, beiClient)
	go bot.NewLocalClient(<-serverConns, "", "BOT_tabula_tabula", "", 1, bgammon.VariantTabula, beiClient)

	// Wait for the bots to finish creating matches.
	time.Sleep(250 * time.Millisecond)

	// Connect the player.
	go g.ConnectLocal(<-serverConns)
}

func (g *Game) handleUpdateTimeLabels() {
	lastTimerHour, lastTimerMinute := -1, -1
	lastClockHour, lastClockMinute := -1, -1

	t := time.NewTicker(3 * time.Second)
	var now time.Time
	var d time.Duration
	var h, m int
	for {
		now = time.Now()

		// Update match timer.
		started := g.Board.gameState.Started
		if started.IsZero() {
			h, m = 0, 0
		} else {
			ended := g.Board.gameState.Ended
			if ended.IsZero() {
				d = now.Sub(started)
			} else {
				d = ended.Sub(started)
			}
			h, m = int(d.Hours()), int(d.Minutes())%60
		}
		if h != lastTimerHour || m != lastTimerMinute {
			g.Board.timerLabel.SetText(fmt.Sprintf("%d:%02d", h, m))
			lastTimerHour, lastTimerMinute = h, m
			scheduleFrame()
		}

		// Update clock.
		h, m = now.Hour()%12, now.Minute()
		if h == 0 {
			h = 12
		}
		if h != lastClockHour || m != lastClockMinute {
			g.Board.clockLabel.SetText(fmt.Sprintf("%d:%02d", h, m))
			lastClockHour, lastClockMinute = h, m
			scheduleFrame()
		}

		<-t.C
	}
}

func (g *Game) setRoot(w etk.Widget) {
	if w != g.Board.frame {
		g.rootWidget = w
	}
	displayFrame.Clear()
	displayFrame.AddChild(w, g.keyboardFrame)
}

func (g *Game) setBufferRects() {
	statusBufferHeight := etk.Scale(75)
	historyRatingHeight := etk.Scale(200)

	createGameContainer.SetRowSizes(-1, statusBufferHeight, g.lobby.buttonBarHeight)
	joinGameContainer.SetRowSizes(-1, statusBufferHeight, g.lobby.buttonBarHeight)
	historyContainer.SetRowSizes(g.itemHeight(), 2, -1, g.lobby.buttonBarHeight, historyRatingHeight, statusBufferHeight, g.lobby.buttonBarHeight)
	listHeaderHeight := g.itemHeight()
	if AutoEnableTouchInput {
		listHeaderHeight /= 2
	}
	listGamesContainer.SetRowSizes(listHeaderHeight, 2, -1, statusBufferHeight, g.lobby.buttonBarHeight)
}

func (g *Game) handleAutoRefresh() {
	g.lastRefresh = time.Now()
	t := time.NewTicker(19 * time.Second)
	for range t.C {
		if viewBoard {
			continue
		}

		if g.Client != nil && g.Client.Username != "" {
			g.Client.Out <- []byte("ls")
			g.lastRefresh = time.Now()
		}
	}
}

func (g *Game) handleEvent(e interface{}) {
	switch ev := e.(type) {
	case *bgammon.EventWelcome:
		g.Client.Username = ev.PlayerName
		g.register = false

		username := ev.PlayerName
		if strings.HasPrefix(username, "Guest_") && !onlyNumbers.MatchString(username[6:]) {
			username = username[6:]
		}
		password := g.connectPassword.Text()
		if password == "" {
			password = g.registerPassword.Text()
		}
		go saveCredentials(username, password)

		var msg string
		if ev.Clients == 1 && ev.Games == 1 {
			msg = gotext.Get("Welcome, %s. There is 1 client playing 1 match.", ev.PlayerName)
		} else if ev.Clients == 1 {
			msg = gotext.Get("Welcome, %s. There is 1 client playing %d matches.", ev.PlayerName, ev.Games)
		} else if ev.Games == 1 {
			msg = gotext.Get("Welcome, %s. There are %d clients playing 1 match.", ev.PlayerName, ev.Clients)
		} else {
			msg = gotext.Get("Welcome, %s. There are %d clients playing %d matches.", ev.PlayerName, ev.Clients, ev.Games)
		}
		l(fmt.Sprintf("*** " + msg))

		if strings.HasPrefix(g.Client.Username, "Guest_") && g.savedUsername == "" {
			g.tutorialFrame.AddChild(NewTutorialWidget())
		}
	case *bgammon.EventHelp:
		l(fmt.Sprintf("*** Help: %s", ev.Message))
	case *bgammon.EventNotice:
		l(fmt.Sprintf("*** %s", ev.Message))
	case *bgammon.EventSay:
		l(fmt.Sprintf("<%s> %s", ev.Player, ev.Message))
		playSoundEffect(effectSay)
	case *bgammon.EventList:
		g.lobby.setGameList(ev.Games)
		if !viewBoard {
			scheduleFrame()
		}
	case *bgammon.EventJoined:
		g.Board.Lock()
		if ev.PlayerNumber == 1 {
			g.Board.gameState.Player1.Name = ev.Player
		} else if ev.PlayerNumber == 2 {
			g.Board.gameState.Player2.Name = ev.Player
		}
		g.Board.playerRoll1, g.Board.playerRoll2, g.Board.playerRoll3 = 0, 0, 0
		g.Board.opponentRoll1, g.Board.opponentRoll2, g.Board.opponentRoll3 = 0, 0, 0
		g.Board.playerRollStale = false
		g.Board.opponentRollStale = false
		g.Board.availableStale = false
		g.Board.playerMoves = nil
		g.Board.opponentMoves = nil
		if g.needLayoutBoard {
			g.layoutBoard()
		}
		g.Board.processState()
		g.Board.Unlock()
		setViewBoard(true)

		if ev.Player == g.Client.Username {
			gameBuffer.SetText("")
			gameLogged = false
			g.Board.rematchButton.SetVisible(false)
		} else {
			lg(gotext.Get("%s joined the match.", ev.Player))
			playSoundEffect(effectJoinLeave)
		}
	case *bgammon.EventFailedJoin:
		l("*** " + gotext.Get("Failed to join match: %s", ev.Reason))
	case *bgammon.EventFailedLeave:
		l("*** " + gotext.Get("Failed to leave match: %s", ev.Reason))
		setViewBoard(false)
	case *bgammon.EventLeft:
		g.Board.Lock()
		if g.Board.gameState.Player1.Name == ev.Player {
			g.Board.gameState.Player1.Name = ""
		} else if g.Board.gameState.Player2.Name == ev.Player {
			g.Board.gameState.Player2.Name = ""
		}
		g.Board.processState()
		g.Board.Unlock()
		if ev.Player == g.Client.Username {
			setViewBoard(false)
		} else {
			lg(gotext.Get("%s left the match.", ev.Player))
			playSoundEffect(effectJoinLeave)
		}
	case *bgammon.EventBoard:
		g.Board.Lock()

		g.Board.stateLock.Lock()
		*g.Board.gameState = ev.GameState
		*g.Board.gameState.Game = *ev.GameState.Game
		if g.Board.gameState.Turn == 0 {
			if g.Board.playerRoll2 != 0 {
				g.Board.playerRoll1, g.Board.playerRoll2, g.Board.playerRoll3 = 0, 0, 0
			}
			if g.Board.opponentRoll1 != 0 {
				g.Board.opponentRoll1, g.Board.opponentRoll2, g.Board.opponentRoll3 = 0, 0, 0
			}
			if g.Board.gameState.Roll1 != 0 {
				g.Board.playerRoll1 = g.Board.gameState.Roll1
			}
			if g.Board.gameState.Roll2 != 0 {
				g.Board.opponentRoll2 = g.Board.gameState.Roll2
			}
		} else if g.Board.gameState.Roll1 != 0 {
			if g.Board.gameState.Turn == 1 {
				g.Board.playerRoll1, g.Board.playerRoll2, g.Board.playerRoll3 = g.Board.gameState.Roll1, g.Board.gameState.Roll2, g.Board.gameState.Roll3
				g.Board.playerRollStale = false
				g.Board.opponentRollStale = true
				if g.Board.opponentRoll1 == 0 || g.Board.opponentRoll2 == 0 {
					g.Board.opponentRoll1, g.Board.opponentRoll2, g.Board.opponentRoll3 = 0, 0, 0
				}
			} else {
				g.Board.opponentRoll1, g.Board.opponentRoll2, g.Board.opponentRoll3 = g.Board.gameState.Roll1, g.Board.gameState.Roll2, g.Board.gameState.Roll3
				g.Board.opponentRollStale = false
				g.Board.playerRollStale = true
				if g.Board.playerRoll1 == 0 || g.Board.playerRoll2 == 0 {
					g.Board.playerRoll1, g.Board.playerRoll2, g.Board.playerRoll3 = 0, 0, 0
				}
				g.Board.dragging = nil
			}
		}
		g.Board.availableStale = false
		g.Board.stateLock.Unlock()

		g.Board.processState()
		g.Board.Unlock()

		setViewBoard(true)
	case *bgammon.EventRolled:
		g.Board.Lock()
		g.Board.stateLock.Lock()
		g.Board.gameState.Roll1 = ev.Roll1
		g.Board.gameState.Roll2 = ev.Roll2
		g.Board.gameState.Roll3 = ev.Roll3
		var diceFormatted string
		if g.Board.gameState.Turn == 0 {
			if g.Board.gameState.Player1.Name == ev.Player {
				diceFormatted = fmt.Sprintf("%d", g.Board.gameState.Roll1)
				g.Board.playerRoll1 = g.Board.gameState.Roll1
				g.Board.playerRollStale = false
			} else {
				diceFormatted = fmt.Sprintf("%d", g.Board.gameState.Roll2)
				g.Board.opponentRoll2 = g.Board.gameState.Roll2
				g.Board.opponentRollStale = false
			}
			if !ev.Selected {
				playSoundEffect(effectDie)
			}
			g.Board.availableStale = false
		} else {
			diceFormatted = fmt.Sprintf("%d-%d", g.Board.gameState.Roll1, g.Board.gameState.Roll2)
			if g.Board.gameState.Player1.Name == ev.Player {
				g.Board.playerRoll1, g.Board.playerRoll2, g.Board.playerRoll3 = g.Board.gameState.Roll1, g.Board.gameState.Roll2, g.Board.gameState.Roll3
				g.Board.playerRollStale = false
			} else {
				g.Board.opponentRoll1, g.Board.opponentRoll2, g.Board.opponentRoll3 = g.Board.gameState.Roll1, g.Board.gameState.Roll2, g.Board.gameState.Roll3
				g.Board.opponentRollStale = false
			}
			if g.Board.gameState.Roll3 != 0 {
				diceFormatted += fmt.Sprintf("-%d", g.Board.gameState.Roll3)
			}
			if !ev.Selected {
				playSoundEffect(effectDice)
			}
			g.Board.availableStale = true
		}
		g.Board.stateLock.Unlock()
		g.Board.processState()
		g.Board.Unlock()
		scheduleFrame()
		lg(gotext.Get("%s rolled %s", ev.Player, diceFormatted))
	case *bgammon.EventFailedRoll:
		l(fmt.Sprintf("*** %s: %s", gotext.Get("Failed to roll"), ev.Reason))
	case *bgammon.EventMoved:
		moves := ev.Moves
		if g.Board.gameState.Turn == 2 && game.Board.traditional {
			moves = bgammon.FlipMoves(moves, 2, g.Board.gameState.Variant)
		}
		lg(gotext.Get("%s moved %s", ev.Player, bgammon.FormatMoves(moves)))
		if ev.Player == g.Client.Username && !g.Board.gameState.Spectating && !g.Board.gameState.Forced {
			return
		}

		g.Board.Lock()
		g.Unlock()
		for _, move := range ev.Moves {
			playSoundEffect(effectMove)
			g.Board.movePiece(move[0], move[1], true)
		}
		g.Lock()
		if g.Board.showMoves {
			moves := g.Board.gameState.Moves
			if g.Board.gameState.Turn == 2 && game.Board.traditional {
				moves = bgammon.FlipMoves(moves, 2, g.Board.gameState.Variant)
			}
			if g.Board.gameState.Turn == 1 {
				g.Board.playerMoves = expandMoves(moves)
			} else if g.Board.gameState.Turn == 2 {
				g.Board.opponentMoves = expandMoves(moves)
			}
		}
		g.Board.Unlock()
	case *bgammon.EventFailedMove:
		g.Client.Out <- []byte("board") // Refresh game state.

		var extra string
		if ev.From != 0 || ev.To != 0 {
			extra = " " + gotext.Get("from %s to %s", bgammon.FormatSpace(ev.From), bgammon.FormatSpace(ev.To))
		}
		l("*** " + gotext.Get("Failed to move checker%s: %s", extra, ev.Reason))
		l("*** " + gotext.Get("Legal moves: %s", bgammon.FormatMoves(g.Board.gameState.Available)))
	case *bgammon.EventFailedOk:
		g.Client.Out <- []byte("board") // Refresh game state.
		l("*** " + gotext.Get("Failed to submit moves: %s", ev.Reason))
	case *bgammon.EventWin:
		g.Board.Lock()
		lg(gotext.Get("%s wins!", ev.Player))
		if (g.Board.gameState.Player1.Points >= g.Board.gameState.Points || g.Board.gameState.Player2.Points >= g.Board.gameState.Points) && !g.Board.gameState.Spectating {
			g.Board.rematchButton.SetVisible(true)
		}
		g.Board.Unlock()
	case *bgammon.EventSettings:
		b := g.Board
		b.Lock()
		if ev.Speed >= bgammon.SpeedSlow && ev.Speed <= bgammon.SpeedInstant {
			b.speed = ev.Speed
			b.selectSpeed.SetSelectedItem(int(b.speed))
		}
		b.highlightAvailable = ev.Highlight
		b.highlightCheckbox.SetSelected(b.highlightAvailable)
		b.showPipCount = ev.Pips
		b.showPipCountCheckbox.SetSelected(b.showPipCount)
		b.showMoves = ev.Moves
		b.showMovesCheckbox.SetSelected(b.showMoves)
		b.flipBoard = ev.Flip
		b.flipBoardCheckbox.SetSelected(b.flipBoard)
		b.traditional = ev.Traditional
		b.traditionalCheckbox.SetSelected(b.traditional)
		if !AutoEnableTouchInput {
			b.advancedMovement = ev.Advanced
			b.advancedMovementCheckbox.SetSelected(b.advancedMovement)
		}
		b.autoPlayCheckbox.SetSelected(ev.AutoPlay)
		if g.needLayoutBoard {
			g.layoutBoard()
		}
		b.setSpaceRects()
		b.updateBackgroundImage()
		b.processState()
		b.updatePlayerLabel()
		b.updateOpponentLabel()
		b.Unlock()
	case *bgammon.EventReplay:
		if game.downloadReplay == ev.ID {
			err := saveReplay(ev.ID, ev.Content)
			if err != nil {
				l("*** " + gotext.Get("Failed to download replay: %s", err))
			}
			game.downloadReplay = 0
			return
		}
		go game.HandleReplay(ev.Content)
	case *bgammon.EventHistory:
		game.lobby.historyMatches = ev.Matches
		game.lobby.historyPage = ev.Page
		game.lobby.historyPages = ev.Pages
		game.lobby.historyPageLabel.SetText(fmt.Sprintf("%d/%d", ev.Page, ev.Pages))
		game.lobby.historyRatingCasualBackgammonSingle.SetText(fmt.Sprintf("%d", ev.CasualBackgammonSingle))
		game.lobby.historyRatingCasualBackgammonMulti.SetText(fmt.Sprintf("%d", ev.CasualBackgammonMulti))
		game.lobby.historyRatingCasualAceySingle.SetText(fmt.Sprintf("%d", ev.CasualAceyDeuceySingle))
		game.lobby.historyRatingCasualAceyMulti.SetText(fmt.Sprintf("%d", ev.CasualAceyDeuceyMulti))
		game.lobby.historyRatingCasualTabulaSingle.SetText(fmt.Sprintf("%d", ev.CasualTabulaSingle))
		game.lobby.historyRatingCasualTabulaMulti.SetText(fmt.Sprintf("%d", ev.CasualTabulaMulti))
		list := game.lobby.historyList
		list.Clear()
		list.SetSelectionMode(etk.SelectRow)
		if len(ev.Matches) == 0 {
			scheduleFrame()
			return
		}
		y := list.Rows()
		for i, match := range ev.Matches {
			result := "W"
			if match.Winner == 2 {
				result = "L"
			}
			dateLabel := newCenteredText(time.Unix(match.Timestamp, 0).Format("2006-01-02"))
			resultLabel := newCenteredText(result)
			opponentLabel := newCenteredText(match.Opponent)
			if AutoEnableTouchInput {
				dateLabel.SetFont(etk.Style.TextFont, etk.Scale(mediumFontSize))
				resultLabel.SetFont(etk.Style.TextFont, etk.Scale(mediumFontSize))
				opponentLabel.SetFont(etk.Style.TextFont, etk.Scale(mediumFontSize))
			}
			list.AddChildAt(dateLabel, 0, y+i)
			list.AddChildAt(resultLabel, 1, y+i)
			list.AddChildAt(opponentLabel, 2, y+i)
		}
		if ev.Page == 1 {
			list.SetSelectedItem(0, 0)
		}
		scheduleFrame()
	case *bgammon.EventPing:
		g.Client.Out <- []byte(fmt.Sprintf("pong %s", ev.Message))
	default:
		l("*** " + gotext.Get("Warning: Received unknown event: %+v", ev))
		l("*** " + gotext.Get("You may need to upgrade your client."))
	}
}

func (g *Game) handleEvents() {
	for e := range g.Client.Events {
		g.Board.Lock()
		g.Lock()
		g.Board.Unlock()
		g.handleEvent(e)
		g.Unlock()
	}
}

func (g *Game) _handleReplay(gs *bgammon.GameState, line []byte, lineNumber int, sendEvent bool, haveLock bool) bool {
	if !haveLock {
		g.Lock()
		defer g.Unlock()
	}

	if !g.replay {
		return false
	}

	split := bytes.Split(line, []byte(" "))
	if len(split) < 2 {
		log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
		return false
	}
	switch {
	case bytes.Equal(split[0], []byte("bgammon-replay")):
		return true
	case bytes.Equal(split[0], []byte("i")):
		if len(split) < 10 {
			log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
			return false
		}

		if sendEvent {
			ev := &bgammon.EventJoined{
				GameID: 1,
			}
			ev.PlayerNumber = 1
			ev.Player = game.Client.Username
			g.Client.Events <- ev
		}

		variant, err := strconv.Atoi(string(split[9]))
		if err != nil || variant < 0 || variant > 2 {
			log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
			return false
		}

		*gs = bgammon.GameState{
			Game:         bgammon.NewGame(int8(variant)),
			PlayerNumber: 1,
			Spectating:   true,
		}
		gs.Turn = 0

		gs.Player1.Name, gs.Player2.Name = string(split[2]), string(split[3])

		points, err := strconv.Atoi(string(split[4]))
		if err != nil || gs.Points < 1 {
			log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
			return false
		}
		gs.Points = int8(points)

		points, err = strconv.Atoi(string(split[5]))
		if err != nil || gs.Player1.Points < 0 {
			log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
			return false
		}
		gs.Player1.Points = int8(points)

		points, err = strconv.Atoi(string(split[6]))
		if err != nil || gs.Player1.Points < 0 {
			log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
			return false
		}
		gs.Player2.Points = int8(points)

		if sendEvent {
			ev := &bgammon.EventBoard{
				GameState: bgammon.GameState{
					Game:         gs.Game.Copy(true),
					PlayerNumber: 1,
					Available:    gs.Available,
					Spectating:   true,
				},
			}
			g.Client.Events <- ev
		}

		timestamp, err := strconv.ParseInt(string(split[1]), 10, 64)
		if err != nil {
			log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
			return false
		}
		gs.Started = time.Unix(timestamp, 0)
		gs.Ended = gs.Started
	case bytes.Equal(split[0], []byte("1")), bytes.Equal(split[0], []byte("2")):
		if len(split) < 2 || (!bytes.Equal(split[1], []byte("t")) && len(split) < 3) {
			log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
			return false
		}
		var player int8 = 1
		if bytes.Equal(split[0], []byte("2")) {
			player = 2
		}
		switch {
		case bytes.Equal(split[1], []byte("d")):
			if len(split) < 4 {
				log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
				return false
			}
			doubleValue, err := strconv.Atoi(string(split[2]))
			if err != nil || doubleValue < 2 {
				log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
				return false
			}
			resultValue, err := strconv.Atoi(string(split[3]))
			if err != nil || resultValue < 0 || resultValue > 1 {
				log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
				return false
			}
			resultText := "accepts"
			if resultValue == 0 {
				resultText = "declines"
			}
			l(fmt.Sprintf("*** %s offers a double (%d points). %s %s.", gs.Player1.Name, doubleValue, gs.Player2.Name, resultText))
		case bytes.Equal(split[1], []byte("r")):
			rollSplit := bytes.Split(split[2], []byte("-"))
			if len(rollSplit) < 2 || len(rollSplit[0]) != 1 || len(rollSplit[1]) != 1 {
				log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
				return false
			}
			r1, err := strconv.Atoi(string(rollSplit[0]))
			if err != nil || r1 < 1 || r1 > 6 {
				log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
				return false
			}
			r2, err := strconv.Atoi(string(rollSplit[1]))
			if err != nil || r2 < 1 || r2 > 6 {
				log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
				return false
			}
			var r3 int
			if len(rollSplit) > 2 {
				r3, err = strconv.Atoi(string(rollSplit[2]))
				if err != nil || r3 < 1 || r3 > 6 {
					log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
					return false
				}
			}

			gs.Moves = nil
			if gs.Turn == 0 {
				gs.Turn = player
				gs.Available = nil
				gs.Moves = nil
				if sendEvent {
					ev := &bgammon.EventBoard{
						GameState: bgammon.GameState{
							Game:         gs.Game.Copy(true),
							PlayerNumber: 1,
							Available:    gs.Available,
							Spectating:   true,
						},
					}
					g.Client.Events <- ev
				}
			}

			playerName := gs.Player1.Name
			if player == 2 {
				playerName = gs.Player2.Name
			}

			if sendEvent {
				ev := &bgammon.EventRolled{
					Roll1: int8(r1),
					Roll2: int8(r2),
					Roll3: int8(r3),
				}
				ev.Player = playerName
				g.Client.Events <- ev
			}

			gs.Roll1, gs.Roll2, gs.Roll3 = int8(r1), int8(r2), int8(r3)
			gs.Turn = player
			gs.Available = gs.LegalMoves(true)
			gs.Moves = nil

			if sendEvent {
				ev := &bgammon.EventBoard{
					GameState: bgammon.GameState{
						Game:         gs.Game.Copy(true),
						PlayerNumber: 1,
						Available:    gs.Available,
						Spectating:   true,
					},
				}
				g.Client.Events <- ev
			}

			if len(split) == 3 {
				return true
			}
			for _, move := range split[3:] {
				moveSplit := bytes.Split(move, []byte("/"))
				if len(moveSplit) != 2 || len(moveSplit[0]) > 3 || len(moveSplit[1]) > 3 {
					log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
					return false
				}
				from, to := bgammon.ParseSpace(string(moveSplit[0])), bgammon.ParseSpace(string(moveSplit[1]))
				if from < 0 || to < 0 || from == to {
					log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
					return false
				} else if from == bgammon.SpaceBarPlayer && player == 2 {
					from = bgammon.SpaceBarOpponent
				} else if from == bgammon.SpaceHomePlayer && player == 2 {
					from = bgammon.SpaceHomeOpponent
				}
				if to == bgammon.SpaceHomePlayer && player == 2 {
					to = bgammon.SpaceHomeOpponent
				}
				if sendEvent {
					ev := &bgammon.EventMoved{
						Moves: [][]int8{{from, to}},
					}
					ev.Player = playerName
					g.Client.Events <- ev
				}
				ok, _ := gs.AddMoves([][]int8{{from, to}}, false)
				if !ok {
					log.Panicf("failed to move checkers during replay from %d to %d", from, to)
				}
			}

			if gs.Winner != 0 {
				playerBar := bgammon.SpaceBarPlayer
				opponentHome := bgammon.SpaceHomeOpponent
				var opponent int8 = 2
				if player == 2 {
					playerBar = bgammon.SpaceBarOpponent
					opponentHome = bgammon.SpaceHomePlayer
					opponent = 1
				}

				backgammon := bgammon.PlayerCheckers(gs.Board[playerBar], opponent) != 0
				if !backgammon {
					homeStart, homeEnd := bgammon.HomeRange(gs.Winner, gs.Variant)
					bgammon.IterateSpaces(homeStart, homeEnd, gs.Variant, func(space int8, spaceCount int8) {
						if bgammon.PlayerCheckers(gs.Board[space], opponent) != 0 {
							backgammon = true
						}
					})
				}

				var winPoints int8
				switch gs.Variant {
				case bgammon.VariantBackgammon:
					if backgammon {
						winPoints = 3 // Award backgammon.
					} else if gs.Board[opponentHome] == 0 {
						winPoints = 2 // Award gammon.
					} else {
						winPoints = 1
					}
				case bgammon.VariantAceyDeucey:
					for space := int8(0); space < bgammon.BoardSpaces; space++ {
						if (space == bgammon.SpaceHomePlayer || space == bgammon.SpaceHomeOpponent) && ((opponent == 1 && gs.Player1.Entered) || (opponent == 2 && gs.Player2.Entered)) {
							continue
						}
						winPoints += bgammon.PlayerCheckers(gs.Board[space], opponent)
					}
				case bgammon.VariantTabula:
					winPoints = 1
				}

				if sendEvent {
					ev := &bgammon.EventWin{
						Points: winPoints * gs.DoubleValue,
					}
					ev.Player = playerName
					g.Client.Events <- ev
				}
			}

			if sendEvent {
				ev := &bgammon.EventBoard{
					GameState: bgammon.GameState{
						Game:         gs.Game.Copy(true),
						PlayerNumber: 1,
						Available:    gs.Available,
						Spectating:   true,
					},
				}
				g.Client.Events <- ev
			}
		case bytes.Equal(split[1], []byte("t")):
			playerName := gs.Player1.Name
			if player == 2 {
				playerName = gs.Player2.Name
			}
			if sendEvent {
				ev := &bgammon.EventNotice{
					Message: gotext.Get("%s resigned.", playerName),
				}
				g.Client.Events <- ev
			}
		default:
			log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
			return false
		}
	default:
		log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
		return false
	}
	return true
}

func (g *Game) showReplayFrame(replayFrame int, showInfo bool) {
	if !g.replay || replayFrame < 0 || replayFrame >= len(g.replayFrames) {
		return
	}

	if g.needLayoutBoard {
		g.layoutBoard()
	}

	if replayFrame == 0 && showInfo {
		g.Board.recreateUIGrid()
	}

	g.replayFrame = replayFrame
	frame := g.replayFrames[replayFrame]

	ev := &bgammon.EventBoard{
		GameState: bgammon.GameState{
			Game:         frame.Game.Copy(true),
			PlayerNumber: 1,
			Available:    frame.Game.LegalMoves(true),
			Spectating:   true,
		},
	}
	g.Client.Events <- ev

	if replayFrame == 0 && showInfo {
		l(fmt.Sprintf("*** "+gotext.Get("Replaying %s vs. %s", "%s", "%s")+" (%s)", frame.Game.Player2.Name, frame.Game.Player1.Name, frame.Game.Started.Format("2006-01-02 15:04")))
	}
}

func (g *Game) HandleReplay(replay []byte) {
	g.Lock()
	if g.replay {
		g.Unlock()
		return
	}
	g.replay = true
	g.replayFrame = 0
	g.replayFrames = g.replayFrames[:0]
	g.replayData = replay
	g.Unlock()

	g.Board.rematchButton.SetVisible(false)

	if !g.loggedIn {
		go g.playOffline()
		time.Sleep(500 * time.Millisecond)
	}

	gs := &bgammon.GameState{
		Game:         bgammon.NewGame(bgammon.VariantBackgammon),
		PlayerNumber: 1,
		Spectating:   true,
	}

	g.replaySummary1 = g.replaySummary1[:0]
	g.replaySummary2 = g.replaySummary2[:0]
	var haveRoll bool
	var wrote1 bool
	var wrote2 bool

	var lineNumber int
	scanner := bufio.NewScanner(bytes.NewReader(replay))
	for scanner.Scan() {
		if !bytes.HasPrefix(scanner.Bytes(), []byte("bgammon-reply")) && !bytes.HasPrefix(scanner.Bytes(), []byte("i ")) {
			g.replayFrames = append(g.replayFrames, &replayFrame{
				Game:  gs.Game.Copy(true),
				Event: scanner.Bytes(),
			})
		}
		lineNumber++
		if !g._handleReplay(gs, scanner.Bytes(), lineNumber, false, false) {
			return
		}

		if bytes.HasPrefix(scanner.Bytes(), []byte("1 r ")) || bytes.HasPrefix(scanner.Bytes(), []byte("2 r ")) {
			player := 1
			if bytes.HasPrefix(scanner.Bytes(), []byte("2")) {
				player = 2
			}

			if player == 1 {
				if wrote1 {
					g.replaySummary1 = append(g.replaySummary1, '\n')
				}
				g.replaySummary1 = append(g.replaySummary1, scanner.Bytes()[4:]...)
				haveRoll = true
				wrote1 = true
			} else {
				if wrote2 {
					g.replaySummary2 = append(g.replaySummary2, '\n')
				}
				g.replaySummary2 = append(g.replaySummary2, scanner.Bytes()[4:]...)
				wrote2 = true
				if !haveRoll {
					g.replaySummary1 = append(g.replaySummary1, '\n')
					haveRoll = true
				}
			}
		}
	}
	if scanner.Err() != nil {
		log.Printf("warning: failed to read replay: %s", scanner.Err())
		return
	}

	if len(g.replayFrames) < 2 {
		log.Printf("warning: failed to read replay: no frames were loaded")
		return
	}

	g.replayFrames = append(g.replayFrames, &replayFrame{
		Game:  gs.Game.Copy(true),
		Event: nil,
	})

	g.Lock()
	g.showReplayFrame(0, true)
	g.Unlock()
}

func (g *Game) Connect() {
	if g.loggedIn {
		return
	}
	g.loggedIn = true

	l("*** " + gotext.Get("Connecting..."))

	if g.Password != "" {
		g.lobby.historyButton.SetVisible(true)
	}

	g.setRoot(listGamesFrame)

	address := g.ServerAddress
	if address == "" {
		address = DefaultServerAddress
	}
	g.Client = newClient(address, g.Username, g.Password, false)
	g.lobby.c = g.Client
	g.Board.Client = g.Client

	go g.handleEvents()

	if g.Password != "" {
		g.Board.recreateAccountGrid()
	}

	c := g.Client

	if g.TV {
		go func() {
			time.Sleep(time.Second)
			c.Out <- []byte("tv")
		}()
	}

	connectTime := time.Now()
	t := time.NewTicker(250 * time.Millisecond)
	go func() {
		for {
			<-t.C
			if c.loggedIn {
				return
			} else if !c.connecting || time.Since(connectTime) >= 20*time.Second {
				if !g.showConnectStatusBuffer {
					connectGrid.AddChildAt(statusBuffer, 0, g.connectGridY+4, 5, 1)
					g.showConnectStatusBuffer = true
				}

				g.loggedIn = false
				g.setRoot(connectFrame)
				scheduleFrame()
				return
			}
		}
	}()

	go c.Connect()

	// TODO

}

func (g *Game) ConnectLocal(conn net.Conn) {
	if g.loggedIn {
		return
	}
	g.loggedIn = true

	l("*** " + gotext.Get("Playing offline."))

	g.setRoot(listGamesFrame)

	g.Client = newClient("", g.Username, g.Password, false)
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
	}

	go c.connectTCP(conn)
}

func (g *Game) selectRegister() error {
	g.showRegister = true
	g.registerUsername.SetText(g.connectUsername.Text())
	g.registerPassword.SetText(g.connectPassword.Text())
	g.setRoot(registerGrid)
	etk.SetFocus(g.registerEmail)
	return nil
}

func (g *Game) selectReset() error {
	g.showReset = true
	g.setRoot(resetGrid)
	etk.SetFocus(g.resetEmail)
	return nil
}

func (g *Game) selectCancel() error {
	g.showRegister = false
	g.showReset = false
	g.setRoot(connectFrame)
	etk.SetFocus(g.connectUsername)
	return nil
}

func (g *Game) selectConfirmRegister() error {
	go hideKeyboard()
	g.Email = g.registerEmail.Text()
	g.Username = g.registerUsername.Text()
	g.Password = g.registerPassword.Text()
	if ShowServerSettings {
		g.ServerAddress = g.connectServer.Text()
	}
	g.register = true
	g.Connect()
	return nil
}

func (g *Game) selectConfirmReset() error {
	go hideKeyboard()
	if g.resetInProgress {
		return nil
	}
	g.resetInProgress = true
	address := g.ServerAddress
	if ShowServerSettings {
		address = g.connectServer.Text()
	}
	client := newClient(address, g.resetEmail.Text(), "", true)
	go client.Connect()
	g.resetInfo.SetText(gotext.Get("Sending password reset request") + "...")
	go func() {
		time.Sleep(10 * time.Second)
		g.resetInfo.SetText(gotext.Get("Check your email for a link to reset your password. Be sure to check your spam folder."))
		g.resetInProgress = false
		scheduleFrame()
	}()
	return nil
}

func (g *Game) selectConnect() error {
	go hideKeyboard()
	g.Username = g.connectUsername.Text()
	g.Password = g.connectPassword.Text()
	if ShowServerSettings {
		g.ServerAddress = g.connectServer.Text()
	}
	g.Connect()
	return nil
}

func (g *Game) searchMatches(username string) {
	go hideKeyboard()
	loadingText := newCenteredText(gotext.Get("Loading..."))
	if AutoEnableTouchInput {
		loadingText.SetFont(etk.Style.TextFont, etk.Scale(mediumFontSize))
	}

	g.lobby.historyList.Clear()
	g.lobby.historyList.SetSelectionMode(etk.SelectNone)
	g.lobby.historyList.AddChildAt(loadingText, 0, 0)
	g.Client.Out <- []byte(fmt.Sprintf("history %s", username))
}

func (g *Game) selectHistory() error {
	go hideKeyboard()
	g.lobby.showHistory = true
	g.setRoot(historyFrame)
	g.lobby.historyUsername.SetText(g.Client.Username)
	g.searchMatches(g.Client.Username)
	etk.SetFocus(g.lobby.historyUsername)
	g.lobby.rebuildButtonsGrid()
	return nil
}

func (g *Game) selectHistorySearch() error {
	go hideKeyboard()
	username := g.lobby.historyUsername.Text()
	if strings.TrimSpace(username) == "" {
		return nil
	}
	g.searchMatches(username)
	return nil
}

func (g *Game) selectHistoryPrevious() error {
	go hideKeyboard()
	if g.lobby.historyUsername.Text() == "" || g.lobby.historyPage == 1 {
		return nil
	}
	g.Client.Out <- []byte(fmt.Sprintf("history %s %d", g.lobby.historyUsername.Text(), g.lobby.historyPage-1))
	return nil
}

func (g *Game) selectHistoryNext() error {
	go hideKeyboard()
	if g.lobby.historyUsername.Text() == "" || g.lobby.historyPage == g.lobby.historyPages {
		return nil
	}
	g.Client.Out <- []byte(fmt.Sprintf("history %s %d", g.lobby.historyUsername.Text(), g.lobby.historyPage+1))
	return nil
}

func (g *Game) handleInput(keys []ebiten.Key) error {
	if len(keys) == 0 {
		return nil
	} else if AutoEnableTouchInput {
		scheduleFrame()
	}

	if !g.loggedIn {
		for _, key := range keys {
			switch key {
			case ebiten.KeyTab:
				focusedWidget := etk.Focused()
				switch focusedWidget {
				case g.connectUsername:
					etk.SetFocus(g.connectPassword)
				case g.connectPassword:
					etk.SetFocus(g.connectUsername)
				case g.registerEmail:
					if ebiten.IsKeyPressed(ebiten.KeyShift) {
						etk.SetFocus(g.registerPassword)
					} else {
						etk.SetFocus(g.registerUsername)
					}
				case g.registerUsername:
					if ebiten.IsKeyPressed(ebiten.KeyShift) {
						etk.SetFocus(g.registerEmail)
					} else {
						etk.SetFocus(g.registerPassword)
					}
				case g.registerPassword:
					if ebiten.IsKeyPressed(ebiten.KeyShift) {
						etk.SetFocus(g.registerUsername)
					} else {
						etk.SetFocus(g.registerEmail)
					}
				}
			case ebiten.KeyEnter, ebiten.KeyKPEnter:
				if g.showRegister {
					g.selectConfirmRegister()
				} else if g.showReset {
					g.selectConfirmReset()
				} else {
					g.selectConnect()
				}
			}
		}
		return nil
	}

	for _, key := range keys {
		switch key {
		case ebiten.KeyEscape:
			if viewBoard {
				if g.Board.menuGrid.Visible() {
					g.Board.menuGrid.SetVisible(false)
				} else if g.Board.settingsGrid.Visible() {
					g.Board.settingsGrid.SetVisible(false)
					g.Board.selectSpeed.SetMenuVisible(false)
				} else if g.Board.changePasswordGrid.Visible() {
					g.Board.hideMenu()
				} else if g.Board.leaveGameGrid.Visible() {
					g.Board.leaveGameGrid.SetVisible(false)
				} else {
					g.Board.menuGrid.SetVisible(true)
				}
				continue
			}
			setViewBoard(!viewBoard)
		}
	}

	if !viewBoard && g.lobby.showCreateGame {
		for _, key := range keys {
			switch key {
			case ebiten.KeyTab:
				focusedWidget := etk.Focused()
				if ebiten.IsKeyPressed(ebiten.KeyShift) {
					switch focusedWidget {
					case g.lobby.createGameName:
						etk.SetFocus(g.lobby.createGamePassword)
					case g.lobby.createGamePoints:
						etk.SetFocus(g.lobby.createGameName)
					case g.lobby.createGamePassword:
						etk.SetFocus(g.lobby.createGamePoints)
					}
				} else {
					switch focusedWidget {
					case g.lobby.createGameName:
						etk.SetFocus(g.lobby.createGamePoints)
					case g.lobby.createGamePoints:
						etk.SetFocus(g.lobby.createGamePassword)
					case g.lobby.createGamePassword:
						etk.SetFocus(g.lobby.createGameName)
					}
				}
			}
		}
	}

	if !viewBoard && (g.lobby.showCreateGame || g.lobby.showJoinGame || g.lobby.showHistory) {
		for _, key := range keys {
			if key == ebiten.KeyEnter || key == ebiten.KeyKPEnter {
				if g.lobby.showCreateGame {
					g.lobby.confirmCreateGame()
				} else if g.lobby.showHistory {
					g.selectHistorySearch()
				} else {
					g.lobby.confirmJoinGame()
				}
			}
		}
	}

	if viewBoard {
		for _, key := range keys {
			switch key {
			case ebiten.KeyTab:
				if g.Board.changePasswordGrid.Visible() {
					focusedWidget := etk.Focused()
					switch focusedWidget {
					case g.Board.changePasswordOld:
						etk.SetFocus(g.Board.changePasswordNew)
					case g.Board.changePasswordNew:
						etk.SetFocus(g.Board.changePasswordOld)
					}
				}
			case ebiten.KeyEnter:
				if g.Board.changePasswordGrid.Visible() {
					g.Board.selectChangePassword()
				}
			}
		}
	}
	return nil
}

// Update is called by Ebitengine only when user input occurs, or a frame is
// explicitly scheduled.
func (g *Game) Update() error {
	if ebiten.IsWindowBeingClosed() {
		g.Exit()
		return nil
	}

	g.drawTick++

	g.Lock()
	defer g.Unlock()

	gameUpdateLock.Lock()
	updatedGame = true
	gameUpdateLock.Unlock()

	switch {
	case g.needLayoutConnect && !g.loggedIn:
		g.layoutConnect()
	case g.needLayoutLobby && g.loggedIn && !viewBoard:
		g.layoutLobby()
	case g.needLayoutBoard && g.loggedIn && viewBoard:
		g.layoutBoard()
	}

	cx, cy := ebiten.CursorPosition()
	if (cx != g.cursorX || cy != g.cursorY) && cx >= 0 && cy >= 0 && cx < g.screenW && cy < g.screenH {
		g.cursorX, g.cursorY = cx, cy
		scheduleFrame()
	}

	wheelX, wheelY := ebiten.Wheel()
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) || ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) || wheelX != 0 || wheelY != 0 {
		scheduleFrame()
	}

	g.pressedKeys = inpututil.AppendPressedKeys(g.pressedKeys[:0])
	if len(g.pressedKeys) > 0 {
		scheduleFrame()
	}

	if !g.loaded {
		g.loaded = true

		// Auto-connect
		if g.Username != "" || g.Password != "" {
			g.Connect()
		}
	}

	if ebiten.IsKeyPressed(ebiten.KeyControl) && inpututil.IsKeyJustPressed(ebiten.KeyP) {
		err := g.toggleProfiling()
		if err != nil {
			return err
		}
	}

	if ebiten.IsKeyPressed(ebiten.KeyControl) && inpututil.IsKeyJustPressed(ebiten.KeyD) {
		Debug++
		if Debug > MaxDebug {
			Debug = 0
		}
		g.Board.debug = Debug
		etk.SetDebug(Debug == 2)
	}

	// Handle touch input.
	if len(ebiten.AppendTouchIDs(g.touchIDs[:0])) != 0 {
		scheduleFrame()
	}

	// Handle physical keyboard.
	g.pressedKeys = inpututil.AppendJustPressedKeys(g.pressedKeys[:0])
	err := g.handleInput(g.pressedKeys)
	if err != nil {
		return err
	}

	if AutoEnableTouchInput {
		g.pressedRunes = ebiten.AppendInputChars(g.pressedRunes[:0])
		if len(g.pressedRunes) != 0 {
			scheduleFrame()
		}
	}

	err = etk.Update()
	if err != nil {
		return err
	}
	if !g.loggedIn {
		return nil
	}

	if !viewBoard {
		if g.lobby.showCreateGame || g.lobby.showJoinGame {
			if g.lobby.showCreateGame {
				if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
					p := image.Point{cx, cy}
					if p.In(g.lobby.createGameName.Rect()) {
						etk.SetFocus(g.lobby.createGameName)
					} else if p.In(g.lobby.createGamePoints.Rect()) {
						etk.SetFocus(g.lobby.createGamePoints)
					} else if p.In(g.lobby.createGamePassword.Rect()) {
						etk.SetFocus(g.lobby.createGamePassword)
					}
				}
			}

			if g.lobby.showCreateGame {
				pointsText := g.lobby.createGamePoints.Text()
				strippedText := strings.Join(anyNumbers.FindAllString(pointsText, -1), "")
				if pointsText != strippedText {
					g.lobby.createGamePoints.SetText(strippedText)
				}
			}
		}
	} else {
		g.Board.Update()
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.Lock()
	defer g.Unlock()

	switch {
	case g.needLayoutConnect && !g.loggedIn:
		g.layoutConnect()
	case g.needLayoutLobby && g.loggedIn && !viewBoard:
		g.layoutLobby()
	case g.needLayoutBoard && g.loggedIn && viewBoard:
		g.layoutBoard()
	}

	gameUpdateLock.Lock()
	if drawScreen <= 0 {
		if g.drawTick < targetFPS {
			gameUpdateLock.Unlock()
			return
		}
		updatedGame = false
		drawScreen = 1
	}
	now := time.Now()
	diff := 1000000000*time.Nanosecond/targetFPS - now.Sub(lastDraw)
	if diff > 0 {
		time.Sleep(diff)
	}
	lastDraw = now
	if updatedGame {
		drawScreen -= 1
	}
	gameUpdateLock.Unlock()

	g.drawTick = 0

	if !viewBoard {
		screen.Fill(frameColor)
	} else {
		screen.Fill(tableColor)
	}

	// Log in screen
	if !g.loggedIn {
		err := etk.Draw(screen)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	if viewBoard {
		g.Board.Draw(screen)
	}

	err := etk.Draw(screen)
	if err != nil {
		log.Fatal(err)
	}

	if Debug > 0 {
		g.drawBuffer.Reset()

		g.spinnerIndex++
		if g.spinnerIndex == 4 {
			g.spinnerIndex = 0
		}

		scale := etk.ScaleFactor()
		if scale != 1.0 {
			g.drawBuffer.Write([]byte(fmt.Sprintf("SCA %0.1f\n", scale)))
		}

		g.drawBuffer.Write([]byte(fmt.Sprintf("FPS %c %0.0f", spinner[g.spinnerIndex], ebiten.ActualFPS())))

		g.debugImg.Clear()

		ebitenutil.DebugPrint(g.debugImg, g.drawBuffer.String())

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(3, 0)
		op.GeoM.Scale(2, 2)
		screen.DrawImage(g.debugImg, op)
	}
}

func (g *Game) portraitView() bool {
	return g.screenH-g.screenW >= 100
}

func (g *Game) layoutConnect() {
	g.needLayoutConnect = false

	headerHeight := etk.Scale(60)
	infoHeight := etk.Scale(108)
	if AutoEnableTouchInput {
		headerHeight = etk.Scale(20)
	}

	if ShowServerSettings {
		connectGrid.SetRowSizes(headerHeight, fieldHeight, fieldHeight, fieldHeight, etk.Scale(baseButtonHeight), etk.Scale(baseButtonHeight), infoHeight)
		registerGrid.SetRowSizes(headerHeight, fieldHeight, fieldHeight, fieldHeight, fieldHeight, etk.Scale(baseButtonHeight), infoHeight)
		resetGrid.SetRowSizes(headerHeight, fieldHeight, fieldHeight, etk.Scale(baseButtonHeight), infoHeight)
	} else {
		connectGrid.SetRowSizes(headerHeight, fieldHeight, fieldHeight, etk.Scale(baseButtonHeight), etk.Scale(baseButtonHeight), infoHeight)
		registerGrid.SetRowSizes(headerHeight, fieldHeight, fieldHeight, fieldHeight, etk.Scale(baseButtonHeight), infoHeight)
		resetGrid.SetRowSizes(headerHeight, fieldHeight, etk.Scale(baseButtonHeight), infoHeight)
	}
}

func (g *Game) layoutLobby() {
	g.needLayoutLobby = false

	g.lobby.buttonBarHeight = etk.Scale(baseButtonHeight)
	g.lobby.rebuildButtonsGrid()
	g.setBufferRects()
}

func (g *Game) layoutBoard() {
	g.needLayoutBoard = false

	if g.portraitView() { // Portrait view.
		g.Board.fullHeight = false
		g.Board.horizontalBorderSize = 0
		g.Board.setRect(0, 0, g.screenW, g.screenW)

		g.Board.uiGrid.SetRect(image.Rect(0, g.Board.h, g.screenW, g.screenH))
	} else { // Landscape view.
		g.Board.fullHeight = true
		g.Board.horizontalBorderSize = 20
		g.Board.setRect(0, 0, g.screenW-g.bufferWidth, g.screenH)

		availableWidth := g.screenW - (g.Board.innerW + int(g.Board.horizontalBorderSize*2))
		if availableWidth > g.bufferWidth {
			g.bufferWidth = availableWidth
			g.Board.setRect(0, 0, g.screenW-g.bufferWidth, g.screenH)
		}

		if g.Board.h > g.Board.w {
			g.Board.fullHeight = false
			g.Board.setRect(0, 0, g.Board.w, g.Board.w)
		}

		g.Board.uiGrid.SetRect(image.Rect(g.Board.w, 0, g.screenW, g.screenH))
	}

	g.setBufferRects()

	g.Board.widget.SetRect(image.Rect(0, 0, g.screenW, g.screenH))
}

func (g *Game) bufferPadding() int {
	if g.bufferWidth > 200 {
		return 8
	} else if g.bufferWidth > 100 {
		return 4
	} else {
		return 2
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	g.Lock()
	defer g.Unlock()

	if !g.initialized {
		g.initialize()
		g.initialized = true
	}

	originalWidth, originalHeight := outsideWidth, outsideHeight

	outsideWidth, outsideHeight = etk.Scale(outsideWidth), etk.Scale(outsideHeight)
	if outsideWidth < minWidth {
		outsideWidth = minWidth
	}
	if outsideHeight < minHeight {
		outsideHeight = minHeight
	}
	if g.screenW == outsideWidth && g.screenH == outsideHeight && !g.forceLayout {
		return outsideWidth, outsideHeight
	}
	g.forceLayout = false

	g.screenW, g.screenH = outsideWidth, outsideHeight
	scheduleFrame()

	fontMutex.Lock()
	g.bufferWidth = etk.BoundString(etk.FontFace(etk.Style.TextFont, etk.Scale(g.Board.fontSize)), strings.Repeat("A", bufferCharacterWidth)).Dx()
	fontMutex.Unlock()
	if g.bufferWidth > int(float64(g.screenW)*maxStatusWidthRatio) {
		g.bufferWidth = int(float64(g.screenW) * maxStatusWidthRatio)
	}

	etk.Layout(originalWidth, originalHeight)

	g.needLayoutConnect = true
	g.needLayoutLobby = true
	g.needLayoutBoard = true
	if !g.loggedIn {
		g.layoutConnect()
	} else if !viewBoard {
		g.layoutLobby()
	} else {
		g.layoutBoard()
	}

	padding := g.bufferPadding()
	statusBuffer.SetPadding(padding)
	gameBuffer.SetPadding(padding)
	inputBuffer.SetPadding(padding)

	old := viewBoard
	viewBoard = !old
	setViewBoard(old)

	g.Board.updateOpponentLabel()
	g.Board.updatePlayerLabel()

	g.keyboard.SetRect(image.Rect(0, game.screenH-game.screenH/3, game.screenW, game.screenH))

	if g.LoadReplay != nil {
		go g.HandleReplay(g.LoadReplay)
		g.LoadReplay = nil
	}

	return outsideWidth, outsideHeight
}

func acceptInput(text string) (handled bool) {
	if len(text) == 0 {
		return true
	}

	if text[0] == '/' {
		text = text[1:]
		if strings.ToLower(text) == "download" {
			if game.replay {
				err := saveReplay(-1, game.replayData)
				if err != nil {
					l("*** " + gotext.Get("Failed to download replay: %s", err))
				}
			} else {
				if game.downloadReplay == 0 {
					game.downloadReplay = -1
					game.Client.Out <- []byte("replay")
				} else {
					l("*** " + gotext.Get("Replay download already in progress."))
				}
			}
			return true
		}
	} else {
		l(fmt.Sprintf("<%s> %s", game.Client.Username, text))
		text = "say " + text
	}

	game.Client.Out <- []byte(text)
	go hideKeyboard()
	return true
}

func (g *Game) itemHeight() int {
	return etk.Scale(48)
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
	_ = g.cpuProfile.Close()
	g.cpuProfile = nil

	log.Println("Profiling stopped")
	return nil
}

func (g *Game) Exit() {
	os.Exit(0)
}

type SoundEffect int

const (
	effectJoinLeave SoundEffect = iota
	effectSay
	effectDie
	effectDice
	effectMove
)

var (
	dieSounds      [][]byte
	dieSoundPlays  int
	diceSounds     [][]byte
	diceSoundPlays int
	moveSounds     [][]byte
	moveSoundPlays int
)

func playSoundEffect(effect SoundEffect) {
	if game.volume == 0 || game.replay {
		return
	}

	var b []byte
	switch effect {
	case effectSay:
		b = SoundSay
	case effectJoinLeave:
		b = SoundJoinLeave
	case effectDie:
		b = dieSounds[dieSoundPlays]

		dieSoundPlays++
		if dieSoundPlays == len(dieSounds)-1 {
			randomizeByteSlice(dieSounds)
			dieSoundPlays = 0
		}
	case effectDice:
		b = diceSounds[diceSoundPlays]

		diceSoundPlays++
		if diceSoundPlays == len(diceSounds)-1 {
			randomizeByteSlice(diceSounds)
			diceSoundPlays = 0
		}
	case effectMove:
		b = moveSounds[moveSoundPlays]

		moveSoundPlays++
		if moveSoundPlays == len(moveSounds)-1 {
			randomizeByteSlice(moveSounds)
			moveSoundPlays = 0
		}
	default:
		log.Panicf("unknown sound effect: %d", effect)
		return
	}

	stream, err := vorbis.DecodeWithoutResampling(bytes.NewReader(b))
	if err != nil {
		panic(err)
	}

	player, err := audioContext.NewPlayer(&oggStream{stream})
	if err != nil {
		panic(err)
	}

	if effect == effectSay {
		player.SetVolume(game.volume / 2)
	} else {
		player.SetVolume(game.volume)
	}

	player.Play()
}

func randomizeByteSlice(b [][]byte) {
	for i := range b {
		j := rand.Intn(i + 1)
		b[i], b[j] = b[j], b[i]
	}
}

func LoadLocale(forceLanguage *language.Tag) error {
	entries, err := assetFS.ReadDir("locales")
	if err != nil {
		return err
	}

	var availableTags = []language.Tag{
		language.MustParse("en_US"),
	}
	var availableNames = []string{
		"",
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		availableTags = append(availableTags, language.MustParse(entry.Name()))
		availableNames = append(availableNames, entry.Name())
	}

	var preferred = []language.Tag{}
	if forceLanguage != nil {
		preferred = append(preferred, *forceLanguage)
	} else {
		systemLocale := os.Getenv("LANG")
		if systemLocale != "" {
			tag, err := language.Parse(systemLocale)
			if err == nil {
				preferred = append(preferred, tag)
			}
		}
	}

	useLanguage, index, _ := language.NewMatcher(availableTags).Match(preferred...)
	useLanguageCode := useLanguage.String()
	if index <= 0 || useLanguageCode == "" || strings.HasPrefix(useLanguageCode, "en") {
		return nil
	}
	useLanguageName := availableNames[index]

	b, err := assetFS.ReadFile(fmt.Sprintf("locales/%s/%s.po", useLanguageName, useLanguageName))
	if err != nil {
		return nil
	}

	po := gotext.NewPo()
	po.Parse(b)
	gotext.GetStorage().AddTranslator("boxcars", po)

	AppLanguage = useLanguageName
	return nil
}

type Input struct {
	*etk.Input
}

func (i *Input) HandleMouse(cursor image.Point, pressed bool, clicked bool) (handled bool, err error) {
	if clicked {
		go showKeyboard()
	}
	return i.Input.HandleMouse(cursor, pressed, clicked)
}

type ClickableText struct {
	*etk.Text
	onSelected func()
}

func (t *ClickableText) HandleMouse(cursor image.Point, pressed bool, clicked bool) (handled bool, err error) {
	if clicked {
		t.onSelected()
	}
	return true, nil
}

func newCenteredText(text string) *etk.Text {
	t := etk.NewText(text)
	t.SetVertical(etk.AlignCenter)
	return t
}

func centerInput(input *Input) {
	input.SetVertical(etk.AlignCenter)
	input.SetPadding(etk.Scale(5))
}

// Short description.
var _ = gotext.Get("Play backgammon online via bgammon.org")

// Long description.
var _ = gotext.Get("Boxcars is a client for playing backgammon via bgammon.org, a free and open source backgammon service.")

// This string is used when targetting WebAssembly and Android.
var _ = gotext.Get("To download this replay visit")
