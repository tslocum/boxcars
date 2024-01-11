package game

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"code.rocket9labs.com/tslocum/bgammon"
	"code.rocket9labs.com/tslocum/bgammon-tabula-bot/bot"
	"code.rocket9labs.com/tslocum/etk"
	"code.rocket9labs.com/tslocum/tabula"
	"code.rocketnine.space/tslocum/messeji"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/leonelquinteros/gotext"
	"github.com/llgcode/draw2d/draw2dimg"
	"golang.org/x/image/font"
)

type board struct {
	x, y int
	w, h int

	fullHeight bool

	innerW, innerH int

	backgroundImage *ebiten.Image
	baseImage       *image.RGBA

	Sprites *Sprites

	spaceRects   [][4]int
	spaceSprites [][]*Sprite // Space contents

	dragging      *Sprite
	draggingSpace int8
	draggingClick bool // Drag started with mouse click
	lastDragClick time.Time
	moving        *Sprite // Moving automatically

	touchIDs []ebiten.TouchID

	spaceWidth           float64
	barWidth             float64
	triangleOffset       float64
	horizontalBorderSize float64
	verticalBorderSize   float64
	overlapSize          float64

	lastPlayerNumber int8
	lastVariant      int8

	gameState *bgammon.GameState

	opponentRoll1, opponentRoll2, opponentRoll3 int8
	opponentRollStale                           bool
	playerRoll1, playerRoll2, playerRoll3       int8
	playerRollStale                             bool

	availableStale bool

	opponentMoves [][]int8
	playerMoves   [][]int8

	debug int8 // Print and draw debug information

	Client *Client

	dragX, dragY int

	highlightSpaces [][]int8

	spaceHighlight *ebiten.Image
	foundMoves     map[int]bool

	buttonsGrid             *etk.Grid
	buttonsOnlyRollGrid     *etk.Grid
	buttonsOnlyUndoGrid     *etk.Grid
	buttonsOnlyOKGrid       *etk.Grid
	buttonsDoubleRollGrid   *etk.Grid
	buttonsResignAcceptGrid *etk.Grid
	buttonsUndoOKGrid       *etk.Grid

	selectRollGrid *etk.Grid

	opponentLabel *Label
	playerLabel   *Label

	opponentMovesLabel *etk.Text
	playerMovesLabel   *etk.Text

	opponentPipCount *etk.Text
	playerPipCount   *etk.Text

	timerLabel     *etk.Text
	clockLabel     *etk.Text
	showMenuButton *etk.Button

	menuGrid *etk.Grid

	changePasswordOld  *etk.Input
	changePasswordNew  *etk.Input
	changePasswordGrid *etk.Grid

	highlightCheckbox    *etk.Checkbox
	showPipCountCheckbox *etk.Checkbox
	showMovesCheckbox    *etk.Checkbox
	flipBoardCheckbox    *etk.Checkbox
	accountGrid          *etk.Grid
	settingsGrid         *etk.Grid

	matchStatusGrid *etk.Grid

	replayAuto        time.Time
	replayPauseButton *etk.Button
	replayGrid        *etk.Grid

	inputGrid          *etk.Grid
	showKeyboardButton *etk.Button
	uiGrid             *etk.Grid
	frame              *etk.Frame

	lastKeyboardToggle time.Time

	leaveGameGrid         *etk.Grid
	confirmLeaveGameFrame *etk.Frame

	chatGrid       *etk.Grid
	floatInputGrid *etk.Grid
	floatChatGrid  *etk.Grid

	fontFace   font.Face
	lineHeight int
	lineOffset int

	highlightAvailable bool
	showPipCount       bool
	showMoves          bool
	flipBoard          bool

	widget *BoardWidget

	repositionLock *sync.Mutex
	stateLock      *sync.Mutex

	*sync.Mutex
}

const (
	baseBoardVerticalSize = 25
)

var (
	colorWhite = color.RGBA{255, 255, 255, 255}
	colorBlack = color.RGBA{0, 0, 0, 255}
)

