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

type boardButton struct {
	label    string
	selected func()
	rect     image.Rectangle
}

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
	moving        *Sprite // Moving automatically

	dragTouchId ebiten.TouchID
	touchIDs    []ebiten.TouchID

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

	buttons []*boardButton

	spaceHighlight *ebiten.Image

	opponentLabel *Label
	playerLabel   *Label

	timerLabel     *etk.Text
	clockLabel     *etk.Text
	showMenuButton *etk.Button

	menuGrid *etk.Grid

	highlightCheckbox *etk.Checkbox
	settingsGrid      *etk.Grid

	matchStatusGrid *etk.Grid

	inputGrid          *etk.Grid
	showKeyboardButton *etk.Button
	uiGrid             *etk.Grid
	frame              *etk.Frame

	bearOffOverlay *etk.Button

	leaveGameGrid         *etk.Grid
	confirmLeaveGameFrame *etk.Frame

	fontFace   font.Face
	lineHeight int
	lineOffset int

	highlightAvailable bool
	availableMoves     [][]int

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
			Game: bgammon.NewGame(),
		},
		spaceHighlight:        ebiten.NewImage(1, 1),
		opponentLabel:         NewLabel(color.RGBA{255, 255, 255, 255}),
		playerLabel:           NewLabel(color.RGBA{0, 0, 0, 255}),
		menuGrid:              etk.NewGrid(),
		settingsGrid:          etk.NewGrid(),
		uiGrid:                etk.NewGrid(),
		frame:                 etk.NewFrame(),
		confirmLeaveGameFrame: etk.NewFrame(),
		fontFace:              mediumFont,
		Mutex:                 &sync.Mutex{},
	}

	b.bearOffOverlay = etk.NewButton(gotext.Get("Drag here to bear off"), func() error {
		return nil
	})
	b.bearOffOverlay.SetVisible(false)

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

		b.highlightCheckbox = etk.NewCheckbox(b.toggleHighlightCheckbox)
		b.highlightCheckbox.SetBorderColor(triangleA)
		b.highlightCheckbox.SetCheckColor(triangleA)
		b.highlightCheckbox.SetSelected(b.highlightAvailable)

		highlightLabel := etk.NewText(gotext.Get("Highlight legal moves"))
		highlightLabel.SetVertical(messeji.AlignCenter)

		checkboxGrid := etk.NewGrid()
		checkboxGrid.SetColumnSizes(-1, -1, -1, -1, -1)
		checkboxGrid.AddChildAt(b.highlightCheckbox, 0, 0, 1, 1)
		checkboxGrid.AddChildAt(highlightLabel, 1, 0, 4, 1)

		b.settingsGrid.SetBackground(color.RGBA{40, 24, 9, 255})
		b.settingsGrid.SetColumnSizes(20, -1, -1, 20)
		b.settingsGrid.SetRowSizes(72, 72, 20, -1)
		b.settingsGrid.AddChildAt(settingsLabel, 1, 0, 2, 1)
		b.settingsGrid.AddChildAt(checkboxGrid, 1, 1, 2, 1)
		b.settingsGrid.AddChildAt(etk.NewBox(), 1, 2, 1, 1)
		b.settingsGrid.AddChildAt(etk.NewButton(gotext.Get("Return"), b.hideMenu), 0, 3, 4, 1)
		b.settingsGrid.SetVisible(false)
	}

	{
		leaveGameLabel := etk.NewText(gotext.Get("Leave match?"))
		leaveGameLabel.SetHorizontal(messeji.AlignCenter)

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

	b.showMenuButton = etk.NewButton("Menu", b.showMenu)
	b.showMenuButton.Label.SetFont(smallFont)

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

	b.frame.AddChild(b.opponentLabel)
	b.frame.AddChild(b.playerLabel)
	b.frame.AddChild(b.uiGrid)
	b.frame.AddChild(b.bearOffOverlay)
	b.frame.AddChild(b.menuGrid)
	b.frame.AddChild(b.settingsGrid)
	b.frame.AddChild(b.leaveGameGrid)

	b.fontUpdated()

	b.buttons = []*boardButton{
		{
			label:    gotext.Get("Roll"),
			selected: b.selectRoll,
		}, {
			label:    gotext.Get("Undo"),
			selected: b.selectUndo,
		}, {
			label:    gotext.Get("OK"),
			selected: b.selectOK,
		}, {
			label:    gotext.Get("Double"),
			selected: b.selectDouble,
		}, {
			label:    gotext.Get("Resign"),
			selected: b.selectResign,
		}, {
			label:    gotext.Get("Accept"),
			selected: b.selectOK,
		},
	}

	for i := range b.Sprites.sprites {
		b.Sprites.sprites[i] = b.newSprite(false)
	}

	b.dragTouchId = -1

	return b
}

