package game

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
	"strconv"
	"sync"
	"time"

	"code.rocket9labs.com/tslocum/bgammon"
	"code.rocket9labs.com/tslocum/etk"
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
	draggingSpace int
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

	lastPlayerNumber int

	gameState *bgammon.GameState

	debug int // Print and draw debug information

	Client *Client

	dragX, dragY int

	availableCache       []int
	updateAvailableCache bool

	spaceHighlight *ebiten.Image

	buttonsGrid             *etk.Grid
	buttonsOnlyRollGrid     *etk.Grid
	buttonsOnlyUndoGrid     *etk.Grid
	buttonsOnlyOKGrid       *etk.Grid
	buttonsDoubleRollGrid   *etk.Grid
	buttonsResignAcceptGrid *etk.Grid
	buttonsUndoOKGrid       *etk.Grid

	opponentLabel *Label
	playerLabel   *Label

	opponentPipCount *etk.Text
	playerPipCount   *etk.Text

	timerLabel     *etk.Text
	clockLabel     *etk.Text
	showMenuButton *etk.Button

	menuGrid *etk.Grid

	showPipCountCheckbox *etk.Checkbox
	highlightCheckbox    *etk.Checkbox
	settingsGrid         *etk.Grid

	matchStatusGrid *etk.Grid

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

	showPipCount       bool
	highlightAvailable bool

	widget *BoardWidget

	repositionLock *sync.Mutex
	stateLock      *sync.Mutex

	*sync.Mutex
}

const (
	baseBoardVerticalSize = 25
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
			Game: bgammon.NewGame(false),
		},
		spaceHighlight:          ebiten.NewImage(1, 1),
		opponentLabel:           NewLabel(color.RGBA{255, 255, 255, 255}),
		playerLabel:             NewLabel(color.RGBA{0, 0, 0, 255}),
		opponentPipCount:        etk.NewText("0"),
		playerPipCount:          etk.NewText("0"),
		buttonsGrid:             etk.NewGrid(),
		buttonsOnlyRollGrid:     etk.NewGrid(),
		buttonsOnlyUndoGrid:     etk.NewGrid(),
		buttonsOnlyOKGrid:       etk.NewGrid(),
		buttonsDoubleRollGrid:   etk.NewGrid(),
		buttonsResignAcceptGrid: etk.NewGrid(),
		buttonsUndoOKGrid:       etk.NewGrid(),
		menuGrid:                etk.NewGrid(),
		settingsGrid:            etk.NewGrid(),
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

	centerText(b.opponentPipCount)
	centerText(b.playerPipCount)

	b.opponentPipCount.SetHorizontal(messeji.AlignEnd)
	b.playerPipCount.SetHorizontal(messeji.AlignStart)

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
		settingsLabel := etk.NewText(gotext.Get("Settings"))
		settingsLabel.SetHorizontal(messeji.AlignCenter)

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

		checkboxGrid := etk.NewGrid()
		checkboxGrid.SetRowSizes(-1, 20, -1)
		checkboxGrid.AddChildAt(b.showPipCountCheckbox, 0, 0, 1, 1)
		checkboxGrid.AddChildAt(pipCountLabel, 1, 0, 4, 1)
		checkboxGrid.AddChildAt(b.highlightCheckbox, 0, 2, 1, 1)
		checkboxGrid.AddChildAt(highlightLabel, 1, 2, 4, 1)

		b.settingsGrid.SetBackground(color.RGBA{40, 24, 9, 255})
		b.settingsGrid.SetColumnSizes(20, -1, -1, 20)
		b.settingsGrid.SetRowSizes(72, 72+20+72, 20, -1)
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
	b.matchStatusGrid.AddChildAt(b.timerLabel, 0, 0, 1, 1)
	b.matchStatusGrid.AddChildAt(b.clockLabel, 1, 0, 1, 1)
	if !AutoEnableTouchInput {
		b.matchStatusGrid.AddChildAt(b.showMenuButton, 2, 0, 1, 1)
	}

	b.uiGrid.AddChildAt(b.matchStatusGrid, 0, 0, 1, 1)
	b.uiGrid.AddChildAt(etk.NewBox(), 0, 1, 1, 1)
	b.uiGrid.AddChildAt(statusBuffer, 0, 2, 1, 1)
	b.uiGrid.AddChildAt(etk.NewBox(), 0, 3, 1, 1)
	b.uiGrid.AddChildAt(gameBuffer, 0, 4, 1, 1)
	b.uiGrid.AddChildAt(etk.NewBox(), 0, 5, 1, 1)
	b.uiGrid.AddChildAt(b.inputGrid, 0, 6, 1, 1)

	b.frame.SetPositionChildren(true)

	{
		f := etk.NewFrame()
		f.AddChild(b.opponentPipCount)
		f.AddChild(b.opponentLabel)
		f.AddChild(b.opponentLabel)
		f.AddChild(b.playerLabel)
		f.AddChild(b.playerPipCount)
		f.AddChild(b.uiGrid)
		b.frame.AddChild(f)
	}

	b.frame.AddChild(b.widget)

	b.frame.AddChild(b.buttonsGrid)

	{
		b.chatGrid.SetBackground(tableColor)
		b.chatGrid.AddChildAt(floatStatusBuffer, 0, 0, 1, 1)
		b.chatGrid.AddChildAt(etk.NewBox(), 0, 1, 1, 1)
		b.chatGrid.AddChildAt(b.floatInputGrid, 0, 2, 1, 1)

		padding := etk.NewBox()
		padding.SetBackground(tableColor)

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
		f.AddChild(b.leaveGameGrid)
		b.frame.AddChild(f)
	}

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

	b.opponentPipCount.SetFont(bufferFont, fontMutex)
	b.playerPipCount.SetFont(bufferFont, fontMutex)
}

