package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"golang.org/x/term"
	"log"
	"maps"
	"math"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

import flag "github.com/spf13/pflag"

type PingState int

const (
	Success PingState = iota
	Failure
)

type TargetPing struct {
	State PingState

	SequenceID  int
	Duration    float64
	AvgDuration float64
	Loss        float64
}

type TargetComponent struct {
	Flex             *tview.Flex
	topStats         *tview.TextView
	targetName       *tview.TextView
	bottomStatsLeft  *tview.TextView
	bottomStatsRight *tview.TextView
	spacer1          *tview.Box
	spacer2          *tview.Box
}

type Target struct {
	Host        string
	fails       int
	probes      int
	UIComponent *TargetComponent
	LastPing    *TargetPing
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
		Flex:             flex,
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

func DrawGrid(grid *tview.Grid, targets *map[string]*Target) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		panic(err)
	}

	// https://www.wolframalpha.com/input?i=x+%2F+y+%3D+w%2Fh%2C+x+*+y+%3D+c
	maxColsF := math.Sqrt(float64(width)) * math.Sqrt(float64(len(*targets))) / math.Sqrt(float64(height*2))
	maxRowsF := math.Sqrt(float64(height*2)) * math.Sqrt(float64(len(*targets))) / math.Sqrt(float64(width))

	// Found a good initial guess for the squarest grid, find the fit that minimizes the amount of unoccupied cells
	maxCols, maxRows, err := findBest(int(math.Ceil(maxColsF)), int(math.Ceil(maxRowsF)), len(*targets))
	if err != nil {
		panic(err)
	}

	row := 0
	col := 0
	for (row*maxCols + col) < len(*targets) {
		target := slices.SortedFunc(maps.Values(*targets), func(t1 *Target, t2 *Target) int {
			return strings.Compare(t1.Host, t2.Host)
		})[row*maxCols+col]

		var color tcell.Color

		if target.LastPing != nil {
			if target.LastPing.State == Success {
				color = tcell.ColorGreen
			} else {
				if target.probes == target.fails {
					color = tcell.ColorDarkRed
				} else if target.fails <= 3 {
					color = tcell.ColorOrange
				} else {
					color = tcell.ColorRed
				}
			}
		} else {
			color = tcell.ColorGrey
		}

		if target.LastPing != nil {
			if target.LastPing.State == Success {
				target.UIComponent.SetBottomStats(fmt.Sprintf(" %.1fms", target.LastPing.AvgDuration), fmt.Sprintf("%.0f%% Loss ", target.LastPing.Loss))
				//target.UIComponent.SetTopStats(fmt.Sprintf("Alive!"))
			} else {
				target.UIComponent.SetBottomStats(fmt.Sprintf(" F: %d", target.fails), fmt.Sprintf("%.0f%% Loss ", target.LastPing.Loss))
				// if target.fails <= 3 {
				// 	target.UIComponent.SetTopStats(fmt.Sprintf("Failing..."))
				// } else {
				// 	target.UIComponent.SetTopStats(fmt.Sprintf("Unreachable!"))
				// }
			}

		}
		target.UIComponent.SetBackgroundColor(color)

		grid.RemoveItem(target.UIComponent.Flex)
		if ((row+1)*maxCols+col) >= len(*targets) && row != maxRows-1 {
			grid.AddItem(target.UIComponent.Flex, row, col, 2, 1, 1, 1, false)
		} else {
			grid.AddItem(target.UIComponent.Flex, row, col, 1, 1, 1, 1, false)
		}

		col++
		if col == maxCols {
			col = 0
			row++
		}
	}
}

func RunTUI(targets *map[string]*Target) {

	app := tview.NewApplication()
	grid := tview.NewGrid().
		SetBorders(true).
		SetBordersColor(tcell.ColorBlack)
	grid.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return nil })

	go func() {
		for {
			app.QueueUpdateDraw(func() {
				DrawGrid(grid, targets)
			})
			time.Sleep(time.Millisecond * time.Duration(*refreshRate))
		}
	}()

	if err := app.SetRoot(grid, true).Run(); err != nil {
		panic(err)
	}
}

func RunFPing(targets *map[string]*Target) {
	args := append([]string{"-l"}, slices.Collect(maps.Keys(*targets))...)
	cmd := exec.Command("fping", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}

	if err := cmd.Start(); err != nil {
		panic(err)
	}

	// Failed pings contain "timed out"
	reFail := regexp.MustCompile(`^([^ ]+) +: \[([0-9]+)], timed out \(([0-9]+(\.[0-9]+)?|NaN) avg, ([0-9]+)% loss\)$`)
	// Successful pings contain "bytes"
	reSuccess := regexp.MustCompile(`^([^ ]+) +: \[([0-9]+)], [0-9]+ bytes, ([0-9]+(\.[0-9]+)?) ms \(([0-9]+(\.[0-9]+)?) avg, ([0-9]+)% loss\)$`)

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {

		if reFail.MatchString(scanner.Text()) {

			data := reFail.FindAllStringSubmatch(scanner.Text(), -1)

			sequenceID, _ := strconv.Atoi(data[0][2])
			avgDuration, _ := strconv.ParseFloat(data[0][3], 64)
			loss, _ := strconv.ParseFloat(data[0][5], 64)

			target := (*targets)[data[0][1]]

			if target.LastPing == nil || sequenceID > target.LastPing.SequenceID {
				target.LastPing = &TargetPing{State: Failure, SequenceID: sequenceID, Duration: -1, AvgDuration: avgDuration, Loss: loss}
				target.fails = target.fails + 1
			} else if target.LastPing != nil && target.LastPing.State == Failure {
				target.fails = target.fails + 1
			}
			target.probes = target.probes + 1

		} else if reSuccess.MatchString(scanner.Text()) {

			data := reSuccess.FindAllStringSubmatch(scanner.Text(), -1)

			sequenceID, _ := strconv.Atoi(data[0][2])
			duration, _ := strconv.ParseFloat(data[0][3], 64)
			avgDuration, _ := strconv.ParseFloat(data[0][5], 64)
			loss, _ := strconv.ParseFloat(data[0][7], 64)

			target := (*targets)[data[0][1]]

			if target.LastPing == nil || sequenceID > target.LastPing.SequenceID {
				target.LastPing = &TargetPing{State: Success, SequenceID: sequenceID, Duration: duration, AvgDuration: avgDuration, Loss: loss}
				target.fails = 0
			}
			target.probes = target.probes + 1

		} else {
			log.Println(scanner.Text())
			panic("target state unclear")
		}
	}

	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
}

var targetStrings *[]string = flag.StringSliceP("target", "t", []string{"1.1.1.1", "1.0.0.1"}, "Targets to ping")
var refreshRate *int = flag.Int("refresh", 1000, "Interval at which to refresh the UI")

func main() {

	flag.Parse()

	targets := make(map[string]*Target)
	for _, target := range *targetStrings {
		targets[target] = &Target{target, 0, 0, NewTargetComponent(target), nil}
	}

	go RunFPing(&targets)
	RunTUI(&targets)
}
