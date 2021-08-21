package game

import (
	"image"
	"image/color"
	"math/rand"

	"github.com/llgcode/draw2d/draw2dimg"

	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/hajimehoshi/ebiten/v2"
)

type board struct {
	Sprites *Sprites
	op      *ebiten.DrawImageOptions

	spaces [][]*Sprite

	backgroundImage *ebiten.Image

	dragging *Sprite

	dragTouchId ebiten.TouchID

	touchIDs []ebiten.TouchID

	x, y int
	w, h int

	spaceWidth           int // spaceWidth is also the width and height of checkers
	barWidth             int
	triangleOffset       float64
	horizontalBorderSize int
	verticalBorderSize   int
	overlapSize          int
}

func NewBoard() *board {
	b := &board{
		barWidth:             100,
		triangleOffset:       float64(50),
		horizontalBorderSize: 50,
		verticalBorderSize:   25,
		overlapSize:          97,
		Sprites: &Sprites{
			sprites: make([]*Sprite, 30),
			num:     30,
		},
		spaces: make([][]*Sprite, 26),
	}

	for i := range b.Sprites.sprites {
		s := &Sprite{}

		r := rand.Intn(2)
		s.colorWhite = r != 1

		s.w, s.h = imgCheckerWhite.Size()

		b.Sprites.sprites[i] = s

		space := (i % 24) + 1
		if i > 25 {
			space = 3
		}

		b.spaces[space] = append(b.spaces[space], s)
	}

	b.op = &ebiten.DrawImageOptions{}

	b.dragTouchId = -1

	return b
}

// relX, relY
func (b *board) spacePosition(index int) (int, int) {
	if index <= 12 {
		return b.spaceWidth * (index - 1), 0
	}
	// TODO add innerW innerH
	return b.spaceWidth * (index - 13), (b.h - (b.verticalBorderSize)*2) - b.spaceWidth
}

func (b *board) updateBackgroundImage() {
	tableColor := color.RGBA{0, 102, 51, 255}

	// Table
	box := image.NewRGBA(image.Rect(0, 0, b.w, b.h))
	img := ebiten.NewImageFromImage(box)
	img.Fill(tableColor)
	b.backgroundImage = ebiten.NewImageFromImage(img)

	// Border
	borderColor := color.RGBA{65, 40, 14, 255}
	borderSize := b.horizontalBorderSize
	if borderSize > b.barWidth/2 {
		borderSize = b.barWidth / 2
	}
	box = image.NewRGBA(image.Rect(0, 0, b.w-((b.horizontalBorderSize-borderSize)*2), b.h))
	img = ebiten.NewImageFromImage(box)
	img.Fill(borderColor)
	b.op.GeoM.Reset()
	b.op.GeoM.Translate(float64(b.horizontalBorderSize-borderSize), float64(b.verticalBorderSize))
	b.backgroundImage.DrawImage(img, b.op)

	// Face
	box = image.NewRGBA(image.Rect(0, 0, b.w-(b.horizontalBorderSize*2), b.h-(b.verticalBorderSize*2)))
	img = ebiten.NewImageFromImage(box)
	img.Fill(color.RGBA{120, 63, 25, 255})
	b.op.GeoM.Reset()
	b.op.GeoM.Translate(float64(b.horizontalBorderSize), float64(b.verticalBorderSize))
	b.backgroundImage.DrawImage(img, b.op)

	baseImg := image.NewRGBA(image.Rect(0, 0, b.w-(b.horizontalBorderSize*2), b.h-(b.verticalBorderSize*2)))
	gc := draw2dimg.NewGraphicContext(baseImg)

	// Bar
	box = image.NewRGBA(image.Rect(0, 0, b.barWidth, b.h))
	img = ebiten.NewImageFromImage(box)
	img.Fill(borderColor)
	b.op.GeoM.Reset()
	b.op.GeoM.Translate(float64((b.w/2)-(b.barWidth/2)), 0)
	b.backgroundImage.DrawImage(img, b.op)

	// Draw triangles
	for i := 0; i < 2; i++ {
		triangleTip := float64(b.h / 2)
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
				gc.SetFillColor(color.RGBA{219.0, 185, 113, 255})
			} else {
				gc.SetFillColor(color.RGBA{120.0, 17.0, 0, 255})
			}

			tx := b.spaceWidth * j
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
}

