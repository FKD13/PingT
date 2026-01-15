package tui

import (
	"errors"
	"fkd13.github.com/pingtui/state"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"golang.org/x/term"
	"math"
	"os"
	"slices"
	"strings"
	"time"
)

type TargetComponent struct {
	flex             *tview.Flex
	topStats         *tview.TextView
	targetName       *tview.TextView
	bottomStatsLeft  *tview.TextView
	bottomStatsRight *tview.TextView
	spacer1          *tview.Box
	spacer2          *tview.Box
}

func NewTargetComponent(ipString string) *TargetComponent {
	targetName := tview.NewTextView().SetTextAlign(tview.AlignCenter).SetText(fmt.Sprintf("[::b]%s[::B]", ipString)).SetDynamicColors(true)
	targetName.SetBackgroundColor(tcell.ColorDarkGray)

	spacer1 := tview.NewBox().SetBackgroundColor(tcell.ColorDarkGray)
	spacer2 := tview.NewBox().SetBackgroundColor(tcell.ColorDarkGray)

	topStats := tview.NewTextView().SetTextAlign(tview.AlignCenter).SetText(" ")
	topStats.SetBackgroundColor(tcell.ColorDarkGray)

	bottomStatsLeft := tview.NewTextView().SetTextAlign(tview.AlignLeft).SetText("  ")
	bottomStatsLeft.SetBackgroundColor(tcell.ColorDarkGray)

	bottomStatsRight := tview.NewTextView().SetTextAlign(tview.AlignRight).SetText("  ")
	bottomStatsRight.SetBackgroundColor(tcell.ColorDarkGray)

	bottomFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(bottomStatsLeft, 0, 1, false).
		AddItem(bottomStatsRight, 0, 1, false)
	bottomFlex.SetBackgroundColor(tcell.ColorDarkGray)

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(topStats, 1, 0, false).
		AddItem(spacer1, 0, 1, false).
		AddItem(targetName, 1, 0, false).
		AddItem(spacer2, 0, 1, false).
		AddItem(bottomFlex, 1, 0, false)
	flex.SetBackgroundColor(tcell.ColorDarkGray)

	return &TargetComponent{
		flex:             flex,
		topStats:         topStats,
		targetName:       targetName,
		bottomStatsLeft:  bottomStatsLeft,
		bottomStatsRight: bottomStatsRight,
		spacer1:          spacer1,
		spacer2:          spacer2,
	}
}

func (t *TargetComponent) SetBackgroundColor(color tcell.Color) {
	t.topStats.SetBackgroundColor(color)
	t.targetName.SetBackgroundColor(color)
	t.bottomStatsLeft.SetBackgroundColor(color)
	t.bottomStatsRight.SetBackgroundColor(color)
	t.spacer1.SetBackgroundColor(color)
	t.spacer2.SetBackgroundColor(color)
}

func (t *TargetComponent) SetTopStats(stats string) {
	t.topStats.SetText(stats)
}

func (t *TargetComponent) SetBottomStats(statsLeft string, statsRight string) {
	t.bottomStatsLeft.SetText(statsLeft)
	t.bottomStatsRight.SetText(statsRight)
}

func findBest(i int, j int, limit int) (int, int, error) {
	if i*j < limit {
		return 0, 0, errors.New("")
	}

	i2, j2, err2 := findBest(i-1, j, limit)
	i3, j3, err3 := findBest(i, j-1, limit)

	if err2 != nil && err3 != nil {
		return i, j, nil
	} else if err2 == nil && err3 == nil {
		if i2*j2 <= i3*j3 {
			return i2, j2, nil
		} else {
			return i3, j3, nil
		}
	} else if err2 == nil {
		return i2, j2, nil
	} else {
		return i3, j3, nil
	}
}

func DrawGrid(tui *TUI) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		panic(err)
	}

	targets := tui.appState.GetTargets()
	slices.SortFunc(targets, func(t1 *state.Target, t2 *state.Target) int {
		return strings.Compare(t1.Host, t2.Host)
	})

	// https://www.wolframalpha.com/input?i=x+%2F+y+%3D+w%2Fh%2C+x+*+y+%3D+c
	maxColsF := math.Sqrt(float64(width)) * math.Sqrt(float64(len(targets))) / math.Sqrt(float64(height*2))
	maxRowsF := math.Sqrt(float64(height*2)) * math.Sqrt(float64(len(targets))) / math.Sqrt(float64(width))

	// Found a good initial guess for the squarest grid, find the fit that minimizes the amount of unoccupied cells
	maxCols, maxRows, err := findBest(int(math.Ceil(maxColsF)), int(math.Ceil(maxRowsF)), len(targets))
	if err != nil {
		panic(err)
	}

	row := 0
	col := 0
	for (row*maxCols + col) < len(targets) {
		target := targets[row*maxCols+col]
		component := tui.Components[target.Host]

		var color tcell.Color
		if target.TargetState == state.Online {
			color = tcell.ColorGreen
		} else if target.TargetState == state.Failing {
			color = tcell.ColorOrange
		} else if target.TargetState == state.Offline {
			color = tcell.ColorRed
		} else {
			color = tcell.ColorGrey
		}
		component.SetBackgroundColor(color)

		if target.TargetState == state.Online {
			component.SetBottomStats(
				fmt.Sprintf(" %.1fms", target.Probe.AverageLatency),
				fmt.Sprintf("%.0f%% Loss ", target.Probe.Loss))

		} else if target.TargetState == state.Failing || target.TargetState == state.Offline {
			component.SetBottomStats(
				fmt.Sprintf(" F: %d", target.RecentFailCount),
				fmt.Sprintf("%.0f%% Loss ", target.Probe.Loss))
		}

		tui.Grid.RemoveItem(component.flex)
		if ((row+1)*maxCols+col) >= len(targets) && row != maxRows-1 {
			tui.Grid.AddItem(component.flex, row, col, 2, 1, 1, 1, false)
		} else {
			tui.Grid.AddItem(component.flex, row, col, 1, 1, 1, 1, false)
		}

		col++
		if col == maxCols {
			col = 0
			row++
		}
	}
}

type TUI struct {
	appState *state.AppState

	App        *tview.Application
	Grid       *tview.Grid
	Components map[string]*TargetComponent
}

func RunTUI(appState *state.AppState) {
	tui := &TUI{
		appState:   appState,
		App:        tview.NewApplication(),
		Grid:       tview.NewGrid(),
		Components: make(map[string]*TargetComponent),
	}

	appState.Lock()
	for _, target := range appState.GetTargets() {
		tui.Components[target.Host] = NewTargetComponent(target.Host)
	}
	appState.Unlock()

	tui.Grid.SetBorders(true).SetBordersColor(tcell.ColorBlack)
	tui.Grid.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return nil })

	go func() {
		for {
			tui.App.QueueUpdateDraw(func() {
				tui.appState.Lock()
				DrawGrid(tui)
				tui.appState.Unlock()
			})
			time.Sleep(time.Millisecond * time.Duration(10))
		}
	}()

	if err := tui.App.SetRoot(tui.Grid, true).Run(); err != nil {
		panic(err)
	}
}
