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
	"code.rocket9labs.com/tslocum/gotext"
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
	lobbyButtonHistoryCancel = iota
	lobbyButtonHistoryDownload
	lobbyButtonHistoryView
)

var (
	lobbyIndentA = 200
	lobbyIndentB = 350
)

func init() {
	if smallScreen {
		lobbyIndentA, lobbyIndentB = lobbyIndentA/3, lobbyIndentB/3
	}
}

var mainButtons []string
var mainShortButtons []string
var createButtons []string
var cancelJoinButtons []string
var historyButtons []string

type lobby struct {
	buttonBarHeight int

	loaded bool
	games  []bgammon.GameListing

	lastClick time.Time

	selected int

	c *Client

	showCreateGame           bool
	createGameName           *Input
	createGamePoints         *Input
	createGamePassword       *Input
	createGameAceyCheckbox   *etk.Checkbox
	createGameTabulaCheckbox *etk.Checkbox

	showJoinGame     bool
	joinGameID       int
	joinGameLabel    *etk.Text
	joinGamePassword *Input

	joiningGameID       int
	joiningGamePassword string
	joiningGameShown    bool

	showHistory                         bool
	historySelected                     int
	historyLastClick                    time.Time
	historyMatches                      []*bgammon.HistoryMatch
	historyUsername                     *Input
	historyList                         *etk.List
	historyPage                         int
	historyPages                        int
	historyPageButton                   *etk.Button
	historyRatingCasualBackgammonSingle *etk.Text
	historyRatingCasualBackgammonMulti  *etk.Text
	historyRatingCasualAceySingle       *etk.Text
	historyRatingCasualAceyMulti        *etk.Text
	historyRatingCasualTabulaSingle     *etk.Text
	historyRatingCasualTabulaMulti      *etk.Text

	historyPageDialog      *etk.Grid
	historyPageDialogInput *NumericInput

	availableMatchesList *etk.List

	historyButton *etk.Button
	buttonsGrid   *etk.Grid
}

func NewLobby() *lobby {
	mainButtons = []string{
		gotext.Get("Refresh matches"),
		gotext.Get("Create match"),
		gotext.Get("Join match"),
	}

	mainShortButtons = []string{
		gotext.Get("Refresh"),
		gotext.Get("Create"),
		gotext.Get("Join"),
	}

	createButtons = []string{
		gotext.Get("Cancel"),
		gotext.Get("Create match"),
	}

	cancelJoinButtons = []string{
		gotext.Get("Cancel"),
		gotext.Get("Join match"),
	}

	historyButtons = []string{
		gotext.Get("Return"),
		gotext.Get("Download replay"),
		gotext.Get("View replay"),
	}

	l := &lobby{
		buttonsGrid: etk.NewGrid(),
	}

	loadingText := newCenteredText(gotext.Get("Loading..."))
	if smallScreen {
		loadingText.SetFont(etk.Style.TextFont, etk.Scale(mediumFontSize))
	}

	indentA, indentB := etk.Scale(lobbyIndentA), etk.Scale(lobbyIndentB)

	matchList := etk.NewList(game.itemHeight(), l.selectMatch)
	matchList.SetSelectionMode(etk.SelectRow)
	matchList.SetColumnSizes(indentA, indentB-indentA, indentB-indentA, -1)
	matchList.SetHighlightColor(color.RGBA{79, 55, 30, 255})
	matchList.AddChildAt(loadingText, 0, 0)
	l.availableMatchesList = matchList
	return l
}

func (l *lobby) toggleVariantAcey() error {
	l.createGameTabulaCheckbox.SetSelected(false)
	return nil
}

func (l *lobby) toggleVariantTabula() error {
	l.createGameAceyCheckbox.SetSelected(false)
	return nil
}

func (l *lobby) gameListingsEqual(a bgammon.GameListing, b bgammon.GameListing) bool {
	return a.ID == b.ID && a.Password == b.Password && a.Points == b.Points && a.Players == b.Players && a.Rating == b.Rating && a.Name == b.Name
}

