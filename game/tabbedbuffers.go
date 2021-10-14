package game

import (
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	windowMinimized = iota
	windowNormal
	WindowMaximized
)

const windowStartingAlpha = 0.8

type tabbedBuffers struct {
	buffers []*textBuffer
	labels  [][]byte

	x, y int
	w, h int

	unfocusedAlpha float64

	buffer      *ebiten.Image
	bufferDirty bool

	op *ebiten.DrawImageOptions

	state int

	focused bool
}

func newTabbedBuffers() *tabbedBuffers {
	return &tabbedBuffers{
		state:          windowNormal,
		unfocusedAlpha: windowStartingAlpha,
		buffer:         ebiten.NewImage(1, 1),
		op:             &ebiten.DrawImageOptions{},
	}
}

func (t *tabbedBuffers) setRect(x, y, w, h int) {
	if t.x == x && t.y == y && t.w == w && t.h == h {
		return
	}

	if t.w != w || t.h != h {
		t.buffer = ebiten.NewImage(w, h)
		t.bufferDirty = true
	}

	t.x, t.y, t.w, t.h = x, y, w, h
}

func (t *tabbedBuffers) drawBuffer() {
	t.buffer.Fill(frameColor)
}

func (t *tabbedBuffers) draw(target *ebiten.Image) {
	if t.buffer == nil {
		return
	}
	return // This feature is not yet finished.

	if t.bufferDirty {
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
