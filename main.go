package main

import (
	"bufio"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"golang.org/x/crypto/ssh/terminal"
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

type PingState int

const (
	Success PingState = iota
	Failure
)

type TargetPing struct {
	State PingState

	Duration    float64
	AvgDuration float64
	Loss        float64
}

type TargetComponent struct {
	Flex     *tview.Flex
	TextView *tview.TextView
	box1     *tview.Box
	box2     *tview.Box
}

type Target struct {
	Host        string
	fails       int
	UIComponent *TargetComponent
	LastPing    *TargetPing
}

func NewTargetComponent(ipString string) *TargetComponent {
	ip := tview.NewTextView().SetTextAlign(tview.AlignCenter).SetText(fmt.Sprintf("[::b]%s[::B]", ipString)).SetDynamicColors(true)
	ip.SetBackgroundColor(tcell.ColorDarkGray)

	box1 := tview.NewBox().SetBackgroundColor(tcell.ColorDarkGray)
	box2 := tview.NewBox().SetBackgroundColor(tcell.ColorDarkGray)

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(box1, 0, 1, false).
		AddItem(ip, 1, 0, false).
		AddItem(box2, 0, 1, false)

	return &TargetComponent{
		Flex:     flex,
		TextView: ip,
		box1:     box1,
		box2:     box2,
	}
}

func (t *TargetComponent) SetBackgroundColor(color tcell.Color) {
	t.box1.SetBackgroundColor(color)
	t.box2.SetBackgroundColor(color)
	t.TextView.SetBackgroundColor(color)
}

func (t *Target) Update(grid *tview.Grid, row int, column int) {
	var color tcell.Color

	if t.LastPing != nil {
		if t.LastPing.State == Success {
			color = tcell.ColorGreen
		} else {
			color = tcell.ColorRed
		}
	} else {
		color = tcell.ColorGrey
	}

	t.UIComponent.SetBackgroundColor(color)

	grid.RemoveItem(t.UIComponent.Flex)
	grid.AddItem(t.UIComponent.Flex, row, column, 1, 1, 1, 1, false)
}

func DrawGrid(grid *tview.Grid, targets *map[string]*Target) {
	width, height, err := terminal.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		panic(err)
	}

	maxCols := 1000
	maxRows := 1000
	bestScore := math.NaN()
	for i := range len(*targets) {
		i++
		for j := range len(*targets) {
			j++

			score := (float64(width) / float64(i)) / (float64(height*2) / (float64(j)))
			if i*j >= len(*targets) && i*j <= maxRows*maxCols && ((score > 1 && (score < bestScore || math.IsNaN(bestScore))) || score < 1 && (score > bestScore || math.IsNaN(bestScore))) {
				bestScore = score
				maxCols = i
				maxRows = j
			}
		}
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
				color = tcell.ColorRed
			}
		} else {
			color = tcell.ColorGrey
		}

		target.UIComponent.SetBackgroundColor(color)

		grid.RemoveItem(target.UIComponent.Flex)
		if ((row+1)*maxCols+col) >= len(*targets) && row == maxRows-2 {
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
		//SetColumns(0, 0).
		SetBorders(true).
		SetBordersColor(tcell.ColorBlack)
	grid.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey { return nil })

	go func() {
		for {
			time.Sleep(time.Millisecond * 500)
			app.QueueUpdateDraw(func() {
				DrawGrid(grid, targets)
			})
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
	reFail := regexp.MustCompile(`^([^ ]+) +: \[[0-9]+], timed out \(([0-9]*\.[0-9]*|NaN) avg, ([0-9]+)% loss\)$`)
	// Successful pings contain "bytes"
	reSuccess := regexp.MustCompile(`^([^ ]+) +: \[[0-9]+], [0-9]+ bytes, ([0-9]*\.[0-9]*) ms \(([0-9]*\.[0-9]*) avg, ([0-9]+)% loss\)$`)

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {

		if reFail.MatchString(scanner.Text()) {

			data := reFail.FindAllStringSubmatch(scanner.Text(), -1)

			avgDuration, _ := strconv.ParseFloat(data[0][2], 64)
			loss, _ := strconv.ParseFloat(data[0][3], 64)

			target := (*targets)[data[0][1]]
			target.LastPing = &TargetPing{State: Failure, Duration: -1, AvgDuration: avgDuration, Loss: loss}
			target.fails = target.fails + 1

		} else if reSuccess.MatchString(scanner.Text()) {

			data := reSuccess.FindAllStringSubmatch(scanner.Text(), -1)

			duration, _ := strconv.ParseFloat(data[0][2], 64)
			avgDuration, _ := strconv.ParseFloat(data[0][3], 64)
			loss, _ := strconv.ParseFloat(data[0][4], 64)

			target := (*targets)[data[0][1]]
			target.LastPing = &TargetPing{State: Success, Duration: duration, AvgDuration: avgDuration, Loss: loss}
			target.fails = 0

		} else {
			panic("target state unclear")
		}
	}

	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
}

func main() {

	targets := make(map[string]*Target)
	targets["1.1.1.1"] = &Target{"1.1.1.1", 0, NewTargetComponent("1.1.1.1"), nil}
	targets["127.0.0.1"] = &Target{"127.0.0.1", 0, NewTargetComponent("127.0.0.1"), nil}
	targets["google.com"] = &Target{"google.com", 0, NewTargetComponent("google.com"), nil}
	targets["10.0.0.0"] = &Target{"10.0.0.0", 0, NewTargetComponent("10.0.0.0"), nil}

	go RunFPing(&targets)

	RunTUI(&targets)
}