func (b *board) recreateInputGrid() {
	if b.inputGrid == nil {
		b.inputGrid = etk.NewGrid()
	} else {
		b.inputGrid.Empty()
	}
	b.floatInputGrid.Empty()

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
	grid.Empty()
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
			grid.Empty()
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

	button := func(label string, onSelected func() error) *etk.Button {
		btn := etk.NewButton(label, onSelected)
		btn.Label.SetFont(largeFont, fontMutex)
		return btn
	}

	doubleButton := button(gotext.Get("Double"), b.selectDouble)
	rollButton := button(gotext.Get("Roll"), b.selectRoll)
	undoButton := button(gotext.Get("Undo"), b.selectUndo)
	okButton := button(gotext.Get("OK"), b.selectOK)
	resignButton := button(gotext.Get("Resign"), b.selectResign)
	acceptButton := button(gotext.Get("Accept"), b.selectOK)

	b.buttonsOnlyRollGrid = buttonGrid(b.buttonsOnlyRollGrid, false, rollButton)
	b.buttonsOnlyUndoGrid = buttonGrid(b.buttonsOnlyUndoGrid, true, undoButton)
	b.buttonsOnlyOKGrid = buttonGrid(b.buttonsOnlyOKGrid, false, okButton)
	b.buttonsDoubleRollGrid = buttonGrid(b.buttonsDoubleRollGrid, false, doubleButton, rollButton)
	b.buttonsResignAcceptGrid = buttonGrid(b.buttonsResignAcceptGrid, false, resignButton, acceptButton)
	b.buttonsUndoOKGrid = buttonGrid(b.buttonsUndoOKGrid, false, undoButton, okButton)
}

func (b *board) cancelLeaveGame() error {
	b.leaveGameGrid.SetVisible(false)
	return nil
}

func (b *board) confirmLeaveGame() error {
	b.Client.Out <- []byte("leave")
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

func (b *board) hideMenu() error {
	b.menuGrid.SetVisible(false)
	b.settingsGrid.SetVisible(false)
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
		game.keyboard.Hide()
		b.floatChatGrid.SetVisible(false)
		b.uiGrid.SetRect(b.uiGrid.Rect())
		b.showKeyboardButton.Label.SetText(gotext.Get("Show Keyboard"))
	} else {
		b.floatChatGrid.SetVisible(true)
		b.floatChatGrid.SetRect(b.floatChatGrid.Rect())
		game.keyboard.Show()
		b.showKeyboardButton.Label.SetText(gotext.Get("Hide Keyboard"))
	}
	return nil
}

