package game

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"code.rocketnine.space/tslocum/fibs"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/llgcode/draw2d/draw2dimg"
)

type stateUpdate struct {
	from int
	to   int
	v    []int
}

type board struct {
	x, y int
	w, h int

	op *ebiten.DrawImageOptions

	backgroundImage *ebiten.Image

	Sprites *Sprites

	spaces     [][]*Sprite // Space contents
	spaceRects [][4]int

	dragging *Sprite
	moving   *Sprite // Moving automatically

	dragTouchId ebiten.TouchID
	touchIDs    []ebiten.TouchID

	spaceWidth           int
	barWidth             int
	triangleOffset       float64
	horizontalBorderSize int
	verticalBorderSize   int
	overlapSize          int

	lastDirection int
	V             []int

	moveQueue chan *stateUpdate

	debug int // Print and draw debug information
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
		spaces:     make([][]*Sprite, 26),
		spaceRects: make([][4]int, 26),
		V:          make([]int, 42),
		moveQueue:  make(chan *stateUpdate, 10),
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

		b.spaces[space] = append(b.spaces[space], s)
	}

	go b.handlePieceMoves()

	b.op = &ebiten.DrawImageOptions{}

	b.dragTouchId = -1

	return b
}

func (b *board) newSprite(white bool) *Sprite {
	s := &Sprite{}
	s.colorWhite = white
	s.w, s.h = imgCheckerWhite.Size()
	return s
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
	b.op.GeoM.Translate(float64(b.horizontalBorderSize-borderSize), 0)
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
		triangleTip := float64((b.h - (b.verticalBorderSize * 2)) / 2)
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

	drawSprite := func(sprite *Sprite) {
		b.op.GeoM.Reset()
		b.op.GeoM.Translate(float64(sprite.x), float64(sprite.y))

		if sprite.colorWhite {
			screen.DrawImage(imgCheckerWhite, b.op)
		} else {
			screen.DrawImage(imgCheckerBlack, b.op)
		}
	}

	b.iterateSpaces(func(space int) {
		var numPieces int
		for _, sprite := range b.spaces[space] {
			if sprite == b.dragging || sprite == b.moving {
				continue
			}
			numPieces++

			drawSprite(sprite)

			if numPieces > 5 {
				label := fmt.Sprintf("%d", numPieces)
				labelColor := color.RGBA{255, 255, 255, 255}
				if sprite.colorWhite {
					labelColor = color.RGBA{0, 0, 0, 255}
				}

				bounds := text.BoundString(mplusNormalFont, label)
				overlayImage := ebiten.NewImage(bounds.Dx()*2, bounds.Dy()*2)
				text.Draw(overlayImage, label, mplusNormalFont, 0, bounds.Dy(), labelColor)

				x, y, w, h := b.stackSpaceRect(space, numPieces)
				x += (w / 2) - (bounds.Dx() / 2)
				y += (h / 2) - (bounds.Dy() / 2)
				x, y = b.offsetPosition(x, y)

				b.op.GeoM.Reset()
				b.op.GeoM.Translate(float64(x), float64(y))
				screen.DrawImage(overlayImage, b.op)
			}
		}
	})

	// Draw moving sprite
	if b.moving != nil {
		drawSprite(b.moving)
	}

	// Draw dragged sprite
	if b.dragging != nil {
		drawSprite(b.dragging)
	}

	if b.debug == 2 {
		b.iterateSpaces(func(space int) {
			x, y, w, h := b.spaceRect(space)
			spaceImage := ebiten.NewImage(w, h)
			if space%2 == 0 {
				spaceImage.Fill(color.RGBA{0, 0, 0, 150})
			} else {
				spaceImage.Fill(color.RGBA{255, 255, 255, 150})
			}

			br := ""
			if b.bottomRow(space) {
				br = "B"
			}

			ebitenutil.DebugPrint(spaceImage, fmt.Sprintf(" %d %s", space, br))

			x, y = b.offsetPosition(x, y)

			b.op.GeoM.Reset()
			b.op.GeoM.Translate(float64(x), float64(y))
			screen.DrawImage(spaceImage, b.op)
		})
	}
}

