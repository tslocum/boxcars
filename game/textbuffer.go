package game

import (
	"bytes"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
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

	// TODO /boardstate results in invalid draw
	ebiten.ScheduleFrame()
}

func (b *textBuffer) wrapContent() {
	b.contentWrapped = nil
	for _, line := range b.content {
		lineStr := string(line)

		if b.tab.wrapWidth == 0 {
			b.contentWrapped = append(b.contentWrapped, lineStr)
			continue
		}

		l := len(lineStr)
		var start int
		var end int
		for start < l {
			for end = l; end > start; end-- {
				bounds := text.BoundString(b.tab.chatFont, lineStr[start:end])
				if bounds.Dx() < b.tab.w-(b.tab.padding*2) {
					// Break on whitespace.
					if end < l && !unicode.IsSpace(rune(lineStr[end])) {
						for endOffset := 0; endOffset < end-start; endOffset++ {
							if unicode.IsSpace(rune(lineStr[end-endOffset])) {
								end = end - endOffset
								break
							}
						}
					}
					b.contentWrapped = append(b.contentWrapped, lineStr[start:end])
					break
				}
			}
			start = end
		}
	}
}