func NewBoard() *board {
	b := &board{
		barWidth:             100,
		triangleOffset:       float64(50),
		horizontalBorderSize: 20,
		verticalBorderSize:   float64(baseBoardVerticalSize),
		overlapSize:          97,
		Sprites: &Sprites{
			sprites: make([]*Sprite, 30),
			num:     30,
		},
		spaceSprites: make([][]*Sprite, bgammon.BoardSpaces),
		spaceRects:   make([][4]int, bgammon.BoardSpaces),
		gameState: &bgammon.GameState{
			Game: bgammon.NewGame(bgammon.VariantBackgammon),
		},
		highlightSpaces:         make([][]int8, 28),
		spaceHighlight:          ebiten.NewImage(1, 1),
		foundMoves:              make(map[int]bool),
		opponentLabel:           NewLabel(colorWhite),
		playerLabel:             NewLabel(colorBlack),
		opponentMovesLabel:      etk.NewText(""),
		playerMovesLabel:        etk.NewText(""),
		opponentPipCount:        etk.NewText("0"),
		playerPipCount:          etk.NewText("0"),
		buttonsGrid:             etk.NewGrid(),
		buttonsOnlyRollGrid:     etk.NewGrid(),
		buttonsOnlyUndoGrid:     etk.NewGrid(),
		buttonsOnlyOKGrid:       etk.NewGrid(),
		buttonsDoubleRollGrid:   etk.NewGrid(),
		buttonsResignAcceptGrid: etk.NewGrid(),
		buttonsUndoOKGrid:       etk.NewGrid(),
		selectRollGrid:          etk.NewGrid(),
		menuGrid:                etk.NewGrid(),
		accountGrid:             etk.NewGrid(),
		settingsGrid:            etk.NewGrid(),
		changePasswordGrid:      etk.NewGrid(),
		uiGrid:                  etk.NewGrid(),
		frame:                   etk.NewFrame(),
		confirmLeaveGameFrame:   etk.NewFrame(),
		chatGrid:                etk.NewGrid(),
		floatChatGrid:           etk.NewGrid(),
		floatInputGrid:          etk.NewGrid(),
		showPipCount:            true,
		highlightAvailable:      true,
		widget:                  NewBoardWidget(),
		fontFace:                mediumFont,
		repositionLock:          &sync.Mutex{},
		stateLock:               &sync.Mutex{},
		Mutex:                   &sync.Mutex{},
	}

	centerText := func(t *etk.Text) {
		t.SetVertical(messeji.AlignCenter)
		t.SetScrollBarVisible(false)
	}

	centerText(b.opponentMovesLabel)
	centerText(b.playerMovesLabel)

	centerText(b.opponentPipCount)
	centerText(b.playerPipCount)

	b.opponentMovesLabel.SetHorizontal(messeji.AlignStart)
	b.playerMovesLabel.SetHorizontal(messeji.AlignEnd)

	b.opponentPipCount.SetHorizontal(messeji.AlignEnd)
	b.playerPipCount.SetHorizontal(messeji.AlignStart)

	b.opponentMovesLabel.SetForegroundColor(color.RGBA{255, 255, 255, 255})
	b.playerMovesLabel.SetForegroundColor(color.RGBA{0, 0, 0, 255})

	b.opponentPipCount.SetForegroundColor(color.RGBA{255, 255, 255, 255})
	b.playerPipCount.SetForegroundColor(color.RGBA{0, 0, 0, 255})

	b.recreateButtonGrid()

	{
		b.menuGrid.AddChildAt(etk.NewButton(gotext.Get("Return"), b.hideMenu), 0, 0, 1, 1)
		b.menuGrid.AddChildAt(etk.NewBox(), 1, 0, 1, 1)
		b.menuGrid.AddChildAt(etk.NewButton(gotext.Get("Settings"), b.showSettings), 2, 0, 1, 1)
		b.menuGrid.AddChildAt(etk.NewBox(), 3, 0, 1, 1)
		b.menuGrid.AddChildAt(etk.NewButton(gotext.Get("Leave"), b.leaveGame), 4, 0, 1, 1)
		b.menuGrid.SetVisible(false)
	}

	{
		headerLabel := etk.NewText(gotext.Get("Change password"))
		headerLabel.SetHorizontal(messeji.AlignCenter)

		oldLabel := &ClickableText{
			Text: etk.NewText(gotext.Get("Current")),
			onSelected: func() {
				b.highlightCheckbox.SetSelected(!b.highlightCheckbox.Selected())
				b.toggleHighlightCheckbox()
			},
		}
		oldLabel.SetVertical(messeji.AlignCenter)

		b.changePasswordOld = etk.NewInput("", "", func(text string) (handled bool) {
			return false
		})
		b.changePasswordOld.Field.SetBackgroundColor(frameColor)
		centerInput(b.changePasswordOld)

		newLabel := &ClickableText{
			Text: etk.NewText(gotext.Get("New")),
			onSelected: func() {
				b.showPipCountCheckbox.SetSelected(!b.showPipCountCheckbox.Selected())
				b.togglePipCountCheckbox()
			},
		}
		newLabel.SetVertical(messeji.AlignCenter)

		b.changePasswordNew = etk.NewInput("", "", func(text string) (handled bool) {
			return false
		})
		b.changePasswordNew.Field.SetBackgroundColor(frameColor)
		centerInput(b.changePasswordNew)

		fieldGrid := etk.NewGrid()
		fieldGrid.SetColumnSizes(-1, -1)
		fieldGrid.SetRowSizes(-1, 20, -1)
		fieldGrid.AddChildAt(oldLabel, 0, 0, 1, 1)
		fieldGrid.AddChildAt(b.changePasswordOld, 1, 0, 2, 1)
		fieldGrid.AddChildAt(newLabel, 0, 2, 1, 1)
		fieldGrid.AddChildAt(b.changePasswordNew, 1, 2, 2, 1)

		b.changePasswordGrid.SetBackground(color.RGBA{40, 24, 9, 255})
		b.changePasswordGrid.SetColumnSizes(20, -1, -1, 20)
		b.changePasswordGrid.SetRowSizes(72, fieldHeight+20+fieldHeight, -1, game.scale(baseButtonHeight))
		b.changePasswordGrid.AddChildAt(headerLabel, 1, 0, 2, 1)
		b.changePasswordGrid.AddChildAt(fieldGrid, 1, 1, 2, 1)
		b.changePasswordGrid.AddChildAt(etk.NewBox(), 1, 2, 1, 1)
		b.changePasswordGrid.AddChildAt(etk.NewButton(gotext.Get("Cancel"), b.hideMenu), 0, 3, 2, 1)
		b.changePasswordGrid.AddChildAt(etk.NewButton(gotext.Get("Submit"), b.selectChangePassword), 2, 3, 2, 1)
		b.changePasswordGrid.SetVisible(false)
	}

	{
		settingsLabel := etk.NewText(gotext.Get("Settings"))
		settingsLabel.SetHorizontal(messeji.AlignCenter)

		b.highlightCheckbox = etk.NewCheckbox(b.toggleHighlightCheckbox)
		b.highlightCheckbox.SetBorderColor(triangleA)
		b.highlightCheckbox.SetCheckColor(triangleA)
		b.highlightCheckbox.SetSelected(b.highlightAvailable)

		highlightLabel := &ClickableText{
			Text: etk.NewText(gotext.Get("Highlight legal moves")),
			onSelected: func() {
				b.highlightCheckbox.SetSelected(!b.highlightCheckbox.Selected())
				b.toggleHighlightCheckbox()
			},
		}
		highlightLabel.SetVertical(messeji.AlignCenter)

		b.showPipCountCheckbox = etk.NewCheckbox(b.togglePipCountCheckbox)
		b.showPipCountCheckbox.SetBorderColor(triangleA)
		b.showPipCountCheckbox.SetCheckColor(triangleA)
		b.showPipCountCheckbox.SetSelected(b.showPipCount)

		pipCountLabel := &ClickableText{
			Text: etk.NewText(gotext.Get("Show pip count")),
			onSelected: func() {
				b.showPipCountCheckbox.SetSelected(!b.showPipCountCheckbox.Selected())
				b.togglePipCountCheckbox()
			},
		}
		pipCountLabel.SetVertical(messeji.AlignCenter)

		b.showMovesCheckbox = etk.NewCheckbox(b.toggleMovesCheckbox)
		b.showMovesCheckbox.SetBorderColor(triangleA)
		b.showMovesCheckbox.SetCheckColor(triangleA)
		b.showMovesCheckbox.SetSelected(b.showMoves)

		movesLabel := &ClickableText{
			Text: etk.NewText(gotext.Get("Show moves")),
			onSelected: func() {
				b.showMovesCheckbox.SetSelected(!b.showMovesCheckbox.Selected())
				b.toggleMovesCheckbox()
			},
		}
		movesLabel.SetVertical(messeji.AlignCenter)

		b.flipBoardCheckbox = etk.NewCheckbox(b.toggleFlipBoardCheckbox)
		b.flipBoardCheckbox.SetBorderColor(triangleA)
		b.flipBoardCheckbox.SetCheckColor(triangleA)
		b.flipBoardCheckbox.SetSelected(b.flipBoard)

		flipBoardLabel := &ClickableText{
			Text: etk.NewText(gotext.Get("Flip board")),
			onSelected: func() {
				b.flipBoardCheckbox.SetSelected(!b.flipBoardCheckbox.Selected())
				b.toggleFlipBoardCheckbox()
			},
		}
		flipBoardLabel.SetVertical(messeji.AlignCenter)

		accountLabel := etk.NewText(gotext.Get("Account"))
		accountLabel.SetVertical(messeji.AlignCenter)

		b.recreateAccountGrid()

		checkboxGrid := etk.NewGrid()
		checkboxGrid.SetColumnSizes(72, 20, -1)
		checkboxGrid.SetRowSizes(-1, 20, -1, 20, -1, 20, -1, 20, -1)
		checkboxGrid.AddChildAt(b.highlightCheckbox, 0, 0, 1, 1)
		checkboxGrid.AddChildAt(highlightLabel, 2, 0, 1, 1)
		checkboxGrid.AddChildAt(b.showPipCountCheckbox, 0, 2, 1, 1)
		checkboxGrid.AddChildAt(pipCountLabel, 2, 2, 1, 1)
		checkboxGrid.AddChildAt(b.showMovesCheckbox, 0, 4, 1, 1)
		checkboxGrid.AddChildAt(movesLabel, 2, 4, 1, 1)
		checkboxGrid.AddChildAt(b.flipBoardCheckbox, 0, 6, 1, 1)
		checkboxGrid.AddChildAt(flipBoardLabel, 2, 6, 1, 1)
		{
			grid := etk.NewGrid()
			grid.AddChildAt(accountLabel, 0, 0, 1, 1)
			grid.AddChildAt(b.accountGrid, 1, 0, 2, 1)
			checkboxGrid.AddChildAt(grid, 0, 8, 3, 1)
		}

		b.settingsGrid.SetBackground(color.RGBA{40, 24, 9, 255})
		b.settingsGrid.SetColumnSizes(20, -1, -1, 20)
		b.settingsGrid.SetRowSizes(72, 72+20+72+20+72+20+72+20+72, 20, -1)
		b.settingsGrid.AddChildAt(settingsLabel, 1, 0, 2, 1)
		b.settingsGrid.AddChildAt(checkboxGrid, 1, 1, 2, 1)
		b.settingsGrid.AddChildAt(etk.NewBox(), 1, 2, 1, 1)
		b.settingsGrid.AddChildAt(etk.NewButton(gotext.Get("Return"), b.hideMenu), 0, 3, 4, 1)
		b.settingsGrid.SetVisible(false)
	}

	{
		leaveGameLabel := etk.NewText(gotext.Get("Leave match?"))
		leaveGameLabel.SetHorizontal(messeji.AlignCenter)
		leaveGameLabel.SetVertical(messeji.AlignCenter)

		b.leaveGameGrid = etk.NewGrid()
		b.leaveGameGrid.SetBackground(color.RGBA{40, 24, 9, 255})
		b.leaveGameGrid.AddChildAt(leaveGameLabel, 0, 0, 2, 1)
		b.leaveGameGrid.AddChildAt(etk.NewButton(gotext.Get("No"), b.cancelLeaveGame), 0, 1, 1, 1)
		b.leaveGameGrid.AddChildAt(etk.NewButton(gotext.Get("Yes"), b.confirmLeaveGame), 1, 1, 1, 1)
		b.leaveGameGrid.SetVisible(false)
	}

	b.showKeyboardButton = etk.NewButton(gotext.Get("Show Keyboard"), b.toggleKeyboard)
	b.recreateInputGrid()

	timerLabel := etk.NewText("0:00")
	timerLabel.SetForegroundColor(triangleA)
	timerLabel.SetScrollBarVisible(false)
	timerLabel.SetSingleLine(true)
	timerLabel.TextField.SetHorizontal(messeji.AlignCenter)
	timerLabel.TextField.SetVertical(messeji.AlignCenter)
	b.timerLabel = timerLabel

	clockLabel := etk.NewText("12:00")
	clockLabel.SetForegroundColor(triangleA)
	clockLabel.SetScrollBarVisible(false)
	clockLabel.SetSingleLine(true)
	clockLabel.TextField.SetHorizontal(messeji.AlignCenter)
	clockLabel.TextField.SetVertical(messeji.AlignCenter)
	b.clockLabel = clockLabel

	b.showMenuButton = etk.NewButton("Menu", b.toggleMenu)

	b.matchStatusGrid = etk.NewGrid()
	if !AutoEnableTouchInput {
		b.matchStatusGrid.SetColumnSizes(int(b.verticalBorderSize/4), -1, -1, -1, int(b.verticalBorderSize/4))
	} else {
		b.matchStatusGrid.SetColumnSizes(int(b.verticalBorderSize/4), -1, -1, int(b.verticalBorderSize/4))
	}
	b.matchStatusGrid.AddChildAt(b.timerLabel, 1, 0, 1, 1)
	b.matchStatusGrid.AddChildAt(b.clockLabel, 2, 0, 1, 1)
	x := 3
	if !AutoEnableTouchInput {
		b.matchStatusGrid.AddChildAt(b.showMenuButton, x, 0, 1, 1)
		x++
	}
	b.matchStatusGrid.AddChildAt(etk.NewBox(), x, 0, 1, 1)

	b.replayPauseButton = etk.NewButton("|>", b.selectReplayPause)

	b.replayGrid = etk.NewGrid()
	b.replayGrid.AddChildAt(etk.NewButton("|<<", b.selectReplayStart), 0, 0, 1, 1)
	b.replayGrid.AddChildAt(etk.NewButton("<<", b.selectReplayJumpBack), 1, 0, 1, 1)
	b.replayGrid.AddChildAt(etk.NewButton("<", b.selectReplayStepBack), 2, 0, 1, 1)
	b.replayGrid.AddChildAt(b.replayPauseButton, 3, 0, 1, 1)
	b.replayGrid.AddChildAt(etk.NewButton(">", b.selectReplayStepForward), 4, 0, 1, 1)
	b.replayGrid.AddChildAt(etk.NewButton(">>", b.selectReplayJumpForward), 5, 0, 1, 1)
	b.replayGrid.AddChildAt(etk.NewButton(">>|", b.selectReplayEnd), 6, 0, 1, 1)

	b.uiGrid.SetBackground(frameColor)
	b.recreateUIGrid()

	b.frame.SetPositionChildren(true)

	{
		f := etk.NewFrame()
		f.AddChild(b.opponentMovesLabel)
		f.AddChild(b.opponentPipCount)
		f.AddChild(b.opponentLabel)
		f.AddChild(b.opponentLabel)
		f.AddChild(b.playerLabel)
		f.AddChild(b.playerPipCount)
		f.AddChild(b.playerMovesLabel)
		f.AddChild(b.uiGrid)
		b.frame.AddChild(f)
	}

	b.frame.AddChild(b.widget)

	{
		const padding = 10
		b.selectRollGrid.SetBackground(frameColor)
		b.selectRollGrid.SetColumnPadding(padding)
		b.selectRollGrid.SetRowPadding(padding)
		b.selectRollGrid.AddChildAt(NewDieButton(1, b.selectRollFunc(1)), 0, 0, 1, 1)
		b.selectRollGrid.AddChildAt(NewDieButton(2, b.selectRollFunc(2)), 1, 0, 1, 1)
		b.selectRollGrid.AddChildAt(NewDieButton(3, b.selectRollFunc(3)), 2, 0, 1, 1)
		b.selectRollGrid.AddChildAt(NewDieButton(4, b.selectRollFunc(4)), 0, 1, 1, 1)
		b.selectRollGrid.AddChildAt(NewDieButton(5, b.selectRollFunc(5)), 1, 1, 1, 1)
		b.selectRollGrid.AddChildAt(NewDieButton(6, b.selectRollFunc(6)), 2, 1, 1, 1)
	}

	b.frame.AddChild(b.buttonsGrid)

	b.frame.AddChild(etk.NewFrame(b.selectRollGrid))

	{
		b.chatGrid.SetBackground(frameColor)
		b.chatGrid.AddChildAt(floatStatusBuffer, 0, 0, 1, 1)
		b.chatGrid.AddChildAt(etk.NewBox(), 0, 1, 1, 1)
		b.chatGrid.AddChildAt(b.floatInputGrid, 0, 2, 1, 1)

		padding := etk.NewBox()
		padding.SetBackground(frameColor)

		g := b.floatChatGrid
		g.SetRowSizes(-1, -1, -1)
		g.AddChildAt(b.chatGrid, 0, 1, 1, 1)
		g.AddChildAt(padding, 0, 2, 1, 1)
		g.SetVisible(false)
		b.frame.AddChild(g)
	}

	b.frame.AddChild(b.floatChatGrid)

	{
		f := etk.NewFrame()
		f.AddChild(b.menuGrid)
		f.AddChild(b.settingsGrid)
		f.AddChild(b.changePasswordGrid)
		f.AddChild(b.leaveGameGrid)
		b.frame.AddChild(f)
	}

	b.frame.AddChild(etk.NewFrame(game.keyboard))

	b.frame.AddChild(game.tutorialFrame)

	b.fontUpdated()

	for i := range b.Sprites.sprites {
		b.Sprites.sprites[i] = b.newSprite(i >= 15)
	}

	return b
}

func (b *board) fontUpdated() {
	fontMutex.Lock()
	m := b.fontFace.Metrics()
	b.lineHeight = m.Height.Round()
	b.lineOffset = m.Ascent.Round()
	fontMutex.Unlock()

	bufferFont := b.fontFace
	if game.scaleFactor <= 1 {
		switch b.fontFace {
		case largeFont:
			bufferFont = mediumFont
		case mediumFont:
			bufferFont = smallFont
		}
	}
	statusBuffer.SetFont(bufferFont, fontMutex)
	floatStatusBuffer.SetFont(bufferFont, fontMutex)
	gameBuffer.SetFont(bufferFont, fontMutex)
	inputBuffer.Field.SetFont(bufferFont, fontMutex)

	if game.TouchInput {
		b.showMenuButton.Label.SetFont(largeFont, fontMutex)
	} else {
		b.showMenuButton.Label.SetFont(smallFont, fontMutex)
	}

	b.showKeyboardButton.Label.SetFont(largeFont, fontMutex)

	b.timerLabel.SetFont(b.fontFace, fontMutex)
	b.clockLabel.SetFont(b.fontFace, fontMutex)

	b.opponentMovesLabel.SetFont(bufferFont, fontMutex)
	b.playerMovesLabel.SetFont(bufferFont, fontMutex)

	b.opponentPipCount.SetFont(bufferFont, fontMutex)
	b.playerPipCount.SetFont(bufferFont, fontMutex)
}

func (b *board) recreateInputGrid() {
	if b.inputGrid == nil {
		b.inputGrid = etk.NewGrid()
	} else {
		b.inputGrid.Clear()
	}
	b.floatInputGrid.Clear()

	if game.TouchInput {
		b.inputGrid.AddChildAt(inputBuffer, 0, 0, 2, 1)
		b.floatInputGrid.AddChildAt(inputBuffer, 0, 0, 2, 1)

		b.inputGrid.AddChildAt(etk.NewBox(), 0, 1, 2, 1)
		b.floatInputGrid.AddChildAt(etk.NewBox(), 0, 1, 2, 1)

		b.inputGrid.AddChildAt(b.showMenuButton, 0, 2, 1, 1)
		b.floatInputGrid.AddChildAt(b.showMenuButton, 0, 2, 1, 1)
		b.inputGrid.AddChildAt(b.showKeyboardButton, 1, 2, 1, 1)
		b.floatInputGrid.AddChildAt(b.showKeyboardButton, 1, 2, 1, 1)

		b.inputGrid.SetRowSizes(52, int(b.horizontalBorderSize/2), -1)
		b.floatInputGrid.SetRowSizes(52, int(b.horizontalBorderSize/2), -1)
	} else {
		b.inputGrid.AddChildAt(inputBuffer, 0, 0, 1, 1)
		b.floatInputGrid.AddChildAt(inputBuffer, 0, 0, 1, 1)
	}
}