func (b *board) setRect(x, y, w, h int) {
	if b.x == x && b.y == y && b.w == w && b.h == h {
		return
	}
	const stackAllowance = 0.97 // TODO configurable

	b.x, b.y, b.w, b.h = x, y, w, h

	b.horizontalBorderSize = 0

	b.triangleOffset = float64(b.h-(b.verticalBorderSize*2)) / 33

	for {
		b.verticalBorderSize = 7 // TODO configurable

		b.spaceWidth = (b.w - (b.horizontalBorderSize * 2)) / 13

		b.barWidth = b.spaceWidth

		b.overlapSize = (((b.h - (b.verticalBorderSize * 2)) - (int(b.triangleOffset) * 2)) / 2) / 5
		o := int(float64(b.spaceWidth) * stackAllowance)
		if b.overlapSize >= o {
			b.overlapSize = o
			break
		}

		b.horizontalBorderSize++
	}

	extraSpace := b.w - (b.spaceWidth * 12)
	largeBarWidth := int(float64(b.spaceWidth) * 1.25)
	if extraSpace >= largeBarWidth {
		b.barWidth = largeBarWidth
	}
	// TODO barwidth in calcs is wrong

	b.horizontalBorderSize = ((b.w - (b.spaceWidth * 12)) - b.barWidth) / 2
	if b.horizontalBorderSize < 0 {
		b.horizontalBorderSize = 0
	}

	loadAssets(b.spaceWidth)
	for i := 0; i < b.Sprites.num; i++ {
		s := b.Sprites.sprites[i]
		s.w, s.h = imgCheckerWhite.Size()
	}

	b.setSpaceRects()
	b.updateBackgroundImage()
	b.positionCheckers()
}

func (b *board) offsetPosition(x, y int) (int, int) {
	return b.x + x + b.horizontalBorderSize, b.y + y + b.verticalBorderSize
}

