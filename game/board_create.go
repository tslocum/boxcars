package game

import (
	"image/color"
	"log"

	"code.rocket9labs.com/tslocum/bgammon"
	"code.rocket9labs.com/tslocum/etk"
	"code.rocket9labs.com/tslocum/gotext"
)

func (b *board) createRatingLabels() {
	for i := 0; i < 2; i++ {
		o := b.opponentRatingLabel
		p := b.playerRatingLabel
		if i == 1 {
			o = b.opponentRatingShadowLabel
			p = b.playerRatingShadowLabel
		}

		o.SetHorizontal(etk.AlignCenter)
		o.SetVertical(etk.AlignStart)
		o.SetScrollBarVisible(false)
		o.SetFont(etk.Style.TextFont, etk.Scale(mediumFontSize))
		o.SetAutoResize(true)

		p.SetHorizontal(etk.AlignCenter)
		p.SetVertical(etk.AlignEnd)
		p.SetScrollBarVisible(false)
		p.SetFont(etk.Style.TextFont, etk.Scale(mediumFontSize))
		p.SetAutoResize(true)
	}
}

func (b *board) createForcedLabels() {
	padding := 15
	b.opponentForcedLabel.SetPadding(padding)
	b.opponentForcedLabel.SetHorizontal(etk.AlignCenter)
	b.opponentForcedLabel.SetVertical(etk.AlignEnd)
	b.opponentForcedLabel.SetScrollBarVisible(false)
	b.opponentForcedLabel.SetVisible(false)
	b.playerForcedLabel.SetPadding(padding)
	b.playerForcedLabel.SetHorizontal(etk.AlignCenter)
	b.playerForcedLabel.SetVertical(etk.AlignEnd)
	b.playerForcedLabel.SetScrollBarVisible(false)
	b.playerForcedLabel.SetVisible(false)
}

func (b *board) createMenu() {
	grid := b.menuGrid
	grid.AddChildAt(etk.NewButton(gotext.Get("Return"), b.hideMenu), 0, 0, 1, 1)
	grid.AddChildAt(etk.NewButton(gotext.Get("Settings"), b.showSettings), 1, 0, 1, 1)
	grid.AddChildAt(etk.NewButton(gotext.Get("Leave"), b.leaveMatch), 2, 0, 1, 1)
	grid.SetVisible(false)
}

func (b *board) createChangePasswordDialog() {
	headerLabel := etk.NewText(gotext.Get("Change password"))
	headerLabel.SetHorizontal(etk.AlignCenter)
	headerLabel.SetVertical(etk.AlignCenter)

	oldLabel := &ClickableText{
		Text: resizeText(gotext.Get("Current")),
		onSelected: func() {
			b.highlightCheckbox.SetSelected(!b.highlightCheckbox.Selected())
			b.toggleHighlightCheckbox()
		},
	}
	oldLabel.SetVertical(etk.AlignCenter)

	b.changePasswordOld = &Input{etk.NewInput("", func(text string) (handled bool) {
		b.selectChangePassword()
		return false
	})}
	b.changePasswordOld.SetBackground(frameColor)
	centerInput(b.changePasswordOld)
	b.changePasswordOld.SetMask('*')

	newLabel := &ClickableText{
		Text: resizeText(gotext.Get("New")),
		onSelected: func() {
			b.showPipCountCheckbox.SetSelected(!b.showPipCountCheckbox.Selected())
			b.togglePipCountCheckbox()
		},
	}
	newLabel.SetVertical(etk.AlignCenter)

	b.changePasswordNew = &Input{etk.NewInput("", func(text string) (handled bool) {
		b.selectChangePassword()
		return false
	})}
	b.changePasswordNew.SetBackground(frameColor)
	centerInput(b.changePasswordNew)
	b.changePasswordNew.SetMask('*')

	fieldGrid := etk.NewGrid()
	fieldGrid.SetColumnSizes(-1, -1)
	fieldGrid.SetRowSizes(-1, 20, -1)
	fieldGrid.AddChildAt(oldLabel, 0, 0, 1, 1)
	fieldGrid.AddChildAt(b.changePasswordOld, 1, 0, 2, 1)
	fieldGrid.AddChildAt(newLabel, 0, 2, 1, 1)
	fieldGrid.AddChildAt(b.changePasswordNew, 1, 2, 2, 1)

	grid := etk.NewGrid()
	grid.SetBackground(color.RGBA{40, 24, 9, 255})
	grid.SetColumnSizes(20, -1, -1, 20)
	grid.SetRowSizes(72, fieldHeight+20+fieldHeight, -1, etk.Scale(baseButtonHeight))
	grid.AddChildAt(headerLabel, 1, 0, 2, 1)
	grid.AddChildAt(fieldGrid, 1, 1, 2, 1)
	grid.AddChildAt(etk.NewBox(), 1, 2, 1, 1)
	grid.AddChildAt(etk.NewButton(gotext.Get("Cancel"), b.hideMenu), 0, 3, 2, 1)
	grid.AddChildAt(etk.NewButton(gotext.Get("Submit"), b.selectChangePassword), 2, 3, 2, 1)

	b.changePasswordDialog = &Dialog{grid}
	b.changePasswordDialog.SetVisible(false)
}

