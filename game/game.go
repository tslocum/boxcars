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
	"strconv"
	"strings"
	"sync"
	"time"

	"codeberg.org/tslocum/bgammon"
	"codeberg.org/tslocum/bgammon-bei-bot/bot"
	"codeberg.org/tslocum/bgammon/pkg/server"
	"codeberg.org/tslocum/etk"
	"codeberg.org/tslocum/etk/kibodo"
	"codeberg.org/tslocum/gotext"
	"codeberg.org/tslocum/tabula"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"
)

const (
	AppVersion           = "v1.4.8"
	baseButtonHeight     = 54
	MaxDebug             = 2
	DefaultServerAddress = "wss://ws.bgammon.org:1338"
)

var AppLanguage = "en"

var (
	anyNumbers  = regexp.MustCompile(`[0-9]+`)
	onlyNumbers = regexp.MustCompile(`^[0-9]+$`)
)

var resizeDuration = 250 * time.Millisecond

//go:embed asset locales
var assetFS embed.FS

var (
	imgCheckerTopLight  *ebiten.Image
	imgCheckerTopDark   *ebiten.Image
	imgCheckerSideLight *ebiten.Image
	imgCheckerSideDark  *ebiten.Image

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

	imgProfileBirthday1 *ebiten.Image

	imgIcon    *ebiten.Image
	ImgIconAlt image.Image

	fontMutex = &sync.Mutex{}
)

var (
	checkerColor = color.RGBA{232, 211, 162, 255}
)

const maxStatusWidthRatio = 0.5

const bufferCharacterWidth = 16

const (
	minWidth  = 320
	minHeight = 240
)

var (
	extraSmallFontSize  = 14
	smallFontSize       = 18
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

	newGameLogMessage   bool
	lastGameLogTime     string
	incomingGameLogRoll bool
	incomingGameLogMove bool

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
	registerFrame   *etk.Frame
	resetFrame      *etk.Frame
	createGameFrame *etk.Frame
	joinGameFrame   *etk.Frame
	historyFrame    *etk.Frame
	listGamesFrame  *etk.Frame
)

const sampleRate = 44100

var (
	audioContext *audio.Context

	SoundDie1, SoundDie2, SoundDie3                *audio.Player
	SoundDice1, SoundDice2, SoundDice3, SoundDice4 *audio.Player
	SoundMove1, SoundMove2, SoundMove3             *audio.Player
	SoundHomeSingle                                *audio.Player
	SoundHomeMulti1, SoundHomeMulti2               *audio.Player
	SoundJoinLeave                                 *audio.Player
	SoundSay                                       *audio.Player
)

func init() {
	gotext.SetDomain("boxcars")

	ImgIconAlt = _loadImage("asset/image/icon.png")
	imgIcon = ebiten.NewImageFromImage(ImgIconAlt)
}