func (b *board) draw(screen *ebiten.Image) {
	b.op.GeoM.Reset()
	b.op.GeoM.Translate(float64(b.x), float64(b.y))
	screen.DrawImage(b.backgroundImage, b.op)

	for i := 0; i < b.Sprites.num; i++ {
		sprite := b.Sprites.sprites[i]
		b.op.GeoM.Reset()
		b.op.GeoM.Translate(float64(sprite.x), float64(sprite.y))

		if sprite.colorWhite {
			screen.DrawImage(imgCheckerWhite, b.op)
		} else {
			screen.DrawImage(imgCheckerBlack, b.op)
		}
	}
}

func (b *board) setRect(x, y, w, h int) {
	if b.x == x && b.y == y && b.w == w && b.h == h {
		return
	}

	b.x, b.y, b.w, b.h = x, y, w, h

	b.horizontalBorderSize = 0

	b.triangleOffset = float64(b.h) / 30

	for {
		b.verticalBorderSize = 0 // TODO configurable

		b.spaceWidth = (b.w - (b.horizontalBorderSize * 2)) / 13

		b.barWidth = b.spaceWidth

		b.overlapSize = (((b.h - (b.verticalBorderSize * 2)) - (int(b.triangleOffset) * 2)) / 2) / 5
		if b.overlapSize >= b.spaceWidth {
			b.overlapSize = b.spaceWidth
			break
		}

		b.horizontalBorderSize++
	}

	b.horizontalBorderSize = ((b.w - (b.spaceWidth * 12)) - b.barWidth) / 2
	if b.horizontalBorderSize < 0 {
		b.horizontalBorderSize = 0
	}

	loadAssets(b.spaceWidth)
	for i := 0; i < b.Sprites.num; i++ {
		s := b.Sprites.sprites[i]
		s.w, s.h = imgCheckerWhite.Size()
	}

	b.updateBackgroundImage()
	b.positionCheckers()
}

func (b *board) offsetPosition(x, y int) (int, int) {
	return b.x + x + b.horizontalBorderSize, b.y + y + b.verticalBorderSize
}

func (b *board) positionCheckers() {
	for space := 1; space < 25; space++ {
		sprites := b.spaces[space]

		for i := range sprites {
			s := sprites[i]
			if b.dragging == s {
				continue
			}

			x, y := b.spacePosition(space)
			s.x, s.y = b.offsetPosition(x, y)
			if (space > 6 && space < 13) || (space > 18 && space < 25) {
				s.x += b.barWidth
			}
			s.x += (b.spaceWidth - s.w) / 2
			s.y += i * b.overlapSize
		}
	}
}

func (b *board) spriteAt(x, y int) *Sprite {
	for i := 0; i < b.Sprites.num; i++ {
		s := b.Sprites.sprites[i]
		if x >= s.x && y >= s.y && x <= s.x+s.w && y <= s.y+s.h {
			// Bring sprite to front
			b.Sprites.sprites = append(b.Sprites.sprites[:i], b.Sprites.sprites[i+1:]...)
			b.Sprites.sprites = append(b.Sprites.sprites, s)

			return s
		}
	}
	return nil
}

func (b *board) update() {
	if b.dragging == nil {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			x, y := ebiten.CursorPosition()

			if b.dragging == nil {
				s := b.spriteAt(x, y)
				if s != nil {
					b.dragging = s
				}
			}
		}

		b.touchIDs = inpututil.AppendJustPressedTouchIDs(b.touchIDs[:0])
		for _, id := range b.touchIDs {
			x, y := ebiten.TouchPosition(id)
			s := b.spriteAt(x, y)
			if s != nil {
				b.dragging = s
				b.dragTouchId = id
			}
		}
	}

	if b.dragTouchId == -1 {
		if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
			b.dragging = nil
		}
	} else if inpututil.IsTouchJustReleased(b.dragTouchId) {
		b.dragging = nil
	}

	if b.dragging != nil {
		x, y := ebiten.CursorPosition()
		if b.dragTouchId != -1 {
			x, y = ebiten.TouchPosition(b.dragTouchId)
		}

		sprite := b.dragging
		sprite.x = x - (sprite.w / 2)
		sprite.y = y - (sprite.h / 2)
	}
}