func (b *board) createMuteSoundsDialog() {
	headerLabel := resizeText(gotext.Get("Mute Sounds"))
	headerLabel.SetHorizontal(etk.AlignCenter)
	headerLabel.SetVertical(etk.AlignCenter)

	rowCount := 5

	b.muteJoinLeaveCheckbox = etk.NewCheckbox(b.toggleMuteJoinLeave)
	b.muteJoinLeaveCheckbox.SetBorderColor(triangleA)
	b.muteJoinLeaveCheckbox.SetCheckColor(triangleA)
	b.muteJoinLeaveCheckbox.SetSelected(b.muteJoinLeave)

	muteJoinLeaveLabel := &ClickableText{
		Text: resizeText(gotext.Get("Join/Leave")),
		onSelected: func() {
			b.muteJoinLeaveCheckbox.SetSelected(!b.muteJoinLeaveCheckbox.Selected())
			b.toggleMuteJoinLeave()
		},
	}
	muteJoinLeaveLabel.SetVertical(etk.AlignCenter)

	b.muteChatCheckbox = etk.NewCheckbox(b.toggleMuteChat)
	b.muteChatCheckbox.SetBorderColor(triangleA)
	b.muteChatCheckbox.SetCheckColor(triangleA)
	b.muteChatCheckbox.SetSelected(b.muteChat)

	muteChatLabel := &ClickableText{
		Text: resizeText(gotext.Get("Chat")),
		onSelected: func() {
			b.muteChatCheckbox.SetSelected(!b.muteChatCheckbox.Selected())
			b.toggleMuteChat()
		},
	}
	muteChatLabel.SetVertical(etk.AlignCenter)

	b.muteRollCheckbox = etk.NewCheckbox(b.toggleMuteRoll)
	b.muteRollCheckbox.SetBorderColor(triangleA)
	b.muteRollCheckbox.SetCheckColor(triangleA)
	b.muteRollCheckbox.SetSelected(b.muteRoll)

	muteRollLabel := &ClickableText{
		Text: resizeText(gotext.Get("Roll Dice")),
		onSelected: func() {
			b.muteRollCheckbox.SetSelected(!b.muteRollCheckbox.Selected())
			b.toggleMuteRoll()
		},
	}
	muteRollLabel.SetVertical(etk.AlignCenter)

	b.muteMoveCheckbox = etk.NewCheckbox(b.toggleMuteMove)
	b.muteMoveCheckbox.SetBorderColor(triangleA)
	b.muteMoveCheckbox.SetCheckColor(triangleA)
	b.muteMoveCheckbox.SetSelected(b.muteMove)

	muteMoveLabel := &ClickableText{
		Text: resizeText(gotext.Get("Move")),
		onSelected: func() {
			b.muteMoveCheckbox.SetSelected(!b.muteMoveCheckbox.Selected())
			b.toggleMuteMove()
		},
	}
	muteMoveLabel.SetVertical(etk.AlignCenter)

	b.muteBearOffCheckbox = etk.NewCheckbox(b.toggleMuteBearOff)
	b.muteBearOffCheckbox.SetBorderColor(triangleA)
	b.muteBearOffCheckbox.SetCheckColor(triangleA)
	b.muteBearOffCheckbox.SetSelected(b.muteBearOff)

	muteBearOffLabel := &ClickableText{
		Text: resizeText(gotext.Get("Bear Off")),
		onSelected: func() {
			b.muteBearOffCheckbox.SetSelected(!b.muteBearOffCheckbox.Selected())
			b.toggleMuteBearOff()
		},
	}
	muteBearOffLabel.SetVertical(etk.AlignCenter)

	checkboxGrid := etk.NewGrid()
	checkboxGrid.SetColumnSizes(72, 20, -1)
	sizes := []int{-1}
	for i := 1; i < rowCount; i++ {
		sizes = append(sizes, 20, -1)
	}
	checkboxGrid.SetRowSizes(sizes...)

	gridY := 0
	checkboxGrid.AddChildAt(cGrid(b.muteJoinLeaveCheckbox), 0, gridY, 1, 1)
	checkboxGrid.AddChildAt(muteJoinLeaveLabel, 2, gridY, 1, 1)
	gridY += 2
	checkboxGrid.AddChildAt(cGrid(b.muteChatCheckbox), 0, gridY, 1, 1)
	checkboxGrid.AddChildAt(muteChatLabel, 2, gridY, 1, 1)
	gridY += 2
	checkboxGrid.AddChildAt(cGrid(b.muteRollCheckbox), 0, gridY, 1, 1)
	checkboxGrid.AddChildAt(muteRollLabel, 2, gridY, 1, 1)
	gridY += 2
	checkboxGrid.AddChildAt(cGrid(b.muteMoveCheckbox), 0, gridY, 1, 1)
	checkboxGrid.AddChildAt(muteMoveLabel, 2, gridY, 1, 1)
	gridY += 2
	checkboxGrid.AddChildAt(cGrid(b.muteBearOffCheckbox), 0, gridY, 1, 1)
	checkboxGrid.AddChildAt(muteBearOffLabel, 2, gridY, 1, 1)

	grid := etk.NewGrid()
	grid.SetBackground(color.RGBA{40, 24, 9, 255})
	grid.SetColumnSizes(20, -1, -1, 20)
	grid.SetRowSizes(72, fieldHeight+((fieldHeight+20)*(rowCount-1)), -1, etk.Scale(baseButtonHeight))
	grid.AddChildAt(headerLabel, 1, 0, 2, 1)
	grid.AddChildAt(checkboxGrid, 1, 1, 2, 1)
	grid.AddChildAt(etk.NewBox(), 1, 2, 1, 1)
	grid.AddChildAt(etk.NewButton(gotext.Get("Return"), b.showSettings), 0, 3, 4, 1)

	b.muteSoundsDialog = &Dialog{grid}
	b.muteSoundsDialog.SetVisible(false)
}

