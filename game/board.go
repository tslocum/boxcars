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
	Sprites Sprites
	op      *ebiten.DrawImageOptions

	backgroundImage *ebiten.Image

	dragging *Sprite

	x, y int
	w, h int
}

func NewBoard() *board {
	b := &board{}

	b.Sprites.sprites = make([]*Sprite, 30)
	b.Sprites.num = 30
	for i := range b.Sprites.sprites {
		s := &Sprite{}

		r := rand.Intn(2)
		if r != 1 {
			s.image = imgCheckerWhite
		} else {
			s.image = imgCheckerBlack
		}

		s.w, s.h = imgCheckerWhite.Size()

		b.Sprites.sprites[i] = s
	}

	b.op = &ebiten.DrawImageOptions{}

	return b
}

func (b *board) updateBackgroundImage() {
	box := image.NewRGBA(image.Rect(0, 0, b.w, b.h))

	img := ebiten.NewImageFromImage(box)
	img.Fill(color.RGBA{0, 0, 0, 255})

	b.backgroundImage = ebiten.NewImageFromImage(img)

	box = image.NewRGBA(image.Rect(0, 0, b.w-10, b.h-10))

	img = ebiten.NewImageFromImage(box)
	img.Fill(color.RGBA{101, 56, 24, 255})

	b.op.GeoM.Reset()
	b.op.GeoM.Translate(float64(5), float64(5))
	b.backgroundImage.DrawImage(img, b.op)

	baseImg := image.NewRGBA(image.Rect(0, 0, b.w-10, b.h-10))

	gc := draw2dimg.NewGraphicContext(baseImg)
	// Set some properties
	gc.SetFillColor(color.RGBA{0, 0, 0, 255})

	// Draw triangles
	for i := 0; i < 2; i++ {
		triangleTip := float64(b.h / 2)
		if i == 0 {
			triangleTip -= 50
		} else {
			triangleTip += 50
		}
		for j := 0; j < 12; j++ {
			gc.MoveTo(float64(100*j), float64(b.h*i))
			gc.LineTo(float64(100*j)+50, triangleTip)
			gc.LineTo(float64(100*j)+100, float64(b.h*i))
			gc.Close()
			gc.Fill()
		}
	}

	img = ebiten.NewImageFromImage(baseImg)

	b.op.GeoM.Reset()
	b.op.GeoM.Translate(float64(5), float64(5))
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
		screen.DrawImage(sprite.image, b.op)
	}
}

func (b *board) setRect(x, y, w, h int) {
	if b.x == x && b.y == y && b.w == w && b.h == h {
		return
	}

	b.x, b.y, b.w, b.h = x, y, w, h

	b.updateBackgroundImage()
	b.positionCheckers()
}

func (b *board) offsetPosition(x, y int) (int, int) {
	const boardPadding = 7
	return b.x + x + boardPadding, b.y + y + boardPadding
}

func (b *board) positionCheckers() {
	for i := 0; i < b.Sprites.num; i++ {
		s := b.Sprites.sprites[i]
		s.x, s.y = b.offsetPosition(s.w*(i%12), s.h*(i/12))
	}
}

func (b *board) update() {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()

		if b.dragging == nil {
			for i := 0; i < b.Sprites.num; i++ {
				s := b.Sprites.sprites[i]
				if x >= s.x && y >= s.y && x <= s.x+s.w && y <= s.y+s.h {
					b.dragging = s

					// Bring sprite to front
					b.Sprites.sprites = append(b.Sprites.sprites[:i], b.Sprites.sprites[i+1:]...)
					b.Sprites.sprites = append(b.Sprites.sprites, s)

					break
				}
			}
		}
	}

	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		b.dragging = nil
	}

	if b.dragging != nil {
		x, y := ebiten.CursorPosition()
		sprite := b.dragging
		sprite.x = x - (sprite.w / 2)
		sprite.y = y - (sprite.h / 2)
	}
}
