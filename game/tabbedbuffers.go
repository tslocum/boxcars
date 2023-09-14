package game

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
)

const (
	windowMinimized = iota
	windowNormal
	WindowMaximized
)

const windowStartingAlpha = 0.9

const (
	smallFontSize  = 20
	monoFontSize   = 10
	mediumFontSize = 24
	largeFontSize  = 32
)

const (
	monoLineHeight     = 14
	standardLineHeight = 24
)

type tabbedBuffers struct {
	buffers []*textBuffer
	labels  [][]byte

	x, y int
	w, h int

	padding int

	unfocusedAlpha float64

	buffer      *ebiten.Image
	bufferDirty bool

	wrapWidth int

	op *ebiten.DrawImageOptions

	state int

	docked bool

	focused bool

	touchIDs []ebiten.TouchID

	incomingBuffer []rune

	inputBuffer []byte

	client *Client

	chatFont       font.Face
	chatFontSize   int
	chatLineHeight int

	acceptInput bool
}

func newTabbedBuffers() *tabbedBuffers {
	tab := &tabbedBuffers{
		state:          windowNormal,
		unfocusedAlpha: windowStartingAlpha,
		buffer:         ebiten.NewImage(1, 1),
		op:             &ebiten.DrawImageOptions{},
		chatFont:       monoFont,
		chatFontSize:   monoFontSize,
		chatLineHeight: monoLineHeight,
	}

	// TODO
	tab.chatFont = smallFont
	tab.chatFontSize = smallFontSize
	tab.chatLineHeight = standardLineHeight

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
		t.wrapWidth = (w - (t.padding * 4)) / t.chatFontSize
		for _, b := range t.buffers {
			b.wrapDirty = true
		}
	}

	t.x, t.y, t.w, t.h = x, y, w, h
}

func (t *tabbedBuffers) linesVisible() int {
	lines := (t.h - (t.padding * 2)) / t.chatLineHeight
	// Leave space for the input buffer.
	if t.acceptInput {
		if lines > 1 {
			lines--
		}
	}
	return lines
}

func (t *tabbedBuffers) drawBuffer() {
	// Draw background.
	t.buffer.Fill(color.RGBA{0, 0, 0, 100})

	// Draw content.

	textColor := triangleALight

	b := t.buffers[0]

	l := len(b.contentWrapped)

	showLines := t.linesVisible()
	if l < showLines {
		showLines = l
	}

	for i := 0; i < showLines; i++ {
		line := b.contentWrapped[l-showLines+i]

		text.Draw(t.buffer, line, t.chatFont, t.padding*2, t.chatLineHeight*(i+1), textColor)
	}

	// Draw input buffer.
	if t.acceptInput {
		text.Draw(t.buffer, "> "+string(t.inputBuffer), t.chatFont, t.padding*2, t.h-(t.padding*4), textColor)
	}
}

func (t *tabbedBuffers) draw(target *ebiten.Image) {
	if t.buffer == nil {
		return
	}

	if !t.docked && t.state == windowMinimized {
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

	// Read user input.
	t.incomingBuffer = ebiten.AppendInputChars(t.incomingBuffer[:0])
	if len(t.incomingBuffer) > 0 {
		t.inputBuffer = append(t.inputBuffer, []byte(string(t.incomingBuffer))...)
		t.bufferDirty = true
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(t.inputBuffer) > 0 {
		b := string(t.inputBuffer)
		if len(b) > 1 {
			t.inputBuffer = []byte(b[:len(b)-1])
		} else {
			t.inputBuffer = nil
		}
		t.bufferDirty = true
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if len(t.inputBuffer) == 0 {
			if !t.docked {
				if t.state == windowMinimized {
					t.state = windowNormal
				} else {
					t.state = windowMinimized
				}

				t.bufferDirty = true
			}
		} else {
			if t.client != nil {
				if len(t.inputBuffer) > 0 {
					if t.inputBuffer[0] == '/' {
						t.client.Out <- t.inputBuffer[1:]
					} else {
						t.client.Out <- []byte(fmt.Sprintf("kibitz %s", t.inputBuffer))
					}
				}
			} else {
				StatusWriter.Write([]byte("* You have not connected to a server yet"))
			}
			t.inputBuffer = nil

			t.bufferDirty = true
		}
	}

	// TODO add show virtual keyboard button
}
