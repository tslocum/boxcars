package game

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"strconv"
	"sync"
	"time"

	"code.rocket9labs.com/tslocum/bgammon"
	"code.rocket9labs.com/tslocum/etk"
	"code.rocketnine.space/tslocum/messeji"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
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

	Sprites *Sprites

	spaceRects   [][4]int
	spaceSprites [][]*Sprite // Space contents

	dragging *Sprite
	moving   *Sprite // Moving automatically

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

	inputGrid          *etk.Grid
	showKeyboardButton *etk.Button
	frame              *etk.Frame

	leaveGameGrid         *etk.Grid
	confirmLeaveGameFrame *etk.Frame

	fontFace   font.Face
	lineHeight int
	lineOffset int

	*sync.Mutex
}

func NewBoard() *board {
	b := &board{
		barWidth:             100,
		triangleOffset:       float64(50),
		horizontalBorderSize: 20,
		verticalBorderSize:   20,
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
		inputGrid:             etk.NewGrid(),
		frame:                 etk.NewFrame(),
		confirmLeaveGameFrame: etk.NewFrame(),
		fontFace:              mediumFont,
		Mutex:                 &sync.Mutex{},
	}
	b.fontUpdated()

	{
		leaveGameLabel := etk.NewText("Leave match?")
		leaveGameLabel.SetHorizontal(messeji.AlignCenter)

		b.leaveGameGrid = etk.NewGrid()
		b.leaveGameGrid.SetBackground(color.RGBA{40, 24, 9, 255})
		b.leaveGameGrid.AddChildAt(leaveGameLabel, 0, 0, 2, 1)
		b.leaveGameGrid.AddChildAt(etk.NewButton("No", b.cancelLeaveGame), 0, 1, 1, 1)
		b.leaveGameGrid.AddChildAt(etk.NewButton("Yes", b.confirmLeaveGame), 1, 1, 1, 1)
		b.leaveGameGrid.SetVisible(false)
	}

	leaveGameButton := etk.NewButton("Leave Match", b.leaveGame)
	b.showKeyboardButton = etk.NewButton("Show Keyboard", b.toggleKeyboard)
	b.inputGrid.AddChildAt(leaveGameButton, 0, 0, 1, 1)
	b.inputGrid.AddChildAt(b.showKeyboardButton, 1, 0, 1, 1)
	b.frame.AddChild(b.inputGrid)
	b.frame.AddChild(b.leaveGameGrid)

	b.buttons = []*boardButton{
		{
			label:    "Roll",
			selected: b.selectRoll,
		}, {
			label:    "Reset",
			selected: b.selectReset,
		}, {
			label:    "OK",
			selected: b.selectOK,
		}, {
			label:    "Double",
			selected: b.selectDouble,
		}, {
			label:    "Resign",
			selected: b.selectResign,
		}, {
			label:    "Accept",
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

	statusBuffer.SetFont(b.fontFace)
	gameBuffer.SetFont(b.fontFace)
	inputBuffer.SetFont(b.fontFace)
}

func (b *board) setKeyboardRect() {
	heightOffset := 76
	if game.portraitView() {
		heightOffset += 44
	}
	game.keyboard.SetRect(0, game.screenH/2, game.screenW, (game.screenH - game.screenH/2 - heightOffset))
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
	b.leaveGameGrid.SetVisible(true)
	return nil
}

func (b *board) toggleKeyboard() error {
	if game.keyboard.Visible() {
		game.keyboard.Hide()
		b.showKeyboardButton.Label.SetText("Show Keyboard")
	} else {
		b.setKeyboardRect()
		game.keyboard.Show()
		b.showKeyboardButton.Label.SetText("Hide Keyboard")
	}
	return nil
}

func (b *board) selectRoll() {
	b.Client.Out <- []byte("roll")
}

func (b *board) selectOK() {
	b.Client.Out <- []byte("ok")
}

func (b *board) selectReset() {
	b.Client.Out <- []byte("reset")
}

func (b *board) selectDouble() {
	b.Client.Out <- []byte("double")
}

func (b *board) selectResign() {
	b.Client.Out <- []byte("resign")
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

	// Table
	b.backgroundImage = ebiten.NewImage(b.w, b.h)
	b.backgroundImage.Fill(tableColor)

	// Frame
	img := ebiten.NewImage(frameW, b.h)
	img.Fill(frameColor)
	{
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(b.horizontalBorderSize-borderSize), 0)
		b.backgroundImage.DrawImage(img, op)
	}

	// Face
	img = ebiten.NewImage(int(innerW), b.h-int(b.verticalBorderSize*2))
	img.Fill(faceColor)
	{
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(b.horizontalBorderSize), float64(b.verticalBorderSize))
		b.backgroundImage.DrawImage(img, op)
	}

	// Bar
	img = ebiten.NewImage(int(b.barWidth), b.h)
	img.Fill(frameColor)
	{
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64((b.w/2)-int(b.barWidth/2)), 0)
		b.backgroundImage.DrawImage(img, op)
	}

	// Draw triangles
	baseImg := image.NewRGBA(image.Rect(0, 0, b.w-int(b.horizontalBorderSize*2), b.h-int(b.verticalBorderSize*2)))
	gc := draw2dimg.NewGraphicContext(baseImg)
	for i := 0; i < 2; i++ {
		triangleTip := (float64(b.h) - (b.verticalBorderSize * 2)) / 2
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
			ty := b.h * i
			if j >= 6 {
				tx += b.barWidth
			}
			gc.MoveTo(float64(tx), float64(ty))
			gc.LineTo(float64(tx+b.spaceWidth/2), triangleTip)
			gc.LineTo(float64(tx+b.spaceWidth), float64(ty))
			gc.Close()
			gc.Fill()
		}
	}
	img = ebiten.NewImageFromImage(baseImg)
	{
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(b.horizontalBorderSize), float64(b.verticalBorderSize))
		b.backgroundImage.DrawImage(img, op)
	}

	// Border
	borderImage := image.NewRGBA(image.Rect(0, 0, b.w, b.h))
	gc = draw2dimg.NewGraphicContext(borderImage)
	gc.SetStrokeColor(borderColor)
	// - Center
	gc.SetLineWidth(2)
	gc.MoveTo(float64(frameW/2), float64(0))
	gc.LineTo(float64(frameW/2), float64(b.h))
	gc.Stroke()
	// - Outside right
	gc.MoveTo(float64(frameW), float64(0))
	gc.LineTo(float64(frameW), float64(b.h))
	gc.Stroke()
	// - Inside left
	gc.SetLineWidth(1)
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
	img = ebiten.NewImageFromImage(borderImage)
	{
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(b.horizontalBorderSize-borderSize, 0)
		b.backgroundImage.DrawImage(img, op)
	}

	// Draw space numbers.
	for space, r := range b.spaceRects {
		if space < 1 || space > 24 {
			continue
		} else if b.gameState.PlayerNumber == 1 {
			space = 24 - space + 1
		}

		sp := strconv.Itoa(space)
		if space < 10 {
			sp = " " + sp
		}
		x := r[0] + r[2]/2 + int(b.horizontalBorderSize/2) + 4
		y := 2
		if b.bottomRow(space) {
			y = b.h - int(b.verticalBorderSize) + 2
		}
		ebitenutil.DebugPrintAt(b.backgroundImage, sp, x, y)
	}
}

