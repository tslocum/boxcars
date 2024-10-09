package game

import (
	"image"
	"image/color"
	"time"

	"code.rocket9labs.com/tslocum/etk"
	"code.rocket9labs.com/tslocum/gotext"
)

type tutorialWidget struct {
	*etk.Frame
	grid      *etk.Grid
	page      int
	lastClick time.Time
}

func NewTutorialWidget() *tutorialWidget {
	w := &tutorialWidget{
		Frame: etk.NewFrame(),
		grid:  etk.NewGrid(),
	}
	w.Frame.SetPositionChildren(true)
	w.Frame.AddChild(w.grid)

	w.setPage(0)
	return w
}

func (w *tutorialWidget) nextPage() error {
	w.setPage(w.page + 1)
	return nil
}

func (w *tutorialWidget) hide() error {
	game.lobby.showCreateGame = false
	game.setRoot(listGamesFrame)
	etk.SetFocus(game.lobby.availableMatchesList)
	setViewBoard(false)
	game.board.gameState.PlayerNumber = 0
	game.savedUsername = "a"
	w.grid.Clear()
	return nil
}

func (w *tutorialWidget) newTutorialBox() *tutorialBox {
	return &tutorialBox{
		Box:     etk.NewBox(),
		handler: w.hide,
	}
}

func (w *tutorialWidget) newDialog(title string, message string) *Dialog {
	titleLabel := resizeText(title)
	titleLabel.SetHorizontal(etk.AlignCenter)
	titleLabel.SetVertical(etk.AlignCenter)

	messageLabel := etk.NewText(message)

	grid := etk.NewGrid()
	grid.SetBackground(color.RGBA{40, 24, 9, 255})
	grid.SetColumnSizes(20, -1, -1, 20)
	grid.SetRowSizes(72, 20, -1, etk.Scale(20), etk.Scale(baseButtonHeight))
	grid.AddChildAt(titleLabel, 1, 0, 2, 1)
	grid.AddChildAt(messageLabel, 1, 2, 2, 1)
	grid.AddChildAt(etk.NewBox(), 1, 3, 1, 1)
	columns := 2
	if w.page == 5 {
		columns = 4
	}
	grid.AddChildAt(etk.NewButton(gotext.Get("Dismiss"), w.hide), 0, 4, columns, 1)
	if w.page < 5 {
		grid.AddChildAt(etk.NewButton(gotext.Get("Next"), w.nextPage), 2, 4, 2, 1)
	}
	return &Dialog{grid}
}

func (w *tutorialWidget) setPage(page int) {
	if time.Since(w.lastClick) < 250*time.Millisecond {
		return
	}
	w.lastClick = time.Now()
	w.page = page

	var title string
	var message string
	switch w.page {
	case 0:
		title = gotext.Get("Tutorial")
		message = gotext.Get("Welcome to the guided tutorial. This program (Boxcars) is the official bgammon.org client. bgammon.org is a free and open source backgammon server.")
	case 1:
		title = gotext.Get("Matches List")
		message = gotext.Get("This screen lists the matches that are currently available. A few bots are always available to play against. You can also spectate ongoing public matches.")
	case 2:
		game.lobby.showCreateGame = true
		game.setRoot(createGameFrame)
		etk.SetFocus(game.lobby.createGameName)
		title = gotext.Get("Create Match")
		message = gotext.Get("Create a match if you would like to play against someone else. Backgammon and several of its variants are supported.")
	case 3:
		game.lobby.showCreateGame = false
		game.setRoot(listGamesFrame)
		etk.SetFocus(game.lobby.availableMatchesList)
		game.board.gameState.PlayerNumber = 1
		if game.needLayoutBoard {
			game.layoutBoard()
		}
		setViewBoard(true)
		title = gotext.Get("Board")
		message = gotext.Get("You have the black checkers. You can move a checker by either clicking it or dragging it.")
	case 4:
		title = gotext.Get("Bearing Off")
		message = gotext.Get("Double click a checker to bear it off. Bear off all 15 checkers to win.")
	case 5:
		title = gotext.Get("Good Luck, Have Fun")
		message = gotext.Get("This concludes the tutorial. Learn how to play backgammon at bgammon.org/faq and share your feedback at bgammon.org/community")
	case 6:
		w.hide()
		return
	}

	w.grid.Clear()
	w.grid.AddChildAt(w.newTutorialBox(), 0, 0, 6, 1)
	w.grid.AddChildAt(w.newTutorialBox(), 0, 1, 1, 2)
	w.grid.AddChildAt(w.newDialog(title, message), 1, 1, 4, 2)
	w.grid.AddChildAt(w.newTutorialBox(), 5, 1, 1, 2)
	w.grid.AddChildAt(w.newTutorialBox(), 0, 3, 6, 1)
}

type tutorialBox struct {
	*etk.Box
	handler func() error
}

func (b *tutorialBox) HandleMouse(cursor image.Point, pressed bool, clicked bool) (handled bool, err error) {
	if !cursor.In(b.Rect()) || !clicked {
		return false, nil
	}
	b.handler()
	return true, nil
}
