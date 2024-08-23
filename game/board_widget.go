package game

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"time"

	"code.rocket9labs.com/tslocum/bgammon"
	"code.rocket9labs.com/tslocum/etk"
	"github.com/hajimehoshi/ebiten/v2"
)

type Label struct {
	*etk.Text
	active      bool
	activeColor color.RGBA
	lastActive  bool
	bg          *ebiten.Image
}

func NewLabel(c color.RGBA) *Label {
	l := &Label{
		Text:        etk.NewText(""),
		activeColor: c,
	}
	l.Text.SetFont(etk.Style.TextFont, etk.Scale(largeFontSize))
	l.Text.SetForeground(c)
	l.Text.SetScrollBarVisible(false)
	l.Text.SetSingleLine(true)
	l.Text.SetHorizontal(etk.AlignCenter)
	l.Text.SetVertical(etk.AlignCenter)
	l.Text.SetAutoResize(true)
	return l
}

func (l *Label) updateBackground() {
	if l.Text.Text() == "" {
		l.bg = nil
		return
	}

	r := l.Rect()
	if l.bg != nil {
		bounds := l.bg.Bounds()
		if bounds.Dx() != r.Dx() || bounds.Dy() != r.Dy() {
			l.bg = ebiten.NewImage(r.Dx(), r.Dy())
		}
	} else {
		l.bg = ebiten.NewImage(r.Dx(), r.Dy())
	}

	bgColor := color.RGBA{0, 0, 0, 20}
	borderSize := 2
	if l.active {
		l.bg.Fill(l.activeColor)

		bounds := l.bg.Bounds()
		l.bg.SubImage(image.Rect(0, 0, bounds.Dx(), bounds.Dy()).Inset(borderSize)).(*ebiten.Image).Fill(bgColor)
	} else {
		l.bg.Fill(bgColor)
	}

	l.lastActive = l.active
}

func (l *Label) SetRect(r image.Rectangle) {
	if r.Dx() == 0 || r.Dy() == 0 {
		l.bg = nil
		l.Text.SetRect(r)
		return
	}

	l.Text.SetRect(r)
	l.updateBackground()
}

func (l *Label) SetActive(active bool) {
	l.active = active
}

func (l *Label) SetText(t string) {
	r := l.Rect()
	if r.Empty() || l.Text.Text() == t {
		return
	}
	l.Text.SetText(t)
	l.updateBackground()
}

func (l *Label) Draw(screen *ebiten.Image) error {
	if l.bg == nil {
		return nil
	}
	if l.active != l.lastActive {
		l.updateBackground()
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(l.Rect().Min.X), float64(l.Rect().Min.Y))
	screen.DrawImage(l.bg, op)
	return l.Text.Draw(screen)
}

type DieButton struct {
	*etk.Button
	Value int8
}

func NewDieButton(value int8, onSelected func() error) *DieButton {
	return &DieButton{
		Button: etk.NewButton(" ", onSelected),
		Value:  value,
	}
}

func (b *DieButton) Draw(screen *ebiten.Image) error {
	dieFace := diceImage(b.Value)
	if dieFace == nil {
		return nil
	}

	err := b.Button.Draw(screen)
	if err != nil {
		return err
	}

	r := b.Rect()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(r.Min.X+(r.Dx()-int(game.Board.spaceWidth))/2), float64(r.Min.Y+(r.Dy()-int(game.Board.spaceWidth))/2))
	screen.DrawImage(dieFace, op)
	return nil
}

type BoardWidget struct {
	*etk.Box
}

func NewBoardWidget() *BoardWidget {
	return &BoardWidget{
		Box: etk.NewBox(),
	}
}