func (b *board) drawButton(target *ebiten.Image, r image.Rectangle, label string) {
	w, h := r.Dx(), r.Dy()

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

	bounds := text.BoundString(b.fontFace, label)
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

func (b *board) Draw(screen *ebiten.Image) {
	{
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(b.x), float64(b.y))
		screen.DrawImage(b.backgroundImage, op)
	}

	drawSprite := func(sprite *Sprite) {
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
			screen.DrawImage(imgCheckerLight, op)
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

		screen.DrawImage(imgCheckerLight, op)
	}

	for space := 0; space < bgammon.BoardSpaces; space++ {
		var numPieces int
		for i, sprite := range b.spaceSprites[space] {
			if sprite == b.dragging || sprite == b.moving {
				continue
			}
			numPieces++

			drawSprite(sprite)

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

			bounds := text.BoundString(b.fontFace, overlayText)
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

	// Draw opponent name and dice

	borderSize := b.horizontalBorderSize
	if borderSize > b.barWidth/2 {
		borderSize = b.barWidth / 2
	}

	playerColor := color.White
	opponentColor := color.Black
	playerBorderColor := lightCheckerColor
	opponentBorderColor := darkCheckerColor
	if b.gameState.PlayerNumber == 1 {
		playerColor = color.Black
		opponentColor = color.White
		playerBorderColor = darkCheckerColor
		opponentBorderColor = lightCheckerColor
	}

	playerRoll := b.gameState.Roll1
	opponentRoll := b.gameState.Roll2
	if b.gameState.PlayerNumber == 2 {
		playerRoll, opponentRoll = opponentRoll, playerRoll
	}

	drawLabel := func(label string, labelColor color.Color, border bool, borderColor color.Color) *ebiten.Image {
		bounds := text.BoundString(b.fontFace, label)

		w := int(float64(bounds.Dx()) * 1.5)
		h := int(float64(bounds.Dy()) * 2)

		baseImg := image.NewRGBA(image.Rect(0, 0, w, h))
		// Draw border
		if border {
			gc := draw2dimg.NewGraphicContext(baseImg)
			gc.SetLineWidth(5)
			gc.SetStrokeColor(borderColor)
			gc.MoveTo(float64(0), float64(0))
			gc.LineTo(float64(w), 0)
			gc.LineTo(float64(w), float64(h))
			gc.LineTo(float64(0), float64(h))
			gc.Close()
			gc.Stroke()
		}

		img := ebiten.NewImageFromImage(baseImg)
		text.Draw(img, label, b.fontFace, (w-bounds.Dx())/2, int(float64(h-(bounds.Max.Y/2))*0.75), labelColor)

		return img
	}

	const diceGap = 10

	opponent := b.gameState.OpponentPlayer()
	if opponent.Name != "" {
		label := fmt.Sprintf("%s", opponent.Name)

		img := drawLabel(label, opponentColor, b.gameState.Turn != b.gameState.PlayerNumber, opponentBorderColor)
		bounds := img.Bounds()

		x := b.x + int(((float64(b.innerW))/4)-(float64(bounds.Dx()/2))) - int(b.horizontalBorderSize)/2
		y := b.y + (b.innerH / 2) - (bounds.Dy() / 2) + int(b.verticalBorderSize)
		{
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(x), float64(y))
			screen.DrawImage(img, op)
		}

		if b.gameState.Turn == 0 {
			if opponentRoll != 0 {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(b.x+(b.innerW/4)-int(b.horizontalBorderSize)/2-diceSize/2), float64(b.y+(b.innerH/2))-(float64(diceSize)*1.4))
				screen.DrawImage(diceImage(opponentRoll), op)
			}
		} else if b.gameState.Turn != b.gameState.PlayerNumber && b.gameState.Roll1 != 0 {
			{
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(b.x+(b.innerW/4)-int(b.horizontalBorderSize)/2-diceSize-diceGap), float64(b.y+(b.innerH/2))-(float64(diceSize)*1.4))
				screen.DrawImage(diceImage(b.gameState.Roll1), op)
			}

			{
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(b.x+(b.innerW/4)-int(b.horizontalBorderSize)/2+diceGap), float64(b.y+(b.innerH/2))-(float64(diceSize)*1.4))
				screen.DrawImage(diceImage(b.gameState.Roll2), op)
			}
		}
	}

	// Draw player name and dice

	player := b.gameState.LocalPlayer()
	if player.Name != "" {
		label := fmt.Sprintf("%s", player.Name)

		img := drawLabel(label, playerColor, b.gameState.Turn == b.gameState.PlayerNumber, playerBorderColor)
		bounds := img.Bounds()

		x := b.x + int((((float64(b.innerW))/4)*3)-(float64(bounds.Dx()/2))) + int(b.horizontalBorderSize)/2
		y := b.y + (b.innerH / 2) - (bounds.Dy() / 2) + int(b.verticalBorderSize)
		{
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(x), float64(y))
			screen.DrawImage(img, op)
		}

		if b.gameState.Turn == 0 {
			if playerRoll != 0 {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(b.x+((b.innerW/4)*3)+int(b.horizontalBorderSize)/2-diceSize/2), float64(b.y+(b.innerH/2))-(float64(diceSize)*1.4))
				screen.DrawImage(diceImage(playerRoll), op)
			}
		} else if b.gameState.Turn == b.gameState.PlayerNumber && b.gameState.Roll1 != 0 {
			{
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(b.x+((b.innerW/4)*3)+int(b.horizontalBorderSize)/2-diceSize-diceGap), float64(b.y+(b.innerH/2))-(float64(diceSize)*1.4))
				screen.DrawImage(diceImage(b.gameState.Roll1), op)
			}

			{
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(b.x+((b.innerW/4)*3)+int(b.horizontalBorderSize)/2+diceGap), float64(b.y+(b.innerH/2))-(float64(diceSize)*1.4))
				screen.DrawImage(diceImage(b.gameState.Roll2), op)
			}
		}
	}

	// Draw moving sprite
	if b.moving != nil {
		drawSprite(b.moving)
	}

	// Draw dragged sprite
	if b.dragging != nil {
		drawSprite(b.dragging)
	}

	b.drawButtons(screen)
}

