package game

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"strconv"
	"time"

	"code.rocket9labs.com/tslocum/bgammon"
	"code.rocket9labs.com/tslocum/etk"
	"code.rocket9labs.com/tslocum/gotext"
)

type replayFrame struct {
	Game  *bgammon.Game
	Event []byte
}

func (g *Game) _handleReplay(gs *bgammon.GameState, line []byte, lineNumber int, sendEvent bool, haveLock bool) bool {
	if !haveLock {
		g.Lock()
		defer g.Unlock()
	}

	if !g.replay {
		return false
	}

	split := bytes.Split(line, []byte(" "))
	if len(split) < 2 {
		log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
		return false
	}
	switch {
	case bytes.Equal(split[0], []byte("bgammon-replay")):
		return true
	case bytes.Equal(split[0], []byte("i")):
		if len(split) < 10 {
			log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
			return false
		}

		if sendEvent {
			ev := &bgammon.EventJoined{
				GameID: 1,
			}
			ev.PlayerNumber = 1
			ev.Player = game.client.Username
			g.client.Events <- ev
		}

		variant, err := strconv.Atoi(string(split[9]))
		if err != nil || variant < 0 || variant > 2 {
			log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
			return false
		}

		*gs = bgammon.GameState{
			Game:         bgammon.NewGame(int8(variant)),
			PlayerNumber: 1,
			Spectating:   true,
		}
		gs.Turn = 0

		gs.Player1.Name, gs.Player2.Name = string(split[2]), string(split[3])

		points, err := strconv.Atoi(string(split[4]))
		if err != nil || gs.Points < 1 {
			log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
			return false
		}
		gs.Points = int8(points)

		points, err = strconv.Atoi(string(split[5]))
		if err != nil || gs.Player1.Points < 0 {
			log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
			return false
		}
		gs.Player1.Points = int8(points)

		points, err = strconv.Atoi(string(split[6]))
		if err != nil || gs.Player1.Points < 0 {
			log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
			return false
		}
		gs.Player2.Points = int8(points)

		if sendEvent {
			ev := &bgammon.EventBoard{
				GameState: bgammon.GameState{
					Game:         gs.Game.Copy(true),
					PlayerNumber: 1,
					Available:    gs.Available,
					Spectating:   true,
				},
			}
			g.client.Events <- ev
		}

		timestamp, err := strconv.ParseInt(string(split[1]), 10, 64)
		if err != nil {
			log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
			return false
		}
		gs.Started = timestamp
		gs.Ended = gs.Started
	case bytes.Equal(split[0], []byte("1")), bytes.Equal(split[0], []byte("2")):
		if len(split) < 2 || (!bytes.Equal(split[1], []byte("t")) && len(split) < 3) {
			log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
			return false
		}
		var player int8 = 1
		if bytes.Equal(split[0], []byte("2")) {
			player = 2
		}
		switch {
		case bytes.Equal(split[1], []byte("d")):
			if len(split) < 4 {
				log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
				return false
			}
			doubleValue, err := strconv.Atoi(string(split[2]))
			if err != nil || doubleValue < 2 {
				log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
				return false
			}
			resultValue, err := strconv.Atoi(string(split[3]))
			if err != nil || resultValue < 0 || resultValue > 1 {
				log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
				return false
			}
			resultText := "accepts"
			if resultValue == 0 {
				resultText = "declines"
			}
			ls(fmt.Sprintf("*** %s offers a double (%d points). %s %s.", gs.Player1.Name, doubleValue, gs.Player2.Name, resultText))
		case bytes.Equal(split[1], []byte("r")):
			rollSplit := bytes.Split(split[2], []byte("-"))
			if len(rollSplit) < 2 || len(rollSplit[0]) != 1 || len(rollSplit[1]) != 1 {
				log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
				return false
			}
			r1, err := strconv.Atoi(string(rollSplit[0]))
			if err != nil || r1 < 1 || r1 > 6 {
				log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
				return false
			}
			r2, err := strconv.Atoi(string(rollSplit[1]))
			if err != nil || r2 < 1 || r2 > 6 {
				log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
				return false
			}
			var r3 int
			if len(rollSplit) > 2 {
				r3, err = strconv.Atoi(string(rollSplit[2]))
				if err != nil || r3 < 1 || r3 > 6 {
					log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
					return false
				}
			}

			gs.Moves = nil
			if gs.Turn == 0 {
				gs.Turn = player
				gs.Available = nil
				gs.Moves = nil
				if sendEvent {
					ev := &bgammon.EventBoard{
						GameState: bgammon.GameState{
							Game:         gs.Game.Copy(true),
							PlayerNumber: 1,
							Available:    gs.Available,
							Spectating:   true,
						},
					}
					g.client.Events <- ev
				}
			}

			playerName := gs.Player1.Name
			if player == 2 {
				playerName = gs.Player2.Name
			}

			if sendEvent {
				ev := &bgammon.EventRolled{
					Roll1: int8(r1),
					Roll2: int8(r2),
					Roll3: int8(r3),
				}
				ev.Player = playerName
				g.client.Events <- ev
			}

			gs.Roll1, gs.Roll2, gs.Roll3 = int8(r1), int8(r2), int8(r3)
			gs.Turn = player
			gs.Available = gs.LegalMoves(true)
			gs.Moves = nil

			if sendEvent {
				ev := &bgammon.EventBoard{
					GameState: bgammon.GameState{
						Game:         gs.Game.Copy(true),
						PlayerNumber: 1,
						Available:    gs.Available,
						Spectating:   true,
					},
				}
				g.client.Events <- ev
			}

			if len(split) == 3 {
				return true
			}
			for _, move := range split[3:] {
				moveSplit := bytes.Split(move, []byte("/"))
				if len(moveSplit) != 2 || len(moveSplit[0]) > 3 || len(moveSplit[1]) > 3 {
					log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
					return false
				}
				from, to := bgammon.ParseSpace(string(moveSplit[0])), bgammon.ParseSpace(string(moveSplit[1]))
				if from < 0 || to < 0 || from == to {
					log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
					return false
				} else if from == bgammon.SpaceBarPlayer && player == 2 {
					from = bgammon.SpaceBarOpponent
				} else if from == bgammon.SpaceHomePlayer && player == 2 {
					from = bgammon.SpaceHomeOpponent
				}
				if to == bgammon.SpaceHomePlayer && player == 2 {
					to = bgammon.SpaceHomeOpponent
				}
				if sendEvent {
					ev := &bgammon.EventMoved{
						Moves: [][]int8{{from, to}},
					}
					ev.Player = playerName
					g.client.Events <- ev
				}
				ok, _ := gs.AddMoves([][]int8{{from, to}}, false)
				if !ok {
					log.Panicf("failed to move checkers during replay from %d to %d", from, to)
				}
			}

			if gs.Winner != 0 {
				playerBar := bgammon.SpaceBarPlayer
				opponentHome := bgammon.SpaceHomeOpponent
				var opponent int8 = 2
				if player == 2 {
					playerBar = bgammon.SpaceBarOpponent
					opponentHome = bgammon.SpaceHomePlayer
					opponent = 1
				}

				backgammon := bgammon.PlayerCheckers(gs.Board[playerBar], opponent) != 0
				if !backgammon {
					homeStart, homeEnd := bgammon.HomeRange(gs.Winner, gs.Variant)
					bgammon.IterateSpaces(homeStart, homeEnd, gs.Variant, func(space int8, spaceCount int8) {
						if bgammon.PlayerCheckers(gs.Board[space], opponent) != 0 {
							backgammon = true
						}
					})
				}

				var winPoints int8
				switch gs.Variant {
				case bgammon.VariantBackgammon:
					if backgammon {
						winPoints = 3 // Award backgammon.
					} else if gs.Board[opponentHome] == 0 {
						winPoints = 2 // Award gammon.
					} else {
						winPoints = 1
					}
				case bgammon.VariantAceyDeucey:
					for space := int8(0); space < bgammon.BoardSpaces; space++ {
						if (space == bgammon.SpaceHomePlayer || space == bgammon.SpaceHomeOpponent) && ((opponent == 1 && gs.Player1.Entered) || (opponent == 2 && gs.Player2.Entered)) {
							continue
						}
						winPoints += bgammon.PlayerCheckers(gs.Board[space], opponent)
					}
				case bgammon.VariantTabula:
					winPoints = 1
				}

				if sendEvent {
					ev := &bgammon.EventWin{
						Points: winPoints * gs.DoubleValue,
					}
					ev.Player = playerName
					g.client.Events <- ev
				}
			}

			if sendEvent {
				ev := &bgammon.EventBoard{
					GameState: bgammon.GameState{
						Game:         gs.Game.Copy(true),
						PlayerNumber: 1,
						Available:    gs.Available,
						Spectating:   true,
					},
				}
				g.client.Events <- ev
			}
		case bytes.Equal(split[1], []byte("t")):
			playerName := gs.Player1.Name
			if player == 2 {
				playerName = gs.Player2.Name
			}
			if sendEvent {
				ev := &bgammon.EventNotice{
					Message: gotext.Get("%s resigned.", playerName),
				}
				g.client.Events <- ev
			}
		default:
			log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
			return false
		}
	default:
		log.Printf("warning: failed to read replay: failed to parse line %d", lineNumber)
		return false
	}
	return true
}

