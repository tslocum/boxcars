package game

import (
	"fmt"
	"image/color"
	"sort"
	"strconv"
	"strings"
	"time"

	"code.rocket9labs.com/tslocum/bgammon"
	"code.rocket9labs.com/tslocum/etk"
	"code.rocketnine.space/tslocum/messeji"
	"github.com/leonelquinteros/gotext"
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

const (
	lobbyIndentA = 200
	lobbyIndentB = 350
)

type lobbyButton struct {
	label string
}

// TODO get button labels dynamically later as it needs to be after gotext loads

var mainButtons []*lobbyButton
var createButtons []*lobbyButton
var cancelJoinButtons []*lobbyButton

type lobby struct {
	buttonBarHeight int

	loaded bool
	games  []bgammon.GameListing

	lastClick time.Time

	itemHeight int

	selected int

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

	availableMatchesList *etk.List

	showKeyboardButton *etk.Button
	buttonsGrid        *etk.Grid
	frame              *etk.Frame
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

	itemHeight := 48
	if defaultFontSize == extraLargeFontSize {
		itemHeight = 72
	}
	l := &lobby{
		refresh:     true,
		buttonsGrid: etk.NewGrid(),
		itemHeight:  itemHeight,
	}

	indentA, indentB := lobbyIndentA, lobbyIndentB
	if defaultFontSize == extraLargeFontSize {
		indentA, indentB = int(float64(indentA)*1.3), int(float64(indentB)*1.3)
	}

	matchList := etk.NewList(l.itemHeight, l.selectMatch)
	matchList.SetSelectionMode(etk.SelectRow)
	matchList.SetColumnSizes(indentA, indentB-indentA, -1)
	matchList.SetHighlightColor(color.RGBA{79, 55, 30, 255})
	l.availableMatchesList = matchList

	l.showKeyboardButton = etk.NewButton(gotext.Get("Show Keyboard"), l.toggleKeyboard)
	l.frame = etk.NewFrame()
	l.frame.AddChild(l.showKeyboardButton)
	return l
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

func (l *lobby) setGameList(games []bgammon.GameListing) {
	l.games = games
	l.loaded = true

	const (
		aceyPrefix = "(Acey-deucey)"
		botPrefix  = "BOT_"
	)
	sort.Slice(l.games, func(i, j int) bool {
		a, b := l.games[i], l.games[j]
		switch {
		case (a.Password) != (b.Password):
			return !a.Password
		case (a.Players) != (b.Players):
			return a.Players < b.Players
		case strings.HasPrefix(a.Name, aceyPrefix) != strings.HasPrefix(b.Name, aceyPrefix):
			return strings.HasPrefix(b.Name, aceyPrefix)
		case strings.HasPrefix(a.Name, botPrefix) != strings.HasPrefix(b.Name, botPrefix):
			return strings.HasPrefix(b.Name, botPrefix)
		default:
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}
	})

	newLabel := func(label string) *etk.Text {
		txt := etk.NewText(label)
		txt.SetFollow(false)
		txt.SetScrollBarVisible(false)
		txt.SetWordWrap(false)
		txt.SetVertical(messeji.AlignCenter)
		return txt
	}

	_, lastSelection := l.availableMatchesList.SelectedItem()

	var status string
	l.availableMatchesList.Clear()
	for i, entry := range l.games {
		if entry.Password {
			status = gotext.Get("Private")
		} else if entry.Players == 2 {
			status = gotext.Get("Started")
		} else {
			status = gotext.Get("Available")
		}
		l.availableMatchesList.AddChildAt(newLabel(status), 0, i)
		l.availableMatchesList.AddChildAt(newLabel(fmt.Sprintf("%d", entry.Points)), 1, i)
		l.availableMatchesList.AddChildAt(newLabel(entry.Name), 2, i)
	}

	if lastSelection >= 0 && lastSelection < len(l.games) {
		l.availableMatchesList.SetSelectedItem(0, lastSelection)
	} else {
		_, selected := l.availableMatchesList.SelectedItem()
		if selected == -1 {
			l.availableMatchesList.SetSelectedItem(0, 0)
		}
	}
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
				l.rebuildButtonsGrid()
				game.setRoot(listGamesFrame)
			case lobbyButtonCreateConfirm:
				l.confirmCreateGame()
			}
			return nil
		} else if l.showJoinGame {
			if buttonIndex == 0 { // Cancel
				l.showJoinGame = false
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
			l.rebuildButtonsGrid()
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

func (l *lobby) selectMatch(index int) bool {
	if index < 0 || index >= len(l.games) {
		return false
	}
	const doubleClickDuration = 200 * time.Millisecond
	if l.selected == index && l.selected >= 0 && l.selected < len(l.games) {
		if time.Since(l.lastClick) <= doubleClickDuration {
			entry := l.games[l.selected]
			if entry.Password {
				l.showJoinGame = true
				game.setRoot(joinGameFrame)
				etk.SetFocus(l.joinGamePassword)
				l.joinGameLabel.SetText(gotext.Get("Join match: %s", entry.Name))
				l.joinGamePassword.Field.SetText("")
				l.joinGameID = entry.ID
				l.rebuildButtonsGrid()
			} else {
				l.c.Out <- []byte(fmt.Sprintf("j %d", entry.ID))
			}
			l.lastClick = time.Time{}
			return true
		}
	}

	l.lastClick = time.Now()
	l.selected = index
	return true
}