func (b *board) fontUpdated() {
	m := b.fontFace.Metrics()
	b.lineHeight = m.Height.Round()
	b.lineOffset = m.Ascent.Round()

	bufferFont := b.fontFace
	if game.scaleFactor <= 1 {
		switch b.fontFace {
		case largeFont:
			bufferFont = mediumFont
		case mediumFont:
			bufferFont = smallFont
		}
	}
	statusBuffer.SetFont(bufferFont)
	gameBuffer.SetFont(bufferFont)
	inputBuffer.Field.SetFont(bufferFont)

	b.timerLabel.SetFont(b.fontFace)
	b.clockLabel.SetFont(b.fontFace)
}

func (b *board) recreateInputGrid() {
	if b.inputGrid == nil {
		b.inputGrid = etk.NewGrid()
	} else {
		*b.inputGrid = *etk.NewGrid()
	}

	if game.TouchInput {
		b.inputGrid.AddChildAt(inputBuffer, 0, 0, 2, 1)

		b.inputGrid.AddChildAt(etk.NewBox(), 0, 1, 2, 1)

		showMenuButton := etk.NewButton(gotext.Get("Menu"), b.showMenu)
		b.inputGrid.AddChildAt(showMenuButton, 0, 2, 1, 1)
		b.inputGrid.AddChildAt(b.showKeyboardButton, 1, 2, 1, 1)

		b.inputGrid.SetRowSizes(52, int(b.horizontalBorderSize/2), -1)
	} else {
		b.inputGrid.AddChildAt(inputBuffer, 0, 0, 1, 1)
	}
}