func ls(s string) {
	m := time.Now().Format("[3:04]") + " " + s
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
	t := time.Now().Format("[3:04]")
	m := t + " " + s
	newGameLogMessage = true
	incomingGameLogRoll = false
	incomingGameLogMove = false
	lastGameLogTime = t
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

func resizeImage(source *ebiten.Image, size int) *ebiten.Image {
	if size == 0 {
		size = 1
	}
	bounds := source.Bounds()
	if bounds.Dx() == size && bounds.Dy() == size {
		return source
	}
	scale, yScale := float64(size)/float64(bounds.Dx()), float64(size)/float64(bounds.Dy())
	if yScale < scale {
		scale = yScale
	}
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	op.GeoM.Scale(scale, scale)
	dx := float64(size) - float64(bounds.Dx())*scale
	op.GeoM.Translate(dx/2, 0)
	dy := float64(size) - float64(bounds.Dy())*scale
	op.GeoM.Translate(0, dy/2)
	img := ebiten.NewImage(size, size)
	img.DrawImage(source, op)
	return img
}

func loadImageAssets(width int) {
	if width == loadedCheckerWidth {
		return
	}
	loadedCheckerWidth = width

	imgCheckerTopLight = loadAsset("asset/image/checker_top_light.png", width)
	imgCheckerTopDark = loadAsset("asset/image/checker_top_dark.png", width)
	imgCheckerSideLight = loadAsset("asset/image/checker_side_light.png", width)
	imgCheckerSideDark = loadAsset("asset/image/checker_side_dark.png", width)

	resizeDice := func(img *ebiten.Image, scale float64) *ebiten.Image {
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
		return resizeImage(img, int(float64(diceSize)*scale))
	}

	const size = 184
	imgDice = ebiten.NewImageFromImage(loadImage("asset/image/dice.png"))
	imgDice1 = resizeDice(imgDice.SubImage(image.Rect(0, 0, size*1, size*1)).(*ebiten.Image), 1)
	imgDice2 = resizeDice(imgDice.SubImage(image.Rect(size*1, 0, size*2, size*1)).(*ebiten.Image), 1)
	imgDice3 = resizeDice(imgDice.SubImage(image.Rect(size*2, 0, size*3, size*1)).(*ebiten.Image), 1)
	imgDice4 = resizeDice(imgDice.SubImage(image.Rect(0, size*1, size*1, size*2)).(*ebiten.Image), 1)
	imgDice5 = resizeDice(imgDice.SubImage(image.Rect(size*1, size*1, size*2, size*2)).(*ebiten.Image), 1)
	imgDice6 = resizeDice(imgDice.SubImage(image.Rect(size*2, size*1, size*3, size*2)).(*ebiten.Image), 1)
	imgCubes = ebiten.NewImageFromImage(loadImage("asset/image/cubes.png"))
	imgCubes2 = resizeDice(imgCubes.SubImage(image.Rect(0, 0, size*1, size*1)).(*ebiten.Image), 0.6)
	imgCubes4 = resizeDice(imgCubes.SubImage(image.Rect(size*1, 0, size*2, size*1)).(*ebiten.Image), 0.6)
	imgCubes8 = resizeDice(imgCubes.SubImage(image.Rect(size*2, 0, size*3, size*1)).(*ebiten.Image), 0.6)
	imgCubes16 = resizeDice(imgCubes.SubImage(image.Rect(0, size*1, size*1, size*2)).(*ebiten.Image), 0.6)
	imgCubes32 = resizeDice(imgCubes.SubImage(image.Rect(size*1, size*1, size*2, size*2)).(*ebiten.Image), 0.6)
	imgCubes64 = resizeDice(imgCubes.SubImage(image.Rect(size*2, size*1, size*3, size*2)).(*ebiten.Image), 0.6)

	imgProfileBirthday1 = ebiten.NewImageFromImage(loadImage("asset/image/profile_birthday_1.png"))
}

func loadAudioAssets() {
	audioContext = audio.NewContext(sampleRate)
	p := "asset/audio/"

	SoundDie1 = LoadOGG(audioContext, p+"die1.ogg")
	SoundDie2 = LoadOGG(audioContext, p+"die2.ogg")
	SoundDie3 = LoadOGG(audioContext, p+"die3.ogg")

	SoundDice1 = LoadOGG(audioContext, p+"dice1.ogg")
	SoundDice2 = LoadOGG(audioContext, p+"dice2.ogg")
	SoundDice3 = LoadOGG(audioContext, p+"dice3.ogg")
	SoundDice4 = LoadOGG(audioContext, p+"dice4.ogg")

	SoundMove1 = LoadOGG(audioContext, p+"move1.ogg")
	SoundMove2 = LoadOGG(audioContext, p+"move2.ogg")
	SoundMove3 = LoadOGG(audioContext, p+"move3.ogg")

	SoundHomeSingle = LoadOGG(audioContext, p+"homesingle.ogg")

	SoundHomeMulti1 = LoadOGG(audioContext, p+"homemulti1.ogg")
	SoundHomeMulti2 = LoadOGG(audioContext, p+"homemulti2.ogg")

	SoundJoinLeave = LoadOGG(audioContext, p+"joinleave.ogg")
	SoundSay = LoadOGG(audioContext, p+"say.ogg")

	dieSounds = []*audio.Player{
		SoundDie1,
		SoundDie2,
		SoundDie3,
	}
	randomizeSounds(dieSounds)

	diceSounds = []*audio.Player{
		SoundDice1,
		SoundDice2,
		SoundDice3,
		SoundDice4,
	}
	randomizeSounds(diceSounds)

	moveSounds = []*audio.Player{
		SoundMove1,
		SoundMove2,
		SoundMove3,
	}
	randomizeSounds(moveSounds)

	homeMultiSounds = []*audio.Player{
		SoundHomeMulti1,
		SoundHomeMulti2,
	}
	randomizeSounds(homeMultiSounds)
}

func _loadImage(assetPath string) image.Image {
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

func loadImage(assetPath string) *ebiten.Image {
	return ebiten.NewImageFromImage(_loadImage(assetPath))
}

func loadAsset(assetPath string, width int) *ebiten.Image {
	img := loadImage(assetPath)
	if width > 0 {
		return resizeImage(img, width)
	}
	return img
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

	game.board.selectRollGrid.SetVisible(false)

	if viewBoard {
		// Exit dialogs.
		game.lobby.showJoinGame = false
		game.lobby.showCreateGame = false
		game.lobby.createGameName.SetText("")
		game.lobby.createGamePassword.SetText("")
		game.lobby.rebuildButtonsGrid()

		game.setRoot(game.board.frame)
		etk.SetFocus(inputBuffer)

		game.board.uiGrid.SetRect(game.board.uiGrid.Rect())
	} else {
		if !game.loggedIn {
			game.setRoot(connectFrame)
		} else if game.lobby.showCreateGame {
			game.setRoot(createGameFrame)
		} else if game.lobby.showJoinGame {
			game.setRoot(joinGameFrame)
		} else if game.lobby.showHistory {
			game.setRoot(historyFrame)
			etk.SetFocus(game.lobby.historyUsername)
		} else {
			game.setRoot(listGamesFrame)
			etk.SetFocus(game.lobby.availableMatchesList)
		}

		game.board.menuGrid.SetVisible(false)
		game.board.settingsDialog.SetVisible(false)
		game.board.selectSpeed.SetMenuVisible(false)
		game.board.leaveMatchDialog.SetVisible(false)

		statusBuffer.SetRect(statusBuffer.Rect())

		game.board.playerRoll1, game.board.playerRoll2, game.board.playerRoll3 = 0, 0, 0
		game.board.playerRollStale = false
		game.board.opponentRoll1, game.board.opponentRoll2, game.board.opponentRoll3 = 0, 0, 0
		game.board.opponentRollStale = false
	}

	if refreshLobby && game.client != nil {
		game.client.Out <- []byte("list")
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

type Game struct {
	screenW, screenH int
	lastResize       time.Time

	drawBuffer bytes.Buffer

	spinnerIndex int

	ServerAddress string
	Email         string
	Username      string
	Password      string
	register      bool
	loggedIn      bool

	JoinGame   int
	Mute       bool
	Instant    bool
	Fullscreen bool

	client *Client

	board *board

	lobby *lobby

	keyboard      *etk.Keyboard
	keyboardFrame *etk.Frame

	volume float64 // Volume range is 0-1.

	runeBuffer []rune

	debugImg *ebiten.Image

	connectUsername *Input
	connectPassword *Input
	connectServer   *Input

	registerEmail    *Input
	registerUsername *Input
	registerPassword *Input

	resetEmail      *Input
	resetInfo       *etk.Text
	resetInProgress bool

	mainStatusGrid *etk.Grid

	tutorialFrame *etk.Frame

	aboutDialog *Dialog
	quitDialog  *Dialog

	bufferFontSize int

	pressedKeys  []ebiten.Key
	pressedRunes []rune

	cursorX, cursorY int

	rootWidget etk.Widget

	touchIDs []ebiten.TouchID

	lastRefresh time.Time

	ignoreEnter bool

	forceLayout bool

	bufferWidth int

	connectGridY int

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

	replay       bool
	replayData   []byte
	replayFrame  int
	replayFrames []*replayFrame

	localServer chan net.Conn

	lastTermination time.Time

	*sync.Mutex
}

func NewGame() *Game {
	ebiten.SetVsyncEnabled(false)
	ebiten.SetScreenClearedEveryFrame(false)
	ebiten.SetTPS(targetFPS)
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetWindowClosingHandled(true)

	if smallScreen {
		etk.Style.ButtonBorderSize /= 2

		extraSmallFontSize /= 2
		smallFontSize /= 2
		mediumFontSize /= 2
		mediumLargeFontSize /= 2
		largeFontSize /= 2
	}

	faceSource, err := text.NewGoTextFaceSource(bytes.NewReader(LoadBytes("asset/font/mplus-1p-regular.ttf")))
	if err != nil {
		log.Fatal(err)
	}

	etk.Style.TextFont = faceSource
	etk.Style.TextSize = largeFontSize

	etk.Style.TextColorLight = triangleA
	etk.Style.TextColorDark = triangleA
	etk.Style.InputBgColor = color.RGBA{40, 24, 9, 255}

	etk.Style.ScrollAreaColor = color.RGBA{26, 15, 6, 255}
	etk.Style.ScrollHandleColor = color.RGBA{180, 154, 108, 255}

	etk.Style.InputBorderSize = 1
	etk.Style.InputBorderFocused = color.RGBA{0, 0, 0, 255}
	etk.Style.InputBorderUnfocused = color.RGBA{0, 0, 0, 255}

	etk.Style.ScrollBorderLeft = color.RGBA{210, 182, 135, 255}
	etk.Style.ScrollBorderTop = color.RGBA{210, 182, 135, 255}

	etk.Style.ButtonTextColor = color.RGBA{0, 0, 0, 255}
	etk.Style.ButtonBgColor = color.RGBA{225, 188, 125, 255}

	etk.Style.ButtonBorderLeft = color.RGBA{233, 207, 170, 255}
	etk.Style.ButtonBorderTop = color.RGBA{233, 207, 170, 255}

	etk.Style.CheckboxBgColor = color.RGBA{40, 24, 9, 255}

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
	if !enableOnScreenKeyboard {
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

	statusBuffer = etk.NewText("")
	gameBuffer = etk.NewText("")
	inputBuffer = &Input{etk.NewInput("", acceptInput)}

	inputBuffer.SetBorderSize(0)

	statusBuffer.SetForeground(bufferTextColor)
	statusBuffer.SetBackground(bufferBackgroundColor)

	gameBuffer.SetForeground(bufferTextColor)
	gameBuffer.SetBackground(bufferBackgroundColor)

	inputBuffer.SetForeground(bufferTextColor)
	inputBuffer.SetBackground(bufferBackgroundColor)
	inputBuffer.SetSuffix("")

	bounds := etk.BoundString(etk.FontFace(etk.Style.TextFont, etk.Scale(largeFontSize)), "A")
	fieldHeight = bounds.Dy() + etk.Scale(5)*2

	displayFrame = etk.NewFrame()
	displayFrame.SetPositionChildren(true)

	g.board = NewBoard()
	g.lobby = NewLobby()

	xPadding := etk.Scale(10)
	yPadding := etk.Scale(20)
	labelWidth := etk.Scale(200)
	if smallScreen {
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
	centerInput(g.connectServer)
	g.connectServer.SetAutoResize(true)

	g.mainStatusGrid = etk.NewGrid()
	g.mainStatusGrid.AddChildAt(statusBuffer, 0, 0, 1, 1)
	g.mainStatusGrid.SetVisible(false)

	var aboutGrid *etk.Grid
	var aboutHeight int
	{
		versionInfo := etk.NewText("Boxcars " + AppVersion)
		versionInfo.SetAutoResize(true)
		versionInfo.SetVertical(etk.AlignCenter)
		aboutLabel := gotext.Get("About")
		bounds := etk.BoundString(etk.FontFace(etk.Style.TextFont, etk.Scale(largeFontSize)), aboutLabel)
		aboutHeight = bounds.Dy() + etk.Scale(5)*2
		if aboutHeight < etk.Scale(baseButtonHeight) {
			aboutHeight = etk.Scale(baseButtonHeight)
		}
		aboutButton := etk.NewButton(aboutLabel, g.showAboutDialog)
		aboutGrid = etk.NewGrid()
		aboutGrid.SetRowSizes(-1, aboutHeight)
		aboutGrid.SetColumnSizes(-1, bounds.Dx()+etk.Scale(50))
		aboutGrid.AddChildAt(versionInfo, 0, 1, 1, 1)
		aboutGrid.AddChildAt(g.mainStatusGrid, 0, 1, 1, 1)
		aboutGrid.AddChildAt(aboutButton, 1, 1, 1, 1)
	}

	mainScreen := func(subGrid *etk.Grid, fields int, buttons int, header string, info string) *etk.Grid {
		if ShowServerSettings {
			fields++
		}
		var wgt etk.Widget
		wgt = subGrid
		if !smallScreen {
			f := etk.NewFrame(subGrid)
			f.SetPositionChildren(true)
			f.SetMaxWidth(1024)
			wgt = f
		}
		headerHeight := fieldHeight
		headerLabel := newCenteredText(header)
		headerLabel.SetAutoResize(true)
		infoLabel := newCenteredText(info)
		infoLabel.SetVertical(etk.AlignStart)
		infoLabel.SetAutoResize(false)
		if smallScreen {
			headerLabel.SetHorizontal(etk.AlignCenter)
			headerLabel.SetFont(etk.Style.TextFont, etk.Scale(largeFontSize))
			infoLabel.SetFont(etk.Style.TextFont, etk.Scale(largeFontSize))
		}
		grid := etk.NewGrid()
		grid.SetColumnPadding(int(g.board.horizontalBorderSize / 2))
		grid.SetRowPadding(int(g.board.horizontalBorderSize / 2))
		grid.SetRowSizes(headerHeight, fields*fieldHeight+buttons*etk.Scale(baseButtonHeight)+yPadding*(fields+1)+yPadding*buttons, -1, aboutHeight)
		grid.AddChildAt(headerLabel, 0, 0, 1, 1)
		grid.AddChildAt(wgt, 0, 1, 1, 1)
		grid.AddChildAt(infoLabel, 0, 2, 1, 1)
		grid.AddChildAt(aboutGrid, 0, 3, 1, 1)
		return grid
	}

	{
		gr := etk.NewGrid()
		var y int
		{
			label := resizeText("Boxcars")
			label.SetHorizontal(etk.AlignCenter)
			label.SetVertical(etk.AlignEnd)
			label.SetFont(etk.Style.TextFont, etk.Scale(largeFontSize))
			gr.AddChildAt(label, 0, y, 5, 1)
			y++
		}

		{
			label := resizeText(fmt.Sprintf(gotext.Get("Created by %s"), "Trevor Slocum"))
			label.SetHorizontal(etk.AlignCenter)
			label.SetVertical(etk.AlignStart)
			label.SetFont(etk.Style.TextFont, etk.Scale(mediumFontSize))
			gr.AddChildAt(label, 1, y, 3, 1)
			y++
		}

		{
			ll := resizeText(gotext.Get("FAQ:"))
			ll.SetVertical(etk.AlignCenter)
			rl := resizeText("bgammon.org/faq")
			rl.SetVertical(etk.AlignCenter)
			link := &ClickableText{rl, func() {
				etk.Open("https://bgammon.org/faq")
			}}
			iconSprite := etk.NewSprite(imgIcon)
			iconSprite.SetHorizontal(etk.AlignEnd)
			iconSprite.SetVertical(etk.AlignStart)
			gr.AddChildAt(ll, 1, y, 1, 1)
			gr.AddChildAt(link, 2, y, 1, 1)
			gr.AddChildAt(iconSprite, 3, y, 1, 3)
			y++
		}

		{
			ll := resizeText(gotext.Get("Source:"))
			ll.SetVertical(etk.AlignCenter)
			rl := resizeText("bgammon.org/code")
			rl.SetVertical(etk.AlignCenter)
			link := &ClickableText{rl, func() {
				etk.Open("https://bgammon.org/code")
			}}
			gr.AddChildAt(ll, 1, y, 1, 1)
			gr.AddChildAt(link, 2, y, 1, 1)
			y++
		}

		{
			ll := resizeText(gotext.Get("Donate:"))
			ll.SetVertical(etk.AlignCenter)
			rl := resizeText("bgammon.org/donate")
			rl.SetVertical(etk.AlignCenter)
			link := &ClickableText{rl, func() {
				etk.Open("https://bgammon.org/donate")
			}}
			gr.AddChildAt(ll, 1, y, 1, 1)
			gr.AddChildAt(link, 2, y, 1, 1)
			y++
		}

		{
			ll := resizeText(gotext.Get("Contact:"))
			ll.SetVertical(etk.AlignCenter)
			rl := resizeText("bgammon.org/community")
			rl.SetVertical(etk.AlignCenter)
			link := &ClickableText{rl, func() {
				etk.Open("https://bgammon.org/community")
			}}
			gr.AddChildAt(ll, 1, y, 1, 1)
			gr.AddChildAt(link, 2, y, 2, 1)
			y++
		}

		gr.AddChildAt(etk.NewBox(), 0, y, 5, 1)

		labelWidth := etk.Scale(150)
		if smallScreen {
			labelWidth /= 2
		}

		gr.SetRowSizes(etk.Scale(baseButtonHeight), -1, -1, -1, -1, -1, etk.Scale(10))
		gr.SetColumnSizes(etk.Scale(10), labelWidth, -1, 144, etk.Scale(10))

		g.aboutDialog = newDialog(etk.NewGrid())

		d := g.aboutDialog
		d.SetRowSizes(-1, etk.Scale(baseButtonHeight))
		d.AddChildAt(&withDialogBorder{gr, image.Rectangle{}}, 0, 0, 1, 1)
		d.AddChildAt(etk.NewButton(gotext.Get("Return"), func() error { g.aboutDialog.SetVisible(false); return nil }), 0, 1, 1, 1)
		d.SetVisible(false)
	}

	{
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

		grid := etk.NewGrid()
		grid.SetColumnPadding(int(g.board.horizontalBorderSize / 2))
		grid.SetRowPadding(yPadding)
		grid.SetColumnSizes(xPadding, labelWidth, -1, -1, xPadding)
		if ShowServerSettings {
			grid.SetRowSizes(fieldHeight, fieldHeight, fieldHeight, fieldHeight, etk.Scale(baseButtonHeight))
		} else {
			grid.SetRowSizes(fieldHeight, fieldHeight, fieldHeight, etk.Scale(baseButtonHeight))
		}
		grid.AddChildAt(emailLabel, 1, 0, 2, 1)
		grid.AddChildAt(g.registerEmail, 2, 0, 2, 1)
		grid.AddChildAt(nameLabel, 1, 1, 2, 1)
		grid.AddChildAt(g.registerUsername, 2, 1, 2, 1)
		grid.AddChildAt(passwordLabel, 1, 2, 2, 1)
		grid.AddChildAt(g.registerPassword, 2, 2, 2, 1)
		y := 3
		if ShowServerSettings {
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
		registerGrid = grid

		header := gotext.Get("Register")
		info := gotext.Get("Please enter a valid email address, or it will not be possible to reset your password.")
		registerFrame = etk.NewFrame(mainScreen(registerGrid, 3, 1, header, info))
		registerFrame.SetPositionChildren(true)
		registerFrame.AddChild(etk.NewFrame(g.aboutDialog))
	}

	{
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

		grid := etk.NewGrid()
		grid.SetColumnPadding(int(g.board.horizontalBorderSize / 2))
		grid.SetRowPadding(yPadding)
		grid.SetColumnSizes(xPadding, labelWidth, -1, -1, xPadding)
		if ShowServerSettings {
			grid.SetRowSizes(fieldHeight, fieldHeight, etk.Scale(baseButtonHeight))
		} else {
			grid.SetRowSizes(fieldHeight, etk.Scale(baseButtonHeight))
		}
		grid.AddChildAt(emailLabel, 1, 0, 2, 1)
		grid.AddChildAt(g.resetEmail, 2, 0, 2, 1)
		y := 1
		if ShowServerSettings {
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
			y++
		}
		grid.AddChildAt(g.resetInfo, 1, y, 3, 1)
		resetGrid = grid

		header := gotext.Get("Reset Password")
		resetFrame = etk.NewFrame(mainScreen(resetGrid, 3, 1, header, ""))
		resetFrame.SetPositionChildren(true)
		resetFrame.AddChild(etk.NewFrame(g.aboutDialog))
	}

	{
		nameLabel := newCenteredText(gotext.Get("Username"))
		passwordLabel := newCenteredText(gotext.Get("Password"))
		serverLabel := newCenteredText(gotext.Get("Server"))

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
		grid.SetColumnPadding(int(g.board.horizontalBorderSize / 2))
		grid.SetRowPadding(yPadding)
		if ShowServerSettings {
			grid.SetRowSizes(fieldHeight, fieldHeight, fieldHeight, etk.Scale(baseButtonHeight), etk.Scale(baseButtonHeight))
		} else {
			grid.SetRowSizes(fieldHeight, fieldHeight, etk.Scale(baseButtonHeight), etk.Scale(baseButtonHeight))
		}
		grid.SetColumnSizes(xPadding, labelWidth, -1, -1, xPadding)
		grid.AddChildAt(nameLabel, 1, 0, 2, 1)
		grid.AddChildAt(g.connectUsername, 2, 0, 2, 1)
		grid.AddChildAt(passwordLabel, 1, 1, 2, 1)
		grid.AddChildAt(g.connectPassword, 2, 1, 2, 1)
		g.connectGridY = 2
		if ShowServerSettings {
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
			g.connectGridY++
		}
		{
			subGrid := etk.NewGrid()
			subGrid.SetColumnSizes(-1, yPadding, -1)
			subGrid.SetRowSizes(-1, yPadding, -1)
			subGrid.AddChildAt(resetButton, 0, 0, 1, 1)
			subGrid.AddChildAt(offlineButton, 2, 0, 1, 1)
			grid.AddChildAt(subGrid, 1, g.connectGridY, 3, 1)
		}
		connectGrid = grid

		{
			label := resizeText(gotext.Get("Exit Boxcars?"))
			label.SetHorizontal(etk.AlignCenter)
			label.SetVertical(etk.AlignCenter)

			grid := etk.NewGrid()
			grid.AddChildAt(label, 0, 0, 1, 1)

			g.quitDialog = newDialog(etk.NewGrid())
			d := g.quitDialog
			d.AddChildAt(&withDialogBorder{grid, image.Rectangle{}}, 0, 0, 2, 1)
			d.AddChildAt(etk.NewButton(gotext.Get("No"), func() error { g.quitDialog.SetVisible(false); return nil }), 0, 1, 1, 1)
			d.AddChildAt(etk.NewButton(gotext.Get("Yes"), func() error { g.Exit(); return nil }), 1, 1, 1, 1)
			d.SetVisible(false)
		}

		{
			header := gotext.Get("%s - Free Online Backgammon", "bgammon.org")
			info := gotext.Get("To log in as a guest, enter a username (if you want) and do not enter a password.")
			connectFrame = etk.NewFrame(mainScreen(connectGrid, 2, 2, header, info))
			connectFrame.SetPositionChildren(true)
			connectFrame.AddChild(etk.NewFrame(g.aboutDialog))
			connectFrame.AddChild(etk.NewFrame(g.quitDialog))
		}
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

		g.lobby.createGamePoints = &NumericInput{etk.NewInput("", func(text string) (handled bool) {
			g.lobby.confirmCreateGame()
			return false
		})}
		centerNumericInput(g.lobby.createGamePoints)

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
		if smallScreen {
			variantPadding = 30
			variantWidth = 500
		}
		variantFlex := etk.NewFlex()
		variantFlex.SetGaps(0, variantPadding)
		variantFlex.SetChildSize(variantWidth, fieldHeight)
		variantFlex.AddChild(aceyDeuceyGrid)
		variantFlex.AddChild(tabulaGrid)

		variantFrame := etk.NewFrame(variantLabel)
		variantFrame.SetPositionChildren(true)
		variantFrame.SetMaxHeight(fieldHeight)

		subGrid := etk.NewGrid()
		subGrid.SetRowPadding(yPadding)
		subGrid.SetRowSizes(fieldHeight, fieldHeight, fieldHeight, -1)
		subGrid.SetColumnSizes(xPadding, labelWidth, -1, xPadding)
		subGrid.AddChildAt(nameLabel, 1, 0, 1, 1)
		subGrid.AddChildAt(g.lobby.createGameName, 2, 0, 1, 1)
		subGrid.AddChildAt(pointsLabel, 1, 1, 1, 1)
		subGrid.AddChildAt(g.lobby.createGamePoints, 2, 1, 1, 1)
		subGrid.AddChildAt(passwordLabel, 1, 2, 1, 1)
		subGrid.AddChildAt(g.lobby.createGamePassword, 2, 2, 1, 1)
		subGrid.AddChildAt(variantFrame, 1, 3, 1, 1)
		subGrid.AddChildAt(variantFlex, 2, 3, 1, 1)

		subFrame := etk.NewFrame(subGrid)
		subFrame.SetPositionChildren(true)
		subFrame.SetMaxWidth(1024)

		grid := etk.NewGrid()
		grid.SetColumnPadding(int(g.board.horizontalBorderSize / 2))
		grid.SetRowPadding(yPadding)
		grid.SetColumnSizes(xPadding, labelWidth, -1, xPadding)
		grid.SetRowSizes(60, -1, fieldHeight)
		grid.AddChildAt(headerLabel, 0, 0, 3, 1)
		grid.AddChildAt(etk.NewBox(), 3, 0, 1, 1)
		grid.AddChildAt(subFrame, 1, 1, 3, 1)
		createGameGrid = grid

		dividerLine := etk.NewBox()
		dividerLine.SetBackground(bufferTextColor)

		createGameContainer = etk.NewGrid()
		createGameContainer.AddChildAt(createGameGrid, 0, 0, 1, 1)
		createGameContainer.AddChildAt(dividerLine, 0, 1, 1, 1)
		createGameContainer.AddChildAt(statusBuffer, 0, 2, 1, 1)
		createGameContainer.AddChildAt(g.lobby.buttonsGrid, 0, 3, 1, 1)

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

		subGrid := etk.NewGrid()
		subGrid.SetRowPadding(yPadding)
		subGrid.SetRowSizes(fieldHeight, fieldHeight, fieldHeight, -1)
		subGrid.SetColumnSizes(xPadding, labelWidth, -1, xPadding)
		subGrid.AddChildAt(passwordLabel, 1, 0, 1, 1)
		subGrid.AddChildAt(g.lobby.joinGamePassword, 2, 0, 1, 1)

		subFrame := etk.NewFrame(subGrid)
		subFrame.SetPositionChildren(true)
		subFrame.SetMaxWidth(1024)

		grid := etk.NewGrid()
		grid.SetColumnPadding(int(g.board.horizontalBorderSize / 2))
		grid.SetRowPadding(yPadding)
		grid.SetColumnSizes(xPadding, labelWidth, -1, xPadding)
		grid.SetRowSizes(60, -1)
		grid.AddChildAt(g.lobby.joinGameLabel, 0, 0, 3, 1)
		grid.AddChildAt(etk.NewBox(), 3, 0, 1, 1)
		grid.AddChildAt(subFrame, 1, 1, 2, 1)
		joinGameGrid = grid

		dividerLine := etk.NewBox()
		dividerLine.SetBackground(bufferTextColor)

		joinGameContainer = etk.NewGrid()
		joinGameContainer.AddChildAt(joinGameGrid, 0, 0, 1, 1)
		joinGameContainer.AddChildAt(dividerLine, 0, 1, 1, 1)
		joinGameContainer.AddChildAt(statusBuffer, 0, 2, 1, 1)
		joinGameContainer.AddChildAt(g.lobby.buttonsGrid, 0, 3, 1, 1)

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

		backgroundBox := etk.NewBox()
		backgroundBox.SetBackground(bufferBackgroundColor)

		dateLabel := newCenteredText(gotext.Get("Date"))
		dateLabel.SetFollow(false)
		dateLabel.SetScrollBarVisible(false)
		resultLabel := newCenteredText(gotext.Get("Result"))
		resultLabel.SetFollow(false)
		resultLabel.SetScrollBarVisible(false)
		opponentLabel := newCenteredText(gotext.Get("Opponent"))
		opponentLabel.SetFollow(false)
		opponentLabel.SetScrollBarVisible(false)
		if smallScreen {
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
		if smallScreen {
			historyItemHeight /= 2
		}
		g.lobby.historyList = etk.NewList(historyItemHeight, nil)
		g.lobby.historyList.SetConfirmedFunc(g.lobby.confirmSelectHistory)
		g.lobby.historyList.SetColumnSizes(int(float64(indentA)*1.25), int(float64(indentB)*1.25)-int(float64(indentA)*1.25), -1)
		g.lobby.historyList.SetHighlightColor(color.RGBA{79, 55, 30, 255})

		headerGrid := etk.NewGrid()
		headerGrid.SetColumnSizes(int(float64(indentA)*1.25), int(float64(indentB)*1.25)-int(float64(indentA)*1.25), -1, 400, 200)
		headerGrid.AddChildAt(backgroundBox, 0, 0, 3, 1)
		headerGrid.AddChildAt(dateLabel, 0, 0, 1, 1)
		headerGrid.AddChildAt(resultLabel, 1, 0, 1, 1)
		headerGrid.AddChildAt(opponentLabel, 2, 0, 1, 1)
		headerGrid.AddChildAt(g.lobby.historyUsername, 3, 0, 1, 1)
		headerGrid.AddChildAt(searchButton, 4, 0, 1, 1)

		newLabel := func(text string, horizontal etk.Alignment) *etk.Text {
			t := etk.NewText(text)
			t.SetVertical(etk.AlignCenter)
			t.SetHorizontal(horizontal)
			t.SetAutoResize(true)
			return t
		}

		g.lobby.historyRatingCasualBackgammonSingle = newLabel("...", etk.AlignEnd)
		g.lobby.historyRatingCasualBackgammonMulti = newLabel("...", etk.AlignEnd)
		g.lobby.historyRatingCasualAceySingle = newLabel("...", etk.AlignEnd)
		g.lobby.historyRatingCasualAceyMulti = newLabel("...", etk.AlignEnd)
		g.lobby.historyRatingCasualTabulaSingle = newLabel("...", etk.AlignEnd)
		g.lobby.historyRatingCasualTabulaMulti = newLabel("...", etk.AlignEnd)

		ratingGrid := func(singleLabel *etk.Text, multiLabel *etk.Text) *etk.Grid {
			g := etk.NewGrid()
			g.AddChildAt(singleLabel, 0, 0, 1, 1)
			g.AddChildAt(newLabel(gotext.Get("Single"), etk.AlignStart), 1, 0, 1, 1)
			g.AddChildAt(multiLabel, 0, 1, 1, 1)
			g.AddChildAt(newLabel(gotext.Get("Multi"), etk.AlignStart), 1, 1, 1, 1)
			return g
		}

		historyDividerLine := etk.NewBox()
		historyDividerLine.SetBackground(bufferTextColor)

		g.lobby.historyPageButton = etk.NewButton("1/1", g.selectHistoryPage)

		{
			g.lobby.historyPageDialog = newDialog(etk.NewGrid())
			g.lobby.historyPageDialogInput = &NumericInput{etk.NewInput("", g.confirmHistoryPage)}
			centerNumericInput(g.lobby.historyPageDialogInput)
			g.lobby.historyPageDialogInput.SetBorderSize(0)
			g.lobby.historyPageDialogInput.SetAutoResize(true)
			label := resizeText(gotext.Get("Go to page:"))
			label.SetHorizontal(etk.AlignCenter)
			label.SetVertical(etk.AlignCenter)

			grid := etk.NewGrid()
			grid.AddChildAt(label, 0, 0, 1, 1)
			grid.AddChildAt(g.lobby.historyPageDialogInput, 0, 1, 1, 1)

			d := g.lobby.historyPageDialog
			d.SetRowSizes(-1, etk.Scale(baseButtonHeight))
			d.AddChildAt(&withDialogBorder{grid, image.Rectangle{}}, 0, 0, 2, 1)
			d.AddChildAt(etk.NewButton(gotext.Get("Cancel"), g.cancelHistoryPage), 0, 1, 1, 1)
			d.AddChildAt(etk.NewButton(gotext.Get("Go"), func() error { g.confirmHistoryPage(g.lobby.historyPageDialogInput.Text()); return nil }), 1, 1, 1, 1)
			d.SetVisible(false)
		}

		pageControlGrid := etk.NewGrid()
		pageControlGrid.AddChildAt(etk.NewButton("<- "+gotext.Get("Previous"), g.selectHistoryPrevious), 0, 0, 1, 1)
		pageControlGrid.AddChildAt(g.lobby.historyPageButton, 1, 0, 1, 1)
		pageControlGrid.AddChildAt(etk.NewButton(gotext.Get("Next")+" ->", g.selectHistoryNext), 2, 0, 1, 1)

		historyRatingBackground := etk.NewBox()
		historyRatingBackground.SetBackground(bufferBackgroundColor)

		historyRatingGrid := etk.NewGrid()
		historyRatingGrid.SetRowSizes(2, -1, -1, -1)
		historyRatingGrid.AddChildAt(historyDividerLine, 0, 0, 3, 1)
		historyRatingGrid.AddChildAt(historyRatingBackground, 0, 1, 3, 3)
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
		historyContainer.AddChildAt(historyRatingGrid, 0, 3, 1, 1)
		historyContainer.AddChildAt(pageControlGrid, 0, 4, 1, 1)
		historyContainer.AddChildAt(statusBuffer, 0, 5, 1, 1)
		historyContainer.AddChildAt(g.lobby.buttonsGrid, 0, 6, 1, 1)

		historyFrame.SetPositionChildren(true)
		historyFrame.AddChild(historyContainer)
		historyFrame.AddChild(etk.NewFrame(g.lobby.historyPageDialog))
	}

	{
		listGamesFrame = etk.NewFrame()

		g.lobby.rebuildButtonsGrid()

		dividerLineTop := etk.NewBox()
		dividerLineTop.SetBackground(bufferTextColor)

		dividerLineBottom := etk.NewBox()
		dividerLineBottom.SetBackground(bufferTextColor)

		backgroundBox := etk.NewBox()
		backgroundBox.SetBackground(bufferBackgroundColor)

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
		if smallScreen {
			statusLabel.SetFont(etk.Style.TextFont, etk.Scale(mediumFontSize))
			ratingLabel.SetFont(etk.Style.TextFont, etk.Scale(mediumFontSize))
			pointsLabel.SetFont(etk.Style.TextFont, etk.Scale(mediumFontSize))
			nameLabel.SetFont(etk.Style.TextFont, etk.Scale(mediumFontSize))
		}

		g.lobby.historyButton = etk.NewButton(gotext.Get("History"), game.selectHistory)

		indentA, indentB := etk.Scale(lobbyIndentA), etk.Scale(lobbyIndentB)

		headerGrid := etk.NewGrid()
		headerGrid.SetColumnSizes(indentA, indentB-indentA, indentB-indentA, -1, 200)
		headerGrid.AddChildAt(backgroundBox, 0, 0, 5, 1)
		headerGrid.AddChildAt(statusLabel, 0, 0, 1, 1)
		headerGrid.AddChildAt(ratingLabel, 1, 0, 1, 1)
		headerGrid.AddChildAt(pointsLabel, 2, 0, 1, 1)
		headerGrid.AddChildAt(nameLabel, 3, 0, 1, 1)
		headerGrid.AddChildAt(g.lobby.historyButton, 4, 0, 1, 1)

		listGamesContainer = etk.NewGrid()
		listGamesContainer.AddChildAt(headerGrid, 0, 0, 1, 1)
		listGamesContainer.AddChildAt(dividerLineTop, 0, 1, 1, 1)
		listGamesContainer.AddChildAt(g.lobby.availableMatchesList, 0, 2, 1, 1)
		listGamesContainer.AddChildAt(dividerLineBottom, 0, 3, 1, 1)
		listGamesContainer.AddChildAt(statusBuffer, 0, 4, 1, 1)
		listGamesContainer.AddChildAt(g.lobby.buttonsGrid, 0, 5, 1, 1)

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

	if g.Mute {
		g.board.MuteSounds()
	}

	if g.Instant {
		g.board.speed = 3
		g.board.selectSpeed.SetSelectedItem(3)
	}

	if g.JoinGame != 0 {
		g.Username = ""
		g.Password = ""
		g.Connect()
		go func() {
			for {
				if g.client.loggedIn {
					g.client.Out <- []byte(fmt.Sprintf("j %d", g.JoinGame))
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
		}()
	}
}

func (g *Game) playOffline() {
	go hideKeyboard()
	if g.loggedIn {
		return
	}

	if g.localServer == nil {
		// Start the local BEI server.
		beiServer := &tabula.BEIServer{
			Verbose: true,
		}
		beiConns := beiServer.ListenLocal()

		// Connect to the local BEI server.
		beiClient := bot.NewLocalBEIClient(<-beiConns, false)

		// Start the local bgammon server.
		op := &server.Options{
			Verbose: true,
		}
		s := server.NewServer(op)
		g.localServer = s.ListenLocal()

		// Connect the bots.
		go bot.NewLocalClient(<-g.localServer, "", "BOT_tabula", "", 1, bgammon.VariantBackgammon, false, beiClient)
		go bot.NewLocalClient(<-g.localServer, "", "BOT_tabula_acey", "", 1, bgammon.VariantAceyDeucey, false, beiClient)
		go bot.NewLocalClient(<-g.localServer, "", "BOT_tabula_tabula", "", 1, bgammon.VariantTabula, false, beiClient)

		// Wait for the bots to finish creating matches.
		time.Sleep(250 * time.Millisecond)
	}

	// Connect the player.
	go g.ConnectLocal(<-g.localServer)
}

func (g *Game) clearBuffers() {
	statusBuffer.SetText("")
	gameBuffer.SetText("")
	inputBuffer.SetText("")

	statusLogged = false
	gameLogged = false
	newGameLogMessage = true
	incomingGameLogRoll = false
	incomingGameLogMove = false
}

func (g *Game) showMainMenu(clearBuffers bool) {
	if !g.loggedIn {
		return
	}
	g.loggedIn = false
	g.register = false

	if g.client == nil {
		return
	}
	g.client.Disconnect()
	g.client = nil

	g.client = nil
	g.lobby.c = nil
	g.board.client = nil

	g.setRoot(connectFrame)
	if g.connectUsername.Text() == "" {
		etk.SetFocus(g.connectUsername)
	} else {
		etk.SetFocus(g.connectPassword)
	}

	if clearBuffers {
		g.clearBuffers()
	}

	loadingText := newCenteredText(gotext.Get("Loading..."))
	if smallScreen {
		loadingText.SetFont(etk.Style.TextFont, etk.Scale(mediumFontSize))
	}
	g.lobby.availableMatchesList.Clear()
	g.lobby.availableMatchesList.AddChildAt(loadingText, 0, 0)

	g.loggedIn = false
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
		started := g.board.gameState.Started
		if started == 0 {
			h, m = 0, 0
		} else {
			ended := g.board.gameState.Ended
			if ended == 0 {
				d = now.Sub(time.Unix(started, 0))
			} else {
				d = time.Unix(ended, 0).Sub(time.Unix(started, 0))
			}
			h, m = int(d.Hours()), int(d.Minutes())%60
		}
		if h != lastTimerHour || m != lastTimerMinute {
			g.board.timerLabel.SetText(fmt.Sprintf("%d:%02d", h, m))
			lastTimerHour, lastTimerMinute = h, m
			scheduleFrame()
		}

		// Update clock.
		h, m = now.Hour()%12, now.Minute()
		if h == 0 {
			h = 12
		}
		if h != lastClockHour || m != lastClockMinute {
			g.board.clockLabel.SetText(fmt.Sprintf("%d:%02d", h, m))
			lastClockHour, lastClockMinute = h, m
			scheduleFrame()
		}

		<-t.C
	}
}

func (g *Game) setRoot(w etk.Widget) {
	if w != g.board.frame {
		g.rootWidget = w
	}
	displayFrame.Clear()
	displayFrame.AddChild(w, g.keyboardFrame)
}

func (g *Game) setBufferRects() {
	var statusBufferHeight int
	{
		fontMutex.Lock()
		m := etk.FontFace(etk.Style.TextFont, g.bufferFontSize).Metrics()
		lineHeight := int(m.HAscent + m.HDescent)
		fontMutex.Unlock()
		statusBufferHeight = lineHeight*3 + g.bufferPadding()*2
	}
	var historyRatingHeight int
	{
		fontMutex.Lock()
		m := etk.FontFace(etk.Style.TextFont, etk.Scale(largeFontSize)).Metrics()
		lineHeight := int(m.HAscent + m.HDescent)
		fontMutex.Unlock()
		historyRatingHeight = lineHeight*3 + etk.Scale(5)*3
	}

	createGameContainer.SetRowSizes(-1, 2, statusBufferHeight, g.lobby.buttonBarHeight)
	joinGameContainer.SetRowSizes(-1, 2, statusBufferHeight, g.lobby.buttonBarHeight)
	historyContainer.SetRowSizes(g.itemHeight(), 2, -1, historyRatingHeight, g.lobby.buttonBarHeight, statusBufferHeight, g.lobby.buttonBarHeight)
	listHeaderHeight := g.itemHeight()
	if smallScreen {
		listHeaderHeight /= 2
	}
	listGamesContainer.SetRowSizes(listHeaderHeight, 2, -1, 2, statusBufferHeight, g.lobby.buttonBarHeight)
}

func (g *Game) handleAutoRefresh() {
	g.lastRefresh = time.Now()
	t := time.NewTicker(19 * time.Second)
	for range t.C {
		if viewBoard {
			continue
		}

		if g.client != nil && g.client.Username != "" {
			g.client.Out <- []byte("ls")
			g.lastRefresh = time.Now()
		}
	}
}

func (g *Game) handleEvent(e interface{}) {
	switch ev := e.(type) {
	case *bgammon.EventWelcome:
		g.client.Username = ev.PlayerName
		g.register = false

		username := ev.PlayerName
		if strings.HasPrefix(username, "Guest_") && !onlyNumbers.MatchString(username[6:]) {
			username = username[6:]
		}
		password := g.connectPassword.Text()
		if password == "" {
			password = g.registerPassword.Text()
		}
		if !g.client.local {
			go saveCredentials(username, password)
		}

		clients := gotext.GetN("There is %d client", "There are %d clients", ev.Clients, ev.Clients)
		matches := gotext.GetN("%d match", "%d matches", ev.Games, ev.Games)
		msg := gotext.Get("Welcome, %[1]s. %[2]s playing %[3]s.", ev.PlayerName, clients, matches)
		ls(fmt.Sprintf("*** " + msg))

		if strings.HasPrefix(g.client.Username, "Guest_") && g.savedUsername == "" && g.JoinGame == 0 {
			g.tutorialFrame.AddChild(NewTutorialWidget())
		}
	case *bgammon.EventNotice:
		if strings.HasPrefix(ev.Message, "Connection terminated") {
			g.lastTermination = time.Now()
		}
		ls(fmt.Sprintf("*** %s", ev.Message))
	case *bgammon.EventSay:
		ls(fmt.Sprintf("<%s> %s", ev.Player, ev.Message))
		playSoundEffect(effectSay)
	case *bgammon.EventList:
		g.lobby.setGameList(ev.Games)
		if !viewBoard {
			scheduleFrame()
		}
	case *bgammon.EventFailedCreate:
		g.lobby.createGamePending, g.lobby.createGameShown = false, false
		g.lobby.rebuildButtonsGrid()

		ls("*** " + gotext.Get("Failed to create match: %s", ev.Reason))
	case *bgammon.EventJoined:
		g.lobby.createGamePending, g.lobby.createGameShown = false, false
		g.lobby.joiningGameID, g.lobby.joiningGamePassword, g.lobby.joiningGameShown = 0, "", false
		g.lobby.rebuildButtonsGrid()

		g.board.Lock()
		if ev.PlayerNumber == 1 {
			g.board.gameState.Player1.Name = ev.Player
		} else if ev.PlayerNumber == 2 {
			g.board.gameState.Player2.Name = ev.Player
		}
		g.board.playerRoll1, g.board.playerRoll2, g.board.playerRoll3 = 0, 0, 0
		g.board.opponentRoll1, g.board.opponentRoll2, g.board.opponentRoll3 = 0, 0, 0
		g.board.playerRollStale = false
		g.board.opponentRollStale = false
		g.board.availableStale = false
		g.board.playerMoves = nil
		g.board.opponentMoves = nil
		if g.needLayoutBoard {
			g.layoutBoard()
		}
		g.board.processState()
		g.board.Unlock()
		setViewBoard(true)

		if ev.Player == g.client.Username {
			gameBuffer.SetText("")
			gameLogged = false
			newGameLogMessage = true
			incomingGameLogRoll = false
			incomingGameLogMove = false
			g.board.rematchButton.SetVisible(false)
		} else {
			lg(gotext.Get("%s joined the match.", ev.Player))
			playSoundEffect(effectJoinLeave)
		}
	case *bgammon.EventFailedJoin:
		g.lobby.joiningGameID, g.lobby.joiningGamePassword, g.lobby.joiningGameShown = 0, "", false
		g.lobby.rebuildButtonsGrid()

		ls("*** " + gotext.Get("Failed to join match: %s", ev.Reason))
	case *bgammon.EventFailedLeave:
		ls("*** " + gotext.Get("Failed to leave match: %s", ev.Reason))
		setViewBoard(false)
	case *bgammon.EventLeft:
		g.board.Lock()
		if g.board.gameState.Player1.Name == ev.Player {
			g.board.gameState.Player1.Name = ""
		} else if g.board.gameState.Player2.Name == ev.Player {
			g.board.gameState.Player2.Name = ""
		}
		g.board.processState()
		g.board.Unlock()
		if ev.Player == g.client.Username {
			setViewBoard(false)
		} else {
			lg(gotext.Get("%s left the match.", ev.Player))
			playSoundEffect(effectJoinLeave)
		}

		if g.JoinGame != 0 && g.board.gameState.Player1.Name == "" && g.board.gameState.Player2.Name == "" {
			g.Exit()
		}
	case *bgammon.EventBoard:
		g.board.Lock()

		g.board.stateLock.Lock()
		*g.board.gameState = ev.GameState
		*g.board.gameState.Game = *ev.GameState.Game
		if g.board.gameState.Turn == 0 {
			if g.board.playerRoll2 != 0 {
				g.board.playerRoll1, g.board.playerRoll2, g.board.playerRoll3 = 0, 0, 0
			}
			if g.board.opponentRoll1 != 0 {
				g.board.opponentRoll1, g.board.opponentRoll2, g.board.opponentRoll3 = 0, 0, 0
			}
			if g.board.gameState.Roll1 != 0 {
				g.board.playerRoll1 = g.board.gameState.Roll1
			}
			if g.board.gameState.Roll2 != 0 {
				g.board.opponentRoll2 = g.board.gameState.Roll2
			}
		} else if g.board.gameState.Roll1 != 0 {
			if g.board.gameState.Turn == 1 {
				g.board.playerRoll1, g.board.playerRoll2, g.board.playerRoll3 = g.board.gameState.Roll1, g.board.gameState.Roll2, g.board.gameState.Roll3
				g.board.playerRollStale = false
				g.board.opponentRollStale = true
				if g.board.opponentRoll1 == 0 || g.board.opponentRoll2 == 0 {
					g.board.opponentRoll1, g.board.opponentRoll2, g.board.opponentRoll3 = 0, 0, 0
				}
			} else {
				g.board.opponentRoll1, g.board.opponentRoll2, g.board.opponentRoll3 = g.board.gameState.Roll1, g.board.gameState.Roll2, g.board.gameState.Roll3
				g.board.opponentRollStale = false
				g.board.playerRollStale = true
				if g.board.playerRoll1 == 0 || g.board.playerRoll2 == 0 {
					g.board.playerRoll1, g.board.playerRoll2, g.board.playerRoll3 = 0, 0, 0
				}
				g.board.dragging = nil
			}
		}
		g.board.availableStale = false
		g.board.stateLock.Unlock()

		g.board.processState()
		g.board.Unlock()

		if incomingGameLogRoll {
			if g.board.gameState.Roll1 != 0 && g.board.gameState.Roll2 != 0 {
				roll := formatRoll(g.board.gameState.Roll1, g.board.gameState.Roll2, g.board.gameState.Roll3)

				var extra string
				if g.board.gameState.Turn != 0 && len(g.board.gameState.Available) > 0 {
					extra = ":"
				}

				name := g.board.gameState.Player1.Name
				if game.board.gameState.Turn == 2 {
					name = g.board.gameState.Player2.Name
				}

				lg(name + " " + roll + extra)
				newGameLogMessage = false
			}
			incomingGameLogRoll = false
		}

		if incomingGameLogMove {
			if g.board.gameState.Roll1 != 0 && g.board.gameState.Roll2 != 0 {
				name := g.board.gameState.Player1.Name
				if game.board.gameState.Turn == 2 {
					name = g.board.gameState.Player2.Name
				}
				var moves string
				if len(g.board.gameState.Moves) > 0 {
					moves = string(bgammon.FormatMoves(g.board.gameState.Moves))
				}
				msg := name + " " + formatRoll(g.board.gameState.Roll1, g.board.gameState.Roll2, g.board.gameState.Roll3) + ": " + moves
				if !newGameLogMessage {
					if lastGameLogTime == "" {
						lastGameLogTime = time.Now().Format("[3:04]")
					}
					gameBuffer.SetLast(lastGameLogTime + " " + msg)
					gameLogged = true
				} else {
					lg(msg)
				}
				newGameLogMessage = false
			}
			incomingGameLogMove = false
		}

		if !g.board.gameState.Spectating && (g.board.gameState.Player1.Points >= g.board.gameState.Points || g.board.gameState.Player2.Points >= g.board.gameState.Points) {
			g.board.rematchButton.SetVisible(true)
		}

		setViewBoard(true)
	case *bgammon.EventRolled:
		playSound := SoundEffect(-1)
		g.board.Lock()
		g.board.stateLock.Lock()
		g.board.gameState.Roll1 = ev.Roll1
		g.board.gameState.Roll2 = ev.Roll2
		g.board.gameState.Roll3 = ev.Roll3
		var roll string
		if g.board.gameState.Turn == 0 {
			if g.board.gameState.Player1.Name == ev.Player {
				roll = formatRoll(g.board.gameState.Roll1, 0, 0)
				g.board.playerRoll1 = g.board.gameState.Roll1
				g.board.playerRollStale = false
			} else {
				roll = formatRoll(0, g.board.gameState.Roll2, 0)
				g.board.opponentRoll2 = g.board.gameState.Roll2
				g.board.opponentRollStale = false
			}
			if !ev.Selected {
				playSound = effectDie
			}
			g.board.availableStale = false
		} else {
			roll = formatRoll(g.board.gameState.Roll1, g.board.gameState.Roll2, g.board.gameState.Roll3)
			if g.board.gameState.Player1.Name == ev.Player {
				g.board.playerRoll1, g.board.playerRoll2, g.board.playerRoll3 = g.board.gameState.Roll1, g.board.gameState.Roll2, g.board.gameState.Roll3
				g.board.playerRollStale = false
			} else {
				g.board.opponentRoll1, g.board.opponentRoll2, g.board.opponentRoll3 = g.board.gameState.Roll1, g.board.gameState.Roll2, g.board.gameState.Roll3
				g.board.opponentRollStale = false
			}
			if !ev.Selected {
				playSound = effectDice
			}
			g.board.availableStale = true
		}
		g.board.stateLock.Unlock()
		g.board.processState()
		g.board.Unlock()
		scheduleFrame()

		if g.board.gameState.Turn == 0 {
			lg(gotext.Get("%s rolled %s", ev.Player, roll))
		} else {
			newGameLogMessage = true
		}
		// Play the sound effect after processing the board state to avoid
		// audio issues when running in a single-threaded environment.
		if playSound != -1 {
			playSoundEffect(playSound)
		}
		if g.board.gameState.Roll1 == 0 || g.board.gameState.Roll2 == 0 {
			return
		}
		if g.board.gameState.Variant == bgammon.VariantBackgammon || g.board.gameState.Turn != 0 {
			incomingGameLogRoll = true
		}
	case *bgammon.EventFailedRoll:
		ls(fmt.Sprintf("*** %s: %s", gotext.Get("Failed to roll"), ev.Reason))
	case *bgammon.EventMoved:
		incomingGameLogMove = true
		if ev.Player == g.client.Username && !g.board.gameState.Spectating && !g.board.gameState.Forced {
			return
		}

		g.board.Lock()
		g.Unlock()
		for _, move := range ev.Moves {
			playSoundEffect(effectMove)
			g.board.movePiece(move[0], move[1], true)
		}
		g.Lock()
		if g.board.showMoves {
			moves := g.board.gameState.Moves
			if g.board.gameState.Turn == 2 && game.board.traditional {
				moves = bgammon.FlipMoves(moves, 2, g.board.gameState.Variant)
			}
			if g.board.gameState.Turn == 1 {
				g.board.playerMoves = expandMoves(moves)
			} else if g.board.gameState.Turn == 2 {
				g.board.opponentMoves = expandMoves(moves)
			}
		}
		g.board.Unlock()
	case *bgammon.EventFailedMove:
		g.client.Out <- []byte("board") // Refresh game state.

		var extra string
		if ev.From != 0 || ev.To != 0 {
			extra = " " + gotext.Get("from %s to %s", bgammon.FormatSpace(ev.From), bgammon.FormatSpace(ev.To))
		}
		ls("*** " + gotext.Get("Failed to move checker%s: %s", extra, ev.Reason))
		ls("*** " + gotext.Get("Legal moves: %s", bgammon.FormatMoves(g.board.gameState.Available)))
	case *bgammon.EventFailedOk:
		g.client.Out <- []byte("board") // Refresh game state.
		ls("*** " + gotext.Get("Failed to submit moves: %s", ev.Reason))
	case *bgammon.EventWin:
		g.board.Lock()
		if ev.Resigned != "" {
			lg(gotext.Get("%s resigned.", ev.Resigned))
		}
		var message string
		if ev.Points <= 1 {
			message = gotext.Get("%s wins!", ev.Player)
		} else {
			message = gotext.GetN("%[1]s wins %[2]d point!", "%[1]s wins %[2]d points!", int(ev.Points), ev.Player, ev.Points)
		}
		if ev.Rating != 0 {
			message += fmt.Sprintf(" (+%d)", ev.Rating)
		}
		lg(message)
		g.board.Unlock()
	case *bgammon.EventSettings:
		g.board.stateLock.Lock()
		g.board.pendingSettings = ev
		g.board.stateLock.Unlock()
	case *bgammon.EventReplay:
		if game.downloadReplay == ev.ID {
			err := saveReplay(ev.ID, ev.Content)
			if err != nil {
				ls("*** " + gotext.Get("Failed to download replay: %s", err))
			}
			game.downloadReplay = 0
			return
		}
		go game.HandleReplay(ev.Content)
	case *bgammon.EventHistory:
		game.lobby.historyMatches = ev.Matches
		game.lobby.historyPage = ev.Page
		game.lobby.historyPages = ev.Pages
		game.lobby.historyPageButton.SetText(fmt.Sprintf("%d/%d", ev.Page, ev.Pages))
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
			noReplaysText := newCenteredText(gotext.Get("No replays found."))
			list.AddChildAt(noReplaysText, 0, 0)
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
			if smallScreen {
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
		g.client.Out <- []byte(fmt.Sprintf("pong %s", ev.Message))
	default:
		ls("*** " + gotext.Get("Warning: Received unknown event: %+v", ev))
		ls("*** " + gotext.Get("You may need to upgrade your client."))
	}
}

func (g *Game) handleEvents(c *Client) {
	for e := range c.Events {
		g.board.Lock()
		g.Lock()
		g.board.Unlock()
		g.handleEvent(e)
		g.Unlock()
	}
}

func (g *Game) Connect() {
	if g.loggedIn {
		return
	}
	g.loggedIn = true

	g.clearBuffers()
	ls("*** " + gotext.Get("Connecting..."))

	g.lobby.historyButton.SetVisible(true)

	g.setRoot(listGamesFrame)
	etk.SetFocus(game.lobby.availableMatchesList)

	address := g.ServerAddress
	if address == "" {
		address = DefaultServerAddress
	}
	g.client = newClient(address, g.Username, g.Password, false)
	g.lobby.c = g.client
	g.board.client = g.client

	g.lobby.loaded = false

	go g.handleEvents(g.client)

	if g.Password != "" {
		g.board.recreateAccountGrid()
	}

	c := g.client

	connectTime := time.Now()
	t := time.NewTicker(250 * time.Millisecond)
	go func() {
		for {
			<-t.C
			if c.loggedIn {
				return
			} else if !c.connecting || time.Since(connectTime) >= 20*time.Second {
				g.mainStatusGrid.SetVisible(true)

				g.showMainMenu(false)
				scheduleFrame()
				return
			}
		}
	}()

	go c.Connect()
}

func (g *Game) ConnectLocal(conn net.Conn) {
	if g.loggedIn {
		return
	}
	g.loggedIn = true

	g.clearBuffers()
	ls("*** " + gotext.Get("Playing offline."))

	g.lobby.historyButton.SetVisible(false)

	g.setRoot(listGamesFrame)
	etk.SetFocus(game.lobby.availableMatchesList)

	g.client = newClient("", g.connectUsername.Text(), "", false)
	g.lobby.c = g.client
	g.board.client = g.client

	g.client.local = true
	g.client.connecting = true

	g.lobby.loaded = false

	go g.handleEvents(g.client)

	go g.client.connectTCP(conn)
}

func (g *Game) selectRegister() error {
	g.closeDialogs()
	g.showRegister = true
	g.registerUsername.SetText(g.connectUsername.Text())
	g.registerPassword.SetText(g.connectPassword.Text())
	g.setRoot(registerFrame)
	etk.SetFocus(g.registerEmail)
	return nil
}

func (g *Game) selectReset() error {
	g.closeDialogs()
	g.showReset = true
	g.setRoot(resetFrame)
	etk.SetFocus(g.resetEmail)
	return nil
}

func (g *Game) selectCancel() error {
	g.closeDialogs()
	g.showRegister = false
	g.showReset = false
	g.setRoot(connectFrame)
	etk.SetFocus(g.connectUsername)
	return nil
}

func (g *Game) selectConfirmRegister() error {
	g.closeDialogs()
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
	g.closeDialogs()
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
	g.closeDialogs()
	go hideKeyboard()
	g.Username = g.connectUsername.Text()
	g.Password = g.connectPassword.Text()
	if ShowServerSettings {
		g.ServerAddress = g.connectServer.Text()
	}
	g.Connect()
	return nil
}

func (g *Game) showAboutDialog() error {
	g.aboutDialog.SetVisible(true)
	return nil
}

func (g *Game) closeDialogs() {
	g.aboutDialog.SetVisible(false)
	g.quitDialog.SetVisible(false)
}

func (g *Game) searchMatches(username string) {
	go hideKeyboard()
	loadingText := newCenteredText(gotext.Get("Loading..."))
	if smallScreen {
		loadingText.SetFont(etk.Style.TextFont, etk.Scale(mediumFontSize))
	}

	g.lobby.historyList.Clear()
	g.lobby.historyList.SetSelectionMode(etk.SelectNone)
	g.lobby.historyList.AddChildAt(loadingText, 0, 0)
	g.client.Out <- []byte(fmt.Sprintf("history %s", username))
}

func (g *Game) selectHistory() error {
	go hideKeyboard()
	g.lobby.showHistory = true
	g.setRoot(historyFrame)
	g.lobby.historyUsername.SetText(g.client.Username)
	g.searchMatches(g.client.Username)
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
	g.client.Out <- []byte(fmt.Sprintf("history %s %d", g.lobby.historyUsername.Text(), g.lobby.historyPage-1))
	return nil
}

func (g *Game) selectHistoryNext() error {
	go hideKeyboard()
	if g.lobby.historyUsername.Text() == "" || g.lobby.historyPage == g.lobby.historyPages {
		return nil
	}
	g.client.Out <- []byte(fmt.Sprintf("history %s %d", g.lobby.historyUsername.Text(), g.lobby.historyPage+1))
	return nil
}

func (g *Game) selectHistoryPage() error {
	go showKeyboard()
	g.lobby.historyPageDialogInput.SetText("")
	g.lobby.historyPageDialog.SetVisible(true)
	etk.SetFocus(g.lobby.historyPageDialogInput)
	return nil
}

func (g *Game) confirmHistoryPage(text string) bool {
	go hideKeyboard()
	g.lobby.historyPageDialog.SetVisible(false)
	page, err := strconv.Atoi(g.lobby.historyPageDialogInput.Text())
	if err != nil || page < 0 {
		page = 0
	} else if page > g.lobby.historyPages {
		page = g.lobby.historyPages
	}
	g.client.Out <- []byte(fmt.Sprintf("history %s %d", g.lobby.historyUsername.Text(), page))
	return true
}

func (g *Game) cancelHistoryPage() error {
	go hideKeyboard()
	g.lobby.historyPageDialog.SetVisible(false)
	return nil
}

func (g *Game) handleInput(keys []ebiten.Key) error {
	if len(keys) == 0 {
		return nil
	} else if mobileDevice {
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
					return nil
				case g.connectPassword:
					etk.SetFocus(g.connectUsername)
					return nil
				case g.registerEmail:
					if ebiten.IsKeyPressed(ebiten.KeyShift) {
						etk.SetFocus(g.registerPassword)
						return nil
					} else {
						etk.SetFocus(g.registerUsername)
						return nil
					}
				case g.registerUsername:
					if ebiten.IsKeyPressed(ebiten.KeyShift) {
						etk.SetFocus(g.registerEmail)
						return nil
					} else {
						etk.SetFocus(g.registerPassword)
						return nil
					}
				case g.registerPassword:
					if ebiten.IsKeyPressed(ebiten.KeyShift) {
						etk.SetFocus(g.registerUsername)
						return nil
					} else {
						etk.SetFocus(g.registerEmail)
						return nil
					}
				}
			case ebiten.KeyEnter, ebiten.KeyKPEnter:
				if g.aboutDialog.Visible() {
					g.aboutDialog.SetVisible(false)
					g.ignoreEnter = true
					return nil
				} else if g.showRegister {
					g.selectConfirmRegister()
					return nil
				} else if g.showReset {
					g.selectConfirmReset()
					return nil
				} else if ShowQuitDialog && g.quitDialog.Visible() {
					g.Exit()
					return nil
				} else {
					g.selectConnect()
					return nil
				}
			case ebiten.KeyEscape:
				if g.aboutDialog.Visible() {
					g.aboutDialog.SetVisible(false)
					return nil
				} else if g.showRegister || g.showReset {
					g.selectCancel()
					return nil
				} else {
					if ShowQuitDialog {
						g.quitDialog.SetVisible(!g.quitDialog.Visible())
					}
					return nil
				}
			}
		}
		return nil
	}

	for _, key := range keys {
		switch key {
		case ebiten.KeyEscape:
			if viewBoard {
				if g.board.menuGrid.Visible() {
					g.board.menuGrid.SetVisible(false)
					return nil
				} else if g.board.settingsDialog.Visible() {
					g.board.settingsDialog.SetVisible(false)
					g.board.selectSpeed.SetMenuVisible(false)
					return nil
				} else if g.board.changePasswordDialog.Visible() {
					g.board.changePasswordDialog.SetVisible(false)
					g.board.settingsDialog.SetVisible(true)
					return nil
				} else if g.board.muteSoundsDialog.Visible() {
					g.board.muteSoundsDialog.SetVisible(false)
					g.board.settingsDialog.SetVisible(true)
					return nil
				} else if g.board.leaveMatchDialog.Visible() {
					g.board.leaveMatchDialog.SetVisible(false)
					return nil
				} else {
					g.board.menuGrid.SetVisible(true)
					return nil
				}
			} else if g.lobby.showHistory {
				if g.lobby.historyPageDialog.Visible() {
					g.lobby.historyPageDialog.SetVisible(false)
					return nil
				} else {
					g.lobby.showHistory = false
					g.lobby.rebuildButtonsGrid()
					g.setRoot(listGamesFrame)
					etk.SetFocus(game.lobby.availableMatchesList)
					return nil
				}
			} else if g.lobby.showCreateGame {
				g.lobby.showCreateGame = false
				g.lobby.rebuildButtonsGrid()
				g.setRoot(listGamesFrame)
				etk.SetFocus(game.lobby.availableMatchesList)
				return nil
			} else if g.lobby.showJoinGame {
				g.lobby.showJoinGame = false
				g.lobby.rebuildButtonsGrid()
				g.setRoot(listGamesFrame)
				etk.SetFocus(game.lobby.availableMatchesList)
				return nil
			} else {
				g.showMainMenu(true)
				return nil
			}
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
						return nil
					case g.lobby.createGamePoints:
						etk.SetFocus(g.lobby.createGameName)
						return nil
					case g.lobby.createGamePassword:
						etk.SetFocus(g.lobby.createGamePoints)
						return nil
					}
				} else {
					switch focusedWidget {
					case g.lobby.createGameName:
						etk.SetFocus(g.lobby.createGamePoints)
						return nil
					case g.lobby.createGamePoints:
						etk.SetFocus(g.lobby.createGamePassword)
						return nil
					case g.lobby.createGamePassword:
						etk.SetFocus(g.lobby.createGameName)
						return nil
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
					return nil
				} else if g.lobby.showHistory {
					g.selectHistorySearch()
					return nil
				} else {
					g.lobby.confirmJoinGame()
					return nil
				}
			}
		}
	}

	if viewBoard {
		for _, key := range keys {
			switch key {
			case ebiten.KeyTab:
				if g.board.changePasswordDialog.Visible() {
					focusedWidget := etk.Focused()
					switch focusedWidget {
					case g.board.changePasswordOld:
						etk.SetFocus(g.board.changePasswordNew)
						return nil
					case g.board.changePasswordNew:
						etk.SetFocus(g.board.changePasswordOld)
						return nil
					}
				}
			case ebiten.KeyBackspace:
				if len(inputBuffer.Text()) == 0 && !g.board.gameState.Spectating && g.board.gameState.Turn == g.board.gameState.PlayerNumber && len(g.board.gameState.Moves) > 0 && !g.board.menuGrid.Visible() && !g.board.settingsDialog.Visible() && !g.board.changePasswordDialog.Visible() && !g.board.leaveMatchDialog.Visible() {
					g.board.selectUndo()
					return nil
				}
			case ebiten.KeyEnter:
				if g.board.changePasswordDialog.Visible() {
					g.board.selectChangePassword()
					return nil
				} else if g.board.leaveMatchDialog.Visible() {
					g.board.confirmLeaveMatch()
					return nil
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

	g.Lock()
	defer g.Unlock()

	if g.ignoreEnter {
		if ebiten.IsKeyPressed(ebiten.KeyEnter) || ebiten.IsKeyPressed(ebiten.KeyKPEnter) {
			return nil
		}
		g.ignoreEnter = false
	}

	if ebiten.IsKeyPressed(ebiten.KeyAlt) && inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		g.Fullscreen = !g.Fullscreen
		ebiten.SetFullscreen(g.Fullscreen)
		g.ignoreEnter = true
		return nil
	}

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

	if ebiten.IsKeyPressed(ebiten.KeyControl) && inpututil.IsKeyJustPressed(ebiten.KeyD) {
		Debug++
		if Debug > MaxDebug {
			Debug = 0
		}
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
	} else if g.ignoreEnter {
		return nil
	}

	if mobileDevice {
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
	} else if viewBoard {
		g.board.Update()
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
		gameUpdateLock.Unlock()
		return
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

	if !viewBoard {
		screen.Fill(frameColor)
	} else {
		screen.Fill(tableColor)
	}

	if !g.loggedIn { // Draw main menu.
		err := etk.Draw(screen)
		if err != nil {
			log.Fatal(err)
		}
	} else { // Draw lobby and board.
		err := etk.Draw(screen)
		if err != nil {
			log.Fatal(err)
		}

		if drawScreen == 0 {
			if g.lobby.createGamePending && !g.lobby.createGameShown {
				typeAndPassword := "public"
				if len(strings.TrimSpace(game.lobby.createGamePassword.Text())) > 0 {
					typeAndPassword = fmt.Sprintf("private %s", strings.ReplaceAll(game.lobby.createGamePassword.Text(), " ", "_"))
				}
				points, err := strconv.Atoi(game.lobby.createGamePoints.Text())
				if err != nil {
					points = 1
				}
				var variant int8
				if game.lobby.createGameAceyCheckbox.Selected() {
					variant = bgammon.VariantAceyDeucey
				} else if game.lobby.createGameTabulaCheckbox.Selected() {
					variant = bgammon.VariantTabula
				}
				g.lobby.c.Out <- []byte(fmt.Sprintf("c %s %d %d %s", typeAndPassword, points, variant, game.lobby.createGameName.Text()))
				g.lobby.createGameShown = true
			} else if g.lobby.joiningGameID != 0 && !g.lobby.joiningGameShown {
				g.lobby.c.Out <- []byte(fmt.Sprintf("j %d %s", g.lobby.joiningGameID, g.lobby.joiningGamePassword))
				g.lobby.joiningGameShown = true
			}
		}
	}

	if Debug == 0 {
		return
	}
	// Draw debug information.
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

func (g *Game) portraitView() bool {
	return g.screenH-g.screenW >= 200
}

func (g *Game) layoutConnect() {
	g.needLayoutConnect = false

	{
		fontMutex.Lock()
		m := etk.FontFace(etk.Style.TextFont, etk.Scale(largeFontSize)).Metrics()
		lineHeight := int(m.HAscent + m.HDescent)
		fontMutex.Unlock()

		dialogWidth := etk.Scale(650)
		if dialogWidth > game.screenW {
			dialogWidth = game.screenW
		}
		dialogHeight := etk.Scale(baseButtonHeight)*2 + lineHeight*5
		if dialogHeight > game.screenH {
			dialogHeight = game.screenH
		}

		x, y := game.screenW/2-dialogWidth/2, game.screenH/2-dialogHeight/2
		if x < 0 {
			x = 0
		}
		if y < 0 {
			y = 0
		}
		g.aboutDialog.SetRect(image.Rect(x, y, x+dialogWidth, y+dialogHeight))
	}

	{
		dialogWidth := etk.Scale(400)
		if dialogWidth > game.screenW {
			dialogWidth = game.screenW
		}
		dialogHeight := etk.Scale(baseButtonHeight) * 2
		if dialogHeight > game.screenH {
			dialogHeight = game.screenH
		}

		x, y := game.screenW/2-dialogWidth/2, game.screenH/2-dialogHeight/2
		if x < 0 {
			x = 0
		}
		if y < 0 {
			y = 0
		}
		g.quitDialog.SetRect(image.Rect(x, y, x+dialogWidth, y+dialogHeight))
	}
}

func (g *Game) layoutLobby() {
	g.needLayoutLobby = false

	g.lobby.buttonBarHeight = etk.Scale(baseButtonHeight)
	g.lobby.rebuildButtonsGrid()
	g.setBufferRects()

	{
		dialogWidth := etk.Scale(400)
		if dialogWidth > game.screenW {
			dialogWidth = game.screenW
		}
		dialogHeight := g.lobby.buttonBarHeight * 3
		if dialogHeight > game.screenH {
			dialogHeight = game.screenH
		}

		x, y := game.screenW/2-dialogWidth/2, game.screenH/2-dialogHeight+int(g.board.verticalBorderSize)
		g.lobby.historyPageDialog.SetRect(image.Rect(x, y, x+dialogWidth, y+dialogHeight))
	}
}

func (g *Game) layoutBoard() {
	g.needLayoutBoard = false

	if g.portraitView() { // Portrait view.
		g.board.fullHeight = false
		g.board.horizontalBorderSize = 0
		g.board.setRect(0, 0, g.screenW, g.screenW)

		g.board.uiGrid.SetRect(image.Rect(0, g.board.h, g.screenW, g.screenH))
	} else { // Landscape view.
		g.board.fullHeight = true
		g.board.horizontalBorderSize = 20
		g.board.setRect(0, 0, g.screenW-g.bufferWidth, g.screenH)

		availableWidth := g.screenW - (g.board.innerW + int(g.board.horizontalBorderSize*2))
		if availableWidth > g.bufferWidth {
			g.bufferWidth = availableWidth
			g.board.setRect(0, 0, g.screenW-g.bufferWidth, g.screenH)
		}

		if g.board.h > g.board.w {
			g.board.fullHeight = false
			g.board.setRect(0, 0, g.board.w, g.board.w)
		}

		g.board.uiGrid.SetRect(image.Rect(g.board.w, 0, g.screenW, g.screenH))
	}

	g.setBufferRects()

	g.board.widget.SetRect(image.Rect(0, 0, g.screenW, g.screenH))
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

	scaledWidth, scaledHeight := etk.Layout(outsideWidth, outsideHeight)
	if scaledWidth == g.screenW && scaledHeight == g.screenH {
		return g.screenW, g.screenH
	}
	g.screenW, g.screenH = scaledWidth, scaledHeight
	scheduleFrame()

	g.bufferWidth = etk.BoundString(etk.FontFace(etk.Style.TextFont, etk.Scale(g.board.fontSize)), strings.Repeat("A", bufferCharacterWidth)).Dx()
	if g.bufferWidth > int(float64(g.screenW)*maxStatusWidthRatio) {
		g.bufferWidth = int(float64(g.screenW) * maxStatusWidthRatio)
	}

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

	g.board.updateOpponentLabel()
	g.board.updatePlayerLabel()

	g.keyboard.SetRect(image.Rect(0, game.screenH-game.screenH/3, game.screenW, game.screenH))

	if g.LoadReplay != nil {
		go g.HandleReplay(g.LoadReplay)
		g.LoadReplay = nil
	}

	return g.screenW, g.screenH
}

func acceptInput(text string) (handled bool) {
	if len(text) == 0 {
		g := game
		if viewBoard && !g.board.menuGrid.Visible() && !g.board.settingsDialog.Visible() && !g.board.changePasswordDialog.Visible() && !g.board.leaveMatchDialog.Visible() {
			if g.board.gameState.MayRoll() {
				g.board.selectRoll()
			} else if g.board.gameState.MayOK() {
				g.board.selectOK()
			}
		}
		return true
	}

	if text[0] == '/' {
		text = text[1:]
		if strings.ToLower(text) == "download" {
			if game.replay {
				err := saveReplay(-1, game.replayData)
				if err != nil {
					ls("*** " + gotext.Get("Failed to download replay: %s", err))
				}
			} else {
				if game.downloadReplay == 0 {
					game.downloadReplay = -1
					game.client.Out <- []byte("replay")
				} else {
					ls("*** " + gotext.Get("Replay download already in progress."))
				}
			}
			return true
		}
	} else {
		ls(fmt.Sprintf("<%s> %s", game.client.Username, text))
		text = "say " + text
	}

	game.client.Out <- []byte(text)
	go hideKeyboard()
	return true
}

func (g *Game) itemHeight() int {
	if mobileDevice {
		return etk.Scale(baseButtonHeight)
	}
	fontSize := largeFontSize
	if smallScreen {
		fontSize = mediumFontSize
	}
	return etk.BoundString(etk.FontFace(etk.Style.TextFont, etk.Scale(fontSize)), "(Ag").Dy() + etk.Scale(5)
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
	effectHomeSingle
	effectHomeMulti
)

var (
	dieSounds           []*audio.Player
	dieSoundPlays       int
	diceSounds          []*audio.Player
	diceSoundPlays      int
	moveSounds          []*audio.Player
	moveSoundPlays      int
	homeMultiSounds     []*audio.Player
	homeMultiSoundPlays int
)

func playSoundEffect(effect SoundEffect) {
	if game.volume == 0 || game.replay {
		return
	}

	var p *audio.Player
	switch effect {
	case effectSay:
		if game.board.muteChat {
			return
		}
		p = SoundSay
	case effectJoinLeave:
		if game.board.muteJoinLeave {
			return
		}
		p = SoundJoinLeave
	case effectDie:
		if game.board.muteRoll {
			return
		}
		p = dieSounds[dieSoundPlays]

		dieSoundPlays++
		if dieSoundPlays == len(dieSounds)-1 {
			randomizeSounds(dieSounds)
			dieSoundPlays = 0
		}
	case effectDice:
		if game.board.muteRoll {
			return
		}
		p = diceSounds[diceSoundPlays]

		diceSoundPlays++
		if diceSoundPlays == len(diceSounds)-1 {
			randomizeSounds(diceSounds)
			diceSoundPlays = 0
		}
	case effectMove:
		if game.board.muteMove {
			return
		}
		p = moveSounds[moveSoundPlays]

		moveSoundPlays++
		if moveSoundPlays == len(moveSounds)-1 {
			randomizeSounds(moveSounds)
			moveSoundPlays = 0
		}
	case effectHomeSingle:
		if game.board.muteBearOff {
			return
		}
		p = SoundHomeSingle
	case effectHomeMulti:
		if game.board.muteBearOff {
			return
		}
		p = homeMultiSounds[homeMultiSoundPlays]

		homeMultiSoundPlays++
		if homeMultiSoundPlays == len(homeMultiSounds)-1 {
			randomizeSounds(homeMultiSounds)
			homeMultiSoundPlays = 0
		}
	default:
		log.Panicf("unknown sound effect: %d", effect)
		return
	}

	p.Pause()
	if effect == effectHomeSingle || effect == effectHomeMulti {
		p.SetVolume(game.volume / 6)
	} else if effect == effectSay {
		p.SetVolume(game.volume / 2)
	} else {
		p.SetVolume(game.volume)
	}
	p.Rewind()
	p.Play()
}

func randomizeSounds(s []*audio.Player) {
	last := s[len(s)-1]
	for {
		for i := range s {
			j := rand.Intn(i + 1)
			s[i], s[j] = s[j], s[i]
		}
		if s[0] != last {
			return
		}
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

type Dialog struct {
	*etk.Grid
}

func newDialog(grid *etk.Grid) *Dialog {
	return &Dialog{
		Grid: grid,
	}
}

func (d *Dialog) Background() color.RGBA {
	return color.RGBA{40, 24, 9, 255}
}

func (d *Dialog) HandleMouse(cursor image.Point, pressed bool, clicked bool) (handled bool, err error) {
	_, err = d.Grid.HandleMouse(cursor, pressed, clicked)
	return true, err
}

type withDialogBorder struct {
	*etk.Grid
	r image.Rectangle
}

func (w *withDialogBorder) Rect() image.Rectangle {
	return w.r
}

func (w *withDialogBorder) SetRect(r image.Rectangle) {
	const borderSize = 4
	w.r = r
	w.Grid.SetRect(image.Rect(r.Min.X+borderSize, r.Min.Y+borderSize, r.Max.X-borderSize, r.Max.Y))
}

func (w *withDialogBorder) Draw(screen *ebiten.Image) error {
	const borderSize = 4
	borderColor := color.RGBA{0, 0, 0, 255}
	err := w.Grid.Draw(screen)
	r := w.Rect()
	screen.SubImage(image.Rect(r.Min.X, r.Min.Y, r.Min.X+borderSize, r.Max.Y)).(*ebiten.Image).Fill(borderColor)
	screen.SubImage(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+borderSize)).(*ebiten.Image).Fill(borderColor)
	screen.SubImage(image.Rect(r.Max.X-borderSize, r.Min.Y, r.Max.X, r.Max.Y)).(*ebiten.Image).Fill(borderColor)
	return err
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

type NumericInput struct {
	*etk.Input
}

func (i *NumericInput) HandleKeyboard(key ebiten.Key, r rune) (handled bool, err error) {
	if r != 0 {
		switch r {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		default:
			return true, nil
		}
	} else if key != ebiten.KeyBackspace && key != ebiten.KeyEnter && key != ebiten.KeyKPEnter {
		return true, nil
	}
	return i.Input.HandleKeyboard(key, r)
}

type ClickableText struct {
	*etk.Text
	onSelected func()
}

func (t *ClickableText) Cursor() ebiten.CursorShapeType {
	return ebiten.CursorShapePointer
}

func (t *ClickableText) HandleMouse(cursor image.Point, pressed bool, clicked bool) (handled bool, err error) {
	if clicked {
		t.onSelected()
	}
	return true, nil
}

func resizeText(text string) *etk.Text {
	t := etk.NewText(text)
	t.SetAutoResize(true)
	return t
}

func newCenteredText(text string) *etk.Text {
	t := resizeText(text)
	t.SetVertical(etk.AlignCenter)
	return t
}

func centerInput(input *Input) {
	input.SetVertical(etk.AlignCenter)
	input.SetPadding(etk.Scale(5))
}

func centerNumericInput(input *NumericInput) {
	input.SetVertical(etk.AlignCenter)
	input.SetPadding(etk.Scale(5))
}

func saveReplay(id int, content []byte) error {
	if id <= 0 {
		return nil
	}

	replayDir := ReplayDir()
	if replayDir == "" {
		ls(fmt.Sprintf("*** %s https://bgammon.org/match/%d", gotext.Get("To download this replay visit"), id))
		return nil
	}

	var (
		timestamp int64
		player1   string
		player2   string
		err       error
	)
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		if bytes.HasPrefix(scanner.Bytes(), []byte("i ")) {
			split := bytes.Split(scanner.Bytes(), []byte(" "))
			if len(split) < 4 {
				return fmt.Errorf("failed to parse replay")
			}

			timestamp, err = strconv.ParseInt(string(split[1]), 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse replay timestamp")
			}

			if bytes.Equal(split[3], []byte(game.client.Username)) {
				player1, player2 = string(split[3]), string(split[2])
			} else {
				player1, player2 = string(split[2]), string(split[3])
			}
		}
	}

	_ = os.MkdirAll(replayDir, 0700)
	filePath := path.Join(replayDir, fmt.Sprintf("%d_%s_%s.match", timestamp, player1, player2))
	err = os.WriteFile(filePath, content, 0600)
	if err != nil {
		return fmt.Errorf("failed to write replay to %s: %s", filePath, err)
	}
	ls(fmt.Sprintf("*** %s: %s", gotext.Get("Downloaded replay"), filePath))
	return nil
}

func showKeyboard() {
	if isSteamDeck() {
		etk.Open("steam://open/keyboard")
		return
	} else if !enableOnScreenKeyboard {
		return
	}
	game.keyboard.SetVisible(true)
	scheduleFrame()
}

func hideKeyboard() {
	if isSteamDeck() {
		etk.Open("steam://close/keyboard")
		return
	} else if !enableOnScreenKeyboard {
		return
	}
	game.keyboard.SetVisible(false)
	scheduleFrame()
}

// Short description.
var _ = gotext.Get("Play backgammon online via bgammon.org")

// Long description.
var _ = gotext.Get("Boxcars is a client for playing backgammon via bgammon.org, a free and open source backgammon service.")

// This string is used when targetting WebAssembly and Android.
var _ = gotext.Get("To download this replay visit")