func (b *board) updateButtonRects() {
	btnRoll := b.buttons[0]
	btnReset := b.buttons[1]
	btnOK := b.buttons[2]
	btnDouble := b.buttons[3]
	btnResign := b.buttons[4]
	btnAccept := b.buttons[5]

	w := 200
	h := 75
	x, y := (b.w-w)/2, (b.h-h)/2

	const padding = 20
	btnReset.rect = image.Rect(x, y, x+w, y+h)
	btnRoll.rect = image.Rect(x, y, x+w, y+h)
	btnResign.rect = image.Rect(b.w/2-padding/2-w, y, b.w/2-padding/2, y+h)
	btnOK.rect = image.Rect(b.w/2+padding/2, y, b.w/2+padding/2+w, y+h)
	btnAccept.rect = image.Rect(b.w/2+padding/2, y, b.w/2+padding/2+w, y+h)

	if b.gameState.MayDouble() && b.gameState.MayRoll() {
		btnDouble.rect = image.Rect(b.w/2-padding/2-w, y, b.w/2-padding/2, y+h)
		btnRoll.rect = image.Rect(b.w/2+padding/2, y, b.w/2+padding/2+w, y+h)
	} else if b.gameState.MayReset() && b.gameState.MayOK() {
		btnReset.rect = image.Rect(b.w/2-padding/2-w, y, b.w/2-padding/2, y+h)
		btnOK.rect = image.Rect(b.w/2+padding/2, y, b.w/2+padding/2+w, y+h)
	}
}

