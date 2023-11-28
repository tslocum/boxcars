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
	"code.rocket9labs.com/tslocum/etk"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/leonelquinteros/gotext"
	"golang.org/x/image/font"
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
}

// TODO get button labels dynamically later as it needs to be after gotext loads

var mainButtons []*lobbyButton
var createButtons []*lobbyButton
var cancelJoinButtons []*lobbyButton

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

	c *Client

	refresh bool

	showCreateGame       bool
	createGameName       *etk.Input
	createGamePoints     *etk.Input
	createGamePassword   *etk.Input
	createGameCheckbox   *etk.Checkbox
	createGameAceyDeucey bool

	showJoinGame     bool
	joinGameID       int
	joinGameLabel    *etk.Text
	joinGamePassword *etk.Input

	showKeyboardButton *etk.Button
	buttonsGrid        *etk.Grid
	frame              *etk.Frame

	fontFace   font.Face
	lineHeight int
	lineOffset int
}

func NewLobby() *lobby {
	mainButtons = []*lobbyButton{
		{gotext.Get("Refresh")},
		{gotext.Get("Create")},
		{gotext.Get("Join")},
	}

	createButtons = []*lobbyButton{
		{gotext.Get("Cancel")},
		{gotext.Get("Create")},
	}

	cancelJoinButtons = []*lobbyButton{
		{gotext.Get("Cancel")},
		{gotext.Get("Join")},
	}

	l := &lobby{
		refresh:     true,
		fontFace:    mediumFont,
		buttonsGrid: etk.NewGrid(),
	}
	l.fontUpdated()
	l.showKeyboardButton = etk.NewButton(gotext.Get("Show Keyboard"), l.toggleKeyboard)
	l.frame = etk.NewFrame()
	l.frame.AddChild(l.showKeyboardButton)
	go l.handleRefreshTimer()
	return l
}

func (l *lobby) fontUpdated() {
	fontMutex.Lock()
	defer fontMutex.Unlock()

	m := l.fontFace.Metrics()
	l.lineHeight = m.Height.Round()
	l.lineOffset = m.Ascent.Round()
}

func (l *lobby) toggleKeyboard() error {
	if game.keyboard.Visible() {
		game.keyboard.Hide()
		l.showKeyboardButton.Label.SetText(gotext.Get("Show Keyboard"))
	} else {
		game.keyboard.Show()
		l.showKeyboardButton.Label.SetText(gotext.Get("Hide Keyboard"))
	}
	return nil
}

func (l *lobby) toggleAceyDeucey() error {
	l.createGameAceyDeucey = !l.createGameAceyDeucey
	return nil
}

