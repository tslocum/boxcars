package game

import (
	"image"
	"image/color"
	"log"
	"math/rand"

	"github.com/llgcode/draw2d/draw2dimg"

	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/hajimehoshi/ebiten/v2"
)

type board struct {
	Sprites *Sprites
	op      *ebiten.DrawImageOptions

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
			sprites: make([]*Sprite, 24),
			num:     24,
		},
	}

	for i := range b.Sprites.sprites {
		s := &Sprite{}

		r := rand.Intn(2)
		s.colorWhite = r != 1

		s.w, s.h = imgCheckerWhite.Size()

		b.Sprites.sprites[i] = s
	}

	b.op = &ebiten.DrawImageOptions{}

	b.dragTouchId = -1

	return b
}

// relX, relY
func (b *board) spacePosition(index int) (int, int) {
	log.Printf("%d", index)
	if index <= 12 {
		return b.spaceWidth * (index - 1), 0
	}
	// TODO add innerW innerH
	return b.spaceWidth * (index - 13), (b.h - (b.verticalBorderSize)*2) - b.spaceWidth
}

func (b *board) updateBackgroundImage() {
	// TODO percentage of screen instead

	borderColor := color.RGBA{65, 40, 14, 255}

	// Border
	box := image.NewRGBA(image.Rect(0, 0, b.w, b.h))
	img := ebiten.NewImageFromImage(box)
	img.Fill(borderColor)
	b.backgroundImage = ebiten.NewImageFromImage(img)

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

	b.spaceWidth = ((b.w - (b.horizontalBorderSize * 2)) - b.barWidth) / 12

	loadAssets(b.spaceWidth)
	for i := 0; i < b.Sprites.num; i++ {
		s := b.Sprites.sprites[i]
		log.Printf("%d-%d", s.w, s.h)
		s.w, s.h = imgCheckerWhite.Size()
		log.Printf("NEW %d-%d", s.w, s.h)
	}

	b.updateBackgroundImage()
	b.positionCheckers()
}

func (b *board) offsetPosition(x, y int) (int, int) {
	return b.x + x + b.horizontalBorderSize, b.y + y + b.verticalBorderSize
}

func (b *board) positionCheckers() {
	// TODO slightly overlap to save space
	for i := 0; i < b.Sprites.num; i++ {
		s := b.Sprites.sprites[i]
		if b.dragging == s {
			continue
		}

		spaceIndex := i + 1

		x, y := b.spacePosition(spaceIndex)
		s.x, s.y = b.offsetPosition(x, y)
		if (spaceIndex > 6 && spaceIndex < 13) || (spaceIndex > 18 && spaceIndex < 25) {
			s.x += b.barWidth
		}
		s.x += (b.spaceWidth - s.w) / 2

		/* multiple pieces
		if i <= 12 {
			s.y += b.overlapSize
		} else {
			s.y -= b.overlapSize
		}*/
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