func (b *board) createSettingsDialog() {
	settingsLabel := resizeText(gotext.Get("Settings"))
	settingsLabel.SetHorizontal(etk.AlignCenter)
	settingsLabel.SetVertical(etk.AlignCenter)

	b.highlightCheckbox = etk.NewCheckbox(b.toggleHighlightCheckbox)
	b.highlightCheckbox.SetBorderColor(triangleA)
	b.highlightCheckbox.SetCheckColor(triangleA)
	b.highlightCheckbox.SetSelected(b.highlightAvailable)

	highlightLabel := &ClickableText{
		Text: resizeText(gotext.Get("Highlight legal moves")),
		onSelected: func() {
			b.highlightCheckbox.SetSelected(!b.highlightCheckbox.Selected())
			b.toggleHighlightCheckbox()
		},
	}
	highlightLabel.SetVertical(etk.AlignCenter)

	b.showPipCountCheckbox = etk.NewCheckbox(b.togglePipCountCheckbox)
	b.showPipCountCheckbox.SetBorderColor(triangleA)
	b.showPipCountCheckbox.SetCheckColor(triangleA)
	b.showPipCountCheckbox.SetSelected(b.showPipCount)

	pipCountLabel := &ClickableText{
		Text: resizeText(gotext.Get("Show pip count")),
		onSelected: func() {
			b.showPipCountCheckbox.SetSelected(!b.showPipCountCheckbox.Selected())
			b.togglePipCountCheckbox()
		},
	}
	pipCountLabel.SetVertical(etk.AlignCenter)

	b.showMovesCheckbox = etk.NewCheckbox(b.toggleMovesCheckbox)
	b.showMovesCheckbox.SetBorderColor(triangleA)
	b.showMovesCheckbox.SetCheckColor(triangleA)
	b.showMovesCheckbox.SetSelected(b.showMoves)

	movesLabel := &ClickableText{
		Text: resizeText(gotext.Get("Show moves")),
		onSelected: func() {
			b.showMovesCheckbox.SetSelected(!b.showMovesCheckbox.Selected())
			b.toggleMovesCheckbox()
		},
	}
	movesLabel.SetVertical(etk.AlignCenter)

	b.flipBoardCheckbox = etk.NewCheckbox(b.toggleFlipBoardCheckbox)
	b.flipBoardCheckbox.SetBorderColor(triangleA)
	b.flipBoardCheckbox.SetCheckColor(triangleA)
	b.flipBoardCheckbox.SetSelected(b.flipBoard)

	flipBoardLabel := &ClickableText{
		Text: resizeText(gotext.Get("Flip board")),
		onSelected: func() {
			b.flipBoardCheckbox.SetSelected(!b.flipBoardCheckbox.Selected())
			b.toggleFlipBoardCheckbox()
		},
	}
	flipBoardLabel.SetVertical(etk.AlignCenter)

	b.traditionalCheckbox = etk.NewCheckbox(b.toggleTraditionalCheckbox)
	b.traditionalCheckbox.SetBorderColor(triangleA)
	b.traditionalCheckbox.SetCheckColor(triangleA)
	b.traditionalCheckbox.SetSelected(b.traditional)

	traditionalLabel := &ClickableText{
		Text: resizeText(gotext.Get("Flip opp. space numbers")),
		onSelected: func() {
			b.traditionalCheckbox.SetSelected(!b.traditionalCheckbox.Selected())
			b.toggleTraditionalCheckbox()
		},
	}
	traditionalLabel.SetVertical(etk.AlignCenter)

	b.advancedMovementCheckbox = etk.NewCheckbox(b.toggleAdvancedMovementCheckbox)
	b.advancedMovementCheckbox.SetBorderColor(triangleA)
	b.advancedMovementCheckbox.SetCheckColor(triangleA)
	b.advancedMovementCheckbox.SetSelected(b.advancedMovement)

	advancedMovementLabel := &ClickableText{
		Text: resizeText(gotext.Get("Advanced movement")),
		onSelected: func() {
			b.advancedMovementCheckbox.SetSelected(!b.advancedMovementCheckbox.Selected())
			b.toggleAdvancedMovementCheckbox()
		},
	}
	advancedMovementLabel.SetVertical(etk.AlignCenter)

	b.autoPlayCheckbox = etk.NewCheckbox(b.toggleAutoPlayCheckbox)
	b.autoPlayCheckbox.SetBorderColor(triangleA)
	b.autoPlayCheckbox.SetCheckColor(triangleA)

	autoPlayLabel := &ClickableText{
		Text: resizeText(gotext.Get("Auto-play forced moves")),
		onSelected: func() {
			b.autoPlayCheckbox.SetSelected(!b.autoPlayCheckbox.Selected())
			b.toggleAutoPlayCheckbox()
		},
	}
	autoPlayLabel.SetVertical(etk.AlignCenter)

	b.recreateAccountGrid()

	checkboxGrid := etk.NewGrid()
	checkboxGrid.SetColumnSizes(72, 20, -1)
	if !enableOnScreenKeyboard {
		checkboxGrid.SetRowSizes(-1, 20, -1, 20, -1, 20, -1, 20, -1, 20, -1, 20, -1, 20, -1, 20, -1, 20, -1)
	} else {
		checkboxGrid.SetRowSizes(-1, 20, -1, 20, -1, 20, -1, 20, -1, 20, -1, 20, -1, 20, -1, 20, -1)
	}
	{
		accountLabel := resizeText(gotext.Get("Account"))
		accountLabel.SetVertical(etk.AlignCenter)

		grid := etk.NewGrid()
		grid.AddChildAt(accountLabel, 0, 0, 1, 1)
		grid.AddChildAt(b.accountGrid, 1, 0, 2, 1)
		checkboxGrid.AddChildAt(grid, 0, 0, 3, 1)
	}
	{
		muteLabel := resizeText(gotext.Get("Sound"))
		muteLabel.SetVertical(etk.AlignCenter)

		openMute := etk.NewButton(gotext.Get("Mute Sounds"), b.showMuteSounds)
		openMute.SetHorizontal(etk.AlignStart)

		grid := etk.NewGrid()
		grid.AddChildAt(muteLabel, 0, 0, 1, 1)
		grid.AddChildAt(openMute, 1, 0, 2, 1)
		checkboxGrid.AddChildAt(grid, 0, 2, 3, 1)
	}
	{
		speedLabel := resizeText(gotext.Get("Speed"))
		speedLabel.SetVertical(etk.AlignCenter)

		b.selectSpeed = etk.NewSelect(game.itemHeight(), b.confirmSelectSpeed)
		b.selectSpeed.SetHighlightColor(color.RGBA{191, 156, 94, 255})
		b.selectSpeed.AddOption(gotext.Get("Slow"))
		b.selectSpeed.AddOption(gotext.Get("Medium"))
		b.selectSpeed.AddOption(gotext.Get("Fast"))
		b.selectSpeed.AddOption(gotext.Get("Instant"))
		b.selectSpeed.SetSelectedItem(int(bgammon.SpeedMedium))

		grid := etk.NewGrid()
		grid.AddChildAt(speedLabel, 0, 0, 1, 1)
		grid.AddChildAt(b.selectSpeed, 1, 0, 2, 1)
		checkboxGrid.AddChildAt(grid, 0, 4, 3, 1)
	}
	gridY := 6
	checkboxGrid.AddChildAt(cGrid(b.highlightCheckbox), 0, gridY, 1, 1)
	checkboxGrid.AddChildAt(highlightLabel, 2, gridY, 1, 1)
	gridY += 2
	checkboxGrid.AddChildAt(cGrid(b.showPipCountCheckbox), 0, gridY, 1, 1)
	checkboxGrid.AddChildAt(pipCountLabel, 2, gridY, 1, 1)
	gridY += 2
	checkboxGrid.AddChildAt(cGrid(b.showMovesCheckbox), 0, gridY, 1, 1)
	checkboxGrid.AddChildAt(movesLabel, 2, gridY, 1, 1)
	gridY += 2
	checkboxGrid.AddChildAt(cGrid(b.flipBoardCheckbox), 0, gridY, 1, 1)
	checkboxGrid.AddChildAt(flipBoardLabel, 2, gridY, 1, 1)
	gridY += 2
	checkboxGrid.AddChildAt(cGrid(b.traditionalCheckbox), 0, gridY, 1, 1)
	checkboxGrid.AddChildAt(traditionalLabel, 2, gridY, 1, 1)
	gridY += 2
	if enableRightClick {
		checkboxGrid.AddChildAt(cGrid(b.advancedMovementCheckbox), 0, gridY, 1, 1)
		checkboxGrid.AddChildAt(advancedMovementLabel, 2, gridY, 1, 1)
		gridY += 2
	}
	checkboxGrid.AddChildAt(cGrid(b.autoPlayCheckbox), 0, gridY, 1, 1)
	checkboxGrid.AddChildAt(autoPlayLabel, 2, gridY, 1, 1)

	gridSize := 72 + 20 + 72 + 20 + 72 + 20 + 72 + 20 + 72 + 20 + 72 + 20 + 72 + 20 + 72
	if enableRightClick {
		gridSize += 20 + 72
	}
	grid := etk.NewGrid()
	grid.SetBackground(color.RGBA{40, 24, 9, 255})
	grid.SetColumnSizes(20, -1, -1, 20)
	grid.SetRowSizes(72, -1, 20, etk.Scale(baseButtonHeight))
	grid.AddChildAt(settingsLabel, 1, 0, 2, 1)
	grid.AddChildAt(checkboxGrid, 1, 1, 2, 1)
	grid.AddChildAt(etk.NewBox(), 1, 2, 1, 1)
	grid.AddChildAt(etk.NewButton(gotext.Get("Return"), b.hideMenu), 0, 3, 4, 1)

	b.settingsDialog = &Dialog{grid}
	b.settingsDialog.SetVisible(false)
}