func (b *board) setKeyboardRect() {
	inputAndButtons := 52
	if game.TouchInput {
		inputAndButtons = 52 + int(b.horizontalBorderSize) + game.scale(baseButtonHeight)
	}
	h := game.screenH / 3
	y := game.screenH - game.screenH/3 - inputAndButtons - int(b.horizontalBorderSize)
	game.keyboard.SetRect(0, y, game.screenW, h)
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

func (b *board) showMenu() error {
	b.menuGrid.SetVisible(true)
	return nil
}

func (b *board) toggleKeyboard() error {
	if game.keyboard.Visible() {
		game.keyboard.Hide()
		b.showKeyboardButton.Label.SetText(gotext.Get("Show Keyboard"))
	} else {
		b.setKeyboardRect()
		game.keyboard.Show()
		b.showKeyboardButton.Label.SetText(gotext.Get("Hide Keyboard"))
	}
	return nil
}

func (b *board) selectRoll() {
	b.Client.Out <- []byte("roll")
}

func (b *board) selectOK() {
	b.Client.Out <- []byte("ok")
}

func (b *board) selectUndo() {
	go func() {
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
		b.movePiece(lastMove[1], lastMove[0])
		b.gameState.Moves = b.gameState.Moves[:l-1]
	}()
}

func (b *board) selectDouble() {
	b.Client.Out <- []byte("double")
}

func (b *board) selectResign() {
	b.Client.Out <- []byte("resign")
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
	innerW := float64(b.w) - b.horizontalBorderSize*2 // Outer board width (including frame)

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
		w, h := int(innerW), b.h-int(b.verticalBorderSize*2)
		b.backgroundImage.SubImage(image.Rect(x, y, x+w, y+h)).(*ebiten.Image).Fill(faceColor)
	}

	// Draw bar.
	{
		x, y := int((b.w/2)-int(b.barWidth/2)), 0
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
	// - Center
	gc.SetLineWidth(borderStrokeSize)
	gc.MoveTo(float64(frameW/2), float64(0))
	gc.LineTo(float64(frameW/2), float64(b.h))
	gc.Stroke()
	// - Outside right
	gc.MoveTo(float64(frameW), float64(0))
	gc.LineTo(float64(frameW), float64(b.h))
	gc.Stroke()
	// - Inside left
	gc.SetLineWidth(borderStrokeSize / 2)
	edge := float64((((innerW) - b.barWidth) / 2) + borderSize)
	gc.MoveTo(float64(borderSize), float64(b.verticalBorderSize))
	gc.LineTo(edge, float64(b.verticalBorderSize))
	gc.LineTo(edge, float64(b.h-int(b.verticalBorderSize)))
	gc.LineTo(float64(borderSize), float64(b.h-int(b.verticalBorderSize)))
	gc.LineTo(float64(borderSize), float64(b.verticalBorderSize))
	gc.Close()
	gc.Stroke()
	// - Inside right
	edgeStart := float64((innerW / 2) + (b.barWidth / 2) + borderSize)
	edgeEnd := float64(innerW + borderSize)
	gc.MoveTo(float64(edgeStart), float64(b.verticalBorderSize))
	gc.LineTo(edgeEnd, float64(b.verticalBorderSize))
	gc.LineTo(edgeEnd, float64(b.h-int(b.verticalBorderSize)))
	gc.LineTo(float64(edgeStart), float64(b.h-int(b.verticalBorderSize)))
	gc.LineTo(float64(edgeStart), float64(b.verticalBorderSize))
	gc.Close()
	gc.Stroke()
	if !b.fullHeight {
		// - Outside left
		gc.SetLineWidth(1)
		gc.MoveTo(float64(0), float64(0))
		gc.LineTo(float64(0), float64(b.h))
		// Top
		gc.MoveTo(0, float64(0))
		gc.LineTo(float64(b.w), float64(0))
		// Bottom
		gc.MoveTo(0, float64(b.h))
		gc.LineTo(float64(b.w), float64(b.h))
		gc.Stroke()
	}
	b.backgroundImage.DrawImage(ebiten.NewImageFromImage(b.baseImage), nil)

	// Draw space numbers.
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

func (b *board) drawButton(target *ebiten.Image, r image.Rectangle, label string) {
	w, h := r.Dx(), r.Dy()
	if w == 0 || h == 0 {
		return
	}

	baseImg := image.NewRGBA(image.Rect(0, 0, w, h))

	gc := draw2dimg.NewGraphicContext(baseImg)
	gc.SetLineWidth(5)
	gc.SetStrokeColor(color.Black)
	gc.MoveTo(0, 0)
	gc.LineTo(float64(w), 0)
	gc.LineTo(float64(w), float64(h))
	gc.LineTo(0, float64(h))
	gc.Close()
	gc.Stroke()

	img := ebiten.NewImage(w, h)
	img.Fill(color.RGBA{225.0, 188, 125, 255})
	img.DrawImage(ebiten.NewImageFromImage(baseImg), nil)

	bounds := etk.BoundString(b.fontFace, label)
	text.Draw(img, label, b.fontFace, (w-bounds.Dx())/2, (h+(bounds.Dy()/2))/2, color.Black)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
	target.DrawImage(img, op)
}

func (b *board) drawButtons(screen *ebiten.Image) {
	for i, btn := range b.buttons {
		if (i == 0 && b.gameState.MayRoll()) ||
			(i == 1 && b.gameState.MayReset()) ||
			(i == 2 && b.gameState.MayOK()) ||
			(i == 3 && b.gameState.MayDouble()) ||
			(i == 4 && b.gameState.MayResign()) ||
			(i == 5 && (b.gameState.MayOK() && b.gameState.MayResign())) {
			b.drawButton(screen, btn.rect, btn.label)
		}
	}
}

func (b *board) drawSprite(target *ebiten.Image, sprite *Sprite) {
	x, y := float64(sprite.x), float64(sprite.y)
	if !sprite.toStart.IsZero() {
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

			bounds := etk.BoundString(b.fontFace, overlayText)
			overlayImage := ebiten.NewImage(bounds.Dx()*2, bounds.Dy()*2)
			text.Draw(overlayImage, overlayText, b.fontFace, 0, bounds.Dy(), labelColor)

			x, y, w, h := b.stackSpaceRect(space, numPieces-1)
			x += (w / 2) - (bounds.Dx() / 2)
			y += (h / 2) - (bounds.Dy() / 2)
			x, y = b.offsetPosition(x, y)

			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(x), float64(y))
			screen.DrawImage(overlayImage, op)
		}
	}

	// Draw space hover overlay when dragging
	if b.dragging != nil {
		if b.highlightAvailable && b.draggingSpace != -1 {
			for _, move := range b.gameState.Available {
				if move[0] == b.draggingSpace {
					x, y, _, _ := b.spaceRect(move[1])
					x, y = b.offsetPosition(x, y)
					op := &ebiten.DrawImageOptions{}
					op.GeoM.Translate(float64(x), float64(y))
					op.ColorScale.Scale(0.2, 0.2, 0.2, 0.2)
					screen.DrawImage(b.spaceHighlight, op)
				}
			}
		}

		dx, dy := b.dragX, b.dragY

		x, y := ebiten.CursorPosition()
		if x != 0 || y != 0 {
			dx, dy = x, y
		}

		space := b.spaceAt(dx, dy)
		if space > 0 && space < 25 {
			x, y, _, _ := b.spaceRect(space)
			x, y = b.offsetPosition(x, y)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(x), float64(y))
			op.ColorScale.Scale(0.1, 0.1, 0.1, 0.1)
			screen.DrawImage(b.spaceHighlight, op)
		}
	}

	// Draw opponent dice

	playerRoll := b.gameState.Roll1
	opponentRoll := b.gameState.Roll2
	if b.gameState.PlayerNumber == 2 {
		playerRoll, opponentRoll = opponentRoll, playerRoll
	}

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
		innerCenter := b.x + (b.w / 4) - int(b.barWidth/4) + int(b.horizontalBorderSize/2)
		if b.gameState.Turn == 0 {
			if opponentRoll != 0 {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(innerCenter-diceSize/2), float64(b.y+(b.innerH/2))-diceGap-float64(diceSize))
				screen.DrawImage(diceImage(opponentRoll), op)
			}
		} else if b.gameState.Turn != b.gameState.PlayerNumber && b.gameState.Roll1 != 0 {
			{
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(innerCenter-diceSize)-diceGap, float64(b.y+(b.innerH/2))-diceGap-float64(diceSize))
				screen.DrawImage(diceImage(b.gameState.Roll1), op)
			}

			{
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(innerCenter)+diceGap, float64(b.y+(b.innerH/2))-diceGap-float64(diceSize))
				screen.DrawImage(diceImage(b.gameState.Roll2), op)
			}
		}
	}

	// Draw player dice

	player := b.gameState.LocalPlayer()
	if player.Name != "" {
		innerCenter := b.x + b.w/2 + b.w/4 + int(b.barWidth/4) - int(b.horizontalBorderSize/2)
		if b.gameState.Turn == 0 {
			if playerRoll != 0 {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(innerCenter-diceSize/2), float64(b.y+(b.innerH/2))-diceGap-float64(diceSize))
				screen.DrawImage(diceImage(playerRoll), op)
			}
		} else if b.gameState.Turn == b.gameState.PlayerNumber && b.gameState.Roll1 != 0 {
			{
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(innerCenter-diceSize)-diceGap, float64(b.y+(b.innerH/2))-diceGap-float64(diceSize))
				screen.DrawImage(diceImage(b.gameState.Roll1), op)
			}

			{
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(innerCenter)+diceGap, float64(b.y+(b.innerH/2))-diceGap-float64(diceSize))
				screen.DrawImage(diceImage(b.gameState.Roll2), op)
			}
		}
	}

	b.drawButtons(screen)
}

