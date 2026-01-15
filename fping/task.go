package fping

import (
	"bufio"
	"fkd13.github.com/pingtui/state"
	"log"
	"os/exec"
	"regexp"
	"strconv"
)

func RunFPing(appState *state.AppState) {
	var hosts []string

	appState.Lock()
	for _, target := range appState.GetTargets() {
		hosts = append(hosts, target.Host)
	}
	appState.Unlock()

	cmd := exec.Command("fping", append([]string{"-l"}, hosts...)...)
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
		line := scanner.Text()
		go handleLine(line, appState, reSuccess, reFail)
	}

	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
}

func handleLine(line string, appState *state.AppState, reSuccess *regexp.Regexp, reFail *regexp.Regexp) {
	if reSuccess.MatchString(line) {
		handleSuccess(line, appState, reSuccess)
	} else if reFail.MatchString(line) {
		handleFail(line, appState, reFail)
	} else {
		// TODO: decide what to do
	}
}

func handleSuccess(line string, appState *state.AppState, reSuccess *regexp.Regexp) {
	data := reSuccess.FindAllStringSubmatch(line, -1)

	host := data[0][1]
	sequenceID, _ := strconv.Atoi(data[0][2])
	duration, _ := strconv.ParseFloat(data[0][3], 64)
	avgDuration, _ := strconv.ParseFloat(data[0][5], 64)
	loss, _ := strconv.ParseFloat(data[0][7], 64)

	appState.Lock()
	appState.GetTarget(host).UpdateProbe(&state.Probe{
		State:          state.Success,
		SequenceID:     sequenceID,
		Latency:        duration,
		AverageLatency: avgDuration,
		Loss:           loss,
	})
	appState.Unlock()
}

func handleFail(line string, appState *state.AppState, reFail *regexp.Regexp) {
	data := reFail.FindAllStringSubmatch(line, -1)

	host := data[0][1]
	sequenceID, _ := strconv.Atoi(data[0][2])
	avgDuration, _ := strconv.ParseFloat(data[0][3], 64)
	loss, _ := strconv.ParseFloat(data[0][5], 64)

	appState.Lock()
	appState.GetTarget(host).UpdateProbe(&state.Probe{
		State:          state.Failure,
		SequenceID:     sequenceID,
		AverageLatency: avgDuration,
		Loss:           loss,
	})
	appState.Unlock()
}