func (l *lobby) handleRefreshTimer() {
	t := time.NewTicker(time.Second)
	for range t.C {
		if !game.loggedIn || viewBoard {
			continue
		}

		l.bufferDirty = true
		scheduleFrame()
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

func (l *lobby) confirmCreateGame() {
	typeAndPassword := "public"
	if len(strings.TrimSpace(game.lobby.createGamePassword.Text())) > 0 {
		typeAndPassword = fmt.Sprintf("private %s", strings.ReplaceAll(game.lobby.createGamePassword.Text(), " ", "_"))
	}
	points, err := strconv.Atoi(game.lobby.createGamePoints.Text())
	if err != nil {
		points = 1
	}
	acey := 0
	if game.lobby.createGameAceyDeucey {
		acey = 1
	}
	l.c.Out <- []byte(fmt.Sprintf("c %s %d %d %s", typeAndPassword, points, acey, game.lobby.createGameName.Text()))
}

func (l *lobby) confirmJoinGame() {
	l.c.Out <- []byte(fmt.Sprintf("j %d %s", l.joinGameID, l.joinGamePassword.Text()))
}

func (l *lobby) selectButton(buttonIndex int) func() error {
	return func() error {
		if l.showCreateGame {
			switch buttonIndex {
			case lobbyButtonCreateCancel:
				game.lobby.showCreateGame = false
				game.lobby.createGameName.Field.SetText("")
				game.lobby.createGamePassword.Field.SetText("")
				l.bufferDirty = true
				l.rebuildButtonsGrid()
				game.setRoot(listGamesFrame)
			case lobbyButtonCreateConfirm:
				l.confirmCreateGame()
			}
			return nil
		} else if l.showJoinGame {
			if buttonIndex == 0 { // Cancel
				l.showJoinGame = false
				l.bufferDirty = true
				l.rebuildButtonsGrid()
				if viewBoard {
					game.setRoot(game.Board.frame)
				} else {
					game.setRoot(listGamesFrame)
				}
			} else {
				l.confirmJoinGame()
			}
			return nil
		}

		switch buttonIndex {
		case lobbyButtonRefresh:
			l.refresh = true
			l.c.Out <- []byte("ls")
		case lobbyButtonCreate:
			if l.c.Username == "" {
				return nil
			}

			l.showCreateGame = true
			game.setRoot(createGameFrame)
			etk.SetFocus(l.createGameName)
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
			l.rebuildButtonsGrid()
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
				return nil
			}

			if l.games[l.selected].Password {
				l.showJoinGame = true
				game.setRoot(joinGameFrame)
				etk.SetFocus(l.joinGamePassword)
				l.joinGameLabel.SetText(gotext.Get("Join match: %s", l.games[l.selected].Name))
				l.joinGamePassword.Field.SetText("")
				l.joinGameID = l.games[l.selected].ID
				l.bufferDirty = true
				l.rebuildButtonsGrid()
			} else {
				l.c.Out <- []byte(fmt.Sprintf("j %d", l.games[l.selected].ID))
				setViewBoard(true)
				scheduleFrame()
			}
		}
		return nil
	}
}

func (l *lobby) rebuildButtonsGrid() {
	r := l.buttonsGrid.Rect()
	l.buttonsGrid.Empty()

	for i, btn := range l.getButtons() {
		l.buttonsGrid.AddChildAt(etk.NewButton(btn.label, l.selectButton(i)), i, 0, 1, 1)
	}

	l.buttonsGrid.SetRect(r)
}

// Draw to the off-screen buffer.
func (l *lobby) drawBuffer() {
	l.buffer.Fill(frameColor)

	if l.showCreateGame || l.showJoinGame {
		// Dialog is drawn by etk.
		return
	}

	titleColor := color.RGBA{R: 205, G: 205, B: 0, A: 255}

	var img *ebiten.Image
	drawEntry := func(cx float64, cy float64, colA string, colB string, colC string, highlight bool, title bool) {
		labelColor := triangleA
		if highlight {
			labelColor = lightCheckerColor
		} else if title {
			labelColor = titleColor
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

		fontMutex.Lock()
		defer fontMutex.Unlock()

		text.Draw(img, colA, l.fontFace, 4, l.lineOffset, labelColor)
		text.Draw(img, colB, l.fontFace, 250, l.lineOffset, labelColor)
		text.Draw(img, colC, l.fontFace, 500, l.lineOffset, labelColor)

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
	drawEntry(cx+l.padding, cy+l.padding-titleOffset, gotext.Get("Status"), gotext.Get("Points"), gotext.Get("Name"), false, true)
	cy += l.entryH

	if len(l.games) == 0 {
		drawEntry(cx+l.padding, cy+l.padding, gotext.Get("No matches available. Please create one."), "", "", false, false)
	} else {
		i := 0
		var status string
		for _, entry := range l.games {
			if i >= l.offset {
				if entry.Password {
					status = gotext.Get("Private")
				} else if entry.Players == 2 {
					status = gotext.Get("Started")
				} else {
					status = gotext.Get("Available")
				}

				drawEntry(cx+l.padding, cy+l.padding, status, strconv.Itoa(entry.Points), entry.Name, i == l.selected, false)

				cy += l.entryH
			}

			i++
		}
	}
}

// Draw to the screen.
func (l *lobby) draw(screen *ebiten.Image) {
	if l.buffer == nil {
		return
	}

	if l.bufferDirty {
		l.drawBuffer()
		l.bufferDirty = false
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(l.x), float64(l.y))
	screen.DrawImage(l.buffer, op)
}

func (l *lobby) setRect(x, y, w, h int) {
	if OptimizeSetRect && l.x == x && l.y == y && l.w == w && l.h == h {
		return
	}

	if game.scaleFactor >= 1.25 {
		if l.fontFace != largeFont {
			l.fontFace = largeFont
			l.fontUpdated()
		}
	} else {
		if l.fontFace != mediumFont {
			l.fontFace = mediumFont
			l.fontUpdated()
		}
	}
	l.padding = 4
	l.entryH = float64(l.lineHeight)

	if l.w != w || l.h != h {
		l.buffer = ebiten.NewImage(w, h)
	}

	l.x, l.y, l.w, l.h = x, y, w, h
	l.bufferDirty = true
}

func (l *lobby) click(x, y int) {
	inRect := l.x <= x && x <= l.x+l.w && l.y <= y && y <= l.y+l.h
	if !inRect {
		return
	}

	// Handle button click
	if y >= l.h-l.buttonBarHeight {
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
					game.setRoot(joinGameFrame)
					etk.SetFocus(l.joinGamePassword)
					l.joinGameLabel.SetText(gotext.Get("Join match: %s", entry.Name))
					l.joinGamePassword.Field.SetText("")
					l.joinGameID = entry.ID
					l.bufferDirty = true
					l.rebuildButtonsGrid()
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
	if !l.showCreateGame && !l.showJoinGame {
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
		game.EnableTouchInput()
		x, y := ebiten.TouchPosition(id)
		l.click(x, y)
	}
}