func (b *board) createLeaveMatchDialog() {
	label := resizeText(gotext.Get("Leave match?"))
	label.SetHorizontal(etk.AlignCenter)
	label.SetVertical(etk.AlignCenter)

	grid := etk.NewGrid()
	grid.SetBackground(color.RGBA{40, 24, 9, 255})
	grid.AddChildAt(label, 0, 0, 2, 1)
	grid.AddChildAt(etk.NewButton(gotext.Get("No"), b.cancelLeaveMatch), 0, 1, 1, 1)
	grid.AddChildAt(etk.NewButton(gotext.Get("Yes"), b.confirmLeaveMatch), 1, 1, 1, 1)

	b.leaveMatchDialog = &Dialog{grid}
	b.leaveMatchDialog.SetVisible(false)
}

func (b *board) createMatchStatus() {
	timerLabel := etk.NewText("0:00")
	timerLabel.SetForeground(triangleA)
	timerLabel.SetScrollBarVisible(false)
	timerLabel.SetSingleLine(true)
	timerLabel.SetHorizontal(etk.AlignCenter)
	timerLabel.SetVertical(etk.AlignCenter)
	b.timerLabel = timerLabel

	clockLabel := etk.NewText("12:00")
	clockLabel.SetForeground(triangleA)
	clockLabel.SetScrollBarVisible(false)
	clockLabel.SetSingleLine(true)
	clockLabel.SetHorizontal(etk.AlignCenter)
	clockLabel.SetVertical(etk.AlignCenter)
	b.clockLabel = clockLabel

	b.showMenuButton = etk.NewButton(gotext.Get("Menu"), b.toggleMenu)
	if !mobileDevice {
		b.showMenuButton.SetBorderSize(etk.Scale(etk.Style.ButtonBorderSize / 2))
	}

	var padding int
	if !mobileDevice {
		padding = int(b.verticalBorderSize / 4)
	}
	b.matchStatusGrid = etk.NewGrid()
	b.matchStatusGrid.SetColumnSizes(padding, -1, -1, -1, padding)
	b.matchStatusGrid.AddChildAt(b.timerLabel, 1, 0, 1, 1)
	b.matchStatusGrid.AddChildAt(b.clockLabel, 2, 0, 1, 1)
	b.matchStatusGrid.AddChildAt(b.showMenuButton, 3, 0, 1, 1)
	b.matchStatusGrid.AddChildAt(etk.NewBox(), 4, 0, 1, 1)
}