func (b *board) positionCheckers() {
	for space := 0; space < 26; space++ {
		sprites := b.spaces[space]

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
	pieces := b.spaces[space]
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

// TODO move to fibs library
func (b *board) iterateSpaces(f func(space int)) {
	if b.V[fibs.StateDirection] == 1 {
		for space := 0; space <= 25; space++ {
			f(space)
		}
		return
	}
	for space := 25; space >= 0; space-- {
		f(space)
	}
}

func (b *board) translateSpace(space int) int {
	if b.V[fibs.StateDirection] == -1 {
		// Spaces range from 24 - 1.
		if space == 0 || space == 25 {
			space = 25 - space
		} else if space <= 12 {
			space = 12 + space
		} else {
			space = space - 12
		}
	}
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
			y = (b.h / 2) - b.verticalBorderSize
		}

		w = b.spaceWidth

		var hspace int // horizontal space
		var add int
		if space == 0 {
			hspace = 6
			w = b.barWidth
		} else if space == 25 {
			hspace = 6
			w = b.barWidth
		} else if space <= 6 {
			hspace = space - 1
		} else if space <= 12 {
			hspace = space - 1
			add = b.barWidth
		} else if space <= 18 {
			hspace = 24 - space
			add = b.barWidth
		} else {
			hspace = 24 - space
		}

		x = (b.spaceWidth * hspace) + add

		h = (b.h - (b.verticalBorderSize * 2)) / 2

		b.spaceRects[trueSpace] = [4]int{x, y, w, h}
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
	bottomBar := 25
	if b.V[fibs.StateDirection] == 1 {
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
	osize := float64(stack)
	var o int
	if stack > 4 {
		osize = 3.5
	}
	if b.bottomRow(space) {
		osize += 1.0
	}
	o = int(osize * float64(b.overlapSize))
	if !b.bottomRow(space) {
		y += o
	} else {
		y = y + (h - o)
	}

	w, h = b.spaceWidth, b.spaceWidth
	if space == 0 || space == 25 {
		w = b.barWidth
	}

	return x, y, w, h
}

func (b *board) SetState(v []int) {
	b.moveQueue <- &stateUpdate{-1, -1, v}
}

func (b *board) ProcessState() {
	v := b.V

	if b.lastDirection != v[fibs.StateDirection] {
		b.setSpaceRects()
	}
	b.lastDirection = v[fibs.StateDirection]

	b.Sprites = &Sprites{}
	b.spaces = make([][]*Sprite, 26)
	for space := 0; space < 26; space++ {
		spaceValue := v[fibs.StateBoardSpace0+space]
		if spaceValue == 0 {
			continue
		}

		white := spaceValue > 0
		// TODO reverse bar spaces - always?
		// TODO take direction into account

		abs := spaceValue
		if abs < 0 {
			abs *= -1
		}

		for i := 0; i < abs; i++ {
			s := b.newSprite(white)
			b.spaces[space] = append(b.spaces[space], s)
			b.Sprites.sprites = append(b.Sprites.sprites, s)
		}
	}
	b.Sprites.num = len(b.Sprites.sprites)

	b.positionCheckers()
}

func (b *board) _movePiece(sprite *Sprite, from int, to int, speed int) {
	moveSize := 1
	moveDelay := time.Duration(2/speed) * time.Millisecond

	stackTo := len(b.spaces[to])
	if stackTo == 1 && sprite.colorWhite != b.spaces[to][0].colorWhite {
		stackTo = 0 // Hit
	}
	x, y, _, _ := b.stackSpaceRect(to, stackTo)
	x, y = b.offsetPosition(x, y)

	if sprite.x != x {
		// Center
		cy := (b.h / 2) - (b.spaceWidth / 2)
		for {
			if sprite.y == cy {
				break
			}
			if sprite.y < cy {
				sprite.y += moveSize
				if sprite.y > cy {
					sprite.y = cy
				}
			} else if sprite.y > cy {
				sprite.y -= moveSize
				if sprite.y < cy {
					sprite.y = cy
				}
			}
			time.Sleep(moveDelay)
		}
		for {
			if sprite.x == x {
				break
			}
			if sprite.x < x {
				sprite.x += moveSize
				if sprite.x > x {
					sprite.x = x
				}
			} else if sprite.x > x {
				sprite.x -= moveSize
				if sprite.x < x {
					sprite.x = x
				}
			}
			time.Sleep(moveDelay / 2)
		}
	}
	for {
		if sprite.x == x && sprite.y == y {
			break
		}
		if sprite.x < x {
			sprite.x += moveSize
			if sprite.x > x {
				sprite.x = x
			}
		} else if sprite.x > x {
			sprite.x -= moveSize
			if sprite.x < x {
				sprite.x = x
			}
		}
		if sprite.y < y {
			sprite.y += moveSize
			if sprite.y > y {
				sprite.y = y
			}
		} else if sprite.y > y {
			sprite.y -= moveSize
			if sprite.y < y {
				sprite.y = y
			}
		}
		time.Sleep(moveDelay)
	}

	// TODO do not add bear off pieces
	b.spaces[to] = append(b.spaces[to], sprite)
	for i, s := range b.spaces[from] {
		if s == sprite {
			b.spaces[from] = append(b.spaces[from][:i], b.spaces[from][i+1:]...)
			break
		}
	}
	b.moving = nil
}

func (b *board) handlePieceMoves() {
	for u := range b.moveQueue {
		if u.from == -1 || u.to == -1 {
			b.V = u.v
			b.ProcessState()
			continue
		}

		from, to := u.from, u.to

		pieces := b.spaces[from]
		if len(pieces) == 0 {
			continue
		}

		sprite := pieces[len(pieces)-1]

		var moveAfter *Sprite
		if len(b.spaces[to]) == 1 {
			if sprite.colorWhite != b.spaces[to][0].colorWhite {
				moveAfter = b.spaces[to][0]
			}
		}

		b._movePiece(sprite, from, to, 1)
		if moveAfter != nil {
			toBar := 0
			if b.V[fibs.StateDirection] == 1 {
				toBar = 25
			}
			b._movePiece(moveAfter, to, toBar, 2)
		}

		b.positionCheckers()
	}
}

// Do not call directly
func (b *board) movePiece(from int, to int) {
	b.moveQueue <- &stateUpdate{from, to, nil}
}

func (b *board) update() {
	if b.dragging == nil {
		// TODO allow grabbing multiple pieces by grabbing further down the stack

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

	x, y := ebiten.CursorPosition()
	if b.dragTouchId != -1 {
		x, y = ebiten.TouchPosition(b.dragTouchId)
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
		// TODO allow dragging anywhere outside of board to bear off
		// allow dragging on to bar to bear off

		index := b.spaceAt(x, y)
		if index >= 0 {
			for space, pieces := range b.spaces {
				for stackIndex, piece := range pieces {
					if piece == dropped {
						b.spaces[space] = append(b.spaces[space][:stackIndex], b.spaces[space][stackIndex+1:]...)
						b.spaces[index] = append(b.spaces[index], dropped)
						break
					}
				}
			}
		}
		b.positionCheckers()
	}

	if b.dragging != nil {
		sprite := b.dragging
		sprite.x = x - (sprite.w / 2)
		sprite.y = y - (sprite.h / 2)
	}
}
