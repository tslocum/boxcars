package game

import "github.com/hajimehoshi/ebiten/v2"

type textBuffer struct {
	// Content of buffer separated by newlines.
	content [][]byte

	// Content as it appears on the screen.
	contentWrapped []byte

	offset int

	tab *tabbedBuffers
}

func (b *textBuffer) Write(p []byte) {
	b.content = append(b.content, p)

	b.wrapContent()

	b.tab.bufferDirty = true
	ebiten.ScheduleFrame()
}

func (b *textBuffer) wrapContent() {
	// TODO
}
