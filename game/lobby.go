package game

import (
	"fmt"
	"image/color"
	"math"
	"sort"
	"strings"

	"code.rocketnine.space/tslocum/fibs"
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

	padding         float64
	entryH          float64
	buttonBarHeight int

	who []*fibs.WhoInfo

	touchIDs []ebiten.TouchID

	offset int

	selected int

	buffer      *ebiten.Image
	bufferDirty bool

	bufferButtons      *ebiten.Image
	bufferButtonsDirty bool

	op *ebiten.DrawImageOptions

	c *fibs.Client

	inviteUser   *fibs.WhoInfo
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

func (l *lobby) setWhoInfo(who []*fibs.WhoInfo) {
	l.who = who

	sort.Slice(l.who, func(i, j int) bool {
		if (l.who[i].Opponent != "") != (l.who[j].Opponent != "") {
			return l.who[j].Opponent != ""
		}
		if l.who[i].Ready != l.who[j].Ready {
			return l.who[i].Ready
		}
		if l.who[i].Rating != l.who[j].Rating {
			return l.who[i].Rating > l.who[j].Rating
		}
		return strings.ToLower(l.who[i].Username) < strings.ToLower(l.who[j].Username)
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
	for ly := 0; ly < 2; ly++ {
		for lx := 0; lx < l.w; lx++ {
			l.bufferButtons.Set(lx, ly, triangleA)
		}
	}

	buttons := l.getButtons()

	buttonWidth := l.w / len(buttons)
	for i, button := range buttons {
		// Draw border
		if i > 0 {
			for ly := 0; ly < l.buttonBarHeight; ly++ {
				for lx := buttonWidth * i; lx < (buttonWidth*i)+2; lx++ {
					l.bufferButtons.Set(lx, ly, triangleA)
				}
			}
		}
		bounds := text.BoundString(mplusNormalFont, button.label)

		labelColor := triangleA

		img := ebiten.NewImage(bounds.Dx()*2, bounds.Dy()*2)
		text.Draw(img, button.label, mplusNormalFont, 0, bounds.Dy(), labelColor)

		l.op.GeoM.Reset()
		l.op.GeoM.Translate(float64(buttonWidth*i)+float64((buttonWidth-bounds.Dx())/2), float64(l.buttonBarHeight-bounds.Dy())/2-float64(bounds.Dy()/2))
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
			bounds := text.BoundString(mplusNormalFont, label)
			labelColor := triangleA
			img := ebiten.NewImage(l.w-int(l.padding*2), int(l.entryH))
			text.Draw(img, label, mplusNormalFont, 4, bounds.Dy(), labelColor)
			l.op.GeoM.Reset()
			l.op.GeoM.Translate(padding, padding+float64(i*lineHeight))
			l.buffer.DrawImage(img, l.op)
		}
	} else {
		var img *ebiten.Image
		drawEntry := func(cx float64, cy float64, colA string, colB string, colC string, highlight bool) {
			boundsA := text.BoundString(mplusNormalFont, colA)
			boundsB := text.BoundString(mplusNormalFont, colB)
			boundsC := text.BoundString(mplusNormalFont, colC)
			y := (boundsA.Dy() + boundsB.Dy() + boundsC.Dy()) / 3 // TODO this isn't correct

			labelColor := triangleA
			if highlight {
				labelColor = color.RGBA{200, 200, 60, 255}
			}

			selectedBorderColor := triangleB

			img = ebiten.NewImage(l.w-int(l.padding*2), int(l.entryH))
			if highlight {
				for x := 0; x < l.w; x++ {
					img.Set(x, 0, selectedBorderColor)
					img.Set(x, int(l.entryH)-1, selectedBorderColor)
				}
				for by := 0; by < int(l.entryH)-1; by++ {
					img.Set(0, by, selectedBorderColor)
					img.Set(l.w-(int(l.padding)*2)-1, by, selectedBorderColor)
				}
			}

			text.Draw(img, colA, mplusNormalFont, 4, y+2, labelColor)
			text.Draw(img, colB, mplusNormalFont, int(250*ebiten.DeviceScaleFactor()), y+2, labelColor)
			text.Draw(img, colC, mplusNormalFont, int(500*ebiten.DeviceScaleFactor()), y+2, labelColor)

			l.op.GeoM.Reset()
			l.op.GeoM.Translate(cx, cy)
			l.buffer.DrawImage(img, l.op)
		}

		if len(l.who) == 0 {
			drawEntry(l.padding, l.padding, "Loading...", "Please wait...", "Loading...", false)
			return
		}

		for ly := -3; ly < -1; ly++ {
			for lx := 0; lx < l.w-int(l.padding*2); lx++ {
				l.buffer.Set(int(l.padding)+lx, int(l.padding)+int(l.entryH)+ly, triangleA)
			}
		}

		cx, cy := 0.0, 0.0 // Cursor
		drawEntry(cx+l.padding, cy+l.padding, "Username", "Rating (Experience)", "Status", false)
		cy += l.entryH
		i := 0
		for _, who := range l.who {
			if i >= l.offset {
				details := fmt.Sprintf("%d   (%d)", who.Rating, who.Experience)

				status := "In the lobby"
				if who.Opponent != "" {
					status = fmt.Sprintf("Playing versus %s", who.Opponent)
				} else if who.Ready {
					status = "Ready to play"
				}

				drawEntry(cx+l.padding, cy+l.padding, who.Username, details, status, i == l.selected)

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
		if len(l.who) > 1 {
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
	l.buttonBarHeight = int(200 * s)

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
			l.c.Out <- []byte(fmt.Sprintf("watch %s", l.who[l.selected].Username))
			viewBoard = true
		case 2:
			l.inviteUser = l.who[l.selected]
			l.invitePoints = 1
			l.bufferDirty = true
			l.bufferButtonsDirty = true
		case 3:
			l.c.Out <- []byte(fmt.Sprintf("join %s", l.who[l.selected].Username))
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
