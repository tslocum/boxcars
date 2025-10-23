package game

import (
	"image"
	"time"

	"codeberg.org/tslocum/etk"
	"codeberg.org/tslocum/gotext"
)

type tutorialWidget struct {
	*etk.Frame
	outerBox  *tutorialBox
	content   *etk.Frame
	page      int
	lastClick time.Time
}

func NewTutorialWidget() *tutorialWidget {
	w := &tutorialWidget{
		Frame:   etk.NewFrame(),
		content: etk.NewFrame(),
	}
	w.outerBox = w.newTutorialBox()

	w.Frame.SetPositionChildren(true)
	w.Frame.AddChild(w.outerBox)
	w.Frame.AddChild(w.content)

	w.content.SetPositionChildren(true)
	w.content.SetHorizontal(etk.AlignCenter)
	w.content.SetVertical(etk.AlignCenter)

	w.setPage(0)
	return w
}

func (w *tutorialWidget) SetRect(r image.Rectangle) {
	maxWidth, maxHeight := etk.Scale(800), etk.Scale(400)
	if smallScreen {
		maxHeight = maxHeight / 3 * 2
	}
	if maxWidth > game.screenW/3*2 {
		maxWidth = game.screenW / 3 * 2
	}
	if maxHeight > game.screenH/3*2 {
		maxHeight = game.screenH / 3 * 2
	}
	w.content.SetMaxWidth(maxWidth)
	w.content.SetMaxHeight(maxHeight)

	w.Frame.SetRect(r)
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
	w.Clear()
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
	messageLabel.SetFont(etk.Style.TextFont, etk.Scale(mediumLargeFontSize))

	grid := etk.NewGrid()
	grid.SetColumnSizes(20, -1, -1, 20)
	grid.SetRowSizes(72, 20, -1, etk.Scale(20), etk.Scale(baseButtonHeight))
	grid.AddChildAt(titleLabel, 1, 0, 2, 1)
	grid.AddChildAt(messageLabel, 1, 2, 2, 1)
	grid.AddChildAt(etk.NewBox(), 1, 3, 1, 1)

	cols := 2
	if w.page == 5 {
		cols = 1
	}
	d := newDialog(etk.NewGrid())
	d.SetRowSizes(-1, etk.Scale(baseButtonHeight))
	d.AddChildAt(&withDialogBorder{grid, image.Rectangle{}}, 0, 0, cols, 1)
	d.AddChildAt(etk.NewButton(gotext.Get("Dismiss"), w.hide), 0, 1, 1, 1)
	if w.page < 5 {
		d.AddChildAt(etk.NewButton(gotext.Get("Next"), w.nextPage), 1, 1, 1, 1)
	}
	return d
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
		if isSteamDeck() {
			message = gotext.Get("You have the black checkers. You can move a checker by tapping or dragging the touchscreen.")
		} else {
			message = gotext.Get("You have the black checkers. You can move a checker by either clicking it or dragging it.")
		}
	case 4:
		title = gotext.Get("Bearing Off")
		if isSteamDeck() {
			message = gotext.Get("Drag a checker off the board to bear it off. Bear off all 15 checkers to win.")
		} else {
			message = gotext.Get("Double click a checker to bear it off. Bear off all 15 checkers to win.")
		}
	case 5:
		title = gotext.Get("Good Luck, Have Fun")
		message = gotext.Get("This concludes the tutorial. Learn how to play backgammon at bgammon.org/faq and share your feedback at bgammon.org/community")
	case 6:
		w.hide()
		return
	}

	w.content.Clear()
	w.content.AddChild(w.newDialog(title, message))
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
