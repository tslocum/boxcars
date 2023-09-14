package game

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"code.rocket9labs.com/tslocum/bgammon"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/llgcode/draw2d/draw2dimg"
)

type board struct {
	x, y int
	w, h int

	fullHeight bool

	innerW, innerH int

	op *ebiten.DrawImageOptions

	backgroundImage *ebiten.Image

	Sprites *Sprites

	spaceSprites     [][]*Sprite // Space contents
	spaceSpriteRects [][4]int

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

	drawFrame chan bool

	debug int // Print and draw debug information

	Client *Client

	dragX, dragY int
}

func NewBoard() *board {
	b := &board{
		barWidth:             100,
		triangleOffset:       float64(50),
		horizontalBorderSize: 20,
		verticalBorderSize:   10,
		overlapSize:          97,
		Sprites: &Sprites{
			sprites: make([]*Sprite, 30),
			num:     30,
		},
		spaceSprites:     make([][]*Sprite, 26),
		spaceSpriteRects: make([][4]int, 26),
		drawFrame:        make(chan bool, 10),
		gameState: &bgammon.GameState{
			Game: bgammon.NewGame(),
		},
	}

	for i := range b.Sprites.sprites {
		s := b.newSprite(i%2 == 1)

		b.Sprites.sprites[i] = s

		space := i
		if space > 25 {
			if space%2 == 0 {
				space = 0
			} else {
				space = 25
			}
		}

		b.spaceSprites[space] = append(b.spaceSprites[space], s)
	}

	go b.handleDraw()

	b.op = &ebiten.DrawImageOptions{}

	b.dragTouchId = -1

	return b
}

func (b *board) handleDraw() {
	drawFreq := time.Second / 144 // TODO
	lastDraw := time.Now()
	for v := range b.drawFrame {
		if !v {
			return
		}
		since := time.Since(lastDraw)
		if since < drawFreq {
			t := time.NewTimer(drawFreq - since)
		DELAYDRAW:
			for {
				select {
				case <-b.drawFrame:
					continue DELAYDRAW
				case <-t.C:
					break DELAYDRAW
				}
			}
		}
		ebiten.ScheduleFrame()
		lastDraw = time.Now()
	}
}

func (b *board) newSprite(white bool) *Sprite {
	s := &Sprite{}
	s.colorWhite = white
	s.w, s.h = imgCheckerLight.Size()
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
	box := image.NewRGBA(image.Rect(0, 0, b.w, b.h))
	img := ebiten.NewImageFromImage(box)
	img.Fill(tableColor)
	b.backgroundImage = ebiten.NewImageFromImage(img)

	// Frame
	box = image.NewRGBA(image.Rect(0, 0, frameW, b.h))
	img = ebiten.NewImageFromImage(box)
	img.Fill(frameColor)
	b.op.GeoM.Reset()
	b.op.GeoM.Translate(float64(b.horizontalBorderSize-borderSize), 0)
	b.backgroundImage.DrawImage(img, b.op)

	// Face
	box = image.NewRGBA(image.Rect(0, 0, int(innerW), b.h-int(b.verticalBorderSize*2)))
	img = ebiten.NewImageFromImage(box)
	img.Fill(faceColor)
	b.op.GeoM.Reset()
	b.op.GeoM.Translate(float64(b.horizontalBorderSize), float64(b.verticalBorderSize))
	b.backgroundImage.DrawImage(img, b.op)

	// Bar
	box = image.NewRGBA(image.Rect(0, 0, int(b.barWidth), b.h))
	img = ebiten.NewImageFromImage(box)
	img.Fill(frameColor)
	b.op.GeoM.Reset()
	b.op.GeoM.Translate(float64((b.w/2)-int(b.barWidth/2)), 0)
	b.backgroundImage.DrawImage(img, b.op)

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
	b.op.GeoM.Reset()
	b.op.GeoM.Translate(float64(b.horizontalBorderSize), float64(b.verticalBorderSize))
	b.backgroundImage.DrawImage(img, b.op)

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
	b.op.GeoM.Reset()
	b.op.GeoM.Translate(b.horizontalBorderSize-borderSize, 0)
	b.backgroundImage.DrawImage(img, b.op)
}

func (b *board) ScheduleFrame() {
	b.drawFrame <- true
}

func (b *board) resetButtonRect() (int, int, int, int) {
	w := 200
	h := 75
	return (b.w - w) / 2, (b.h - h) / 2, w, h
}

