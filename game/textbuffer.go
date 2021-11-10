package game

import (
	"bytes"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
)

type textBuffer struct {
	// Content of buffer separated by newlines.
	content [][]byte

	wrapDirty bool

	// Content as it appears on the screen.
	contentWrapped []string

	offset int

	tab *tabbedBuffers
}

func (b *textBuffer) Write(p []byte) {
	b.content = append(b.content, bytes.TrimRightFunc(p, unicode.IsSpace))

	b.wrapDirty = true
	b.tab.bufferDirty = true
	ebiten.ScheduleFrame()
}

func (b *textBuffer) wrapContent() {
	b.contentWrapped = nil
	for _, line := range b.content {
		if b.tab.wrapWidth == 0 {
			b.contentWrapped = append(b.contentWrapped, string(line))
			continue
		}

		lineStr := string(line)
		l := len(lineStr)
		for start := 0; start < l; start += b.tab.wrapWidth {
			end := start + b.tab.wrapWidth
			if end > l {
				end = l
			}
			b.contentWrapped = append(b.contentWrapped, lineStr[start:end])
		}
	}
}
