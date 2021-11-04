package game

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/exp/shiny/materialdesign/colornames"
)

const (
	windowMinimized = iota
	windowNormal
	WindowMaximized
)

const windowStartingAlpha = 0.9

const bufferCharacterWidth = 12

type tabbedBuffers struct {
	buffers []*textBuffer
	labels  [][]byte

	x, y int
	w, h int

	unfocusedAlpha float64

	buffer      *ebiten.Image
	bufferDirty bool

	wrapWidth int

	op *ebiten.DrawImageOptions

	state int

	focused bool

	touchIDs []ebiten.TouchID
}

func newTabbedBuffers() *tabbedBuffers {
	tab := &tabbedBuffers{
		state:          windowNormal,
		unfocusedAlpha: windowStartingAlpha,
		buffer:         ebiten.NewImage(1, 1),
		op:             &ebiten.DrawImageOptions{},
	}

	b := &textBuffer{
		tab: tab,
	}
	tab.buffers = []*textBuffer{b}

	return tab
}

func (t *tabbedBuffers) setRect(x, y, w, h int) {
	if t.x == x && t.y == y && t.w == w && t.h == h {
		return
	}

	if t.w != w || t.h != h {
		t.buffer = ebiten.NewImage(w, h)
		t.bufferDirty = true
	}

	if t.w != w {
		t.wrapWidth = w / bufferCharacterWidth
		for _, b := range t.buffers {
			b.wrapDirty = true
		}
	}

	t.x, t.y, t.w, t.h = x, y, w, h
}

func (t *tabbedBuffers) drawBuffer() {
	t.buffer.Fill(borderColor)

	sub := t.buffer.SubImage(image.Rect(1, 1, t.w-1, t.h-1)).(*ebiten.Image)
	sub.Fill(frameColor)

	b := t.buffers[0]

	l := len(b.contentWrapped)

	lineHeight := 16
	showLines := t.h / lineHeight
	if showLines > 1 {
		showLines--
	}
	if showLines > 1 {
		showLines--
	}

	if l < showLines {
		showLines = l
	}
	for i := 0; i < showLines; i++ {
		line := b.contentWrapped[l-showLines+i]

		bounds := text.BoundString(monoFont, line)
		_ = bounds
		text.Draw(t.buffer, line, monoFont, 0, (lineHeight * (i + 1)), colornames.White)
	}

	text.Draw(t.buffer, "Say: Input buffer test", monoFont, 0, t.h-lineHeight, colornames.White)
}

func (t *tabbedBuffers) draw(target *ebiten.Image) {
	if t.buffer == nil {
		return
	}

	if t.state == windowMinimized {
		return
	}

	if t.bufferDirty {
		for _, b := range t.buffers {
			if b.wrapDirty {
				b.wrapContent()

				b.wrapDirty = false
			}
		}

		t.drawBuffer()

		t.bufferDirty = false
	}

	alpha := t.unfocusedAlpha
	if t.focused {
		alpha = 1.0
	}

	t.op.GeoM.Reset()
	t.op.GeoM.Translate(float64(t.x), float64(t.y))
	t.op.ColorM.Reset()
	t.op.ColorM.Scale(1, 1, 1, alpha)
	target.DrawImage(t.buffer, t.op)
}

func (t *tabbedBuffers) click(x, y int) {

}

func (t *tabbedBuffers) update() {
	// TODO accept keyboard input

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if t.state == windowMinimized {
			t.state = windowNormal
		} else {
			t.state = windowMinimized
		}
		t.bufferDirty = true
	}

	// Enter brings up keyboard and hides it when there is no input

	// TODO switch tabs

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		t.click(x, y)
	}

	t.touchIDs = inpututil.AppendJustPressedTouchIDs(t.touchIDs[:0])
	for _, id := range t.touchIDs {
		x, y := ebiten.TouchPosition(id)
		t.click(x, y)
	}

	// TODO add show virtual keyboard button
}