func (bw *BoardWidget) finishClick(cursor image.Point, double bool) {
	game.Board.Lock()
	game.Lock()
	game.Board.Unlock()
	defer game.Unlock()
	if game.Board.draggingSpace == -1 || len(game.Board.gameState.Available) == 0 {
		return
	}
	rolls := game.Board.gameState.DiceRolls()
	if len(rolls) == 0 {
		return
	}
	space := game.Board.spaceAt(cursor.X, cursor.Y)
	if space == -1 || space != game.Board.draggingSpace {
		return
	} else if !double {
		lowest := int8(math.MaxInt8)
		highest := int8(math.MinInt8)
		for _, roll := range rolls {
			if roll < lowest {
				lowest = roll
			}
			if roll > highest {
				highest = roll
			}
		}
		var roll int8
		if game.Board.draggingRightClick {
			roll = lowest
		} else {
			roll = highest
		}
		var useMove []int8
		for _, move := range game.Board.gameState.Available {
			if move[0] != space {
				continue
			}
			diff := bgammon.SpaceDiff(move[0], move[1], game.Board.gameState.Variant)
			haveRoll := diff == roll && game.Board.gameState.Game.HaveDiceRoll(move[0], move[1]) > 0
			if !haveRoll && (move[1] == bgammon.SpaceHomePlayer || move[1] == bgammon.SpaceHomeOpponent) {
				haveRoll = diff <= roll && game.Board.gameState.Game.HaveBearOffDiceRoll(diff) > 0
			}
			if haveRoll {
				useMove = move
				break
			}
		}
		if len(useMove) == 0 {
			return
		}
		playSoundEffect(effectMove)
		game.Unlock()
		game.Board.Lock()
		game.Board.movePiece(useMove[0], useMove[1], false)
		game.Board.gameState.AddLocalMove([]int8{useMove[0], useMove[1]})
		game.Board.gameState.Moves = append(game.Board.gameState.Moves, []int8{useMove[0], useMove[1]})
		game.Board.processState()
		game.Board.Unlock()
		game.Lock()
		game.Client.Out <- []byte(fmt.Sprintf("mv %d/%d", useMove[0], useMove[1]))
		return
	}

	var useMoves [][]int8
FINDMOVE:
	for _, move := range game.Board.gameState.Available {
		expanded := expandMoves([][]int8{{move[0], space}})
		gc := game.Board.gameState.Game.Copy(true)
		for _, m := range expanded {
			var found bool
			for _, m2 := range gc.LegalMoves(false) {
				if m2[0] == m[0] && m2[1] == m[1] {
					found = true
					break
				}
			}
			if !found {
				continue FINDMOVE
			}
			diff := bgammon.SpaceDiff(m[0], m[1], game.Board.gameState.Variant)
			haveRoll := game.Board.gameState.Game.HaveDiceRoll(m[0], m[1]) > 0
			if !haveRoll && (m[1] == bgammon.SpaceHomePlayer || m[1] == bgammon.SpaceHomeOpponent) {
				haveRoll = game.Board.gameState.Game.HaveBearOffDiceRoll(diff) > 0
			}
			if !haveRoll {
				continue FINDMOVE
			}
			ok, _ := gc.AddMoves([][]int8{m}, false)
			if !ok {
				continue FINDMOVE
			}
		}
		useMoves = expanded
		break
	}
	if len(useMoves) == 0 {
		return
	}
	game.Unlock()
	game.Board.Lock()
	for _, move := range useMoves {
		playSoundEffect(effectMove)
		game.Board.movePiece(move[0], move[1], false)
		game.Board.gameState.AddMoves([][]int8{{move[0], move[1]}}, true)
		game.Board.gameState.Moves = append(game.Board.gameState.Moves, []int8{move[0], move[1]})
		game.Board.processState()
	}
	game.Board.Unlock()
	game.Lock()
	for _, move := range useMoves {
		game.Client.Out <- []byte(fmt.Sprintf("mv %d/%d", move[0], move[1]))
	}
}

func (bw *BoardWidget) HandleMouse(cursor image.Point, pressed bool, clicked bool) (handled bool, err error) {
	if !pressed && !clicked && game.Board.dragging == nil {
		return false, nil
	}

	b := game.Board
	if b.Client == nil || !b.playerTurn() {
		return false, nil
	}

	cx, cy := cursor.X, cursor.Y

	if b.dragging == nil {
		if b.advancedMovement && clicked {
			if b.moving != nil {
				return false, nil
			}
			const doubleClickDuration = 250 * time.Millisecond
			space := b.spaceAt(cx, cy)
			if space != -1 {
				if time.Since(b.lastDragClick) >= doubleClickDuration {
					setTime := time.Now()
					b.draggingSpace = space
					b.draggingRightClick = ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
					b.lastDragClick = setTime
					go func() {
						time.Sleep(doubleClickDuration)
						if !b.lastDragClick.Equal(setTime) {
							return
						}
						bw.finishClick(cursor, false)
						b.lastDragClick = time.Now()
					}()
					return true, nil
				}
				go bw.finishClick(cursor, true)
				b.lastDragClick = time.Now()
				return true, nil
			}
			return false, nil
		}

		// TODO allow grabbing multiple pieces by grabbing further down the stack
		if !handled && b.playerTurn() && clicked && (b.lastDragClick.IsZero() || time.Since(b.lastDragClick) >= 50*time.Millisecond) {
			s, space := b.spriteAt(cx, cy)
			if s != nil && s.colorWhite == (b.flipBoard || b.gameState.PlayerNumber == 2) && space != bgammon.SpaceHomeOpponent && (game.Board.gameState.Variant == bgammon.VariantBackgammon || space != bgammon.SpaceHomePlayer || !game.Board.gameState.Player1.Entered) {
				b.startDrag(s, space, false)
				handled = true
			}
		}
	}

	x, y := cx, cy
	b.finishDrag(x, y, clicked)

	if b.dragging != nil {
		sprite := b.dragging
		sprite.x = x - (sprite.w / 2)
		sprite.y = y - (sprite.h / 2)
	}
	return handled, nil
}

type BoardMovingWidget struct {
	*etk.Box
}

func NewBoardMovingWidget() *BoardMovingWidget {
	return &BoardMovingWidget{
		Box: etk.NewBox(),
	}
}

func (w *BoardMovingWidget) Draw(screen *ebiten.Image) error {
	b := game.Board
	if b.moving != nil {
		b.drawSprite(screen, b.moving)
	}
	return nil
}

type BoardDraggedWidget struct {
	*etk.Box
}

func NewBoardDraggedWidget() *BoardDraggedWidget {
	return &BoardDraggedWidget{
		Box: etk.NewBox(),
	}
}

func (w *BoardDraggedWidget) Draw(screen *ebiten.Image) error {
	b := game.Board
	if b.dragging != nil {
		b.drawSprite(screen, b.dragging)
	}
	return nil
}