func (b *board) setRect(x, y, w, h int) {
	if b.x == x && b.y == y && b.w == w && b.h == h {
		return
	}

	s := ebiten.DeviceScaleFactor()
	if s >= 1.25 {
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

	borderSize := b.horizontalBorderSize
	if borderSize > b.barWidth/2 {
		borderSize = b.barWidth / 2
	}
	b.innerW = int(float64(b.w) - (b.horizontalBorderSize * 2))
	b.innerH = int(float64(b.h) - (b.verticalBorderSize * 2))

	loadAssets(int(b.spaceWidth))

	for i := 0; i < b.Sprites.num; i++ {
		s := b.Sprites.sprites[i]
		s.w, s.h = imgCheckerLight.Bounds().Dx(), imgCheckerLight.Bounds().Dy()
	}

	b.setSpaceRects()
	b.updateBackgroundImage()
	b.processState()

	dialogWidth := 400
	dialogHeight := 100
	b.leaveGameGrid.SetRect(image.Rect(game.screenW/2-dialogWidth/2, game.screenH/2-dialogHeight/2, game.screenW/2+dialogWidth/2, game.screenH/2+dialogHeight/2))

	if viewBoard && game.keyboard.Visible() {
		b.setKeyboardRect()
	}
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

func (b *board) spriteAt(x, y int) *Sprite {
	space := b.spaceAt(x, y)
	if space == -1 {
		return nil
	}
	pieces := b.spaceSprites[space]
	if len(pieces) == 0 {
		return nil
	}
	return pieces[len(pieces)-1]
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
	x, y, w, h = b.spaceRect(space)

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

	if b.dragging == nil && b.playerTurn() {
		// TODO allow grabbing multiple pieces by grabbing further down the stack

		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && !handled && len(b.gameState.Available) > 0 {
			s := b.spriteAt(cx, cy)
			if s != nil && s.colorWhite == (b.gameState.PlayerNumber == 2) {
				b.dragging = s
			}
		}

		b.touchIDs = inpututil.AppendJustPressedTouchIDs(b.touchIDs[:0])
		for _, id := range b.touchIDs {
			game.enableTouchInput()
			x, y := ebiten.TouchPosition(id)
			handled := b.handleClick(x, y)
			if !handled && len(b.gameState.Available) > 0 {
				b.dragX, b.dragY = x, y

				s := b.spriteAt(x, y)
				if s != nil && s.colorWhite == (b.gameState.PlayerNumber == 2) {
					b.dragging = s
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
		}
	} else if inpututil.IsTouchJustReleased(b.dragTouchId) {
		dropped = b.dragging
		b.dragging = nil
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