func (b *board) draw(screen *ebiten.Image) {
	b.op.GeoM.Reset()
	b.op.GeoM.Translate(float64(b.x), float64(b.y))
	screen.DrawImage(b.backgroundImage, b.op)

	if b.debug == 2 {
		b.iterateSpaces(func(space int) {
			x, y, w, h := b.spaceRect(space)
			spaceImage := ebiten.NewImage(w, h)
			if space%2 == 0 {
				spaceImage.Fill(color.RGBA{50, 50, 50, 150})
			} else {
				spaceImage.Fill(color.RGBA{255, 255, 255, 150})
			}
			x, y = b.offsetPosition(x, y)
			b.op.GeoM.Reset()
			b.op.GeoM.Translate(float64(x), float64(y))
			screen.DrawImage(spaceImage, b.op)
		})
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
			ebiten.ScheduleFrame()
		}

		// Draw shadow.

		b.op.GeoM.Reset()
		b.op.GeoM.Translate(x, y)
		b.op.ColorM.Scale(0, 0, 0, 1)

		b.op.Filter = ebiten.FilterLinear

		screen.DrawImage(imgCheckerLight, b.op)

		b.op.ColorM.Reset()

		// Draw checker.

		checkerScale := 0.94

		b.op.GeoM.Reset()
		b.op.GeoM.Translate(-b.spaceWidth/2, -b.spaceWidth/2)
		b.op.GeoM.Scale(checkerScale, checkerScale)
		b.op.GeoM.Translate((b.spaceWidth/2)+x, (b.spaceWidth/2)+y)

		c := lightCheckerColor
		if !sprite.colorWhite {
			c = darkCheckerColor
		}
		b.op.ColorM.Scale(0.0, 0.0, 0.0, 1)
		r := float64(c.R) / 0xff
		g := float64(c.G) / 0xff
		bl := float64(c.B) / 0xff
		b.op.ColorM.Translate(r, g, bl, 0)

		screen.DrawImage(imgCheckerLight, b.op)

		b.op.ColorM.Reset()

		b.op.Filter = ebiten.FilterNearest
	}

	b.iterateSpaces(func(space int) {
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

			bounds := text.BoundString(mediumFont, overlayText)
			overlayImage := ebiten.NewImage(bounds.Dx()*2, bounds.Dy()*2)
			text.Draw(overlayImage, overlayText, mediumFont, 0, bounds.Dy(), labelColor)

			x, y, w, h := b.stackSpaceRect(space, numPieces-1)
			x += (w / 2) - (bounds.Dx() / 2)
			y += (h / 2) - (bounds.Dy() / 2)
			x, y = b.offsetPosition(x, y)

			b.op.GeoM.Reset()
			b.op.GeoM.Translate(float64(x), float64(y))
			screen.DrawImage(overlayImage, b.op)
		}
	})

	// Draw hover overlay

	if b.dragging != nil {
		dx, dy := b.dragX, b.dragY

		x, y := ebiten.CursorPosition()
		if x != 0 || y != 0 {
			dx, dy = x, y
		}

		space := b.spaceAt(dx, dy)
		if space > 0 && space < 25 {
			x, y, w, h := b.spaceRect(space)
			spaceImage := ebiten.NewImage(w, h)
			spaceImage.Fill(color.RGBA{255, 255, 255, 50})
			x, y = b.offsetPosition(x, y)
			b.op.GeoM.Reset()
			b.op.GeoM.Translate(float64(x), float64(y))
			screen.DrawImage(spaceImage, b.op)
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

	drawLabel := func(label string, labelColor color.Color, border bool, borderColor color.Color) *ebiten.Image {
		bounds := text.BoundString(mediumFont, label)

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
		text.Draw(img, label, mediumFont, (w-bounds.Dx())/2, int(float64(h-(bounds.Max.Y/2))*0.75), labelColor)

		return img
	}

	opponent := b.gameState.OpponentPlayer()
	if opponent.Name != "" {
		label := fmt.Sprintf("%s  %d %d", opponent.Name, b.gameState.Roll1, b.gameState.Roll2)

		img := drawLabel(label, opponentColor, b.gameState.Turn != b.gameState.PlayerNumber, opponentBorderColor)
		bounds := img.Bounds()

		x := int(((float64(b.innerW) - borderSize) / 4) - (float64(bounds.Dx()) / 2))
		y := (b.innerH / 2) - (bounds.Dy() / 2)
		x, y = b.offsetPosition(x, y)
		b.op.GeoM.Reset()
		b.op.GeoM.Translate(float64(x), float64(y))
		screen.DrawImage(img, b.op)
	}

	// Draw player name and dice

	player := b.gameState.LocalPlayer()
	if player.Name != "" {
		label := fmt.Sprintf("%s  %d %d", player.Name, b.gameState.Roll1, b.gameState.Roll2)

		img := drawLabel(label, playerColor, b.gameState.Turn == b.gameState.PlayerNumber, playerBorderColor)
		bounds := img.Bounds()

		x := ((b.innerW / 4) * 3) - (bounds.Dx() / 2)
		y := (b.innerH / 2) - (bounds.Dy() / 2)
		x, y = b.offsetPosition(x, y)
		b.op.GeoM.Reset()
		b.op.GeoM.Translate(float64(x), float64(y))
		screen.DrawImage(img, b.op)

	}

	if len(b.gameState.Moves) > 0 {
		x, y, w, h := b.resetButtonRect()
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

		label := "Reset"
		bounds := text.BoundString(mediumFont, label)
		text.Draw(img, label, mediumFont, (w-bounds.Dx())/2, (h+(bounds.Dy()/2))/2, color.Black)

		b.op.GeoM.Reset()
		b.op.GeoM.Translate(float64(x), float64(y))
		screen.DrawImage(img, b.op)
	}

	// Draw moving sprite
	if b.moving != nil {
		drawSprite(b.moving)
	}

	// Draw dragged sprite
	if b.dragging != nil {
		drawSprite(b.dragging)
	}

	if b.debug == 2 {
		homeStart, homeEnd := bgammon.HomeRange(b.gameState.PlayerNumber)
		b.iterateSpaces(func(space int) {
			x, y, w, h := b.spaceRect(space)
			spaceImage := ebiten.NewImage(w, h)
			br := ""
			if space >= homeStart && space <= homeEnd {
				br += "H"
			}
			if space == bgammon.SpaceBarPlayer {
				br += "(PB)"
			} else if space == bgammon.SpaceBarOpponent {
				br += "(OB)"
			}
			ebitenutil.DebugPrint(spaceImage, fmt.Sprintf(" %d %s", space, br))
			x, y = b.offsetPosition(x, y)
			b.op.GeoM.Reset()
			b.op.GeoM.Scale(2, 2)
			b.op.GeoM.Translate(float64(x), float64(y))
			screen.DrawImage(spaceImage, b.op)
		})
	}
}

func (b *board) setRect(x, y, w, h int) {
	if b.x == x && b.y == y && b.w == w && b.h == h {
		return
	}

	b.x, b.y, b.w, b.h = x, y, w, h

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
		s.w, s.h = imgCheckerLight.Size()
	}

	b.setSpaceRects()
	b.updateBackgroundImage()
	b.ProcessState()
}

