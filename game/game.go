package game

import (
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
	"strings"
	"sync"
	"time"

	"code.rocket9labs.com/tslocum/bgammon"
	"code.rocket9labs.com/tslocum/bgammon-tabula-bot/bot"
	"code.rocket9labs.com/tslocum/bgammon/pkg/server"
	"code.rocket9labs.com/tslocum/etk"
	"code.rocketnine.space/tslocum/kibodo"
	"code.rocketnine.space/tslocum/messeji"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/leonelquinteros/gotext"
	"github.com/nfnt/resize"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/text/language"
)

const version = "v1.1.6"

const MaxDebug = 2

const baseButtonHeight = 56

var onlyNumbers = regexp.MustCompile(`[0-9]+`)

//go:embed asset locales
var assetFS embed.FS

var debugExtra []byte

var (
	imgCheckerLight *ebiten.Image
	//imgCheckerDark  *ebiten.Image

	imgDice  *ebiten.Image
	imgDice1 *ebiten.Image
	imgDice2 *ebiten.Image
	imgDice3 *ebiten.Image
	imgDice4 *ebiten.Image
	imgDice5 *ebiten.Image
	imgDice6 *ebiten.Image

	smallFont  font.Face
	mediumFont font.Face
	largeFont  font.Face

	fontMutex = &sync.Mutex{}
)

var (
	lightCheckerColor = color.RGBA{232, 211, 162, 255}
	darkCheckerColor  = color.RGBA{0, 0, 0, 255}
)

const maxStatusWidthRatio = 0.5

const bufferCharacterWidth = 28

const fieldHeight = 50

const (
	minWidth  = 320
	minHeight = 240
)

const (
	smallFontSize  = 20
	mediumFontSize = 24
	largeFontSize  = 36
)

var (
	bufferTextColor       = triangleALight
	bufferBackgroundColor = color.RGBA{40, 24, 9, 255}
)

