package game

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
	"strings"

	"code.rocket9labs.com/tslocum/bgammon"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
)

type lobbyButton struct {
	label string
	f     func()
}

var mainButtons = []*lobbyButton{
	{"Refresh", func() {}},
	{"Watch", func() {}},
	{"Invite", func() {}},
	{"Join", func() {}},
}

var inviteButtons = []*lobbyButton{
	{"Cancel", func() {}},
	{"- Point", func() {}},
	{"+ Point", func() {}},
	{"Send", func() {}},
}

type lobby struct {
	x, y int
	w, h int

	fullscreen bool

	padding         float64
	entryH          float64
	buttonBarHeight int

	loaded bool
	games  []bgammon.GameListing

	touchIDs []ebiten.TouchID

	offset int

	selected int

	buffer      *ebiten.Image
	bufferDirty bool

	bufferButtons      *ebiten.Image
	bufferButtonsDirty bool

	op *ebiten.DrawImageOptions

	c *Client

	inviteUser   *WhoInfo
	invitePoints int

	refresh bool
}

func NewLobby() *lobby {
	l := &lobby{
		refresh: true,
		op:      &ebiten.DrawImageOptions{},
	}
	return l
}

func (l *lobby) setGameList(games []bgammon.GameListing) {
	l.games = games
	l.loaded = true

	sort.Slice(l.games, func(i, j int) bool {
		if (l.games[i].Players) != (l.games[j].Players) {
			return l.games[i].Players < l.games[j].Players
		}
		if (l.games[i].Password) != (l.games[j].Password) {
			return !l.games[i].Password
		}
		return strings.ToLower(l.games[i].Name) < strings.ToLower(l.games[j].Name)
	})

	l.bufferDirty = true
}

func (l *lobby) getButtons() []*lobbyButton {
	if l.inviteUser != nil {
		return inviteButtons
	}
	return mainButtons
}

func (l *lobby) _drawBufferButtons() {
	l.bufferButtons.Fill(frameColor)

	// Draw border
	for ly := 0; ly < 1; ly++ {
		for lx := 0; lx < l.w; lx++ {
			l.bufferButtons.Set(lx, ly, borderColor)
		}
	}

	buttons := l.getButtons()

	buttonWidth := l.w / len(buttons)
	for i, button := range buttons {
		// Draw border
		if i > 0 {
			for ly := 0; ly < l.buttonBarHeight; ly++ {
				for lx := buttonWidth * i; lx < (buttonWidth*i)+1; lx++ {
					l.bufferButtons.Set(lx, ly, borderColor)
				}
			}
		}
		bounds := text.BoundString(mediumFont, button.label)

		labelColor := lightCheckerColor

		img := ebiten.NewImage(bounds.Dx()*2, bounds.Dy()*2)
		text.Draw(img, button.label, mediumFont, 0, standardLineHeight, labelColor)

		l.op.GeoM.Reset()
		l.op.GeoM.Translate(float64(buttonWidth*i)+float64((buttonWidth-bounds.Dx())/2), float64(l.buttonBarHeight-standardLineHeight*1.5)/2)
		l.bufferButtons.DrawImage(img, l.op)
	}
}

// Draw to the off-screen buffer.
func (l *lobby) drawBuffer() {
	l.buffer.Fill(frameColor)

	// Draw invite user dialog
	if l.inviteUser != nil {
		labels := []string{
			fmt.Sprintf("Invite %s to a %d point match.", l.inviteUser.Username, l.invitePoints),
			"",
			fmt.Sprintf("    Rating: %d", l.inviteUser.Rating),
			fmt.Sprintf("    Experience: %d", l.inviteUser.Experience),
		}

		lineHeight := 30
		padding := 24.0
		for i, label := range labels {
			bounds := text.BoundString(mediumFont, label)
			labelColor := triangleA
			img := ebiten.NewImage(l.w-int(l.padding*2), int(l.entryH))
			text.Draw(img, label, mediumFont, 4, bounds.Dy(), labelColor)
			l.op.GeoM.Reset()
			l.op.GeoM.Translate(padding, padding+float64(i*lineHeight))
			l.buffer.DrawImage(img, l.op)
		}
	} else {
		var img *ebiten.Image
		drawEntry := func(cx float64, cy float64, colA string, colB string, highlight bool, title bool) {
			labelColor := triangleA
			if highlight {
				labelColor = lightCheckerColor
			} else if title {
				labelColor = lightCheckerColor
			}

			img = ebiten.NewImage(l.w-int(l.padding*2), int(l.entryH))
			if highlight {
				highlightColor := color.RGBA{triangleA.R, triangleA.G, triangleA.B, 15}
				img.SubImage(image.Rect(0, 0, l.w, int(l.entryH))).(*ebiten.Image).Fill(highlightColor)

				div := 1.75
				highlightBorderColor := color.RGBA{uint8(float64(frameColor.R) / div), uint8(float64(frameColor.G) / div), uint8(float64(frameColor.B) / div), 200}
				for x := 0; x < l.w; x++ {
					img.Set(x, 0, highlightBorderColor)
					img.Set(x, int(l.entryH)-1, highlightBorderColor)
				}
				for by := 0; by < int(l.entryH)-1; by++ {
					img.Set(0, by, highlightBorderColor)
					img.Set(l.w-(int(l.padding)*2)-1, by, highlightBorderColor)
				}
			}

			text.Draw(img, colA, mediumFont, 4, standardLineHeight, labelColor)
			text.Draw(img, colB, mediumFont, int(250*ebiten.DeviceScaleFactor()), standardLineHeight, labelColor)
			//text.Draw(img, colC, mediumFont, int(500*ebiten.DeviceScaleFactor()), standardLineHeight, labelColor)

			l.op.GeoM.Reset()
			l.op.GeoM.Translate(cx, cy)
			l.buffer.DrawImage(img, l.op)
		}

		titleOffset := 2.0

		if !l.loaded {
			drawEntry(l.padding, l.padding-titleOffset, "Loading...", "Please wait...", false, true)
			return
		}

		for ly := -2; ly < -1; ly++ {
			for lx := 0; lx < l.w; lx++ {
				l.buffer.Set(lx, int(l.padding)+int(l.entryH)+ly, borderColor)
			}
		}

		cx, cy := 0.0, 0.0 // Cursor
		drawEntry(cx+l.padding, cy+l.padding-titleOffset, "Status", "Name", false, true)
		cy += l.entryH
		i := 0
		var status string
		for _, entry := range l.games {
			if i >= l.offset {
				if entry.Players == 2 {
					status = "Full"
				} else {
					if !entry.Password {
						status = "Open"
					} else {
						status = "Private"
					}
				}

				drawEntry(cx+l.padding, cy+l.padding, status, entry.Name, i == l.selected, false)

				cy += l.entryH
			}

			i++
		}
	}

	if l.bufferButtonsDirty {
		l._drawBufferButtons()
		l.bufferButtonsDirty = false
	}

	l.op.GeoM.Reset()
	l.op.GeoM.Translate(float64(l.x), float64(l.h-l.buttonBarHeight))
	l.buffer.DrawImage(l.bufferButtons, l.op)
}