func (b *board) offsetPosition(x, y int) (int, int) {
	return b.x + x + int(b.horizontalBorderSize), b.y + y + int(b.verticalBorderSize)
}

// Do not call _positionCheckers directly.  Call ProcessState instead.
func (b *board) _positionCheckers() {
	for space := 0; space < 26; space++ {
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

	b.ScheduleFrame()
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
	for i := 0; i < 26; i++ {
		sx, sy, sw, sh := b.spaceRect(i)
		sx, sy = b.offsetPosition(sx, sy)
		if x >= sx && x <= sx+sw && y >= sy && y <= sy+sh {
			return i
		}
	}
	return -1
}

func (b *board) iterateSpaces(f func(space int)) {
	for space := 0; space <= 25; space++ {
		f(space)
	}
}

func (b *board) translateSpace(space int) int {
	/*if b.gameState.PlayerNumber == 2 {
		// Spaces range from 24 - 1.
		if space == 0 || space == 25 {
			space = 25 - space
		} else if space <= 12 {
			space = 12 + space
		} else {
			space = space - 12
		}
	}*/
	// TODO
	return space
}

func (b *board) setSpaceRects() {
	var x, y, w, h int
	for i := 0; i < 26; i++ {
		trueSpace := i

		space := b.translateSpace(i)
		if !b.bottomRow(trueSpace) {
			y = 0
		} else {
			y = int((float64(b.h) / 2) - b.verticalBorderSize)
		}

		w = int(b.spaceWidth)

		var hspace int // horizontal space
		var add int
		if space == 0 {
			hspace = 6
			w = int(b.barWidth)
		} else if space == 25 {
			hspace = 6
			w = int(b.barWidth)
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

		b.spaceSpriteRects[trueSpace] = [4]int{x, y, w, h}
	}
}

// relX, relY
func (b *board) spaceRect(space int) (x, y, w, h int) {
	rect := b.spaceSpriteRects[space]
	return rect[0], rect[1], rect[2], rect[3]
}

func (b *board) bottomRow(space int) bool {
	bottomStart := 1
	bottomEnd := 12
	bottomBar := 25
	if b.gameState.PlayerNumber == 2 {
		bottomStart = 13
		bottomEnd = 24
		bottomBar = 0
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

func (b *board) ProcessState() {
	if b.lastPlayerNumber != b.gameState.PlayerNumber {
		b.setSpaceRects()
	}
	b.lastPlayerNumber = b.gameState.PlayerNumber

	b.Sprites = &Sprites{}
	b.spaceSprites = make([][]*Sprite, 26)
	for space := 0; space < bgammon.BoardSpaces; space++ {
		spaceValue := b.gameState.Board[space]

		white := spaceValue > 0
		if spaceValue == 0 {
			white = b.gameState.PlayerNumber == 2
		}

		abs := spaceValue
		if abs < 0 {
			abs *= -1
		}

		var preMovesTo int
		var preMovesFrom int
		/*if b.Client != nil {
			preMovesTo = b.Client.Board.Premoveto[space]
			preMovesFrom = b.Client.Board.Premovefrom[space]
		}*/
		// TODO

		for i := 0; i < abs+(preMovesTo-preMovesFrom); i++ {
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
	ebiten.ScheduleFrame()
	time.Sleep(moveTime)

	sprite.x = x
	sprite.y = y
	sprite.toStart = time.Time{}
	ebiten.ScheduleFrame()

	/*homeSpace := b.Client.Board.PlayerHomeSpace()
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
		log.Printf("%d-%d: NO PIECES AT SPACE %d", from, to, from)
		os.Exit(1)
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
		bar := bgammon.SpaceBarOpponent
		if b.gameState.Turn == b.gameState.PlayerNumber {
			bar = bgammon.SpaceBarPlayer
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
	return b.playingGame() && b.gameState.Turn == b.gameState.PlayerNumber
}

func (b *board) update() {
	if b.Client == nil {
		return
	}

	if b.dragging == nil && b.playerTurn() {
		// TODO allow grabbing multiple pieces by grabbing further down the stack

		handleReset := func(x, y int) bool {
			/*if len(b.gameState.Moves) > 0 {
				rx, ry, rw, rh := b.resetButtonRect()
				if x >= rx && x <= rx+rw && y >= ry && y <= ry+rh {
					b.Client.Board.ResetPreMoves()
					b.ProcessState()
					return true
				}
			}*/
			panic("RESET") // TODO
			return false
		}

		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			if b.dragging == nil {
				x, y := ebiten.CursorPosition()
				handled := handleReset(x, y)
				if !handled {
					s := b.spriteAt(x, y)
					if s != nil {
						b.dragging = s
						// TODO set dragFrom instead of calculating later
					}
				}
			}
		}

		b.touchIDs = inpututil.AppendJustPressedTouchIDs(b.touchIDs[:0])
		for _, id := range b.touchIDs {
			x, y := ebiten.TouchPosition(id)
			handled := handleReset(x, y)
			if !handled {
				b.dragX, b.dragY = x, y

				s := b.spriteAt(x, y)
				if s != nil {
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
			// TODO check if all pieces are home
			index = bgammon.SpaceHomePlayer
		}
		if index >= 0 && b.Client != nil {
		ADDPREMOVE:
			for space, pieces := range b.spaceSprites {
				for _, piece := range pieces {
					if piece == dropped {
						if space != index {
							//b.Client.Board.SetSelection(1, space)
							b.Client.Out <- []byte(fmt.Sprintf("move %d/%d", space, index))
						}
						break ADDPREMOVE
					}
				}
			}
		}

		b.ProcessState()
	}

	if b.dragging != nil {
		sprite := b.dragging
		sprite.x = x - (sprite.w / 2)
		sprite.y = y - (sprite.h / 2)
	}
}