var (
	statusBuffer      = etk.NewText("")
	floatStatusBuffer = etk.NewText("")
	gameBuffer        = etk.NewText("")
	inputBuffer       = etk.NewInput("", "", acceptInput)

	statusLogged bool
	gameLogged   bool

	lobbyStatusBufferHeight = 75

	Debug int

	game *Game

	diceSize int

	connectGrid    *etk.Grid
	createGameGrid *etk.Grid
	joinGameGrid   *etk.Grid

	createGameContainer *etk.Grid
	joinGameContainer   *etk.Grid
	listGamesContainer  *etk.Grid

	createGameFrame *etk.Frame
	joinGameFrame   *etk.Frame
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

func l(s string) {
	m := time.Now().Format("3:04") + " " + s
	if statusLogged {
		_, _ = statusBuffer.Write([]byte("\n" + m))
		_, _ = floatStatusBuffer.Write([]byte("\n" + m))
		scheduleFrame()
		return
	}
	_, _ = statusBuffer.Write([]byte(m))
	_, _ = floatStatusBuffer.Write([]byte(m))
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

func init() {
	gotext.SetDomain("boxcars")

	initializeFonts()

	loadAudioAssets()

	etk.Style.TextFont = largeFont
	etk.Style.TextFontMutex = fontMutex

	etk.Style.TextColorLight = triangleA
	etk.Style.TextColorDark = triangleA
	etk.Style.InputBgColor = color.RGBA{40, 24, 9, 255}

	etk.Style.ScrollAreaColor = color.RGBA{26, 15, 6, 255}
	etk.Style.ScrollHandleColor = color.RGBA{180, 154, 108, 255}

	etk.Style.ButtonTextColor = color.RGBA{0, 0, 0, 255}
	etk.Style.ButtonBgColor = color.RGBA{225, 188, 125, 255}

	statusBuffer.SetForegroundColor(bufferTextColor)
	statusBuffer.SetBackgroundColor(bufferBackgroundColor)

	floatStatusBuffer.SetForegroundColor(bufferTextColor)
	floatStatusBuffer.SetBackgroundColor(bufferBackgroundColor)

	gameBuffer.SetForegroundColor(bufferTextColor)
	gameBuffer.SetBackgroundColor(bufferBackgroundColor)

	inputBuffer.Field.SetForegroundColor(bufferTextColor)
	inputBuffer.Field.SetBackgroundColor(bufferBackgroundColor)
	inputBuffer.Field.SetSuffix("")
}

var loadedCheckerWidth = -1

func loadImageAssets(width int) {
	if width == loadedCheckerWidth {
		return
	}
	loadedCheckerWidth = width

	imgCheckerLight = loadAsset("asset/image/checker_white.png", width)
	//imgCheckerDark = loadAsset("asset/image/checker_white.png", width)
	//imgCheckerDark = loadAsset("assets/checker_black.png", width)

	resizeDice := func(img image.Image) *ebiten.Image {
		if game == nil {
			panic("nil game")
		}

		maxSize := game.scale(100)
		if maxSize > game.screenW/10 {
			maxSize = game.screenW / 10
		}
		if maxSize > game.screenH/10 {
			maxSize = game.screenH / 10
		}

		diceSize = game.scale(width)
		if diceSize > maxSize {
			diceSize = maxSize
		}
		return ebiten.NewImageFromImage(resize.Resize(uint(diceSize), 0, img, resize.Lanczos3))
	}

	const size = 184
	imgDice = ebiten.NewImageFromImage(loadImage("asset/image/dice.png"))
	imgDice1 = resizeDice(imgDice.SubImage(image.Rect(0, 0, size*1, size*1)))
	imgDice2 = resizeDice(imgDice.SubImage(image.Rect(size*1, 0, size*2, size*1)))
	imgDice3 = resizeDice(imgDice.SubImage(image.Rect(size*2, 0, size*3, size*1)))
	imgDice4 = resizeDice(imgDice.SubImage(image.Rect(0, size*1, size*1, size*2)))
	imgDice5 = resizeDice(imgDice.SubImage(image.Rect(size*1, size*1, size*2, size*2)))
	imgDice6 = resizeDice(imgDice.SubImage(image.Rect(size*2, size*1, size*3, size*2)))
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

	g := game
	if g.keyboard.Visible() || g.Board.floatChatGrid.Visible() {
		g.keyboard.Hide()
		g.Board.floatChatGrid.SetVisible(false)
		g.connectKeyboardButton.Label.SetText(gotext.Get("Show Keyboard"))
		g.lobby.showKeyboardButton.Label.SetText(gotext.Get("Show Keyboard"))
		g.Board.showKeyboardButton.Label.SetText(gotext.Get("Show Keyboard"))
	}

	game.Board.selectRollGrid.SetVisible(false)

	if viewBoard {
		// Exit dialogs.
		game.lobby.showJoinGame = false
		game.lobby.showCreateGame = false
		game.lobby.createGameName.Field.SetText("")
		game.lobby.createGamePassword.Field.SetText("")
		game.lobby.bufferDirty = true
		game.lobby.rebuildButtonsGrid()

		etk.SetRoot(game.Board.frame)
		etk.SetFocus(inputBuffer)

		game.Board.uiGrid.SetRect(game.Board.uiGrid.Rect())
	} else {
		if !game.loggedIn {
			game.setRoot(connectGrid)
		} else if game.lobby.showCreateGame {
			game.setRoot(createGameFrame)
		} else if game.lobby.showJoinGame {
			game.setRoot(joinGameFrame)
		} else {
			game.setRoot(listGamesFrame)
		}

		game.Board.menuGrid.SetVisible(false)
		game.Board.settingsGrid.SetVisible(false)
		game.Board.leaveGameGrid.SetVisible(false)

		statusBuffer.SetRect(statusBuffer.Rect())
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

var spinner = []byte(`-\|/`)

var viewBoard bool // View board or lobby

var (
	drawScreen     int
	updatedGame    bool
	gameUpdateLock = &sync.Mutex{}
)

func scheduleFrame() {
	gameUpdateLock.Lock()
	defer gameUpdateLock.Unlock()

	updatedGame = false
	drawScreen = 2
}

type Game struct {
	screenW, screenH int

	drawBuffer bytes.Buffer

	spinnerIndex int

	ServerAddress string
	Username      string
	Password      string
	loggedIn      bool

	Watch bool
	TV    bool

	Client *Client

	Board *board

	lobby *lobby

	volume float64 // Volume range is 0-1.

	runeBuffer []rune

	debugImg *ebiten.Image

	keyboard      *kibodo.Keyboard
	keyboardInput []*kibodo.Input

	cpuProfile *os.File

	connectUsername       *etk.Input
	connectPassword       *etk.Input
	connectServer         *etk.Input
	connectKeyboardButton *etk.Button

	pressedKeys []ebiten.Key

	cursorX, cursorY int
	TouchInput       bool

	rootWidget etk.Widget

	touchIDs []ebiten.TouchID

	lastRefresh time.Time

	skipUpdate  bool
	forceLayout bool

	scaleFactor float64

	bufferWidth int

	Instant bool

	needLayoutConnect bool
	needLayoutLobby   bool
	needLayoutBoard   bool

	loadedConnect bool
	loadedLobby   bool
	loadedBoard   bool

	loaded bool

	*sync.Mutex
}

func NewGame() *Game {
	ebiten.SetVsyncEnabled(true)
	ebiten.SetScreenClearedEveryFrame(false)
	ebiten.SetTPS(144)
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetWindowClosingHandled(true)

	g := &Game{
		runeBuffer: make([]rune, 24),

		keyboard: kibodo.NewKeyboard(),

		TouchInput: AutoEnableTouchInput,

		debugImg:    ebiten.NewImage(200, 200),
		volume:      1,
		scaleFactor: 1,

		Mutex: &sync.Mutex{},
	}
	game = g

	loadImageAssets(0)

	g.Board = NewBoard()
	g.lobby = NewLobby()

	if AutoEnableTouchInput {
		g.keyboard.SetKeys(kibodo.KeysMobileQWERTY)
		g.keyboard.SetExtendedKeys(kibodo.KeysMobileSymbols)
	} else {
		g.keyboard.SetKeys(kibodo.KeysQWERTY)
	}

	{
		headerLabel := etk.NewText(gotext.Get("Welcome to %s", "bgammon.org"))
		nameLabel := etk.NewText(gotext.Get("Username"))
		passwordLabel := etk.NewText(gotext.Get("Password"))

		g.connectKeyboardButton = etk.NewButton(gotext.Get("Show Keyboard"), func() error {
			if g.keyboard.Visible() {
				g.keyboard.Hide()
				g.connectKeyboardButton.Label.SetText(gotext.Get("Show Keyboard"))
				g.lobby.showKeyboardButton.Label.SetText(gotext.Get("Show Keyboard"))
				g.Board.showKeyboardButton.Label.SetText(gotext.Get("Show Keyboard"))
			} else {
				g.EnableTouchInput()
				g.keyboard.Show()
				g.connectKeyboardButton.Label.SetText(gotext.Get("Hide Keyboard"))
			}
			return nil
		})

		infoLabel := etk.NewText(gotext.Get("To log in as a guest, enter a username (if you want) and do not enter a password."))

		footerLabel := etk.NewText("Boxcars " + version)
		footerLabel.SetHorizontal(messeji.AlignEnd)
		footerLabel.SetVertical(messeji.AlignEnd)

		g.connectUsername = etk.NewInput("", "", func(text string) (handled bool) {
			return false
		})
		centerInput(g.connectUsername)

		g.connectPassword = etk.NewInput("", "", func(text string) (handled bool) {
			return false
		})
		centerInput(g.connectPassword)

		connectButton := etk.NewButton(gotext.Get("Connect"), func() error {
			g.selectConnect()
			return nil
		})

		offlineButton := etk.NewButton(gotext.Get("Play Offline"), func() error {
			g.playOffline()
			return nil
		})

		grid := etk.NewGrid()
		grid.SetColumnPadding(int(g.Board.horizontalBorderSize / 2))
		grid.SetRowPadding(20)
		grid.SetColumnSizes(10, 200, -1, -1, 10)
		grid.AddChildAt(headerLabel, 0, 0, 4, 1)
		grid.AddChildAt(etk.NewBox(), 4, 0, 1, 1)
		grid.AddChildAt(nameLabel, 1, 1, 2, 1)
		grid.AddChildAt(g.connectUsername, 2, 1, 2, 1)
		grid.AddChildAt(passwordLabel, 1, 2, 2, 1)
		grid.AddChildAt(g.connectPassword, 2, 2, 2, 1)
		y := 3
		if ShowServerSettings {
			connectAddress := game.ServerAddress
			if connectAddress == "" {
				connectAddress = DefaultServerAddress
			}
			g.connectServer = etk.NewInput("", connectAddress, func(text string) (handled bool) {
				return false
			})
			centerInput(g.connectServer)
			grid.AddChildAt(etk.NewText(gotext.Get("Server")), 1, y, 2, 1)
			grid.AddChildAt(g.connectServer, 2, y, 2, 1)
			y++
		}
		grid.AddChildAt(infoLabel, 1, y, 3, 1)
		grid.AddChildAt(connectButton, 2, y+1, 1, 1)
		grid.AddChildAt(offlineButton, 3, y+1, 1, 1)
		grid.AddChildAt(footerLabel, 1, y+2, 3, 1)
		connectGrid = grid
	}

	{
		headerLabel := etk.NewText(gotext.Get("Create match"))
		nameLabel := etk.NewText(gotext.Get("Name"))
		pointsLabel := etk.NewText(gotext.Get("Points"))
		passwordLabel := etk.NewText(gotext.Get("Password"))
		variantLabel := etk.NewText(gotext.Get("Variant"))

		g.lobby.createGameName = etk.NewInput("", "", func(text string) (handled bool) {
			return false
		})
		centerInput(g.lobby.createGameName)

		g.lobby.createGamePoints = etk.NewInput("", "", func(text string) (handled bool) {
			return false
		})
		centerInput(g.lobby.createGamePoints)

		g.lobby.createGamePassword = etk.NewInput("", "", func(text string) (handled bool) {
			return false
		})
		centerInput(g.lobby.createGamePassword)

		g.lobby.createGameCheckbox = etk.NewCheckbox(g.lobby.toggleAceyDeucey)
		g.lobby.createGameCheckbox.SetBorderColor(triangleA)
		g.lobby.createGameCheckbox.SetCheckColor(triangleA)
		g.lobby.createGameCheckbox.SetSelected(false)

		textField := etk.NewText(gotext.Get("Acey-deucey"))
		aceyDeuceyLabel := &ClickableText{
			Text: textField,
			onSelected: func() {
				g.lobby.createGameCheckbox.SetSelected(!g.lobby.createGameCheckbox.Selected())
				g.lobby.toggleAceyDeucey()
			},
		}
		aceyDeuceyLabel.SetVertical(messeji.AlignCenter)

		aceyDeuceyGrid := etk.NewGrid()
		aceyDeuceyGrid.SetColumnSizes(50, -1)
		aceyDeuceyGrid.SetRowSizes(50, -1)
		aceyDeuceyGrid.AddChildAt(g.lobby.createGameCheckbox, 0, 0, 1, 1)
		aceyDeuceyGrid.AddChildAt(aceyDeuceyLabel, 1, 0, 1, 1)

		grid := etk.NewGrid()
		grid.SetColumnPadding(int(g.Board.horizontalBorderSize / 2))
		grid.SetRowPadding(20)
		grid.SetColumnSizes(10, 200, -1, 10)
		grid.SetRowSizes(60, fieldHeight, fieldHeight, fieldHeight)
		grid.AddChildAt(headerLabel, 0, 0, 3, 1)
		grid.AddChildAt(etk.NewBox(), 3, 0, 1, 1)
		grid.AddChildAt(nameLabel, 1, 1, 1, 1)
		grid.AddChildAt(g.lobby.createGameName, 2, 1, 1, 1)
		grid.AddChildAt(pointsLabel, 1, 2, 1, 1)
		grid.AddChildAt(g.lobby.createGamePoints, 2, 2, 1, 1)
		grid.AddChildAt(passwordLabel, 1, 3, 1, 1)
		grid.AddChildAt(g.lobby.createGamePassword, 2, 3, 1, 1)
		grid.AddChildAt(variantLabel, 1, 4, 1, 1)
		grid.AddChildAt(aceyDeuceyGrid, 2, 4, 1, 1)
		createGameGrid = grid

		createGameContainer = etk.NewGrid()
		createGameContainer.AddChildAt(createGameGrid, 0, 0, 1, 1)
		createGameContainer.AddChildAt(statusBuffer, 0, 1, 1, 1)
		createGameContainer.AddChildAt(g.lobby.buttonsGrid, 0, 2, 1, 1)

		createGameFrame = etk.NewFrame()
		createGameFrame.SetPositionChildren(true)
		createGameFrame.AddChild(createGameContainer)
		createGameFrame.AddChild(etk.NewFrame(g.lobby.showKeyboardButton))
	}

	{
		g.lobby.joinGameLabel = etk.NewText(gotext.Get("Join match"))

		passwordLabel := etk.NewText(gotext.Get("Password"))

		g.lobby.joinGamePassword = etk.NewInput("", "", func(text string) (handled bool) {
			return false
		})
		centerInput(g.lobby.joinGamePassword)

		grid := etk.NewGrid()
		grid.SetColumnPadding(int(g.Board.horizontalBorderSize / 2))
		grid.SetRowPadding(20)
		grid.SetColumnSizes(10, 200, -1, 10)
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
		joinGameFrame.AddChild(etk.NewFrame(g.lobby.showKeyboardButton))
	}

	{
		listGamesFrame = etk.NewFrame()

		g.lobby.rebuildButtonsGrid()

		listGamesContainer = etk.NewGrid()
		listGamesContainer.AddChildAt(etk.NewBox(), 0, 0, 1, 1)
		listGamesContainer.AddChildAt(statusBuffer, 0, 1, 1, 1)
		listGamesContainer.AddChildAt(g.lobby.buttonsGrid, 0, 2, 1, 1)

		listGamesFrame.SetPositionChildren(true)
		listGamesFrame.AddChild(listGamesContainer)
	}

	g.needLayoutConnect = true
	g.needLayoutLobby = true
	g.needLayoutBoard = true

	g.setRoot(connectGrid)
	etk.SetFocus(g.connectUsername)

	go g.handleAutoRefresh()
	go g.handleUpdateTimeLabels()

	scheduleFrame()
	return g
}

func (g *Game) playOffline() {
	if g.loggedIn {
		return
	}

	// Start the local server.
	server := server.NewServer("", "", false, false)
	conns := server.ListenLocal()

	// Connect the bot.
	botConn := <-conns
	go bot.NewLocalClient(botConn, "", "BOT_tabula", "", 1, 0)

	// Wait for the bot to finish creating a match.
	time.Sleep(10 * time.Millisecond)

	// Connect the player.
	playerConn := <-conns
	go g.ConnectLocal(playerConn)
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
	etk.SetRoot(w)
}

func (g *Game) setBufferRects() {
	statusBufferHeight := g.scale(75)

	createGameContainer.SetRowSizes(-1, statusBufferHeight, g.lobby.buttonBarHeight)
	joinGameContainer.SetRowSizes(-1, statusBufferHeight, g.lobby.buttonBarHeight)
	listGamesContainer.SetRowSizes(-1, statusBufferHeight, g.lobby.buttonBarHeight)
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
		g.Board.processState()
		g.Board.Unlock()
		setViewBoard(true)

		if ev.Player == g.Client.Username {
			gameBuffer.SetText("")
			gameLogged = false
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
		g.Board.stateLock.Unlock()
		g.Board.processState()
		g.Board.Unlock()
		setViewBoard(true)
	case *bgammon.EventRolled:
		g.Board.Lock()
		g.Board.stateLock.Lock()
		g.Board.gameState.Roll1 = ev.Roll1
		g.Board.gameState.Roll2 = ev.Roll2
		var diceFormatted string
		if g.Board.gameState.Turn == 0 {
			if g.Board.gameState.Player1.Name == ev.Player {
				diceFormatted = fmt.Sprintf("%d", g.Board.gameState.Roll1)
			} else {
				diceFormatted = fmt.Sprintf("%d", g.Board.gameState.Roll2)
			}
			playSoundEffect(effectDie)
		} else {
			diceFormatted = fmt.Sprintf("%d-%d", g.Board.gameState.Roll1, g.Board.gameState.Roll2)
			playSoundEffect(effectDice)
		}
		g.Board.stateLock.Unlock()
		g.Board.processState()
		g.Board.Unlock()
		scheduleFrame()
		lg(gotext.Get("%s rolled %s.", ev.Player, diceFormatted))
	case *bgammon.EventFailedRoll:
		l(fmt.Sprintf("*** Failed to roll: %s", ev.Reason))
	case *bgammon.EventMoved:
		lg(gotext.Get("%s moved %s.", ev.Player, bgammon.FormatMoves(ev.Moves)))
		if ev.Player == g.Client.Username {
			return
		}
		g.Board.Lock()
		g.Unlock()
		for _, move := range ev.Moves {
			playSoundEffect(effectMove)
			g.Board.movePiece(move[0], move[1])
		}
		g.Lock()
		g.Board.Unlock()
	case *bgammon.EventFailedMove:
		g.Client.Out <- []byte("board") // Refresh game state.

		var extra string
		if ev.From != 0 || ev.To != 0 {
			extra = fmt.Sprintf(" from %s to %s", bgammon.FormatSpace(ev.From), bgammon.FormatSpace(ev.To))
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
			lg(gotext.Get("Type %s to offer a rematch.", "/rematch"))
		}
		g.Board.Unlock()
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

func (g *Game) Connect() {
	if g.loggedIn {
		return
	}
	g.loggedIn = true

	l("*** " + gotext.Get("Connecting..."))

	g.keyboard.Hide()
	g.connectKeyboardButton.Label.SetText(gotext.Get("Show Keyboard"))
	g.lobby.showKeyboardButton.Label.SetText(gotext.Get("Show Keyboard"))
	g.Board.showKeyboardButton.Label.SetText(gotext.Get("Show Keyboard"))

	g.setRoot(listGamesFrame)

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

func (g *Game) ConnectLocal(conn net.Conn) {
	if g.loggedIn {
		return
	}
	g.loggedIn = true

	l("*** " + gotext.Get("Playing offline."))

	g.keyboard.Hide()
	g.connectKeyboardButton.Label.SetText(gotext.Get("Show Keyboard"))
	g.lobby.showKeyboardButton.Label.SetText(gotext.Get("Show Keyboard"))
	g.Board.showKeyboardButton.Label.SetText(gotext.Get("Show Keyboard"))

	g.setRoot(listGamesFrame)

	g.Client = newClient("", g.Username, g.Password)
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

	go c.connectTCP(conn)
}

func (g *Game) selectConnect() error {
	g.Username = g.connectUsername.Text()
	g.Password = g.connectPassword.Text()
	if ShowServerSettings {
		g.ServerAddress = g.connectServer.Text()
	}
	g.Connect()
	return nil
}

func (g *Game) handleInput(keys []ebiten.Key) error {
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
				}
			case ebiten.KeyEnter, ebiten.KeyKPEnter:
				g.selectConnect()
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

	if !viewBoard && (g.lobby.showCreateGame || g.lobby.showJoinGame) {
		for _, key := range keys {
			if key == ebiten.KeyEnter || key == ebiten.KeyKPEnter {
				if g.lobby.showCreateGame {
					g.lobby.confirmCreateGame()
				} else {
					g.lobby.confirmJoinGame()
				}
			}
		}
	}
	return nil
}

func (g *Game) handleTouch(p image.Point) {
	if p.X == 0 && p.Y == 0 {
		return
	}
	w := etk.At(p)
	if w == nil {
		return
	}
	switch w.(type) {
	case *etk.Input:
		g.keyboard.Show()
		var btn *etk.Button
		if !g.loggedIn {
			btn = g.connectKeyboardButton
		} else if !viewBoard {
			btn = g.lobby.showKeyboardButton
		} else {
			btn = g.Board.showKeyboardButton
			g.Board.lastKeyboardToggle = time.Now()
			g.Board.floatChatGrid.SetVisible(true)
			g.Board.floatChatGrid.SetRect(g.Board.floatChatGrid.Rect())
		}
		btn.Label.SetText(gotext.Get("Hide Keyboard"))
	}
}

// Update is called by Ebitengine only when user input occurs, or a frame is
// explicitly scheduled.
func (g *Game) Update() error {
	if ebiten.IsWindowBeingClosed() {
		g.Exit()
		return nil
	}

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
	if cx != g.cursorX || cy != g.cursorY {
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

	// Handle physical keyboard.
	g.pressedKeys = inpututil.AppendJustPressedKeys(g.pressedKeys[:0])
	err := g.handleInput(g.pressedKeys)
	if err != nil {
		return err
	}

	// Handle on-screen keyboard.
	err = g.keyboard.Update()
	if err != nil {
		return err
	}
	g.keyboardInput = g.keyboard.AppendInput(g.keyboardInput[:0])
	g.pressedKeys = g.pressedKeys[:0]
	for _, input := range g.keyboardInput {
		if input.Rune == 0 {
			g.pressedKeys = append(g.pressedKeys, input.Key)
		}
	}
	if len(g.pressedKeys) > 0 {
		err = g.handleInput(g.pressedKeys)
		if err != nil {
			return err
		}
	}

	var pressed bool
	var pressedTouch image.Point
	if cx == 0 && cy == 0 {
		g.touchIDs = inpututil.AppendJustPressedTouchIDs(g.touchIDs[:0])
		for _, id := range g.touchIDs {
			game.EnableTouchInput()
			cx, cy = ebiten.TouchPosition(id)
			if cx != 0 || cy != 0 {
				pressed = true
				pressedTouch = image.Point{cx, cy}
				break
			}
		}
	} else {
		pressed = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	}
	var skipUpdate bool
	if pressed && g.keyboard.Visible() {
		p := image.Point{X: cx, Y: cy}
		if p.In(g.keyboard.Rect()) {
			skipUpdate = true
			g.skipUpdate = true
		}
	}
	if g.skipUpdate && !skipUpdate {
		skipUpdate = true
		g.skipUpdate = false
	}

	if !g.loggedIn {
		if len(g.keyboardInput) > 0 {
			w := etk.Focused()
			if w != nil {
				for _, event := range g.keyboardInput {
					if event.Rune > 0 {
						w.HandleKeyboard(-1, event.Rune)
					} else {
						w.HandleKeyboard(event.Key, 0)
					}
				}
			}
		}

		if skipUpdate {
			return nil
		}
		err = etk.Update()
		if err != nil {
			return err
		}
		g.handleTouch(pressedTouch)
		return nil
	}

	if !viewBoard {
		g.lobby.update()

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

			w := etk.Focused()
			if w != nil {
				for _, event := range g.keyboardInput {
					if event.Rune > 0 {
						w.HandleKeyboard(-1, event.Rune)
					} else {
						w.HandleKeyboard(event.Key, 0)
					}
				}
			}

			if g.lobby.showCreateGame {
				pointsText := g.lobby.createGamePoints.Text()
				if pointsText != "" {
					g.lobby.createGamePoints.Field.SetText(strings.Join(onlyNumbers.FindAllString(pointsText, -1), ""))
				}
			}
		}
	} else {
		g.Board.Update()

		for _, event := range g.keyboardInput {
			if event.Rune > 0 {
				inputBuffer.HandleKeyboard(-1, event.Rune)
			} else {
				inputBuffer.HandleKeyboard(event.Key, 0)
			}
		}
	}

	if skipUpdate {
		return nil
	}
	err = etk.Update()
	if err != nil {
		return err
	}
	g.handleTouch(pressedTouch)
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

	if OptimizeDraw {
		gameUpdateLock.Lock()
		if drawScreen <= 0 {
			gameUpdateLock.Unlock()
			return
		} else if updatedGame {
			drawScreen -= 1
		}
		gameUpdateLock.Unlock()
	}

	screen.Fill(tableColor)

	// Log in screen
	if !g.loggedIn {
		err := etk.Draw(screen)
		if err != nil {
			log.Fatal(err)
		}
		game.keyboard.Draw(screen)
		return
	}

	if !viewBoard { // Lobby
		g.lobby.draw(screen)
	} else { // Game board
		g.Board.Draw(screen)
	}

	err := etk.Draw(screen)
	if err != nil {
		log.Fatal(err)
	}

	g.Board.drawDraggedCheckers(screen)

	game.keyboard.Draw(screen)

	if Debug > 0 {
		g.drawBuffer.Reset()

		g.spinnerIndex++
		if g.spinnerIndex == 4 {
			g.spinnerIndex = 0
		}

		if g.scaleFactor != 1.0 {
			g.drawBuffer.Write([]byte(fmt.Sprintf("SCA %0.1f\n", g.scaleFactor)))
		}

		g.drawBuffer.Write([]byte(fmt.Sprintf("FPS %c %0.0f", spinner[g.spinnerIndex], ebiten.ActualFPS())))

		if debugExtra != nil {
			g.drawBuffer.WriteRune('\n')
			g.drawBuffer.Write(debugExtra)
		}

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

func (g *Game) scale(v int) int {
	return int(float64(v) * g.scaleFactor)
}

func (g *Game) layoutConnect() {
	g.needLayoutConnect = false

	if ShowServerSettings {
		connectGrid.SetRowSizes(60, fieldHeight, fieldHeight, fieldHeight, 108, g.scale(baseButtonHeight))
	} else {
		connectGrid.SetRowSizes(60, fieldHeight, fieldHeight, 108, g.scale(baseButtonHeight))
	}

	if !g.loadedConnect {
		updateAllButtons(connectGrid)
		g.loadedConnect = true
	}
}

func (g *Game) layoutLobby() {
	g.needLayoutLobby = false

	if g.portraitView() { // Portrait view.
		g.lobby.fullscreen = true
		g.lobby.setRect(0, 0, g.screenW, g.screenH-lobbyStatusBufferHeight-g.lobby.buttonBarHeight)
	} else { // Landscape view.
		g.lobby.fullscreen = true
		g.lobby.setRect(0, 0, g.screenW, g.screenH-lobbyStatusBufferHeight-g.lobby.buttonBarHeight)
	}

	g.lobby.buttonBarHeight = g.scale(baseButtonHeight)
	g.setBufferRects()

	g.lobby.showKeyboardButton.SetVisible(g.TouchInput)
	g.lobby.showKeyboardButton.SetRect(image.Rect(g.screenW-400, 0, g.screenW, int(76)))

	if !g.loadedLobby {
		updateAllButtons(game.lobby.buttonsGrid)
		g.loadedLobby = true
	}
}

func (g *Game) layoutBoard() {
	g.needLayoutBoard = false

	if g.portraitView() { // Portrait view.
		g.Board.fullHeight = false
		g.Board.setRect(0, 0, g.screenW, g.screenW)

		g.Board.uiGrid.SetRect(image.Rect(0, g.Board.h, g.screenW, g.screenH))
	} else { // Landscape view.
		g.Board.fullHeight = true
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

		bufferPaddingX := int(g.Board.horizontalBorderSize / 2)
		g.Board.uiGrid.SetRect(image.Rect(g.Board.w+bufferPaddingX, bufferPaddingX, g.screenW-bufferPaddingX, g.screenH-bufferPaddingX))
	}

	g.setBufferRects()

	g.Board.widget.SetRect(image.Rect(0, 0, g.screenW, g.screenH))

	if !g.loadedBoard {
		updateAllButtons(game.Board.menuGrid)
		updateAllButtons(game.Board.leaveGameGrid)
		updateAllButtons(game.Board.floatChatGrid)
		g.loadedBoard = true
	}
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

	s := ebiten.DeviceScaleFactor()
	outsideWidth, outsideHeight = int(float64(outsideWidth)*s), int(float64(outsideHeight)*s)
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
	g.scaleFactor = s
	scheduleFrame()

	b := g.Board
	if s >= 1.25 {
		lobbyStatusBufferHeight = int(50 * s)
		g.Board.verticalBorderSize = baseBoardVerticalSize * 1.5
		if b.fontFace != largeFont {
			b.fontFace = largeFont
			b.fontUpdated()
		}
	} else {
		if b.fontFace != mediumFont {
			b.fontFace = mediumFont
			b.fontUpdated()
		}
	}

	statusBuffer.SetScrollBarColors(etk.Style.ScrollAreaColor, etk.Style.ScrollHandleColor)
	floatStatusBuffer.SetScrollBarColors(etk.Style.ScrollAreaColor, etk.Style.ScrollHandleColor)
	gameBuffer.SetScrollBarColors(etk.Style.ScrollAreaColor, etk.Style.ScrollHandleColor)
	inputBuffer.Field.SetScrollBarColors(etk.Style.ScrollAreaColor, etk.Style.ScrollHandleColor)

	{
		scrollBarWidth := g.scale(32)
		statusBuffer.SetScrollBarWidth(scrollBarWidth)
		floatStatusBuffer.SetScrollBarWidth(scrollBarWidth)
		gameBuffer.SetScrollBarWidth(scrollBarWidth)
		inputBuffer.Field.SetScrollBarWidth(scrollBarWidth)
	}

	fontMutex.Lock()
	g.bufferWidth = etk.BoundString(g.Board.fontFace, strings.Repeat("A", bufferCharacterWidth)).Dx()
	fontMutex.Unlock()
	if g.bufferWidth > int(float64(g.screenW)*maxStatusWidthRatio) {
		g.bufferWidth = int(float64(g.screenW) * maxStatusWidthRatio)
	}

	etk.Layout(g.screenW, g.screenH)

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
	floatStatusBuffer.SetPadding(padding)
	gameBuffer.SetPadding(padding)
	inputBuffer.Field.SetPadding(padding)

	old := viewBoard
	viewBoard = !old
	setViewBoard(old)

	g.Board.updateOpponentLabel()
	g.Board.updatePlayerLabel()

	g.keyboard.SetRect(0, game.screenH-game.screenH/3, game.screenW, game.screenH/3)

	return outsideWidth, outsideHeight
}

func acceptInput(text string) (handled bool) {
	if len(text) == 0 {
		return true
	}

	if text[0] == '/' {
		text = text[1:]
	} else {
		l(fmt.Sprintf("<%s> %s", game.Client.Username, text))
		text = "say " + text
	}

	game.Client.Out <- []byte(text)
	return true
}

func (g *Game) EnableTouchInput() {
	if g.TouchInput {
		return
	}
	g.TouchInput = true

	g.keyboard.SetKeys(kibodo.KeysMobileQWERTY)
	g.keyboard.SetExtendedKeys(kibodo.KeysMobileSymbols)

	// Update layout.
	g.forceLayout = true

	b := g.Board
	b.matchStatusGrid.Empty()
	b.matchStatusGrid.AddChildAt(b.timerLabel, 0, 0, 1, 1)
	b.matchStatusGrid.AddChildAt(b.clockLabel, 1, 0, 1, 1)

	b.fontUpdated()
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
	if game.volume == 0 {
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

	var available = []language.Tag{
		language.MustParse("en_US"),
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		available = append(available, language.MustParse(entry.Name()))
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

	useLanguage, _, _ := language.NewMatcher(available).Match(preferred...)
	useLanguageCode := useLanguage.String()
	if useLanguageCode == "" || strings.HasPrefix(useLanguageCode, "en") {
		return nil
	}

	b, err := assetFS.ReadFile(fmt.Sprintf("locales/%s/%s.po", strings.ReplaceAll(useLanguageCode, "-", "_"), strings.ReplaceAll(useLanguageCode, "-", "_")))
	if err != nil {
		return nil
	}

	po := gotext.NewPo()
	po.Parse(b)

	gotext.GetStorage().AddTranslator("boxcars", po)
	return nil
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

func centerInput(input *etk.Input) {
	input.Field.SetVertical(messeji.AlignCenter)
}

func updateAllButtons(w etk.Widget) {
	for _, c := range w.Children() {
		updateAllButtons(c)
	}

	btn, ok := w.(*etk.Button)
	if ok && btn != game.Board.showMenuButton {
		btn.Label.SetFont(largeFont, fontMutex)
	}
}

// Short description.
var _ = gotext.Get("Play backgammon online via bgammon.org")

// Long description.
var _ = gotext.Get("Boxcars is a client for playing backgammon via bgammon.org, a free and open source backgammon service.")