// Draw to the screen.
func (l *lobby) draw(screen *ebiten.Image) {
	if l.buffer == nil {
		return
	}

	var p bool

	if l.bufferDirty {
		if len(l.games) > 1 {
			p = true
			//debugGame.toggleProfiling()
		}

		l.drawBuffer()
		l.bufferDirty = false
	}

	l.op.GeoM.Reset()
	l.op.GeoM.Translate(float64(l.x), float64(l.y))
	screen.DrawImage(l.buffer, l.op)

	if p {
		//debugGame.toggleProfiling()
		//os.Exit(0)
	}
}

func (l *lobby) setRect(x, y, w, h int) {
	if l.x == x && l.y == y && l.w == w && l.h == h {
		return
	}

	s := ebiten.DeviceScaleFactor()
	l.padding = 4 * s
	l.entryH = 32 * s

	if l.w != w || l.h != h {
		l.buffer = ebiten.NewImage(w, h)
		l.bufferButtons = ebiten.NewImage(w, l.buttonBarHeight)
	}

	l.x, l.y, l.w, l.h = x, y, w, h
	l.bufferDirty = true
	l.bufferButtonsDirty = true
}

func (l *lobby) click(x, y int) {
	inRect := l.x <= x && x <= l.x+l.w && l.y <= y && y <= l.y+l.h
	if !inRect {
		return
	}

	// Handle button click
	if y >= l.h-l.buttonBarHeight {
		if l.c == nil {
			// Not yet connected
			return
		}

		buttonWidth := l.w / len(l.getButtons())
		buttonIndex := x / buttonWidth

		if l.inviteUser != nil {
			switch buttonIndex {
			case 0:
				l.inviteUser = nil
				l.bufferDirty = true
				l.bufferButtonsDirty = true
			case 1:
				l.invitePoints--
				if l.invitePoints < 1 {
					l.invitePoints = 1
				}
				l.bufferDirty = true
			case 2:
				l.invitePoints++
				l.bufferDirty = true
			case 3:
				l.c.Out <- []byte(fmt.Sprintf("invite %s %d", l.inviteUser.Username, l.invitePoints))

				l.inviteUser = nil
				l.bufferDirty = true
				l.bufferButtonsDirty = true

				viewBoard = true
			}
			return
		}

		switch buttonIndex {
		case 0:
			l.refresh = true
			l.c.Out <- []byte("rawwho")
		case 1:
			l.c.Out <- []byte(fmt.Sprintf("watch %d", l.games[l.selected].ID))
			viewBoard = true
		case 2:
			/*l.inviteUser = l.games[l.selected]
			l.invitePoints = 1
			l.bufferDirty = true
			l.bufferButtonsDirty = true*/
		case 3:
			l.c.Out <- []byte(fmt.Sprintf("join %d", l.games[l.selected].ID))
			viewBoard = true
		}
		return
	}

	// Handle entry click
	clickedEntry := (((y - int(l.padding)) - l.y) / int(l.entryH)) - 1
	if clickedEntry >= 0 {
		l.selected = l.offset + clickedEntry
		l.bufferDirty = true
	}
}

func (l *lobby) update() {
	scrollLength := 3

	if _, y := ebiten.Wheel(); y != 0 {
		scroll := int(math.Ceil(y))
		if scroll < -1 {
			scroll = -1
		} else if scroll > 1 {
			scroll = 1
		}
		l.offset -= scroll * scrollLength
		l.bufferDirty = true
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		l.selected++
		l.bufferDirty = true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		l.selected--
		l.bufferDirty = true
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
		l.offset += scrollLength * 4
		l.bufferDirty = true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPageUp) {
		l.offset -= scrollLength * 4
		l.bufferDirty = true
	}

	if l.selected < 0 {
		l.selected = 0
	}
	if l.offset < 0 {
		l.offset = 0
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		l.click(x, y)
	}

	l.touchIDs = inpututil.AppendJustPressedTouchIDs(l.touchIDs[:0])
	for _, id := range l.touchIDs {
		x, y := ebiten.TouchPosition(id)
		l.click(x, y)
	}
}
