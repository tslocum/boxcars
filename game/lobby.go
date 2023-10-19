package game

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"code.rocket9labs.com/tslocum/bgammon"
	"code.rocketnine.space/tslocum/etk"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
)

const (
	lobbyButtonCreateCancel = iota
	lobbyButtonCreateConfirm
)

const (
	lobbyButtonRefresh = iota
	lobbyButtonCreate
	lobbyButtonJoin
)

type lobbyButton struct {
	label string
	f     func()
}

var mainButtons = []*lobbyButton{
	{"Refresh", func() {}},
	{"Create", func() {}},
	{"Join", func() {}},
}

var createButtons = []*lobbyButton{
	{"Cancel", func() {}},
	{"Create", func() {}},
}

var cancelConfirmButtons = []*lobbyButton{
	{"Cancel", func() {}},
	{"Confirm", func() {}},
}

var cancelJoinButtons = []*lobbyButton{
	{"Cancel", func() {}},
	{"Join", func() {}},
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

	lastClick time.Time

	touchIDs []ebiten.TouchID

	offset int

	selected int

	buffer      *ebiten.Image
	bufferDirty bool

	bufferButtons      *ebiten.Image
	bufferButtonsDirty bool

	c *Client

	refresh bool

	runeBuffer []rune

	showCreateGame         bool
	createGameNamePrev     string
	createGamePasswordPrev string
	createGameName         *etk.Input
	createGamePoints       *etk.Input
	createGamePassword     *etk.Input
	createGameFocus        int

	showJoinGame     bool
	joinGameID       int
	joinGameLabel    *etk.Text
	joinGamePassword *etk.Input
}

func NewLobby() *lobby {
	return &lobby{
		refresh: true,
	}
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
	if l.showCreateGame {
		return createButtons
	} else if l.showJoinGame {
		return cancelJoinButtons
	}
	return mainButtons
}

func (l *lobby) _drawBufferButtons() {
	l.bufferButtons.Fill(frameColor)

	// Draw border
	for lx := 0; lx < l.w; lx++ {
		l.bufferButtons.Set(lx, 0, borderColor)
		l.bufferButtons.Set(lx, l.buttonBarHeight-1, borderColor)
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

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(buttonWidth*i)+float64((buttonWidth-bounds.Dx())/2), float64(l.buttonBarHeight-standardLineHeight*1.5)/2)
		l.bufferButtons.DrawImage(img, op)
	}
}