func (b *board) createReplayControls() {
	b.replayPauseButton = etk.NewButton("|>", b.selectReplayPause)

	b.replayGrid = etk.NewGrid()
	b.replayGrid.AddChildAt(etk.NewButton("|<<", b.selectReplayStart), 0, 0, 1, 1)
	b.replayGrid.AddChildAt(etk.NewButton("<<", b.selectReplayJumpBack), 1, 0, 1, 1)
	b.replayGrid.AddChildAt(etk.NewButton("<", b.selectReplayStepBack), 2, 0, 1, 1)
	b.replayGrid.AddChildAt(b.replayPauseButton, 3, 0, 1, 1)
	b.replayGrid.AddChildAt(etk.NewButton(">", b.selectReplayStepForward), 4, 0, 1, 1)
	b.replayGrid.AddChildAt(etk.NewButton(">>", b.selectReplayJumpForward), 5, 0, 1, 1)
	b.replayGrid.AddChildAt(etk.NewButton(">>|", b.selectReplayEnd), 6, 0, 1, 1)
}

func (b *board) createReplayList() {
	scrollBarWidth := etk.Scale(32)
	b.replayList = etk.NewList(etk.Scale(baseButtonHeight), nil)
	b.replayList.SetScrollBarColors(etk.Style.ScrollAreaColor, etk.Style.ScrollHandleColor)
	b.replayList.SetScrollBarWidth(scrollBarWidth)
}