func (l *lobby) sortGameListings(games []bgammon.GameListing) {
	sort.Slice(games, func(i, j int) bool {
		const (
			aceyPrefix   = "(Acey-deucey)"
			tabulaPrefix = "(Tabula)"
			botPrefix    = "BOT_"
		)
		a, b := games[i], games[j]
		switch {
		case (a.Password) != (b.Password):
			return !a.Password
		case (a.Players) != (b.Players):
			return a.Players < b.Players
		case strings.HasPrefix(a.Name, tabulaPrefix) != strings.HasPrefix(b.Name, tabulaPrefix):
			return strings.HasPrefix(b.Name, tabulaPrefix)
		case strings.HasPrefix(a.Name, aceyPrefix) != strings.HasPrefix(b.Name, aceyPrefix):
			return strings.HasPrefix(b.Name, aceyPrefix)
		case strings.HasPrefix(a.Name, botPrefix) != strings.HasPrefix(b.Name, botPrefix):
			return strings.HasPrefix(b.Name, botPrefix)
		default:
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}
	})
}

func (l *lobby) setGameList(games []bgammon.GameListing) {
	l.sortGameListings(games)
	if l.loaded && len(games) == len(l.games) {
		var changed bool
		for i := range games {
			if !l.gameListingsEqual(games[i], l.games[i]) {
				changed = true
				break
			}
		}
		if !changed {
			return
		}
	}
	l.games = games
	l.loaded = true

	newLabel := func(label string) *etk.Text {
		txt := etk.NewText(label)
		txt.SetFollow(false)
		txt.SetScrollBarVisible(false)
		txt.SetWordWrap(false)
		txt.SetVertical(etk.AlignCenter)
		if smallScreen {
			txt.SetFont(etk.Style.TextFont, etk.Scale(mediumFontSize))
		}
		return txt
	}

	_, lastSelection := l.availableMatchesList.SelectedItem()

	var status, rating string
	l.availableMatchesList.Clear()
	for i, entry := range l.games {
		if entry.Password {
			status = gotext.Get("Private")
		} else if entry.Players == 2 {
			status = gotext.Get("Started")
		} else {
			status = gotext.Get("Available")
		}
		if entry.Rating == 0 {
			rating = gotext.Get("None")
		} else {
			rating = fmt.Sprintf("%d", entry.Rating)
		}
		nameLabel := newLabel(entry.Name)
		if smallScreen {
			nameLabel.SetWordWrap(true)
		}
		l.availableMatchesList.AddChildAt(newLabel(status), 0, i)
		l.availableMatchesList.AddChildAt(newLabel(rating), 1, i)
		l.availableMatchesList.AddChildAt(newLabel(fmt.Sprintf("%d", entry.Points)), 2, i)
		l.availableMatchesList.AddChildAt(nameLabel, 3, i)
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

func (l *lobby) getButtons() []string {
	if l.showCreateGame {
		return createButtons
	} else if l.showJoinGame {
		return cancelJoinButtons
	} else if l.showHistory {
		return historyButtons
	} else if smallScreen && game.portraitView() {
		return mainShortButtons
	}
	return mainButtons
}

func (l *lobby) confirmCreateGame() {
	go hideKeyboard()
	typeAndPassword := "public"
	if len(strings.TrimSpace(game.lobby.createGamePassword.Text())) > 0 {
		typeAndPassword = fmt.Sprintf("private %s", strings.ReplaceAll(game.lobby.createGamePassword.Text(), " ", "_"))
	}
	points, err := strconv.Atoi(game.lobby.createGamePoints.Text())
	if err != nil {
		points = 1
	}
	var variant int8
	if game.lobby.createGameAceyCheckbox.Selected() {
		variant = bgammon.VariantAceyDeucey
	} else if game.lobby.createGameTabulaCheckbox.Selected() {
		variant = bgammon.VariantTabula
	}
	l.c.Out <- []byte(fmt.Sprintf("c %s %d %d %s", typeAndPassword, points, variant, game.lobby.createGameName.Text()))
}

func (l *lobby) confirmJoinGame() {
	go hideKeyboard()
	l.joiningGameID = l.joinGameID
	l.joiningGamePassword = l.joinGamePassword.Text()
	l.rebuildButtonsGrid()
	scheduleFrame()
}

func (l *lobby) selectButton(buttonIndex int) func() error {
	return func() error {
		if l.showCreateGame {
			switch buttonIndex {
			case lobbyButtonCreateCancel:
				game.lobby.showCreateGame = false
				game.lobby.createGameName.SetText("")
				game.lobby.createGamePassword.SetText("")
				l.rebuildButtonsGrid()
				game.setRoot(listGamesFrame)
				etk.SetFocus(nil)
			case lobbyButtonCreateConfirm:
				l.confirmCreateGame()
			}
			return nil
		} else if l.showJoinGame {
			if buttonIndex == 0 { // Cancel
				l.showJoinGame = false
				l.rebuildButtonsGrid()
				if viewBoard {
					game.setRoot(game.board.frame)
				} else {
					game.setRoot(listGamesFrame)
					etk.SetFocus(nil)
				}
			} else {
				l.confirmJoinGame()
			}
			return nil
		} else if l.showHistory {
			switch buttonIndex {
			case lobbyButtonCreateCancel:
				l.showHistory = false
				l.rebuildButtonsGrid()
				game.setRoot(listGamesFrame)
				etk.SetFocus(nil)
			case lobbyButtonHistoryDownload:
				if game.downloadReplay != 0 {
					return nil
				}
				_, selected := l.historyList.SelectedItem()
				if selected >= 0 && selected < len(l.historyMatches) {
					match := l.historyMatches[selected]
					game.downloadReplay = match.ID
					game.client.Out <- []byte(fmt.Sprintf("replay %d", match.ID))
				}
			case lobbyButtonHistoryView:
				_, selected := l.historyList.SelectedItem()
				if selected >= 0 && selected < len(l.historyMatches) {
					match := l.historyMatches[selected]
					game.client.Out <- []byte(fmt.Sprintf("replay %d", match.ID))
				}
			}
			return nil
		}
		switch buttonIndex {
		case lobbyButtonRefresh:
			l.c.Out <- []byte("ls")
		case lobbyButtonCreate:
			if l.c.Username == "" {
				return nil
			} else if l.c.local {
				ls("*** Failed to create match: Offline human versus human matches are not supported yet. Stay tuned.")
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
			l.createGameName.SetText(namePlural + " match")
			l.createGamePoints.SetText("1")
			l.createGamePassword.SetText("")
			l.rebuildButtonsGrid()
			scheduleFrame()
		case lobbyButtonJoin:
			if l.selected < 0 || l.selected >= len(l.games) {
				return nil
			}

			if l.games[l.selected].Password {
				l.showJoinGame = true
				game.setRoot(joinGameFrame)
				etk.SetFocus(l.joinGamePassword)
				l.joinGameLabel.SetText(gotext.Get("Join match: %s", l.games[l.selected].Name))
				l.joinGamePassword.SetText("")
				l.joinGameID = l.games[l.selected].ID
				l.rebuildButtonsGrid()
			} else {
				l.joiningGameID = l.games[l.selected].ID
				l.joiningGamePassword = ""
				l.rebuildButtonsGrid()
				scheduleFrame()
			}
		}
		return nil
	}
}

func (l *lobby) rebuildButtonsGrid() {
	r := l.buttonsGrid.Rect()
	l.buttonsGrid.Clear()

	buttons := l.getButtons()
	if l.joiningGameID != 0 {
		btns := make([]string, len(buttons))
		for i, label := range buttons {
			if label == gotext.Get("Join match") || label == gotext.Get("Join") {
				btns[i] = gotext.Get("Joining...")
			} else {
				btns[i] = label
			}
		}
		buttons = btns
	}
	for i, label := range buttons {
		l.buttonsGrid.AddChildAt(etk.NewButton(label, l.selectButton(i)), i, 0, 1, 1)
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
				l.joinGamePassword.SetText("")
				l.joinGameID = entry.ID
				l.rebuildButtonsGrid()
			} else {
				l.joiningGameID = entry.ID
				l.joiningGamePassword = ""
				l.rebuildButtonsGrid()
				scheduleFrame()
			}
			l.lastClick = time.Time{}
			return true
		}
	}

	l.lastClick = time.Now()
	l.selected = index
	return true
}

func (l *lobby) selectHistory(index int) bool {
	if index < 0 || index >= len(l.historyMatches) {
		return false
	}
	const doubleClickDuration = 200 * time.Millisecond
	if l.historySelected == index && l.historySelected >= 0 && l.historySelected < len(l.historyMatches) {
		if time.Since(l.historyLastClick) <= doubleClickDuration {
			match := l.historyMatches[l.historySelected]
			l.c.Out <- []byte(fmt.Sprintf("replay %d", match.ID))
			l.historyLastClick = time.Time{}
			return true
		}
	}

	l.historyLastClick = time.Now()
	l.historySelected = index
	return true
}