func (b *board) recreateUIGrid() {
	b.uiGrid.Clear()
	b.uiGrid.AddChildAt(etk.NewBox(), 0, 0, 1, 1)
	b.uiGrid.AddChildAt(b.matchStatusGrid, 0, 1, 1, 1)
	b.uiGrid.AddChildAt(etk.NewBox(), 0, 2, 1, 1)
	if game.replay {
		f := smallFont
		if defaultFontSize == extraLargeFontSize {
			f = mediumFont
		}
		summary1 := etk.NewText("")
		summary1.SetFont(f, fontMutex)
		summary1.Write(game.replaySummary1)
		summary2 := etk.NewText("")
		summary2.SetFont(f, fontMutex)
		summary2.Write(game.replaySummary2)
		subGrid := etk.NewGrid()
		subGrid.SetBackground(bufferBackgroundColor)
		subGrid.AddChildAt(summary2, 0, 0, 1, 1)
		subGrid.AddChildAt(summary1, 1, 0, 1, 1)

		g := etk.NewGrid()
		g.SetRowSizes(game.scale(baseButtonHeight), int(b.verticalBorderSize/2), -1, int(b.verticalBorderSize/2), lobbyStatusBufferHeight*2)
		g.AddChildAt(b.replayGrid, 0, 0, 1, 1)
		g.AddChildAt(subGrid, 0, 2, 1, 1)
		g.AddChildAt(statusBuffer, 0, 4, 1, 1)
		b.uiGrid.AddChildAt(g, 0, 3, 1, 3)
	} else {
		b.uiGrid.AddChildAt(statusBuffer, 0, 3, 1, 1)
		b.uiGrid.AddChildAt(etk.NewBox(), 0, 4, 1, 1)
		b.uiGrid.AddChildAt(gameBuffer, 0, 5, 1, 1)
	}
	b.uiGrid.AddChildAt(etk.NewBox(), 0, 6, 1, 1)
	b.uiGrid.AddChildAt(b.inputGrid, 0, 7, 1, 1)
}

func (b *board) showButtonGrid(buttonGrid *etk.Grid) {
	b.buttonsOnlyRollGrid.SetVisible(false)
	b.buttonsOnlyUndoGrid.SetVisible(false)
	b.buttonsOnlyOKGrid.SetVisible(false)
	b.buttonsDoubleRollGrid.SetVisible(false)
	b.buttonsResignAcceptGrid.SetVisible(false)
	b.buttonsUndoOKGrid.SetVisible(false)
	if buttonGrid == nil {
		b.buttonsGrid.SetVisible(false)
		return
	}

	grid := b.buttonsGrid
	grid.SetColumnSizes(int(b.horizontalBorderSize)*2+b.innerW, -1)
	grid.SetRowSizes(int(b.verticalBorderSize)*2+b.innerH, -1)
	grid.Clear()
	grid.AddChildAt(buttonGrid, 0, 0, 1, 1)

	buttonGrid.SetVisible(true)
	b.buttonsGrid.SetVisible(true)
}

func (b *board) recreateButtonGrid() {
	buttonGrid := func(grid *etk.Grid, reverse bool, widgets ...etk.Widget) *etk.Grid {
		w := game.scale(250)
		if w > b.innerW/4 {
			w = b.innerW / 4
		}
		if w > b.innerH/4 {
			w = b.innerH / 4
		}
		h := game.scale(125)
		if h > b.innerW/8 {
			h = b.innerW / 8
		}
		if h > b.innerH/8 {
			h = b.innerH / 8
		}
		padding := int(b.barWidth - b.horizontalBorderSize*2)
		if padding < 0 {
			padding = 0
		}

		if grid == nil {
			grid = etk.NewGrid()
		} else {
			grid.Clear()
		}
		grid.SetColumnSizes(-1, w, -1, padding, -1, w, -1, w, -1)
		grid.SetRowSizes(-1, h+75, h, -1)
		x := 1
		for i, w := range widgets {
			if !reverse && len(widgets) == 1 {
				x += 2
				i++
			}
			if i == 1 {
				x += 2
			}
			grid.AddChildAt(w, x, 2, 1, 1)
			x += 2
			if reverse && len(widgets) == 1 {
				x += 4
			}
		}
		grid.AddChildAt(etk.NewBox(), x-1, 2, 1, 1)
		grid.AddChildAt(etk.NewBox(), x-1, 3, 1, 1)
		return grid
	}

	doubleButton := etk.NewButton(gotext.Get("Double"), b.selectDouble)
	rollButton := etk.NewButton(gotext.Get("Roll"), b.selectRoll)
	undoButton := etk.NewButton(gotext.Get("Undo"), b.selectUndo)
	okButton := etk.NewButton(gotext.Get("OK"), b.selectOK)
	resignButton := etk.NewButton(gotext.Get("Resign"), b.selectResign)
	acceptButton := etk.NewButton(gotext.Get("Accept"), b.selectOK)

	b.buttonsOnlyRollGrid = buttonGrid(b.buttonsOnlyRollGrid, false, rollButton)
	b.buttonsOnlyUndoGrid = buttonGrid(b.buttonsOnlyUndoGrid, true, undoButton)
	b.buttonsOnlyOKGrid = buttonGrid(b.buttonsOnlyOKGrid, false, okButton)
	b.buttonsDoubleRollGrid = buttonGrid(b.buttonsDoubleRollGrid, false, doubleButton, rollButton)
	b.buttonsResignAcceptGrid = buttonGrid(b.buttonsResignAcceptGrid, false, resignButton, acceptButton)
	b.buttonsUndoOKGrid = buttonGrid(b.buttonsUndoOKGrid, false, undoButton, okButton)
}

func (b *board) recreateAccountGrid() {
	var w etk.Widget
	if b.Client == nil || (game.Password == "" && b.Client.Password == "") {
		guestLabel := etk.NewText(gotext.Get("Logged in as guest"))
		guestLabel.SetFont(mediumFont, fontMutex)
		guestLabel.SetVertical(messeji.AlignCenter)
		w = guestLabel
	} else {
		changePasswordButton := etk.NewButton(gotext.Get("Change password"), b.showChangePassword)
		w = changePasswordButton
	}
	b.accountGrid.Clear()
	b.accountGrid.AddChildAt(w, 0, 0, 1, 1)
}

func (b *board) cancelLeaveGame() error {
	b.leaveGameGrid.SetVisible(false)
	return nil
}

func (b *board) confirmLeaveGame() error {
	if game.replay {
		game.replay = false
		ev := &bgammon.EventLeft{}
		ev.Player = b.Client.Username
		b.Client.Events <- ev
		if !b.replayAuto.IsZero() {
			b.replayAuto = time.Time{}
			b.replayPauseButton.Label.SetText("|>")
		}
		b.recreateUIGrid()
	} else {
		b.Client.Out <- []byte("leave")
	}
	return nil
}

func (b *board) leaveGame() error {
	b.menuGrid.SetVisible(false)
	b.leaveGameGrid.SetVisible(true)
	return nil
}

func (b *board) showSettings() error {
	b.menuGrid.SetVisible(false)
	b.settingsGrid.SetVisible(true)
	return nil
}

func (b *board) showChangePassword() error {
	b.settingsGrid.SetVisible(false)
	b.changePasswordGrid.SetVisible(true)
	etk.SetFocus(b.changePasswordOld)
	return nil
}

func (b *board) hideMenu() error {
	b.menuGrid.SetVisible(false)
	b.settingsGrid.SetVisible(false)
	b.changePasswordGrid.SetVisible(false)
	b.changePasswordOld.Field.SetText("")
	b.changePasswordNew.Field.SetText("")
	return nil
}

func (b *board) toggleMenu() error {
	if b.menuGrid.Visible() {
		b.menuGrid.SetVisible(false)
		b.settingsGrid.SetVisible(false)
	} else {
		b.menuGrid.SetVisible(true)
	}

	return nil
}

func (b *board) toggleKeyboard() error {
	t := time.Now()
	if !b.lastKeyboardToggle.IsZero() && t.Sub(b.lastKeyboardToggle) < 200*time.Millisecond {
		return nil
	}
	b.lastKeyboardToggle = t

	if game.keyboard.Visible() {
		game.keyboard.SetVisible(false)
		b.floatChatGrid.SetVisible(false)
		b.uiGrid.SetRect(b.uiGrid.Rect())
		b.showKeyboardButton.Label.SetText(gotext.Get("Show Keyboard"))
	} else {
		b.floatChatGrid.SetVisible(true)
		b.floatChatGrid.SetRect(b.floatChatGrid.Rect())
		game.keyboard.SetVisible(true)
		b.showKeyboardButton.Label.SetText(gotext.Get("Hide Keyboard"))
	}
	return nil
}

func (b *board) selectRoll() error {
	b.Client.Out <- []byte("roll")
	return nil
}

func (b *board) selectRollFunc(value int) func() error {
	return func() error {
		b.Client.Out <- []byte(fmt.Sprintf("ok %d", value))
		return nil
	}
}

func (b *board) selectOK() error {
	if b.gameState.MayChooseRoll() {
		b.selectRollGrid.SetVisible(true)
		return nil
	}
	b.Client.Out <- []byte("ok")
	return nil
}

func (b *board) _selectUndo() {
	b.Lock()
	defer b.Unlock()

	if b.gameState.Turn != b.gameState.PlayerNumber {
		return
	}

	l := len(b.gameState.Moves)
	if l == 0 {
		return
	}

	b.dragging = nil
	b.dragX, b.dragY = 0, 0
	b._positionCheckers()

	lastMove := b.gameState.Moves[l-1]
	b.Client.Out <- []byte(fmt.Sprintf("mv %d/%d", lastMove[1], lastMove[0]))

	playSoundEffect(effectMove)
	b.movePiece(lastMove[1], lastMove[0])
	b.gameState.Moves = b.gameState.Moves[:l-1]
}

func (b *board) selectUndo() error {
	go b._selectUndo()
	return nil
}

func (b *board) selectDouble() error {
	b.Client.Out <- []byte("double")
	return nil
}

func (b *board) selectResign() error {
	b.Client.Out <- []byte("resign")
	return nil
}

func (b *board) selectChangePassword() error {
	b.Client.Out <- []byte(fmt.Sprintf("password %s %s", strings.ReplaceAll(b.changePasswordOld.Text(), " ", "_"), strings.ReplaceAll(b.changePasswordNew.Text(), " ", "_")))
	return b.hideMenu()
}

func (b *board) selectReplayStart() error {
	if !game.replay {
		return nil
	}

	if !b.replayAuto.IsZero() {
		b.replayAuto = time.Time{}
		b.replayPauseButton.Label.SetText("|>")
	}

	b.playerRoll1, b.playerRoll2, b.playerRoll3 = 0, 0, 0
	b.opponentRoll1, b.opponentRoll2, b.opponentRoll3 = 0, 0, 0
	game.showReplayFrame(0, false)
	return nil
}

func (b *board) selectReplayJumpBack() error {
	if !game.replay {
		return nil
	}

	if !b.replayAuto.IsZero() {
		b.replayAuto = time.Time{}
		b.replayPauseButton.Label.SetText("|>")
	}

	b.playerRoll1, b.playerRoll2, b.playerRoll3 = 0, 0, 0
	b.opponentRoll1, b.opponentRoll2, b.opponentRoll3 = 0, 0, 0
	replayFrame := game.replayFrame
	replayFrame--
	if replayFrame < 0 {
		replayFrame = 0
	}
	game.showReplayFrame(replayFrame, false)
	return nil
}

func (b *board) selectReplayStepBack() error {
	if !game.replay {
		return nil
	}

	if !b.replayAuto.IsZero() {
		b.replayAuto = time.Time{}
		b.replayPauseButton.Label.SetText("|>")
	}

	// TODO Stepping back moves checkers backwards.

	b.playerRoll1, b.playerRoll2, b.playerRoll3 = 0, 0, 0
	b.opponentRoll1, b.opponentRoll2, b.opponentRoll3 = 0, 0, 0
	replayFrame := game.replayFrame
	replayFrame--
	if replayFrame < 0 {
		replayFrame = 0
	}
	game.showReplayFrame(replayFrame, false)
	return nil
}