func (b *board) recreateUIGrid() {
	b.uiGrid.Clear()
	var gridY int
	if !mobileDevice || game.replay {
		b.uiGrid.AddChildAt(etk.NewBox(), 0, 0, 1, 1)
		b.uiGrid.AddChildAt(b.matchStatusGrid, 0, 1, 1, 1)
		b.uiGrid.AddChildAt(etk.NewBox(), 0, 2, 1, 1)
		gridY = 3
	}
	if game.replay {
		g := etk.NewGrid()
		g.SetRowSizes(etk.Scale(baseButtonHeight), int(b.verticalBorderSize/2), -1, int(b.verticalBorderSize/2), etk.Scale(baseButtonHeight*2))
		g.AddChildAt(b.replayGrid, 0, 0, 1, 1)
		g.AddChildAt(b.replayList, 0, 2, 1, 1)
		g.AddChildAt(statusBuffer, 0, 4, 1, 1)
		b.uiGrid.AddChildAt(g, 0, gridY, 1, 3)
		gridY++
	} else {
		if mobileDevice {
			b.uiGrid.AddChildAt(b.inputGrid, 0, gridY, 1, 1)
			b.uiGrid.AddChildAt(etk.NewBox(), 0, gridY+1, 1, 1)
			gridY += 2
		}
		b.uiGrid.AddChildAt(statusBuffer, 0, gridY, 1, 1)
		b.uiGrid.AddChildAt(etk.NewBox(), 0, gridY+1, 1, 1)
		b.uiGrid.AddChildAt(gameBuffer, 0, gridY+2, 1, 1)
		gridY += 3
		if mobileDevice {
			b.uiGrid.AddChildAt(etk.NewBox(), 0, gridY, 1, 1)
			b.uiGrid.AddChildAt(b.matchStatusGrid, 0, gridY+1, 1, 1)
		} else {
			b.uiGrid.AddChildAt(etk.NewBox(), 0, gridY, 1, 1)
			b.uiGrid.AddChildAt(b.inputGrid, 0, gridY+1, 1, 1)
		}
	}
}