func (b *board) selectRoll() error {
	b.Client.Out <- []byte("roll")
	return nil
}

func (b *board) selectOK() error {
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

func (b *board) togglePipCountCheckbox() error {
	b.showPipCount = b.showPipCountCheckbox.Selected()
	b.updatePlayerLabel()
	b.updateOpponentLabel()
	return nil
}

func (b *board) toggleHighlightCheckbox() error {
	b.highlightAvailable = b.highlightCheckbox.Selected()
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
	// Outside right.
	gc.MoveTo(float64(frameW), float64(0))
	gc.LineTo(float64(frameW), float64(b.h))
	gc.Stroke()
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
	edge = leftEdge + float64((b.innerW-int(b.barWidth))/2)
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
	if !b.fullHeight {
		// Outside left.
		gc.SetLineWidth(1)
		gc.MoveTo(float64(0), float64(0))
		gc.LineTo(float64(0), float64(b.h))
		// Top.
		gc.MoveTo(0, float64(0))
		gc.LineTo(float64(b.w), float64(0))
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
		bounds := etk.BoundString(b.fontFace, sp)
		x := r[0] + r[2]/2 + int(b.horizontalBorderSize) - bounds.Dx()/2 - 2
		if space == 1 || space > 9 {
			x -= 2
		}
		y := 0
		if b.bottomRow(space) {
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
	return b.x + int(b.horizontalBorderSize) + (b.innerW / 4) - int(b.horizontalBorderSize*1.5)
}

func (b *board) Draw(screen *ebiten.Image) {
	{
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(b.x), float64(b.y))
		screen.DrawImage(b.backgroundImage, op)
	}

	for space := 0; space < bgammon.BoardSpaces; space++ {
		var numPieces int
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
	playerRoll := b.gameState.Roll1
	opponentRoll := b.gameState.Roll2
	roll1 := b.gameState.Roll1
	roll2 := b.gameState.Roll2
	if b.gameState.PlayerNumber == 2 {
		playerRoll, opponentRoll = opponentRoll, playerRoll
	}
	var highlightSpaces []int
	dragging := b.dragging
	if b.dragging != nil && b.highlightAvailable && b.draggingSpace != -1 {
		highlightSpaces = b.allAvailableMoves()
	}
	b.stateLock.Unlock()

	// Draw space hover overlay when dragging
	if dragging != nil {
		for _, m := range highlightSpaces {
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

	// Draw opponent dice

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
		if b.gameState.Turn == 0 {
			if opponentRoll != 0 {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(innerCenter-diceSize/2), float64(b.y+(b.innerH/2))-diceGap-float64(diceSize))
				screen.DrawImage(diceImage(opponentRoll), op)
			}
		} else if b.gameState.Turn != b.gameState.PlayerNumber && roll1 != 0 {
			{
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(innerCenter-diceSize)-diceGap, float64(b.y+(b.innerH/2))-diceGap-float64(diceSize))
				screen.DrawImage(diceImage(roll1), op)
			}

			{
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(innerCenter)+diceGap, float64(b.y+(b.innerH/2))-diceGap-float64(diceSize))
				screen.DrawImage(diceImage(roll2), op)
			}
		}
	}

	// Draw player dice

	player := b.gameState.LocalPlayer()
	if player.Name != "" {
		innerCenter := b.innerBoardCenter(true)
		if b.gameState.Turn == 0 {
			if playerRoll != 0 {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(innerCenter-diceSize/2), float64(b.y+(b.innerH/2))-diceGap-float64(diceSize))
				screen.DrawImage(diceImage(playerRoll), op)
			}
		} else if b.gameState.Turn == b.gameState.PlayerNumber && roll1 != 0 {
			{
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(innerCenter-diceSize)-diceGap, float64(b.y+(b.innerH/2))-diceGap-float64(diceSize))
				screen.DrawImage(diceImage(roll1), op)
			}

			{
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(innerCenter)+diceGap, float64(b.y+(b.innerH/2))-diceGap-float64(diceSize))
				screen.DrawImage(diceImage(roll2), op)
			}
		}
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
	maxWidth := int(float64(b.h) * 1.2)
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
		inputAndButtons = 52 + int(b.horizontalBorderSize)/2 + game.scale(baseButtonHeight)
	}
	matchStatus := 36
	if game.scaleFactor >= 1.25 {
		matchStatus = 44
	}
	b.uiGrid.SetRowSizes(matchStatus, int(b.horizontalBorderSize/2), -1, int(b.horizontalBorderSize/2), -1, int(b.horizontalBorderSize/2), int(inputAndButtons))

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
		dialogHeight := 72 + 72 + 20 + 72 + 20 + game.scale(baseButtonHeight)
		if dialogHeight > game.screenH {
			dialogHeight = game.screenH
		}

		x, y := game.screenW/2-dialogWidth/2, game.screenH/2-dialogHeight/2
		b.settingsGrid.SetRect(image.Rect(x, y, x+dialogWidth, y+dialogHeight))
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

	b.chatGrid.SetRowSizes(-1, int(b.horizontalBorderSize)/2, inputAndButtons)

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
	b.opponentPipCount.SetPadding(padding / 2)
	b.playerPipCount.SetPadding(padding)
}