func (b *board) selectReplayPause() error {
	if !game.replay {
		return nil
	} else if !b.replayAuto.IsZero() {
		b.replayAuto = time.Time{}
		b.replayPauseButton.Label.SetText("|>")
		return nil
	} else if game.replayFrame >= len(game.replayFrames)-1 {
		return nil
	}
	b.replayAuto = time.Now()
	b.replayPauseButton.Label.SetText("| |")
	autoStart := b.replayAuto
	go func() {
		t := time.NewTicker(3 * time.Second)
		for {
			if b.replayAuto != autoStart {
				return
			}

			frame := game.replayFrames[game.replayFrame]
			if len(frame.Event) != 0 {
				game._handleReplay(&bgammon.GameState{
					Game:         frame.Game.Copy(true),
					PlayerNumber: 1,
					Available:    frame.Game.LegalMoves(true),
					Spectating:   true,
				}, frame.Event, 0, true, true)
			}

			replayFrame := game.replayFrame
			replayFrame++
			if replayFrame >= len(game.replayFrames)-1 {
				time.Sleep(2 * time.Second)
				b.replayAuto = time.Time{}
				b.replayPauseButton.Label.SetText("|>")
				scheduleFrame()
				return
			}

			game.replayFrame = replayFrame
			game.showReplayFrame(replayFrame, false)

			<-t.C
		}
	}()
	return nil
}

func (b *board) selectReplayStepForward() error {
	if !game.replay {
		return nil
	}

	if !b.replayAuto.IsZero() {
		b.replayAuto = time.Time{}
		b.replayPauseButton.Label.SetText("|>")
	}

	frame := game.replayFrames[game.replayFrame]
	if len(frame.Event) != 0 {
		game._handleReplay(&bgammon.GameState{
			Game:         frame.Game.Copy(true),
			PlayerNumber: 1,
			Available:    frame.Game.LegalMoves(true),
			Spectating:   true,
		}, frame.Event, 0, true, true)
	}

	replayFrame := game.replayFrame
	replayFrame++
	if replayFrame < len(game.replayFrames) {
		game.replayFrame = replayFrame
		game.showReplayFrame(replayFrame, false)
	}
	return nil
}

func (b *board) selectReplayJumpForward() error {
	if !game.replay {
		return nil
	}

	if !b.replayAuto.IsZero() {
		b.replayAuto = time.Time{}
		b.replayPauseButton.Label.SetText("|>")
	}

	replayFrame := game.replayFrame
	replayFrame++
	if replayFrame < len(game.replayFrames) {
		game.replayFrame = replayFrame
		game.showReplayFrame(replayFrame, false)
	}
	return nil
}

func (b *board) selectReplayEnd() error {
	if !game.replay {
		return nil
	}

	if !b.replayAuto.IsZero() {
		b.replayAuto = time.Time{}
		b.replayPauseButton.Label.SetText("|>")
	}

	b.playerRoll1, b.playerRoll2, b.playerRoll3 = 0, 0, 0
	b.opponentRoll1, b.opponentRoll2, b.opponentRoll3 = 0, 0, 0
	game.showReplayFrame(len(game.replayFrames)-1, false)
	return nil
}

func (b *board) toggleHighlightCheckbox() error {
	b.highlightAvailable = b.highlightCheckbox.Selected()
	highlight := 0
	if b.highlightAvailable {
		highlight = 1
	}
	b.Client.Out <- []byte(fmt.Sprintf("set highlight %d", highlight))
	return nil
}

func (b *board) togglePipCountCheckbox() error {
	b.showPipCount = b.showPipCountCheckbox.Selected()
	b.updatePlayerLabel()
	b.updateOpponentLabel()
	pips := 0
	if b.showPipCount {
		pips = 1
	}
	b.Client.Out <- []byte(fmt.Sprintf("set pips %d", pips))
	return nil
}

func (b *board) toggleMovesCheckbox() error {
	b.showMoves = b.showMovesCheckbox.Selected()
	b.processState()
	moves := 0
	if b.showMoves {
		moves = 1
	}
	b.Client.Out <- []byte(fmt.Sprintf("set moves %d", moves))
	return nil
}

func (b *board) toggleFlipBoardCheckbox() error {
	b.flipBoard = b.flipBoardCheckbox.Selected()
	b.setSpaceRects()
	b.updateBackgroundImage()

	flipBoard := 0
	if b.flipBoard {
		flipBoard = 1
	}
	b.Client.Out <- []byte(fmt.Sprintf("set flip %d", flipBoard))
	b.Client.Out <- []byte("board")
	return nil
}

func (b *board) newSprite(white bool) *Sprite {
	s := &Sprite{}
	s.colorWhite = white
	s.w, s.h = imgCheckerLight.Bounds().Dx(), imgCheckerLight.Bounds().Dy()
	return s
}

func (b *board) updateBackgroundImage() {
	borderSize := b.horizontalBorderSize
	if borderSize > b.barWidth/2 {
		borderSize = b.barWidth / 2
	}
	frameW := b.w - int((b.horizontalBorderSize-borderSize)*2)

	if b.backgroundImage == nil {
		b.backgroundImage = ebiten.NewImage(b.w, b.h)
		b.baseImage = image.NewRGBA(image.Rect(0, 0, b.w, b.h))
	} else {
		bounds := b.backgroundImage.Bounds()
		if bounds.Dx() != b.w || bounds.Dy() != b.h {
			b.backgroundImage = ebiten.NewImage(b.w, b.h)
			b.baseImage = image.NewRGBA(image.Rect(0, 0, b.w, b.h))
		}
	}

	// Draw table.
	b.backgroundImage.Fill(tableColor)

	// Draw frame.
	{
		x, y := int(b.horizontalBorderSize-borderSize), 0
		w, h := frameW, b.h
		b.backgroundImage.SubImage(image.Rect(x, y, x+w, y+h)).(*ebiten.Image).Fill(frameColor)
	}

	// Draw face.
	{
		x, y := int(b.horizontalBorderSize), int(b.verticalBorderSize)
		w, h := int(b.w)+int(b.horizontalBorderSize), b.h-int(b.verticalBorderSize*2)
		b.backgroundImage.SubImage(image.Rect(x, y, x+w, y+h)).(*ebiten.Image).Fill(faceColor)
	}

	// Draw right edge of frame.
	{
		x, y := int(b.horizontalBorderSize)+b.innerW, int(b.verticalBorderSize)
		w, h := int(b.horizontalBorderSize), b.h
		b.backgroundImage.SubImage(image.Rect(x, y, x+w, y+h)).(*ebiten.Image).Fill(frameColor)
	}

	// Draw bar.
	{
		x, y := int((b.w/2)-int(b.spaceWidth/2)-int(b.barWidth/2)), 0
		w, h := int(b.barWidth), b.h
		b.backgroundImage.SubImage(image.Rect(x, y, x+w, y+h)).(*ebiten.Image).Fill(frameColor)
	}

	// Draw triangles.
	draw.Draw(b.baseImage, image.Rect(0, 0, b.w, b.h), image.NewUniform(color.RGBA{0, 0, 0, 0}), image.Point{}, draw.Src)
	gc := draw2dimg.NewGraphicContext(b.baseImage)
	offsetX, offsetY := float64(b.horizontalBorderSize), float64(b.verticalBorderSize)
	for i := 0; i < 2; i++ {
		triangleTip := float64(b.innerH) / 2
		if i == 0 {
			triangleTip -= b.triangleOffset
		} else {
			triangleTip += b.triangleOffset
		}
		for j := 0; j < 12; j++ {
			colorA := j%2 == 0
			if i == 1 {
				colorA = !colorA
			}

			if colorA {
				gc.SetFillColor(triangleA)
			} else {
				gc.SetFillColor(triangleB)
			}

			tx := b.spaceWidth * float64(j)
			ty := b.innerH * i
			if j >= 6 {
				tx += b.barWidth
			}
			gc.MoveTo(offsetX+float64(tx), offsetY+float64(ty))
			gc.LineTo(offsetX+float64(tx+b.spaceWidth/2), offsetY+triangleTip)
			gc.LineTo(offsetX+float64(tx+b.spaceWidth), offsetY+float64(ty))
			gc.Close()
			gc.Fill()
		}
	}

	// Draw border.
	borderStrokeSize := 2.0
	gc.SetStrokeColor(borderColor)
	// Center.
	gc.SetLineWidth(borderStrokeSize)
	gc.MoveTo(float64(frameW-int(b.spaceWidth))/2-1, float64(0))
	gc.LineTo(float64(frameW-int(b.spaceWidth))/2-1, float64(b.h))
	gc.Stroke()
	if !game.portraitView() {
		// Outside right.
		gc.MoveTo(float64(frameW), float64(0))
		gc.LineTo(float64(frameW), float64(b.h))
		gc.Stroke()
	}
	// Inside left.
	gc.SetLineWidth(borderStrokeSize / 2)
	edge := float64(((float64(b.innerW) + 2 - b.barWidth) / 2) + borderSize)
	gc.MoveTo(float64(borderSize), float64(b.verticalBorderSize))
	gc.LineTo(edge, float64(b.verticalBorderSize))
	gc.LineTo(edge, float64(b.h-int(b.verticalBorderSize)))
	gc.LineTo(float64(borderSize), float64(b.h-int(b.verticalBorderSize)))
	gc.LineTo(float64(borderSize), float64(b.verticalBorderSize))
	gc.Close()
	gc.Stroke()
	// Inside right.
	leftEdge := float64((b.innerW-int(b.barWidth))/2) + borderSize + b.barWidth
	edge = leftEdge + math.Ceil(float64((b.innerW-int(b.barWidth)))/2)
	gc.MoveTo(leftEdge, float64(b.verticalBorderSize))
	gc.LineTo(edge, float64(b.verticalBorderSize))
	gc.LineTo(edge, float64(b.h-int(b.verticalBorderSize)))
	gc.LineTo(leftEdge, float64(b.h-int(b.verticalBorderSize)))
	gc.LineTo(leftEdge, float64(b.verticalBorderSize))
	gc.Close()
	gc.Stroke()
	// Home spaces.
	{
		edgeStart := b.horizontalBorderSize + float64(b.innerW) + b.horizontalBorderSize
		edgeEnd := edgeStart + b.spaceWidth
		gc.MoveTo(float64(edgeStart), float64(b.verticalBorderSize))
		gc.LineTo(edgeEnd, float64(b.verticalBorderSize))
		gc.LineTo(edgeEnd, float64(b.h-int(b.verticalBorderSize)))
		gc.LineTo(float64(edgeStart), float64(b.h-int(b.verticalBorderSize)))
		gc.LineTo(float64(edgeStart), float64(b.verticalBorderSize))
		gc.Close()
		gc.Stroke()
	}
	// Home space divider.
	extraSpace := b.h - int(b.verticalBorderSize)*2 - int(b.overlapSize*10) - 4
	if extraSpace > 0 {
		edgeStart := b.horizontalBorderSize + float64(b.innerW) + b.horizontalBorderSize
		edgeEnd := edgeStart + b.spaceWidth
		divStart := float64(b.h/2 - (extraSpace / 2))
		divEnd := float64(b.h/2 + (extraSpace / 2))

		gc.MoveTo(float64(edgeStart)-1, divStart)
		gc.LineTo(edgeEnd, divStart)
		gc.LineTo(edgeEnd, divEnd)
		gc.LineTo(edgeStart-1, divEnd)
		gc.Close()
		gc.SetFillColor(frameColor)
		gc.Fill()

		gc.MoveTo(float64(edgeStart), divStart)
		gc.LineTo(edgeEnd, divStart)
		gc.Stroke()
		gc.MoveTo(float64(edgeStart), divEnd)
		gc.LineTo(edgeEnd, divEnd)
		gc.Stroke()
	}
	if b.h < game.screenH {
		// Bottom.
		gc.MoveTo(0, float64(b.h))
		gc.LineTo(float64(b.w), float64(b.h))
		gc.Stroke()
	}
	b.backgroundImage.DrawImage(ebiten.NewImageFromImage(b.baseImage), nil)

	// Draw space numbers.
	fontMutex.Lock()
	defer fontMutex.Unlock()

	spaceLabelColor := color.RGBA{121, 96, 60, 255}
	for space, r := range b.spaceRects {
		if space < 1 || space > 24 {
			continue
		} else if b.gameState.PlayerNumber == 1 {
			space = 24 - space + 1
		}

		sp := strconv.Itoa(space)
		if b.gameState.Variant == bgammon.VariantTabula {
			sp = romanNumerals(space)
		}
		bounds := etk.BoundString(b.fontFace, sp)
		x := r[0] + r[2]/2 + int(b.horizontalBorderSize) - bounds.Dx()/2 - 2
		if space == 1 || space > 9 {
			x -= 2
		}
		y := 0
		if b.bottomRow(int8(space)) {
			y = b.h - int(b.verticalBorderSize)
		}
		text.Draw(b.backgroundImage, sp, b.fontFace, x, y+(int(b.verticalBorderSize)-b.lineHeight)/2+b.lineOffset, spaceLabelColor)
	}
}