func (b *board) createSelectRollDialog() {
	const padding = 10
	b.selectRollGrid.SetBackground(frameColor)
	b.selectRollGrid.SetColumnPadding(padding)
	b.selectRollGrid.SetRowPadding(padding)
	b.selectRollGrid.AddChildAt(NewDieButton(1, b.selectRollFunc(1)), 0, 0, 1, 1)
	b.selectRollGrid.AddChildAt(NewDieButton(2, b.selectRollFunc(2)), 1, 0, 1, 1)
	b.selectRollGrid.AddChildAt(NewDieButton(3, b.selectRollFunc(3)), 2, 0, 1, 1)
	b.selectRollGrid.AddChildAt(NewDieButton(4, b.selectRollFunc(4)), 0, 1, 1, 1)
	b.selectRollGrid.AddChildAt(NewDieButton(5, b.selectRollFunc(5)), 1, 1, 1, 1)
	b.selectRollGrid.AddChildAt(NewDieButton(6, b.selectRollFunc(6)), 2, 1, 1, 1)
}

func (b *board) createFrame() {
	b.frame.SetPositionChildren(true)
	b.frame.AddChild(NewBoardBackgroundWidget())
	b.frame.AddChild(NewBoardMovingWidget())

	f := etk.NewFrame()
	f.AddChild(b.opponentRatingShadowLabel)
	f.AddChild(b.opponentRatingLabel)
	f.AddChild(b.opponentForcedLabel)
	f.AddChild(b.opponentMovesLabel)
	f.AddChild(b.opponentPipCount)
	f.AddChild(b.opponentLabel)
	f.AddChild(b.playerLabel)
	f.AddChild(b.playerPipCount)
	f.AddChild(b.playerMovesLabel)
	f.AddChild(b.playerForcedLabel)
	f.AddChild(b.playerRatingShadowLabel)
	f.AddChild(b.playerRatingLabel)
	f.AddChild(b.uiGrid)
	f.AddChild(b.rematchButton)
	b.frame.AddChild(f)

	b.frame.AddChild(b.widget)
	b.frame.AddChild(b.buttonsGrid)
	b.frame.AddChild(etk.NewFrame(b.selectRollGrid))
	b.frame.AddChild(NewBoardDraggedWidget())

	f = etk.NewFrame()
	f.AddChild(b.menuGrid)
	f.AddChild(b.settingsDialog)
	children := b.selectSpeed.Children()
	if len(children) == 0 {
		log.Panicf("failed to find speed selection list")
	}
	f.AddChild(children[0])
	f.AddChild(b.changePasswordDialog)
	f.AddChild(b.muteSoundsDialog)
	f.AddChild(b.leaveMatchDialog)
	b.frame.AddChild(f)

	b.frame.AddChild(game.tutorialFrame)
}

func cGrid(checkbox *etk.Checkbox) *etk.Grid {
	g := etk.NewGrid()
	g.SetColumnSizes(7, -1)
	g.AddChildAt(checkbox, 1, 0, 1, 1)
	return g
}