func (g *Game) showReplayFrame(replayFrame int, showInfo bool) {
	if !g.replay || replayFrame < 0 || replayFrame >= len(g.replayFrames) {
		return
	}

	if g.needLayoutBoard {
		g.layoutBoard()
	}

	if replayFrame == 0 && showInfo {
		g.board.recreateUIGrid()
	}

	g.replayFrame = replayFrame
	frame := g.replayFrames[replayFrame]

	ev := &bgammon.EventBoard{
		GameState: bgammon.GameState{
			Game:         frame.Game.Copy(true),
			PlayerNumber: 1,
			Available:    frame.Game.LegalMoves(true),
			Spectating:   true,
		},
	}
	g.client.Events <- ev

	if replayFrame == 0 && showInfo {
		ls(fmt.Sprintf("*** "+gotext.Get("Replaying %s vs. %s", "%s", "%s")+" (%s)", frame.Game.Player2.Name, frame.Game.Player1.Name, time.Unix(frame.Game.Started, 0).Format("2006-01-02 15:04")))
	}
}

func (g *Game) HandleReplay(replay []byte) {
	g.Lock()
	if g.replay {
		g.Unlock()
		return
	}
	g.replay = true
	g.replayFrame = 0
	g.replayFrames = g.replayFrames[:0]
	g.replayData = replay
	g.Unlock()

	g.board.rematchButton.SetVisible(false)

	if !g.loggedIn {
		go g.playOffline()
		time.Sleep(500 * time.Millisecond)
	}

	gs := &bgammon.GameState{
		Game:         bgammon.NewGame(bgammon.VariantBackgammon),
		PlayerNumber: 1,
		Spectating:   true,
	}

	g.board.replayList.Clear()
	var listY int

	var lineNumber int
	scanner := bufio.NewScanner(bytes.NewReader(replay))
	for scanner.Scan() {
		if !bytes.HasPrefix(scanner.Bytes(), []byte("bgammon-reply")) && !bytes.HasPrefix(scanner.Bytes(), []byte("i ")) {
			g.replayFrames = append(g.replayFrames, &replayFrame{
				Game:  gs.Game.Copy(true),
				Event: scanner.Bytes(),
			})
		}
		lineNumber++
		if !g._handleReplay(gs, scanner.Bytes(), lineNumber, false, false) {
			return
		}

		if bytes.HasPrefix(scanner.Bytes(), []byte("1 r ")) || bytes.HasPrefix(scanner.Bytes(), []byte("2 r ")) {
			player := 1
			if bytes.HasPrefix(scanner.Bytes(), []byte("2")) {
				player = 2
			}

			label := scanner.Bytes()[4:]
			split := bytes.SplitN(label, []byte(" "), 2)
			var roll []byte
			var move []byte
			l := len(split)
			if l > 0 {
				roll = split[0]
				if l == 2 {
					move = split[1]
				}
			}

			var x int
			if player == 1 {
				x = 1
			}
			frame := len(g.replayFrames)
			btn := etk.NewButton("", func() error {
				if !game.board.replayAuto.IsZero() {
					game.board.replayAuto = time.Time{}
					game.board.replayPauseButton.SetText("|>")
				}
				g.showReplayFrame(frame, true)
				return nil
			})
			rollWidth := 85
			if gs.Variant == bgammon.VariantTabula {
				rollWidth = 110
			}
			grid := etk.NewGrid()
			grid.SetColumnSizes(rollWidth, -1)
			rollLabel := etk.NewText(string(roll))
			rollLabel.SetPadding(etk.Scale(etk.Style.ButtonBorderSize + 2))
			rollLabel.SetVertical(etk.AlignCenter)
			rollLabel.SetAutoResize(true)
			rollLabel.SetForeground(etk.Style.ButtonTextColor)
			moveLabel := etk.NewText(string(move))
			moveLabel.SetPadding(etk.Scale(etk.Style.ButtonBorderSize + 2))
			moveLabel.SetVertical(etk.AlignCenter)
			moveLabel.SetAutoResize(true)
			moveLabel.SetForeground(etk.Style.ButtonTextColor)
			grid.AddChildAt(&etk.WithoutMouse{Widget: rollLabel}, 0, 0, 1, 1)
			grid.AddChildAt(&etk.WithoutMouse{Widget: moveLabel}, 1, 0, 1, 1)
			btn.AddChild(&etk.WithoutMouse{Widget: grid})
			g.board.replayList.AddChildAt(btn, x, listY)

			if player == 1 {
				listY++
			}
		}
	}
	if scanner.Err() != nil {
		log.Printf("warning: failed to read replay: %s", scanner.Err())
		return
	}

	if len(g.replayFrames) < 2 {
		log.Printf("warning: failed to read replay: no frames were loaded")
		return
	}

	g.replayFrames = append(g.replayFrames, &replayFrame{
		Game:  gs.Game.Copy(true),
		Event: nil,
	})

	g.Lock()
	g.showReplayFrame(0, true)
	g.Unlock()
}