func (b *board) drawSprite(target *ebiten.Image, sprite *Sprite) {
	x, y := float64(sprite.x), float64(sprite.y)
	if sprite == b.dragging {
		cx, cy := ebiten.CursorPosition()
		if cx != 0 || cy != 0 {
			x, y = float64(cx-sprite.w/2), float64(cy-sprite.h/2)
		} else {
			x, y = float64(b.dragX), float64(b.dragY)
		}
	} else if !sprite.toStart.IsZero() {
		progress := float64(time.Since(sprite.toStart)) / float64(sprite.toTime)
		if x == float64(sprite.toX) && y == float64(sprite.toY) {
			sprite.toStart = time.Time{}
			sprite.x = sprite.toX
			sprite.y = sprite.toY
		} else {
			if x < float64(sprite.toX) {
				x += (float64(sprite.toX) - x) * progress
				if x > float64(sprite.toX) {
					x = float64(sprite.toX)
				}
			} else if x > float64(sprite.toX) {
				x -= (x - float64(sprite.toX)) * progress
				if x < float64(sprite.toX) {
					x = float64(sprite.toX)
				}
			}

			if y < float64(sprite.toY) {
				y += (float64(sprite.toY) - y) * progress
				if y > float64(sprite.toY) {
					y = float64(sprite.toY)
				}
			} else if y > float64(sprite.toY) {
				y -= (y - float64(sprite.toY)) * progress
				if y < float64(sprite.toY) {
					y = float64(sprite.toY)
				}
			}
		}
		// Schedule another frame
		scheduleFrame()
	}

	// Draw shadow.
	{
		op := &ebiten.DrawImageOptions{}
		op.Filter = ebiten.FilterLinear
		op.GeoM.Translate(x, y)
		op.ColorScale.Scale(0, 0, 0, 1)
		target.DrawImage(imgCheckerLight, op)
	}

	// Draw checker.

	checkerScale := 0.94

	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	op.GeoM.Translate(-b.spaceWidth/2, -b.spaceWidth/2)
	op.GeoM.Scale(checkerScale, checkerScale)
	op.GeoM.Translate((b.spaceWidth/2)+x, (b.spaceWidth/2)+y)

	c := lightCheckerColor
	if !sprite.colorWhite {
		c = darkCheckerColor
	}
	op.ColorScale.Scale(0, 0, 0, 1)
	r := float32(c.R) / 0xff
	g := float32(c.G) / 0xff
	bl := float32(c.B) / 0xff
	op.ColorScale.SetR(r)
	op.ColorScale.SetG(g)
	op.ColorScale.SetB(bl)

	target.DrawImage(imgCheckerLight, op)
}

func (b *board) innerBoardCenter(right bool) int {
	if right {
		return b.x + int(b.horizontalBorderSize) + b.innerW - (b.innerW / 4) + int(b.barWidth/4)
	}
	return b.x + int(b.horizontalBorderSize) + b.innerW/4 - int(b.barWidth/4)
}

func (b *board) Draw(screen *ebiten.Image) {
	{
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(b.x), float64(b.y))
		screen.DrawImage(b.backgroundImage, op)
	}

	for space := int8(0); space < bgammon.BoardSpaces; space++ {
		var numPieces int8
		for i, sprite := range b.spaceSprites[space] {
			if sprite == b.dragging || sprite == b.moving {
				continue
			}
			numPieces++

			b.drawSprite(screen, sprite)

			var overlayText string
			if i > 5 {
				overlayText = fmt.Sprintf("%d", numPieces)
			}
			if sprite.premove {
				if overlayText != "" {
					overlayText += " "
				}
				overlayText += "%"
			}
			if overlayText == "" {
				continue
			}

			labelColor := color.RGBA{255, 255, 255, 255}
			if sprite.colorWhite {
				labelColor = color.RGBA{0, 0, 0, 255}
			}

			fontMutex.Lock()
			bounds := etk.BoundString(b.fontFace, overlayText)
			overlayImage := ebiten.NewImage(bounds.Dx()*2, bounds.Dy()*2)
			text.Draw(overlayImage, overlayText, b.fontFace, 0, bounds.Dy(), labelColor)
			fontMutex.Unlock()

			x, y, w, h := b.stackSpaceRect(space, numPieces-1)
			x += (w / 2) - (bounds.Dx() / 2)
			y += (h / 2) - (bounds.Dy() / 2)
			x, y = b.offsetPosition(space, x, y)

			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(x), float64(y))
			screen.DrawImage(overlayImage, op)
		}
	}

	b.stateLock.Lock()
	var highlightSpaces [][]int8
	dragging := b.dragging
	if b.dragging != nil && b.highlightAvailable && b.draggingSpace != -1 {
		highlightSpaces = b.highlightSpaces
	}
	b.stateLock.Unlock()

	// Draw space hover overlay when dragging.
	if dragging != nil {
		for i := range highlightSpaces[b.draggingSpace] {
			m := highlightSpaces[b.draggingSpace][i]
			x, y, _, _ := b.spaceRect(m)
			x, y = b.offsetPosition(m, x, y)
			if b.bottomRow(m) {
				y += b.h/2 - int(b.overlapSize*5) - int(b.verticalBorderSize) - 4
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(x), float64(y))
			op.ColorScale.Scale(0.2, 0.2, 0.2, 0.2)
			screen.DrawImage(b.spaceHighlight, op)
		}

		dx, dy := b.dragX, b.dragY

		x, y := ebiten.CursorPosition()
		if x != 0 || y != 0 {
			dx, dy = x, y
		}

		space := b.spaceAt(dx, dy)
		if space >= 0 && space <= 25 {
			x, y, _, _ := b.spaceRect(space)
			x, y = b.offsetPosition(space, x, y)
			if b.bottomRow(space) {
				y += b.h/2 - int(b.overlapSize*5) - int(b.verticalBorderSize) - 4
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(x), float64(y))
			op.ColorScale.Scale(0.1, 0.1, 0.1, 0.1)
			screen.DrawImage(b.spaceHighlight, op)
		}
	}

	// Draw opponent dice.

	const diceFadeAlpha = 0.1
	diceGap := 10.0
	if game.screenW < 800 {
		v := 10.0 * (float64(game.screenW) / 800)
		if v < diceGap {
			diceGap = v
		}
	}
	if game.screenH < 800 {
		v := 10.0 * (float64(game.screenH) / 800)
		if v < diceGap {
			diceGap = v
		}
	}

	opponent := b.gameState.OpponentPlayer()
	if opponent.Name != "" {
		innerCenter := b.innerBoardCenter(false)
		alpha := float32(1.0)
		if b.gameState.Turn == 0 {
			if b.gameState.Roll2 == 0 {
				alpha = diceFadeAlpha
			}
		} else if b.opponentRollStale || b.gameState.Turn == 1 {
			alpha = diceFadeAlpha
		}

		op := &ebiten.DrawImageOptions{}
		op.ColorScale.ScaleAlpha(alpha)

		d1, d2, d3 := b.opponentRoll1, b.opponentRoll2, b.opponentRoll3

		if b.gameState.Turn == 0 {
			if d2 != 0 {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(innerCenter-diceSize/2), float64(b.y+(b.innerH/2))-diceGap-float64(diceSize))
				screen.DrawImage(diceImage(d2), op)
			}
		} else {
			if d1 != 0 && d2 != 0 {
				if d3 != 0 {
					{
						op.GeoM.Translate(float64(innerCenter-diceSize)-diceGap-float64(diceSize/2)-diceGap, float64(b.y+(b.innerH/2))-diceGap-float64(diceSize))
						screen.DrawImage(diceImage(d1), op)
					}

					{
						op.GeoM.Reset()
						op.GeoM.Translate(float64(innerCenter)-float64(diceSize)/2, float64(b.y+(b.innerH/2))-diceGap-float64(diceSize))
						screen.DrawImage(diceImage(d2), op)
					}

					{
						op.GeoM.Reset()
						op.GeoM.Translate(float64(innerCenter)+diceGap+float64(diceSize/2)+diceGap, float64(b.y+(b.innerH/2))-diceGap-float64(diceSize))
						screen.DrawImage(diceImage(d3), op)
					}
				} else {
					{
						op.GeoM.Translate(float64(innerCenter-diceSize)-diceGap, float64(b.y+(b.innerH/2))-diceGap-float64(diceSize))
						screen.DrawImage(diceImage(d1), op)
					}

					{
						op.GeoM.Reset()
						op.GeoM.Translate(float64(innerCenter)+diceGap, float64(b.y+(b.innerH/2))-diceGap-float64(diceSize))
						screen.DrawImage(diceImage(d2), op)
					}
				}
			}
		}
	}

	// Draw player dice.

	player := b.gameState.LocalPlayer()
	if player.Name != "" {
		innerCenter := b.innerBoardCenter(true)
		alpha := float32(1.0)
		if b.gameState.Turn == 0 {
			if b.gameState.Roll1 == 0 {
				alpha = diceFadeAlpha
			}
		} else if b.playerRollStale || b.gameState.Turn == 2 {
			alpha = diceFadeAlpha
		}

		op := &ebiten.DrawImageOptions{}
		op.ColorScale.ScaleAlpha(alpha)

		d1, d2, d3 := b.playerRoll1, b.playerRoll2, b.playerRoll3

		if b.gameState.Turn == 0 {
			if d1 != 0 {
				op.GeoM.Translate(float64(innerCenter-diceSize/2), float64(b.y+(b.innerH/2))-diceGap-float64(diceSize))
				screen.DrawImage(diceImage(d1), op)
			}
		} else {
			if d1 != 0 && d2 != 0 {
				if d3 != 0 {
					{
						op.GeoM.Translate(float64(innerCenter-diceSize)-diceGap-float64(diceSize/2)-diceGap, float64(b.y+(b.innerH/2))-diceGap-float64(diceSize))
						screen.DrawImage(diceImage(d1), op)
					}

					{
						op.GeoM.Reset()
						op.GeoM.Translate(float64(innerCenter)-float64(diceSize)/2, float64(b.y+(b.innerH/2))-diceGap-float64(diceSize))
						screen.DrawImage(diceImage(d2), op)
					}

					{
						op.GeoM.Reset()
						op.GeoM.Translate(float64(innerCenter)+diceGap+float64(diceSize/2)+diceGap, float64(b.y+(b.innerH/2))-diceGap-float64(diceSize))
						screen.DrawImage(diceImage(d3), op)
					}
				} else {
					{
						op.GeoM.Translate(float64(innerCenter-diceSize)-diceGap, float64(b.y+(b.innerH/2))-diceGap-float64(diceSize))
						screen.DrawImage(diceImage(d1), op)
					}

					{
						op.GeoM.Reset()
						op.GeoM.Translate(float64(innerCenter)+diceGap, float64(b.y+(b.innerH/2))-diceGap-float64(diceSize))
						screen.DrawImage(diceImage(d2), op)
					}
				}
			}
		}
	}

	// Draw sidebar border.
	if !game.portraitView() && b.h < game.screenH {
		screen.SubImage(image.Rect(b.w-1, 0, b.w, game.screenH)).(*ebiten.Image).Fill(color.RGBA{0, 0, 0, 255})
	}
}

func (b *board) drawDraggedCheckers(screen *ebiten.Image) {
	if b.moving != nil {
		b.drawSprite(screen, b.moving)
	}
	if b.dragging != nil {
		b.drawSprite(screen, b.dragging)
	}
}