// Draw to the off-screen buffer.
func (l *lobby) drawBuffer() {
	l.buffer.Fill(frameColor)

	if l.showCreateGame || l.showJoinGame {
		// Create game dialog is drawn by etk.
	} else {
		var img *ebiten.Image
		drawEntry := func(cx float64, cy float64, colA string, colB string, colC string, highlight bool, title bool) {
			labelColor := triangleA
			if highlight {
				labelColor = lightCheckerColor
			} else if title {
				labelColor = lightCheckerColor
			}

			img = ebiten.NewImage(l.w-int(l.padding*2), int(l.entryH))
			if highlight {
				highlightColor := color.RGBA{17, 17, 17, 10}
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
			text.Draw(img, colC, mediumFont, int(500*ebiten.DeviceScaleFactor()), standardLineHeight, labelColor)
			//text.Draw(img, colC, mediumFont, int(500*ebiten.DeviceScaleFactor()), standardLineHeight, labelColor)

			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(cx, cy)
			l.buffer.DrawImage(img, op)
		}

		titleOffset := 2.0

		if !l.loaded {
			drawEntry(l.padding, l.padding-titleOffset, "Loading...", "Please", "wait...", false, true)
			return
		}

		for ly := -2; ly < -1; ly++ {
			for lx := 0; lx < l.w; lx++ {
				l.buffer.Set(lx, int(l.padding)+int(l.entryH)+ly, borderColor)
			}
		}

		cx, cy := 0.0, 0.0 // Cursor
		drawEntry(cx+l.padding, cy+l.padding-titleOffset, "Status", "Points", "Name", false, true)
		cy += l.entryH

		if len(l.games) == 0 {
			drawEntry(cx+l.padding, cy+l.padding, "No matches available. Please create one.", "", "", false, false)
		} else {
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

					drawEntry(cx+l.padding, cy+l.padding, status, strconv.Itoa(entry.Points), entry.Name, i == l.selected, false)

					cy += l.entryH
				}

				i++
			}
		}
	}

	if l.bufferButtonsDirty {
		l._drawBufferButtons()
		l.bufferButtonsDirty = false
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(l.x), float64(l.h-l.buttonBarHeight))
	l.buffer.DrawImage(l.bufferButtons, op)
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

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(l.x), float64(l.y))
	screen.DrawImage(l.buffer, op)

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

		if l.showCreateGame {
			switch buttonIndex {
			case lobbyButtonCreateCancel:
				game.lobby.showCreateGame = false
				game.lobby.createGameFocus = 0
				game.lobby.createGameName.Field.SetText("")
				game.lobby.createGamePassword.Field.SetText("")
				l.bufferDirty = true
				l.bufferButtonsDirty = true
			case lobbyButtonCreateConfirm:
				typeAndPassword := "public"
				if len(strings.TrimSpace(game.lobby.createGamePassword.Text())) > 0 {
					typeAndPassword = fmt.Sprintf("private %s", strings.ReplaceAll(game.lobby.createGamePassword.Text(), " ", "_"))
				}
				points, err := strconv.Atoi(game.lobby.createGamePoints.Text())
				if err != nil {
					points = 1
				}
				l.c.Out <- []byte(fmt.Sprintf("c %s %d %s", typeAndPassword, points, game.lobby.createGameName.Text()))
			}
			return
		} else if l.showJoinGame {
			if buttonIndex == 0 { // Cancel
				l.showJoinGame = false
				l.bufferDirty = true
				l.bufferButtonsDirty = true
			} else {
				l.c.Out <- []byte(fmt.Sprintf("j %d %s", l.joinGameID, l.joinGamePassword.Text()))
			}
			return
		}

		switch buttonIndex {
		case lobbyButtonRefresh:
			l.refresh = true
			l.c.Out <- []byte("list")
		case lobbyButtonCreate:
			l.showCreateGame = true
			etk.SetRoot(createGameGrid)
			namePlural := l.c.Username
			lastLetter := namePlural[len(namePlural)-1]
			if lastLetter == 's' || lastLetter == 'S' {
				namePlural += "'"
			} else {
				namePlural += "'s"
			}
			l.createGameName.Field.SetText(namePlural + " match")
			l.createGamePoints.Field.SetText("1")
			l.createGamePassword.Field.SetText("")
			l.bufferDirty = true
			l.bufferButtonsDirty = true
			l.drawBuffer()
			scheduleFrame()
		/*case lobbyButtonWatch:
		if l.selected < 0 || l.selected >= len(l.games) {
			return
		}
		l.c.Out <- []byte(fmt.Sprintf("watch %d", l.games[l.selected].ID))
		setViewBoard(true)*/
		case lobbyButtonJoin:
			if l.selected < 0 || l.selected >= len(l.games) {
				return
			}
			if l.games[l.selected].Password {
				l.showJoinGame = true
				etk.SetRoot(joinGameGrid)
				l.joinGameLabel.SetText(fmt.Sprintf("Join match: %s", l.games[l.selected].Name))
				l.joinGamePassword.Field.SetText("")
				l.joinGameID = l.games[l.selected].ID
				l.bufferDirty = true
				l.bufferButtonsDirty = true
			} else {
				l.c.Out <- []byte(fmt.Sprintf("j %d", l.games[l.selected].ID))
				setViewBoard(true)
				scheduleFrame()
			}
		}
		return
	}

	// Handle entry click
	clickedEntry := (((y - int(l.padding)) - l.y) / int(l.entryH)) - 1
	if clickedEntry >= 0 {
		const doubleClickDuration = 200 * time.Millisecond
		newSelection := l.offset + clickedEntry
		if l.selected == newSelection && l.selected >= 0 && l.selected < len(l.games) {
			if time.Since(l.lastClick) <= doubleClickDuration {
				entry := l.games[l.selected]
				if entry.Password {
					l.showJoinGame = true
					etk.SetRoot(joinGameGrid)
					l.joinGameLabel.SetText(fmt.Sprintf("Join match: %s", entry.Name))
					l.joinGamePassword.Field.SetText("")
					l.joinGameID = entry.ID
					l.bufferDirty = true
					l.bufferButtonsDirty = true
				} else {
					l.c.Out <- []byte(fmt.Sprintf("j %d", entry.ID))
				}
				l.lastClick = time.Time{}
				return
			}
		}

		l.lastClick = time.Now()
		l.selected = newSelection
		l.bufferDirty = true
	}
}

func (l *lobby) update() {
	if l.showCreateGame || l.showJoinGame {
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			if l.showCreateGame {
				typeAndPassword := "public"
				if len(strings.TrimSpace(game.lobby.createGamePassword.Text())) > 0 {
					typeAndPassword = fmt.Sprintf("private %s", strings.ReplaceAll(game.lobby.createGamePassword.Text(), " ", "_"))
				}
				l.c.Out <- []byte(fmt.Sprintf("c %s %s", typeAndPassword, game.lobby.createGameName.Text()))
			} else {
				l.c.Out <- []byte(fmt.Sprintf("j %d %s", l.joinGameID, l.joinGamePassword.Text()))
			}
			return
		}
	} else {
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