func (b *board) drawDraggedCheckers(screen *ebiten.Image) {
	if b.moving != nil {
		b.drawSprite(screen, b.moving)
	}
	if b.dragging != nil {
		b.drawSprite(screen, b.dragging)
	}
}

func (b *board) updateButtonRects() {
	btnRoll := b.buttons[0]
	btnReset := b.buttons[1]
	btnOK := b.buttons[2]
	btnDouble := b.buttons[3]
	btnResign := b.buttons[4]
	btnAccept := b.buttons[5]

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
	x, y := int(b.horizontalBorderSize)+(b.innerW-w)/2, int(b.verticalBorderSize)+(b.innerH-h)/2

	padding := 75

	if w >= b.innerW/8 {
		btnReset.rect = image.Rect(x, y, x+w, y+h)
		btnRoll.rect = image.Rect(x, y, x+w, y+h)
		btnResign.rect = image.Rect(x, y+h+padding/2, x+w, y+h+padding/2+h)
		btnOK.rect = image.Rect(x, y, x+w, y+h)
		btnAccept.rect = image.Rect(x, y, x+w, y+h)

		if b.gameState.MayDouble() && b.gameState.MayRoll() {
			y -= h/2 + padding/2
			btnDouble.rect = image.Rect(x, y+h+padding/2, x+w, y+h+padding/2+h)
			btnRoll.rect = image.Rect(x, y, x+w, y+h)
		} else if b.gameState.MayReset() && b.gameState.MayOK() {
			y -= h/2 + padding/2
			btnReset.rect = image.Rect(x, y+h+padding/2, x+w, y+h+padding/2+h)
			btnOK.rect = image.Rect(x, y, x+w, y+h)
		}
		return
	}

	btnReset.rect = image.Rect(x, y, x+w, y+h)
	btnRoll.rect = image.Rect(x, y, x+w, y+h)
	btnResign.rect = image.Rect(x-w/2-padding/2, y, x+w/2-padding/2, y+h)
	btnOK.rect = image.Rect(x, y, x+w, y+h)
	btnAccept.rect = image.Rect(x+w/2+padding/2, y, x+w/2+w+padding/2, y+h)

	if b.gameState.MayDouble() && b.gameState.MayRoll() {
		btnDouble.rect = image.Rect(x-w/2-padding/2, y, x+w/2-padding/2, y+h)
		btnRoll.rect = image.Rect(x+w/2+padding/2, y, x+w/2+w+padding/2, y+h)
	} else if b.gameState.MayReset() && b.gameState.MayOK() {
		btnReset.rect = image.Rect(x-w/2-padding/2, y, x+w/2-padding/2, y+h)
		btnOK.rect = image.Rect(x+w/2+padding/2, y, x+w/2+w+padding/2, y+h)
	}
}