func (b *board) setRect(x, y, w, h int) {
	if OptimizeSetRect && b.x == x && b.y == y && b.w == w && b.h == h {
		b.recreateButtonGrid()
		return
	}

	b.x, b.y, b.w, b.h = x, y, w, h
	maxWidth := int(float64(b.h) * 1.333)
	if b.w > maxWidth {
		b.w = maxWidth
	}

	b.triangleOffset = (float64(b.h) - (b.verticalBorderSize * 2)) / 15

	const horizontalSpaces = 14
	b.spaceWidth = (float64(b.w) - (b.horizontalBorderSize * 2)) / horizontalSpaces
	b.barWidth = b.spaceWidth

	b.overlapSize = (((float64(b.h) - (b.verticalBorderSize * 2)) - (b.triangleOffset * 2)) / 2) / 5
	if b.overlapSize > b.spaceWidth*0.94 {
		b.overlapSize = b.spaceWidth * 0.94
	}

	b.barWidth = float64(b.spaceWidth) + b.horizontalBorderSize
	b.spaceWidth = ((float64(b.w) - (b.horizontalBorderSize * 2)) - b.barWidth) / (horizontalSpaces - 1)
	if b.barWidth < 1 {
		b.barWidth = 1
	}
	if b.spaceWidth < 1 {
		b.spaceWidth = 1
	}

	b.innerW = int(float64(b.w) - (b.horizontalBorderSize * 2) - b.spaceWidth)
	b.innerH = int(float64(b.h) - (b.verticalBorderSize * 2))

	b.triangleOffset = (float64(b.innerH)+b.verticalBorderSize-b.spaceWidth*10)/2 + b.spaceWidth/12

	loadImageAssets(int(b.spaceWidth))

	for i := 0; i < b.Sprites.num; i++ {
		s := b.Sprites.sprites[i]
		s.w, s.h = imgCheckerLight.Bounds().Dx(), imgCheckerLight.Bounds().Dy()
	}

	b.setSpaceRects()
	b.updateBackgroundImage()
	b.recreateInputGrid()
	b.processState()

	inputAndButtons := 52
	if game.TouchInput {
		inputAndButtons = 52 + int(b.verticalBorderSize)/2 + game.scale(baseButtonHeight)
	}
	matchStatus := 36
	if game.scaleFactor >= 1.25 {
		matchStatus = 44
	}
	b.uiGrid.SetRowSizes(int(b.verticalBorderSize/2), matchStatus, int(b.verticalBorderSize/2), -1, int(b.verticalBorderSize/2), -1, int(b.verticalBorderSize/2), int(inputAndButtons))

	{
		dialogWidth := game.scale(620)
		if dialogWidth > game.screenW {
			dialogWidth = game.screenW
		}
		dialogHeight := game.scale(100)
		if dialogHeight > game.screenH {
			dialogHeight = game.screenH
		}

		x, y := game.screenW/2-dialogWidth/2, game.screenH/2-dialogHeight+int(b.verticalBorderSize)
		b.menuGrid.SetRect(image.Rect(x, y, x+dialogWidth, y+dialogHeight))
	}

	{
		dialogWidth := game.scale(620)
		if dialogWidth > game.screenW {
			dialogWidth = game.screenW
		}
		dialogHeight := 72 + 72 + 20 + 72 + 20 + 72 + 20 + 72 + 20 + 72 + 20 + game.scale(baseButtonHeight)
		if dialogHeight > game.screenH {
			dialogHeight = game.screenH
		}

		x, y := game.screenW/2-dialogWidth/2, game.screenH/2-dialogHeight/2
		b.settingsGrid.SetRect(image.Rect(x, y, x+dialogWidth, y+dialogHeight))
		b.changePasswordGrid.SetRect(image.Rect(x, y, x+dialogWidth, y+dialogHeight))
		b.selectRollGrid.SetRect(image.Rect(x, y, x+dialogWidth, y+dialogHeight))
	}

	{
		dialogWidth := game.scale(400)
		if dialogWidth > game.screenW {
			dialogWidth = game.screenW
		}
		dialogHeight := game.scale(100)
		if dialogHeight > game.screenH {
			dialogHeight = game.screenH
		}

		x, y := game.screenW/2-dialogWidth/2, game.screenH/2-dialogHeight+int(b.verticalBorderSize)
		b.leaveGameGrid.SetRect(image.Rect(x, y, x+dialogWidth, y+dialogHeight))
	}

	b.updateOpponentLabel()
	b.updatePlayerLabel()

	b.recreateButtonGrid()

	b.menuGrid.SetColumnSizes(-1, game.scale(10), -1, game.scale(10), -1)

	b.chatGrid.SetRowSizes(-1, int(b.verticalBorderSize)/2, inputAndButtons)

	var padding int
	if b.w >= 600 {
		padding = 20
	} else if b.w >= 400 {
		padding = 12
	} else if b.w >= 300 {
		padding = 10
	} else if b.w >= 200 {
		padding = 7
	} else if b.w >= 100 {
		padding = 5
	}
	b.opponentMovesLabel.SetPadding(padding)
	b.playerMovesLabel.SetPadding(padding)
	b.opponentPipCount.SetPadding(padding)
	b.playerPipCount.SetPadding(padding)
}

func (b *board) updateOpponentLabel() {
	player := b.gameState.OpponentPlayer()
	label := b.opponentLabel

	var text string
	if b.gameState.Points > 1 && len(player.Name) > 0 {
		text = fmt.Sprintf("%s (%d)", player.Name, player.Points)
	} else if len(player.Name) > 0 {
		text = player.Name
	} else if b.gameState.Started.IsZero() {
		text = fmt.Sprintf("%s...", gotext.Get("Waiting"))
	} else {
		text = gotext.Get("Left match")
	}
	if label.Text.Text() != text {
		label.SetText(text)
	}

	label.active = b.gameState.Turn == player.Number
	label.Text.TextField.SetForegroundColor(label.activeColor)

	fontMutex.Lock()
	bounds := etk.BoundString(largeFont, text)
	fontMutex.Unlock()

	padding := 13
	innerCenter := b.innerBoardCenter(false)
	x := innerCenter - bounds.Dx()/2
	y := b.y + (b.innerH / 2) - (bounds.Dy() / 2) + int(b.verticalBorderSize)
	r := image.Rect(x, y, x+bounds.Dx(), y+bounds.Dy())

	if r.Eq(label.Rect()) && r.Dx() != 0 && r.Dy() != 0 {
		label.updateBackground()
		return
	}
	{
		newRect := image.Rect(int(b.horizontalBorderSize), y-bounds.Dy()*2, x, y+bounds.Dy()*4)
		b.opponentMovesLabel.SetRect(newRect)
	}
	{
		newRect := r.Inset(-padding)
		if !label.Rect().Eq(newRect) {
			label.SetRect(newRect)
		}
	}
	{
		newRect := image.Rect(x+bounds.Dx(), y-bounds.Dy(), b.innerW/2-int(b.barWidth)/2+int(b.horizontalBorderSize), y+bounds.Dy()*2)
		b.opponentPipCount.SetRect(newRect)
	}

	var moves []byte
	if len(b.opponentMoves) != 0 {
		moves = bytes.ReplaceAll(bgammon.FormatMoves(b.opponentMoves), []byte(" "), []byte("\n"))
	}
	if b.opponentMovesLabel.Text() != string(moves) {
		b.opponentMovesLabel.SetText(string(moves))
	}

	if b.showPipCount {
		b.opponentPipCount.SetVisible(true)
		pipCount := strconv.Itoa(b.gameState.Pips(player.Number))
		if b.opponentPipCount.Text() != pipCount {
			b.opponentPipCount.SetText(pipCount)
		}
	} else {
		b.opponentPipCount.SetVisible(false)
	}
}

func (b *board) updatePlayerLabel() {
	player := b.gameState.LocalPlayer()
	label := b.playerLabel

	var text string
	if b.gameState.Points > 1 && len(player.Name) > 0 {
		text = fmt.Sprintf("%s (%d)", player.Name, player.Points)
	} else if len(player.Name) > 0 {
		text = player.Name
	} else if b.gameState.Started.IsZero() {
		text = fmt.Sprintf("%s...", gotext.Get("Waiting"))
	} else {
		text = gotext.Get("Left match")
	}
	if label.Text.Text() != text {
		label.SetText(text)
	}

	label.active = b.gameState.Turn == player.Number
	label.Text.TextField.SetForegroundColor(label.activeColor)

	fontMutex.Lock()
	bounds := etk.BoundString(largeFont, text)
	defer fontMutex.Unlock()

	padding := 13
	innerCenter := b.innerBoardCenter(true)
	x := innerCenter - int(float64(bounds.Dx()/2))
	y := b.y + (b.innerH / 2) - (bounds.Dy() / 2) + int(b.verticalBorderSize)
	r := image.Rect(x, y, x+bounds.Dx(), y+bounds.Dy())
	if r.Eq(label.Rect()) && r.Dx() != 0 && r.Dy() != 0 {
		label.updateBackground()
		return
	}
	{
		newRect := image.Rect(x+bounds.Dx(), y-bounds.Dy()*2, int(b.horizontalBorderSize)+b.innerW, y+bounds.Dy()*4)
		b.playerMovesLabel.SetRect(newRect)
	}
	{
		newRect := r.Inset(-padding)
		if !label.Rect().Eq(newRect) {
			label.SetRect(newRect)
		}
	}
	{
		newRect := image.Rect(b.innerW/2+int(b.barWidth)/2+int(b.horizontalBorderSize), y-bounds.Dy(), x, y+bounds.Dy()*2)
		if !b.playerPipCount.Rect().Eq(newRect) {
			b.playerPipCount.SetRect(newRect)
		}
	}

	var moves []byte
	if len(b.playerMoves) != 0 {
		moves = bytes.ReplaceAll(bgammon.FormatMoves(b.playerMoves), []byte(" "), []byte("\n"))
	}
	if b.playerMovesLabel.Text() != string(moves) {
		b.playerMovesLabel.SetText(string(moves))
	}

	if b.showPipCount {
		b.playerPipCount.SetVisible(true)
		pipCount := strconv.Itoa(b.gameState.Pips(player.Number))
		if b.playerPipCount.Text() != pipCount {
			b.playerPipCount.SetText(pipCount)
		}
	} else {
		b.playerPipCount.SetVisible(false)
	}
}

func (b *board) offsetPosition(space int8, x, y int) (int, int) {
	if space == bgammon.SpaceHomePlayer || space == bgammon.SpaceHomeOpponent {
		x += 1
	}
	return b.x + x + int(b.horizontalBorderSize), b.y + y + int(b.verticalBorderSize)
}

// Do not call _positionCheckers directly.  Call processState instead.
func (b *board) _positionCheckers() {
	for space := int8(0); space < bgammon.BoardSpaces; space++ {
		sprites := b.spaceSprites[space]

		for i := range sprites {
			s := sprites[i]
			if b.dragging == s {
				continue
			}

			x, y, w, _ := b.stackSpaceRect(space, int8(i))
			s.x, s.y = b.offsetPosition(space, x, y)
			// Center piece in space
			s.x += (w - s.w) / 2
		}
	}
}

func (b *board) spriteAt(x, y int) (*Sprite, int8) {
	space := b.spaceAt(x, y)
	if space == -1 {
		return nil, -1
	}
	pieces := b.spaceSprites[space]
	if len(pieces) == 0 {
		return nil, -1
	}
	return pieces[len(pieces)-1], space
}

func (b *board) spaceAt(x, y int) int8 {
	for i := int8(0); i < bgammon.BoardSpaces; i++ {
		sx, sy, sw, sh := b.spaceRect(i)
		sx, sy = b.offsetPosition(i, sx, sy)
		if x >= sx && x < sx+sw && y >= sy && y < sy+sh {
			return i
		}
	}
	return -1
}

func (b *board) setSpaceRects() {
	var x, y, w, h int
	for space := int8(0); space < bgammon.BoardSpaces; space++ {
		if !b.bottomRow(space) {
			y = 0
		} else {
			y = int((float64(b.h) / 2) - b.verticalBorderSize)
		}

		w = int(b.spaceWidth)

		var hspace int8 // horizontal space
		var add int
		if space == bgammon.SpaceBarPlayer || space == bgammon.SpaceBarOpponent {
			hspace = 6
			add = 1
			w = int(b.barWidth)
		} else if space == bgammon.SpaceHomePlayer || space == bgammon.SpaceHomeOpponent {
			hspace = 13
			add = int(b.horizontalBorderSize)
		} else if space <= 6 {
			hspace = space - 1
		} else if space <= 12 {
			hspace = space - 1
			add = int(b.barWidth)
		} else if space <= 18 {
			hspace = 24 - space
			add = int(b.barWidth)
		} else {
			hspace = 24 - space
		}

		x = int((b.spaceWidth * float64(hspace)) + float64(add))

		h = int((float64(b.h) - (b.verticalBorderSize * 2)) / 2)

		if space == bgammon.SpaceHomePlayer || space == bgammon.SpaceHomeOpponent {
			x += int(b.horizontalBorderSize)
		}

		b.spaceRects[space] = [4]int{x, y, w, h}
	}

	// Flip board.
	if b.gameState.PlayerNumber == 1 && !b.flipBoard {
		for i := 0; i < 6; i++ {
			j, k, l, m := 1+i, 12-i, 13+i, 24-i
			b.spaceRects[j], b.spaceRects[k], b.spaceRects[l], b.spaceRects[m] = b.spaceRects[k], b.spaceRects[j], b.spaceRects[m], b.spaceRects[l]
		}
	}

	r := b.spaceRects[1]
	highlightHeight := int(b.overlapSize*5) + 4
	bounds := b.spaceHighlight.Bounds()
	if bounds.Dx() != r[2] || bounds.Dy() != highlightHeight {
		b.spaceHighlight = ebiten.NewImage(r[2], highlightHeight)
		b.spaceHighlight.Fill(color.RGBA{255, 255, 255, 51})
	}
}

