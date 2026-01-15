package main

import (
	"fkd13.github.com/pingtui/fping"
	"fkd13.github.com/pingtui/ntfy"
	"fkd13.github.com/pingtui/state"
	"fkd13.github.com/pingtui/tui"
)

import flag "github.com/spf13/pflag"

var targetStrings *[]string = flag.StringSliceP("target", "t", []string{"1.1.1.1", "1.0.0.1"}, "targets to ping")
var refreshRate *int = flag.Int("refresh", 1000, "Interval at which to refresh the UI")

func main() {

	flag.Parse()
	appState := state.NewAppState(targetStrings)

	go ntfy.RunNtfy(appState)
	go fping.RunFPing(appState)
	tui.RunTUI(appState)
}