func (b *board) allAvailableMoves() []int {
	if !b.updateAvailableCache {
		return b.availableCache
	}
	var all []int
	found := make(map[int]bool)
	for _, move := range b.gameState.Available {
		if move[0] == b.draggingSpace {
			for _, m := range allMoves(b.gameState.Game, move[0], move[1]) {
				if !found[m] {
					all = append(all, m)
					found[m] = true
				}
			}
		}
	}
	b.availableCache = all
	b.updateAvailableCache = false
	return b.availableCache
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
		text = "Waiting..."
	} else {
		text = "Left match"
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
	x := innerCenter - int(float64(bounds.Dx()/2))
	y := b.y + (b.innerH / 2) - (bounds.Dy() / 2) + int(b.verticalBorderSize)
	r := image.Rect(x, y, x+bounds.Dx(), y+bounds.Dy())

	if r.Eq(label.Rect()) && r.Dx() != 0 && r.Dy() != 0 {
		label.updateBackground()
		return
	}
	{
		newRect := r.Inset(-padding)
		if !label.Rect().Eq(newRect) {
			label.SetRect(newRect)
		}
	}
	{
		newRect := image.Rect(x+bounds.Dx(), y-bounds.Dy(), b.innerW/2-int(b.barWidth)/2+int(b.horizontalBorderSize), y+bounds.Dy()*2)
		if !b.opponentPipCount.Rect().Eq(newRect) {
			b.opponentPipCount.SetRect(newRect)
		}
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
		text = "Waiting..."
	} else {
		text = "Left match"
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

func (b *board) offsetPosition(space, x, y int) (int, int) {
	if space == bgammon.SpaceHomePlayer || space == bgammon.SpaceHomeOpponent {
		x += 1
	}
	return b.x + x + int(b.horizontalBorderSize), b.y + y + int(b.verticalBorderSize)
}

// Do not call _positionCheckers directly.  Call processState instead.
func (b *board) _positionCheckers() {
	for space := 0; space < bgammon.BoardSpaces; space++ {
		sprites := b.spaceSprites[space]

		for i := range sprites {
			s := sprites[i]
			if b.dragging == s {
				continue
			}

			x, y, w, _ := b.stackSpaceRect(space, i)
			s.x, s.y = b.offsetPosition(space, x, y)
			// Center piece in space
			s.x += (w - s.w) / 2
		}
	}
}

func (b *board) spriteAt(x, y int) (*Sprite, int) {
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

func (b *board) spaceAt(x, y int) int {
	for i := 0; i < bgammon.BoardSpaces; i++ {
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
	for space := 0; space < bgammon.BoardSpaces; space++ {
		if !b.bottomRow(space) {
			y = 0
		} else {
			y = int((float64(b.h) / 2) - b.verticalBorderSize)
		}

		w = int(b.spaceWidth)

		var hspace int // horizontal space
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
	if b.gameState.PlayerNumber == 1 {
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
func (b *board) spaceRect(space int) (x, y, w, h int) {
	rect := b.spaceRects[space]
	return rect[0], rect[1], rect[2], rect[3]
}

func (b *board) bottomRow(space int) bool {
	bottomStart := 1
	bottomEnd := 12
	bottomBar := bgammon.SpaceBarPlayer
	bottomHome := bgammon.SpaceHomePlayer
	if b.gameState.PlayerNumber == 2 {
		bottomStart = 1
		bottomEnd = 12
	}
	return space == bottomBar || space == bottomHome || (space >= bottomStart && space <= bottomEnd)
}

// relX, relY
func (b *board) stackSpaceRect(space int, stack int) (x, y, w, h int) {
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

	if b.lastPlayerNumber != b.gameState.PlayerNumber {
		b.setSpaceRects()
		b.updateBackgroundImage()
	}
	b.lastPlayerNumber = b.gameState.PlayerNumber

	b.updateAvailableCache = true

	var showGrid *etk.Grid
	if !b.gameState.Spectating {
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

		abs := spaceValue
		if abs < 0 {
			abs *= -1
		}
		for i := 0; i < abs; i++ {
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

	b.updateOpponentLabel()
	b.updatePlayerLabel()
}

// _movePiece returns after moving the piece.
func (b *board) _movePiece(sprite *Sprite, from int, to int, speed int, pause bool) {
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

	x, y, w, _ := b.stackSpaceRect(space, stack)
	x, y = b.offsetPosition(space, x, y)
	// Center piece in space
	x += (w - int(b.spaceWidth)) / 2

	sprite.toX = x
	sprite.toY = y
	sprite.toTime = moveTime
	sprite.toStart = time.Now()

	time.Sleep(moveTime)

	sprite.x = x
	sprite.y = y
	sprite.toStart = time.Time{}

	/*homeSpace := b.ClientWebSocket.Board.PlayerHomeSpace()
	if b.gameState.Turn != b.gameState.Player {
		homeSpace = 25 - homeSpace
	}

	if to != homeSpace {*/
	b.spaceSprites[to] = append(b.spaceSprites[to], sprite)
	/*}*/
	for i, s := range b.spaceSprites[from] {
		if s == sprite {
			b.spaceSprites[from] = append(b.spaceSprites[from][:i], b.spaceSprites[from][i+1:]...)
			break
		}
	}
	b.moving = nil

	if pause {
		time.Sleep(pauseTime)
	} else {
		time.Sleep(50 * time.Millisecond)
	}
}

// movePiece returns when finished moving the piece.
func (b *board) movePiece(from int, to int) {
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

func (b *board) startDrag(s *Sprite, space int, click bool) {
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
			for space, pieces := range b.spaceSprites {
				for _, piece := range pieces {
					if piece == dropped {
						if space != index {
							playSoundEffect(effectMove)
							b.gameState.AddLocalMove([]int{space, index})
							b.processState()
							scheduleFrame()
							processed = true
							b.Client.Out <- []byte(fmt.Sprintf("mv %d/%d", space, index))
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

	if game.keyboard.Visible() && cursor.In(game.keyboard.Rect()) {
		return false, nil
	}

	cx, cy := cursor.X, cursor.Y

	if b.dragging == nil {
		// TODO allow grabbing multiple pieces by grabbing further down the stack
		if !handled && b.playerTurn() && clicked && (b.lastDragClick.IsZero() || time.Since(b.lastDragClick) >= 50*time.Millisecond) {
			s, space := b.spriteAt(cx, cy)
			if s != nil && s.colorWhite == (b.gameState.PlayerNumber == 2) && space != bgammon.SpaceHomeOpponent && (space != bgammon.SpaceHomePlayer || !game.Board.gameState.Acey || !game.Board.gameState.Player1.Entered) {
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

func allMoves(in *bgammon.Game, from int, to int) []int {
	gc := in.Copy()
	ok, _ := gc.AddMoves([][]int{{from, to}}, true)
	if !ok {
		return nil
	}

	moves := []int{to}
	var found = make(map[int]bool)
	found[to] = true
	for _, m := range gc.LegalMoves(true) {
		if m[0] == to && !found[m[1]] {
			for _, move := range allMoves(gc, m[0], m[1]) {
				if !found[move] {
					moves = append(moves, move)
					found[move] = true
				}
			}
		}
	}
	return moves
}