// relX, relY
func (b *board) spaceRect(space int8) (x, y, w, h int) {
	rect := b.spaceRects[space]
	return rect[0], rect[1], rect[2], rect[3]
}

func (b *board) bottomRow(space int8) bool {
	var bottomStart int8 = 1
	var bottomEnd int8 = 12
	bottomBar := bgammon.SpaceBarPlayer
	bottomHome := bgammon.SpaceHomePlayer
	if b.gameState.Variant == bgammon.VariantTabula {
		bottomStart = 13
		bottomEnd = 24
	} else if b.flipBoard || b.gameState.PlayerNumber == 2 {
		bottomStart = 1
		bottomEnd = 12
	}
	return space == bottomBar || space == bottomHome || (space >= bottomStart && space <= bottomEnd)
}

// relX, relY
func (b *board) stackSpaceRect(space int8, stack int8) (x, y, w, h int) {
	x, y, _, h = b.spaceRect(space)

	// Stack pieces
	var o int
	osize := float64(stack)
	if stack > 4 {
		osize = 3.5
	}
	if b.bottomRow(space) {
		osize += 1.0
	}
	o = int(osize * float64(b.overlapSize))
	padding := int(b.spaceWidth - b.overlapSize)
	if b.bottomRow(space) {
		o += padding
	} else {
		o -= padding - 3
	}
	if !b.bottomRow(space) {
		y += o
	} else {
		y = y + (h - o)
	}

	w, h = int(b.spaceWidth), int(b.spaceWidth)
	if space == bgammon.SpaceBarPlayer || space == bgammon.SpaceBarOpponent {
		w = int(b.barWidth)
	}

	return x, y, w, h
}

func (b *board) processState() {
	b.stateLock.Lock()
	defer b.stateLock.Unlock()

	if b.lastPlayerNumber != b.gameState.PlayerNumber || b.lastVariant != b.gameState.Variant {
		b.setSpaceRects()
		b.updateBackgroundImage()
	}
	b.lastPlayerNumber = b.gameState.PlayerNumber
	b.lastVariant = b.gameState.Variant

	if b.flipBoard || b.gameState.PlayerNumber == 2 {
		if b.opponentLabel.activeColor != colorBlack {
			b.opponentLabel.activeColor = colorBlack
			b.opponentLabel.SetForegroundColor(colorBlack)
			b.opponentPipCount.SetForegroundColor(colorBlack)
			b.opponentMovesLabel.SetForegroundColor(colorBlack)
			b.opponentLabel.lastActive = !b.opponentLabel.active
			b.opponentLabel.updateBackground()
		}
		if b.playerLabel.activeColor != colorWhite {
			b.playerLabel.activeColor = colorWhite
			b.playerLabel.SetForegroundColor(colorWhite)
			b.playerPipCount.SetForegroundColor(colorWhite)
			b.playerMovesLabel.SetForegroundColor(colorWhite)
			b.playerLabel.lastActive = !b.opponentLabel.active
			b.playerLabel.updateBackground()
		}
	} else {
		if b.opponentLabel.activeColor != colorWhite {
			b.opponentLabel.activeColor = colorWhite
			b.opponentLabel.SetForegroundColor(colorWhite)
			b.opponentPipCount.SetForegroundColor(colorWhite)
			b.opponentMovesLabel.SetForegroundColor(colorWhite)
			b.opponentLabel.lastActive = !b.opponentLabel.active
			b.opponentLabel.updateBackground()
		}
		if b.playerLabel.activeColor != colorBlack {
			b.playerLabel.activeColor = colorBlack
			b.playerLabel.SetForegroundColor(colorBlack)
			b.playerPipCount.SetForegroundColor(colorBlack)
			b.playerMovesLabel.SetForegroundColor(colorBlack)
			b.playerLabel.lastActive = !b.opponentLabel.active
			b.playerLabel.updateBackground()
		}
	}

	var showGrid *etk.Grid
	if !b.gameState.Spectating && !b.availableStale {
		if b.gameState.MayRoll() {
			if b.gameState.MayDouble() {
				showGrid = b.buttonsDoubleRollGrid
			} else {
				showGrid = b.buttonsOnlyRollGrid
			}
		} else if b.gameState.MayOK() {
			if b.gameState.MayResign() {
				showGrid = b.buttonsResignAcceptGrid
			} else if len(b.gameState.Moves) != 0 {
				showGrid = b.buttonsUndoOKGrid
			} else {
				showGrid = b.buttonsOnlyOKGrid
			}
		} else if b.gameState.Winner == 0 && b.gameState.Turn != 0 && b.gameState.Turn == b.gameState.PlayerNumber && len(b.gameState.Moves) != 0 {
			showGrid = b.buttonsOnlyUndoGrid
		}
	}
	b.showButtonGrid(showGrid)

	var nextWhite int
	var nextBlack int

	for space := 0; space < bgammon.BoardSpaces; space++ {
		b.spaceSprites[space] = b.spaceSprites[space][:0]
		spaceValue := b.gameState.Board[space]

		white := spaceValue < 0
		if b.flipBoard {
			white = !white
		}

		abs := spaceValue
		if abs < 0 {
			abs *= -1
		}
		for i := int8(0); i < abs; i++ {
			var s *Sprite
			if !white {
				for i := range b.Sprites.sprites[nextBlack:] {
					if !b.Sprites.sprites[nextBlack+i].colorWhite {
						s = b.Sprites.sprites[nextBlack+i]
						nextBlack = nextBlack + 1 + i
						break
					}
				}
			} else {
				for i := range b.Sprites.sprites[nextWhite:] {
					if b.Sprites.sprites[nextWhite+i].colorWhite {
						s = b.Sprites.sprites[nextWhite+i]
						nextWhite = nextWhite + 1 + i
						break
					}
				}
			}
			if s == nil {
				panic("no checker sprite available")
			}

			if i >= abs {
				s.premove = true
			}
			b.spaceSprites[space] = append(b.spaceSprites[space], s)
		}
	}

	b._positionCheckers()

	if b.showMoves && b.gameState.Turn == 1 {
		b.playerMoves = expandMoves(b.gameState.Moves)
	} else if b.showMoves && b.gameState.Turn == 2 {
		b.opponentMoves = expandMoves(b.gameState.Moves)
	} else {
		b.playerMoves, b.opponentMoves = nil, nil
	}

	b.updateOpponentLabel()
	b.updatePlayerLabel()

	if b.gameState.Turn != b.gameState.PlayerNumber {
		return
	}

	tabulaBoard := bot.TabulaBoard(b.gameState.Game.Board)
	tabulaBoard[tabula.SpaceRoll1], tabulaBoard[tabula.SpaceRoll2], tabulaBoard[tabula.SpaceRoll3], tabulaBoard[tabula.SpaceRoll4] = int8(b.gameState.Game.Roll1), int8(b.gameState.Game.Roll2), 0, 0
	if b.gameState.Variant == bgammon.VariantTabula {
		tabulaBoard[tabula.SpaceRoll3] = int8(b.gameState.Game.Roll3)
	} else if b.gameState.Game.Roll1 == b.gameState.Game.Roll2 {
		tabulaBoard[tabula.SpaceRoll3], tabulaBoard[tabula.SpaceRoll4] = int8(b.gameState.Game.Roll1), int8(b.gameState.Game.Roll2)
	}
	enteredPlayer, enteredOpponent := int8(1), int8(1)
	if b.gameState.Variant != bgammon.VariantBackgammon {
		if !b.gameState.Player1.Entered {
			enteredPlayer = 0
		}
		if !b.gameState.Player2.Entered {
			enteredOpponent = 0
		}
	}
	tabulaBoard[tabula.SpaceEnteredPlayer], tabulaBoard[tabula.SpaceEnteredOpponent], tabulaBoard[tabula.SpaceVariant] = enteredPlayer, enteredOpponent, b.gameState.Variant
	for _, m := range b.gameState.Moves {
		delta := int8(bgammon.SpaceDiff(m[0], m[1], b.gameState.Variant))
		switch {
		case tabulaBoard[tabula.SpaceRoll1] == delta:
			tabulaBoard[tabula.SpaceRoll1] = 0
			continue
		case tabulaBoard[tabula.SpaceRoll2] == delta:
			tabulaBoard[tabula.SpaceRoll2] = 0
			continue
		case tabulaBoard[tabula.SpaceRoll3] == delta:
			tabulaBoard[tabula.SpaceRoll3] = 0
			continue
		case tabulaBoard[tabula.SpaceRoll4] == delta:
			tabulaBoard[tabula.SpaceRoll4] = 0
			continue
		}
		switch {
		case tabulaBoard[tabula.SpaceRoll1] > delta:
			tabulaBoard[tabula.SpaceRoll1] = 0
			continue
		case tabulaBoard[tabula.SpaceRoll2] > delta:
			tabulaBoard[tabula.SpaceRoll2] = 0
			continue
		case tabulaBoard[tabula.SpaceRoll3] > delta:
			tabulaBoard[tabula.SpaceRoll3] = 0
			continue
		case tabulaBoard[tabula.SpaceRoll4] > delta:
			tabulaBoard[tabula.SpaceRoll4] = 0
			continue
		}
	}
	onBar := tabulaBoard[tabula.SpaceBarPlayer] != 0
	available, _ := tabulaBoard.Available(1)
	for space := 0; space < 28; space++ {
		b.highlightSpaces[space] = b.highlightSpaces[space][:0]
	}
	for i := range available {
		var moves [][2]int8
		for _, m := range available[i] {
			if m[0] == 0 && m[1] == 0 {
				break
			}
			moves = append(moves, m)
		}
		sort.Slice(moves, func(i, j int) bool {
			if moves[i][0] != moves[j][0] {
				return moves[i][0] > moves[j][0]
			}
			return moves[i][1] > moves[j][1]
		})

		originalFrom := int8(-1)
		lastFrom := int8(-1)
		lastTo := int8(-1)
		for j := range moves {
			move := moves[j]
			if lastFrom != -1 {
				if move[0] == lastTo {
					var exists bool
					for _, existing := range b.highlightSpaces[originalFrom] {
						if int8(existing) == move[1] {
							exists = true
							break
						}
					}
					if !exists {
						b.highlightSpaces[originalFrom] = append(b.highlightSpaces[originalFrom], move[1])
					}

					exists = false
					for _, existing := range b.highlightSpaces[lastFrom] {
						if existing == move[1] {
							exists = true
							break
						}
					}
					if !exists {
						b.highlightSpaces[lastFrom] = append(b.highlightSpaces[lastFrom], move[1])
					}
				} else {
					originalFrom = move[0]
				}
			} else {
				originalFrom = move[0]
			}
			lastFrom = move[0]
			lastTo = move[1]

			var exists bool
			for _, existing := range b.highlightSpaces[move[0]] {
				if existing == move[1] {
					exists = true
					break
				}
			}
			if !exists {
				b.highlightSpaces[move[0]] = append(b.highlightSpaces[move[0]], move[1])
			}
		}
	}
	if onBar {
		for space := 0; space <= 25; space++ {
			b.highlightSpaces[space] = b.highlightSpaces[space][:0]
		}
	}
}

