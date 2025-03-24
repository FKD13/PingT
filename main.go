package main

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"math/rand"
	"time"
)

type TargetComponent struct {
	Flex     *tview.Flex
	TextView *tview.TextView
	box1     *tview.Box
	box2     *tview.Box
}

type Target struct {
	IP          string
	fails       int
	UIComponent *TargetComponent
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

func (t *TargetComponent) Update(grid *tview.Grid, row int, column int) {
	if rand.Intn(10) > 5 {
		t.SetBackgroundColor(tcell.ColorRed)
	} else {
		t.SetBackgroundColor(tcell.ColorGreen)
	}

	grid.RemoveItem(t.Flex)
	grid.AddItem(t.Flex, row, column, 1, 1, 1, 1, false)
}

func main() {

	comp1 := NewTargetComponent("1.1.1.1")
	comp2 := NewTargetComponent("8.8.8.8")
	comp3 := NewTargetComponent("1.0.0.1")
	comp4 := NewTargetComponent("8.4.4.8")

	app := tview.NewApplication()
	grid := tview.NewGrid().
		SetColumns(0, 0).
		SetBorders(true).
		SetBordersColor(tcell.ColorBlack).
		AddItem(comp1.Flex, 0, 0, 1, 1, 1, 1, false).
		AddItem(comp2.Flex, 0, 1, 1, 1, 1, 1, false).
		AddItem(comp3.Flex, 0, 2, 1, 1, 1, 1, false).
		AddItem(comp4.Flex, 1, 0, 1, 3, 1, 1, false)

	go func() {
		for {
			time.Sleep(time.Second)
			app.QueueUpdateDraw(func() {
				comp1.Update(grid, 0, 0)
				comp2.Update(grid, 0, 1)
				comp3.Update(grid, 0, 2)
				comp4.Update(grid, 1, 0)
			})
		}
	}()

	//box := tview.NewBox().SetBorder(true).SetTitle("Hello, world!")
	if err := app.SetRoot(grid, true).Run(); err != nil {
		panic(err)
	}
}