func (b *board) setRect(x, y, w, h int) {
	if OptimizeSetRect && b.x == x && b.y == y && b.w == w && b.h == h {
		return
	}

	if game.scaleFactor >= 1.25 {
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

	b.x, b.y, b.w, b.h = x, y, w, h
	if b.w > b.h {
		b.w = b.h
	}
	b.updateButtonRects()

	b.triangleOffset = (float64(b.h) - (b.verticalBorderSize * 2)) / 15

	b.spaceWidth = (float64(b.w) - (b.horizontalBorderSize * 2)) / 13
	b.barWidth = b.spaceWidth

	b.overlapSize = (((float64(b.h) - (b.verticalBorderSize * 2)) - (b.triangleOffset * 2)) / 2) / 5
	if b.overlapSize > b.spaceWidth*0.94 {
		b.overlapSize = b.spaceWidth * 0.94
	}

	extraSpace := float64(b.w) - (b.spaceWidth * 12)
	largeBarWidth := float64(b.spaceWidth) * 1.25
	if extraSpace >= largeBarWidth {
		b.barWidth = largeBarWidth
		b.spaceWidth = ((float64(b.w) - (b.horizontalBorderSize * 2)) - b.barWidth) / 12
	}

	if b.barWidth < 1 {
		b.barWidth = 1
	}
	if b.spaceWidth < 1 {
		b.spaceWidth = 1
	}

	b.innerW = int(float64(b.w) - (b.horizontalBorderSize * 2))
	b.innerH = int(float64(b.h) - (b.verticalBorderSize * 2))

	b.triangleOffset = (float64(b.innerH)+b.verticalBorderSize-b.spaceWidth*10)/2 + b.spaceWidth/12

	loadImageAssets(int(b.spaceWidth))

	for i := 0; i < b.Sprites.num; i++ {
		s := b.Sprites.sprites[i]
		s.w, s.h = imgCheckerLight.Bounds().Dx(), imgCheckerLight.Bounds().Dy()
	}

	b.setSpaceRects()
	b.updateBackgroundImage()
	b.processState()

	b.recreateInputGrid()

	inputAndButtons := 52
	if game.TouchInput {
		inputAndButtons = 52 + int(b.horizontalBorderSize) + game.scale(baseButtonHeight)
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
		dialogHeight := 72 + 72 + 20 + game.scale(baseButtonHeight)
		if dialogHeight > game.screenH {
			dialogHeight = game.screenH
		}

		x, y := game.screenW/2-dialogWidth/2, game.screenH/2-dialogHeight+int(b.verticalBorderSize)
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

	b.menuGrid.SetColumnSizes(-1, game.scale(10), -1, game.scale(10), -1)

	if viewBoard && game.keyboard.Visible() {
		b.setKeyboardRect()
	}
}

func (b *board) updateOpponentLabel() {
	player := b.gameState.OpponentPlayer()
	label := b.opponentLabel

	label.SetText(player.Name)

	if player.Number == 1 {
		label.activeColor = color.RGBA{0, 0, 0, 255}
	} else {
		label.activeColor = color.RGBA{255, 255, 255, 255}
	}
	label.active = b.gameState.Turn == player.Number
	label.Text.TextField.SetForegroundColor(label.activeColor)

	bounds := etk.BoundString(largeFont, player.Name)
	padding := 13
	innerCenter := b.x + (b.w / 4) - int(b.barWidth/4) + int(b.horizontalBorderSize/2)
	x := innerCenter - int(float64(bounds.Dx()/2))
	y := b.y + (b.innerH / 2) - (bounds.Dy() / 2) + int(b.verticalBorderSize)
	r := image.Rect(x, y, x+bounds.Dx(), y+bounds.Dy())
	if r.Eq(label.Rect()) && r.Dx() != 0 && r.Dy() != 0 {
		label.updateBackground()
		return
	}
	label.SetRect(r.Inset(-padding))
}

func (b *board) updatePlayerLabel() {
	player := b.gameState.LocalPlayer()
	label := b.playerLabel

	label.SetText(player.Name)

	if player.Number == 1 {
		label.activeColor = color.RGBA{0, 0, 0, 255}
	} else {
		label.activeColor = color.RGBA{255, 255, 255, 255}
	}
	label.active = b.gameState.Turn == player.Number
	label.Text.TextField.SetForegroundColor(label.activeColor)

	bounds := etk.BoundString(largeFont, player.Name)
	padding := 13
	innerCenter := b.x + b.w/2 + b.w/4 + int(b.barWidth/4) - int(b.horizontalBorderSize/2)
	x := innerCenter - int(float64(bounds.Dx()/2))
	y := b.y + (b.innerH / 2) - (bounds.Dy() / 2) + int(b.verticalBorderSize)
	r := image.Rect(x, y, x+bounds.Dx(), y+bounds.Dy())
	if r.Eq(label.Rect()) && r.Dx() != 0 && r.Dy() != 0 {
		label.updateBackground()
		return
	}
	label.SetRect(r.Inset(-padding))
}

func (b *board) offsetPosition(x, y int) (int, int) {
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
			s.x, s.y = b.offsetPosition(x, y)
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
		sx, sy = b.offsetPosition(sx, sy)
		if x >= sx && x <= sx+sw && y >= sy && y <= sy+sh {
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
		if space == bgammon.SpaceBarPlayer {
			hspace = 6
			w = int(b.barWidth)
			add = int(b.barWidth)/2 - int(b.spaceWidth)/2
		} else if space == bgammon.SpaceBarOpponent {
			hspace = 6
			w = int(b.barWidth)
			add = int(b.barWidth)/2 - int(b.spaceWidth)/2
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
			x = -int(b.spaceWidth * 2)
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
	bounds := b.spaceHighlight.Bounds()
	if bounds.Dx() != r[2] || bounds.Dy() != r[3] {
		b.spaceHighlight = ebiten.NewImage(r[2], r[3])
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
	if b.gameState.PlayerNumber == 2 {
		bottomStart = 1
		bottomEnd = 12
	}
	return space == bottomBar || (space >= bottomStart && space <= bottomEnd)
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
	if space == 0 || space == 25 {
		w = int(b.barWidth)
	}

	return x, y, w, h
}

func (b *board) processState() {
	if b.lastPlayerNumber != b.gameState.PlayerNumber {
		b.setSpaceRects()
		b.updateBackgroundImage()
	}
	b.updateButtonRects()
	b.lastPlayerNumber = b.gameState.PlayerNumber

	b.Sprites = &Sprites{}
	b.spaceSprites = make([][]*Sprite, bgammon.BoardSpaces)
	for space := 0; space < bgammon.BoardSpaces; space++ {
		spaceValue := b.gameState.Board[space]

		white := spaceValue < 0

		abs := spaceValue
		if abs < 0 {
			abs *= -1
		}
		for i := 0; i < abs; i++ {
			s := b.newSprite(white)
			if i >= abs {
				s.colorWhite = b.gameState.PlayerNumber == 2
				s.premove = true
			}
			b.spaceSprites[space] = append(b.spaceSprites[space], s)
			b.Sprites.sprites = append(b.Sprites.sprites, s)
		}
	}
	b.Sprites.num = len(b.Sprites.sprites)

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
	x, y = b.offsetPosition(x, y)
	// Center piece in space
	x += (w - int(b.spaceWidth)) / 2

	sprite.toX = x
	sprite.toY = y
	sprite.toTime = moveTime
	sprite.toStart = time.Now()

	t := time.NewTimer(moveTime)
	mt := time.NewTicker(time.Second / 144)
DRAWMOVE:
	for {
		select {
		case <-t.C:
			mt.Stop()
			break DRAWMOVE
		case <-mt.C:
			scheduleFrame()
		}

	}

	sprite.x = x
	sprite.y = y
	sprite.toStart = time.Time{}
	scheduleFrame()

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

// WatchingGame returns whether the active game is being watched.
func (b *board) watchingGame() bool {
	return !b.playingGame() && false // TODO
}

// PlayingGame returns whether the active game is being played.
func (b *board) playingGame() bool {
	return b.gameState.Player1.Name != "" || b.gameState.Player2.Name != ""
}

func (b *board) playerTurn() bool {
	return b.playingGame() && (b.gameState.MayRoll() || b.gameState.Turn == b.gameState.PlayerNumber)
}

func (b *board) handleClick(x int, y int) bool {
	p := image.Point{x, y}
	for i := len(b.buttons) - 1; i >= 0; i-- {
		btn := b.buttons[i]
		if (i == 0 && b.gameState.MayRoll()) ||
			(i == 1 && b.gameState.MayReset()) ||
			(i == 2 && b.gameState.MayOK()) ||
			(i == 3 && b.gameState.MayDouble()) ||
			(i == 4 && b.gameState.MayResign()) ||
			(i == 5 && (b.gameState.MayOK() && b.gameState.MayResign())) {
			if p.In(btn.rect) {
				btn.selected()
				return true
			}
		}
	}
	return false
}

func (b *board) startDrag(s *Sprite, space int) {
	b.dragging = s
	b.draggingSpace = space
	if bgammon.CanBearOff(b.gameState.Board, b.gameState.PlayerNumber, true) && b.gameState.Board[bgammon.SpaceHomePlayer] == 0 {
		b.bearOffOverlay.SetVisible(true)
	}
}

func (b *board) Update() {
	if b.Client == nil {
		return
	}

	var handled bool
	cx, cy := ebiten.CursorPosition()
	if (cx != 0 || cy != 0) && game.keyboard.Visible() {
		p := image.Point{X: cx, Y: cy}
		if p.In(game.keyboard.Rect()) {
			return
		}
	}

	if b.dragging == nil && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		handled = b.handleClick(cx, cy)
	}

	if b.dragging == nil {
		// TODO allow grabbing multiple pieces by grabbing further down the stack

		if !handled && b.playerTurn() && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			s, space := b.spriteAt(cx, cy)
			if s != nil && s.colorWhite == (b.gameState.PlayerNumber == 2) {
				b.startDrag(s, space)
			}
		}

		b.touchIDs = inpututil.AppendJustPressedTouchIDs(b.touchIDs[:0])
		for _, id := range b.touchIDs {
			game.EnableTouchInput()
			x, y := ebiten.TouchPosition(id)
			handled := b.handleClick(x, y)
			if !handled && b.playerTurn() {
				b.dragX, b.dragY = x, y

				s, space := b.spriteAt(x, y)
				if s != nil && s.colorWhite == (b.gameState.PlayerNumber == 2) {
					b.startDrag(s, space)
					b.dragTouchId = id
				}
			}
		}
	}

	x, y := ebiten.CursorPosition()
	if b.dragTouchId != -1 {
		x, y = ebiten.TouchPosition(b.dragTouchId)

		if x != 0 || y != 0 { // 0,0 is returned when the touch is released
			b.dragX, b.dragY = x, y
		} else {
			x, y = b.dragX, b.dragY
		}
	}

	var dropped *Sprite
	if b.dragTouchId == -1 {
		if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
			dropped = b.dragging
			b.dragging = nil
			b.bearOffOverlay.SetVisible(false)
		}
	} else if inpututil.IsTouchJustReleased(b.dragTouchId) {
		dropped = b.dragging
		b.dragging = nil
		b.bearOffOverlay.SetVisible(false)
	}
	if dropped != nil {
		index := b.spaceAt(x, y)
		// Bear off by dragging outside the board.
		if index == -1 {
			index = bgammon.SpaceHomePlayer
		}
		var processed bool
		if index >= 0 && b.Client != nil {
		ADDPREMOVE:
			for space, pieces := range b.spaceSprites {
				for _, piece := range pieces {
					if piece == dropped {
						if space != index {
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
	}

	if b.dragging != nil {
		sprite := b.dragging
		sprite.x = x - (sprite.w / 2)
		sprite.y = y - (sprite.h / 2)
	}
}

type Label struct {
	*etk.Text
	active      bool
	activeColor color.RGBA
	lastActive  bool
	bg          *ebiten.Image
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

func NewLabel(c color.RGBA) *Label {
	l := &Label{
		Text:        etk.NewText(""),
		activeColor: c,
	}
	l.Text.SetFont(largeFont)
	l.Text.SetForegroundColor(c)
	l.Text.SetScrollBarVisible(false)
	l.Text.SetSingleLine(true)
	l.Text.SetHorizontal(messeji.AlignCenter)
	l.Text.SetVertical(messeji.AlignCenter)
	return l
}