// _movePiece returns after moving the piece.
func (b *board) _movePiece(sprite *Sprite, from int8, to int8, speed int8, pause bool) {
	moveTime := (650 * time.Millisecond) / time.Duration(speed)
	pauseTime := 250 * time.Millisecond

	b.moving = sprite

	space := to // Immediately go to target space

	stack := len(b.spaceSprites[space])
	if stack == 1 && sprite.colorWhite != b.spaceSprites[space][0].colorWhite {
		stack = 0 // Hit
	} else if space != to {
		stack++
	}

	x, y, w, _ := b.stackSpaceRect(space, int8(stack))
	x, y = b.offsetPosition(space, x, y)
	// Center piece in space
	x += (w - int(b.spaceWidth)) / 2

	if !game.Instant {
		sprite.toX = x
		sprite.toY = y
		sprite.toTime = moveTime
		sprite.toStart = time.Now()

		time.Sleep(moveTime)
	}

	sprite.x = x
	sprite.y = y
	sprite.toStart = time.Time{}

	b.spaceSprites[to] = append(b.spaceSprites[to], sprite)
	for i, s := range b.spaceSprites[from] {
		if s == sprite {
			b.spaceSprites[from] = append(b.spaceSprites[from][:i], b.spaceSprites[from][i+1:]...)
			break
		}
	}
	b.moving = nil

	if game.Instant {
		return
	} else if pause {
		time.Sleep(pauseTime)
	} else {
		time.Sleep(50 * time.Millisecond)
	}
}

// movePiece returns when finished moving the piece.
func (b *board) movePiece(from int8, to int8) {
	pieces := b.spaceSprites[from]
	if len(pieces) == 0 {
		log.Printf("ERROR: NO SPRITE FOR MOVE %d/%d", from, to)
		return
	}

	sprite := pieces[len(pieces)-1]

	var moveAfter *Sprite
	if len(b.spaceSprites[to]) == 1 {
		if sprite.colorWhite != b.spaceSprites[to][0].colorWhite {
			moveAfter = b.spaceSprites[to][0]
		}
	}

	b._movePiece(sprite, from, to, 1, moveAfter == nil)
	if moveAfter != nil {
		bar := bgammon.SpaceBarPlayer
		if b.gameState.Turn == b.gameState.PlayerNumber {
			bar = bgammon.SpaceBarOpponent
		}
		b._movePiece(moveAfter, to, bar, 1, true)
	}
}

// PlayingGame returns whether the active game is being played.
func (b *board) playingGame() bool {
	return (b.gameState.Player1.Name != "" || b.gameState.Player2.Name != "") && !b.gameState.Spectating
}

func (b *board) playerTurn() bool {
	return b.playingGame() && (b.gameState.MayRoll() || b.gameState.Turn == b.gameState.PlayerNumber)
}

func (b *board) startDrag(s *Sprite, space int8, click bool) {
	b.dragging = s
	b.draggingSpace = space
	b.draggingClick = click
	b.lastDragClick = time.Now()
}

// finishDrag calls processState. It does not need to be locked.
func (b *board) finishDrag(x int, y int, click bool) {
	if b.dragging == nil {
		return
	} else if b.draggingClick && !click {
		return
	}

	if x != 0 || y != 0 { // 0,0 is returned when the touch is released
		b.dragX, b.dragY = x, y
	}

	var dropped *Sprite
	if ((b.draggingClick && click) || (!b.draggingClick && len(ebiten.AppendTouchIDs(b.touchIDs[:0])) == 0 && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft))) && time.Since(b.lastDragClick) >= 50*time.Millisecond {
		dropped = b.dragging
		b.dragging = nil
	}
	if dropped != nil {
		if x == 0 && y == 0 {
			x, y = ebiten.CursorPosition()
			if x == 0 && y == 0 {
				x, y = b.dragX, b.dragY
			}
		}

		index := b.spaceAt(x, y)
		// Bear off by dragging outside the board.
		if index == -1 {
			index = bgammon.SpaceHomePlayer
		}

		if !b.draggingClick && index == b.draggingSpace && !b.lastDragClick.IsZero() && time.Since(b.lastDragClick) < 500*time.Millisecond {
			b.startDrag(dropped, index, true)
			if game.TouchInput {
				r := b.spaceRects[index]
				offset := int(b.spaceWidth) + int(b.overlapSize)*4
				if !b.bottomRow(index) {
					b.dragX, b.dragY = int(b.horizontalBorderSize)+r[0], r[1]+offset
				} else {
					b.dragX, b.dragY = int(b.horizontalBorderSize)+r[0], r[1]+r[3]-offset
				}
				if index == bgammon.SpaceBarPlayer || index == bgammon.SpaceBarOpponent {
					b.dragX += int(b.horizontalBorderSize / 2)
				}
			}
			b.processState()
			scheduleFrame()
			b.lastDragClick = time.Now()
			return
		}

		var processed bool
		if index >= 0 && b.Client != nil {
		ADDPREMOVE:
			for sp, pieces := range b.spaceSprites {
				space := int8(sp)
				for _, piece := range pieces {
					if piece == dropped {
						if space != index {
							playSoundEffect(effectMove)
							b.gameState.AddLocalMove([]int8{space, index})
							b.processState()
							scheduleFrame()
							processed = true
							b.Client.Out <- []byte(fmt.Sprintf("mv %d/%d", space, index))
						} else if time.Since(b.lastDragClick) < 500*time.Millisecond && b.gameState.MayBearOff(b.gameState.PlayerNumber, true) {
							homeStart, homeEnd := bgammon.HomeRange(b.gameState.PlayerNumber, b.gameState.Variant)
							if homeEnd < homeStart {
								homeStart, homeEnd = homeEnd, homeStart
							}
							if index >= homeStart && index <= homeEnd {
								b.Client.Out <- []byte(fmt.Sprintf("mv %d/off", index))
							}
						} else if time.Since(b.lastDragClick) < 500*time.Millisecond && space == bgammon.SpaceHomePlayer && !b.gameState.Player1.Entered {
							var found bool
							for _, m := range b.gameState.Available {
								if m[0] == bgammon.SpaceHomePlayer && bgammon.SpaceDiff(m[0], m[1], b.gameState.Variant) == b.gameState.Roll1 {
									b.Client.Out <- []byte(fmt.Sprintf("mv %d/%d", m[0], m[1]))
									found = true
									break
								}
							}
							if !found {
								for _, m := range b.gameState.Available {
									if m[0] == bgammon.SpaceHomePlayer {
										b.Client.Out <- []byte(fmt.Sprintf("mv %d/%d", m[0], m[1]))
										break
									}
								}
							}
						}
						break ADDPREMOVE
					}
				}
			}
		}
		if !processed {
			b.processState()
			scheduleFrame()
		}
		b.lastDragClick = time.Now()
	}
}

func (b *board) Update() {
	b.finishDrag(0, 0, inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft))
	if b.dragging != nil && b.draggingClick {
		x, y := ebiten.CursorPosition()
		if x != 0 || y != 0 {
			sprite := b.dragging
			sprite.x = x - (sprite.w / 2)
			sprite.y = y - (sprite.h / 2)
		}
	}

	if b.moving != nil || b.dragging != nil {
		scheduleFrame()
	}
}

type Label struct {
	*etk.Text
	active      bool
	activeColor color.RGBA
	lastActive  bool
	bg          *ebiten.Image
}

func NewLabel(c color.RGBA) *Label {
	l := &Label{
		Text:        etk.NewText(""),
		activeColor: c,
	}
	l.Text.SetFont(largeFont, fontMutex)
	l.Text.SetForegroundColor(c)
	l.Text.SetScrollBarVisible(false)
	l.Text.SetSingleLine(true)
	l.Text.SetHorizontal(messeji.AlignCenter)
	l.Text.SetVertical(messeji.AlignCenter)
	return l
}

func (l *Label) updateBackground() {
	if l.TextField.Text() == "" {
		l.bg = nil
		return
	}

	r := l.Rect()
	if l.bg != nil {
		bounds := l.bg.Bounds()
		if bounds.Dx() != r.Dx() || bounds.Dy() != r.Dy() {
			l.bg = ebiten.NewImage(r.Dx(), r.Dy())
		}
	} else {
		l.bg = ebiten.NewImage(r.Dx(), r.Dy())
	}

	bgColor := color.RGBA{0, 0, 0, 20}
	borderSize := 2
	if l.active {
		l.bg.Fill(l.activeColor)

		bounds := l.bg.Bounds()
		l.bg.SubImage(image.Rect(0, 0, bounds.Dx(), bounds.Dy()).Inset(borderSize)).(*ebiten.Image).Fill(bgColor)
	} else {
		l.bg.Fill(bgColor)
	}

	l.lastActive = l.active
}

func (l *Label) SetRect(r image.Rectangle) {
	if r.Dx() == 0 || r.Dy() == 0 {
		l.bg = nil
		l.Text.SetRect(r)
		return
	}

	l.Text.SetRect(r)
	l.updateBackground()
}

func (l *Label) SetActive(active bool) {
	l.active = active
}

func (l *Label) SetText(t string) {
	r := l.Rect()
	if r.Empty() || l.TextField.Text() == t {
		return
	}
	l.TextField.SetText(t)
	l.updateBackground()
}

func (l *Label) Draw(screen *ebiten.Image) error {
	if l.bg == nil {
		return nil
	}
	if l.active != l.lastActive {
		l.updateBackground()
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(l.Rect().Min.X), float64(l.Rect().Min.Y))
	screen.DrawImage(l.bg, op)
	return l.Text.Draw(screen)
}

type DieButton struct {
	*etk.Button
	Value int8
}

func NewDieButton(value int8, onSelected func() error) *DieButton {
	return &DieButton{
		Button: etk.NewButton(" ", onSelected),
		Value:  value,
	}
}

func (b *DieButton) Draw(screen *ebiten.Image) error {
	dieFace := diceImage(b.Value)
	if dieFace == nil {
		return nil
	}

	err := b.Button.Draw(screen)
	if err != nil {
		return err
	}

	r := b.Rect()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(r.Min.X+(r.Dx()-int(game.Board.spaceWidth))/2), float64(r.Min.Y+(r.Dy()-int(game.Board.spaceWidth))/2))
	screen.DrawImage(dieFace, op)
	return nil
}

type BoardWidget struct {
	*etk.Box
}

func NewBoardWidget() *BoardWidget {
	return &BoardWidget{
		Box: etk.NewBox(),
	}
}

func (bw *BoardWidget) HandleMouse(cursor image.Point, pressed bool, clicked bool) (handled bool, err error) {
	if !pressed && !clicked && game.Board.dragging == nil {
		return false, nil
	}

	b := game.Board
	if b.Client == nil || !b.playerTurn() {
		return false, nil
	}

	cx, cy := cursor.X, cursor.Y

	if b.dragging == nil {
		// TODO allow grabbing multiple pieces by grabbing further down the stack
		if !handled && b.playerTurn() && clicked && (b.lastDragClick.IsZero() || time.Since(b.lastDragClick) >= 50*time.Millisecond) {
			s, space := b.spriteAt(cx, cy)
			if s != nil && s.colorWhite == (b.flipBoard || b.gameState.PlayerNumber == 2) && space != bgammon.SpaceHomeOpponent && (game.Board.gameState.Variant == bgammon.VariantBackgammon || space != bgammon.SpaceHomePlayer || !game.Board.gameState.Player1.Entered) {
				b.startDrag(s, space, false)
				handled = true
			}
		}
	}

	x, y := cx, cy
	b.finishDrag(x, y, clicked)

	if b.dragging != nil {
		sprite := b.dragging
		sprite.x = x - (sprite.w / 2)
		sprite.y = y - (sprite.h / 2)
	}
	return handled, nil
}

func expandMoves(moves [][]int8) [][]int8 {
	var expanded bool
	for _, m := range moves {
		expandedMoves, ok := game.Board.gameState.ExpandMove(m, m[0], nil, true)
		if !ok {
			return moves
		}
		if len(expandedMoves) != 1 {
			expanded = true
			break
		}
	}
	if !expanded {
		return moves
	}
	var newMoves [][]int8
	for _, m := range moves {
		expandedMoves, ok := game.Board.gameState.ExpandMove(m, m[0], nil, true)
		if !ok {
			return moves
		}
		newMoves = append(newMoves, expandedMoves...)
	}
	return newMoves
}

func romanNumerals(i int) string {
	var roman string = ""
	var numbers = []int{1, 4, 5, 9, 10}
	var numerals = []string{"I", "IV", "V", "IX", "X"}
	var index = len(numerals) - 1
	for i > 0 {
		for numbers[index] <= i {
			roman += numerals[index]
			i -= numbers[index]
		}
		index -= 1
	}
	return roman
}
