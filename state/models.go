package state

import (
	"maps"
	"slices"
	"sync"
)

type ProbeState int
type TargetState int

const (
	Success ProbeState = iota
	Failure
)

const (
	Unknown TargetState = iota
	Online
	Failing
	Offline
)

type AppState struct {
	sync.Mutex

	targets map[string]*Target
}

func NewAppState(hosts *[]string) *AppState {
	targets := make(map[string]*Target)
	for _, target := range *hosts {
		targets[target] = NewTarget(target)
	}

	return &AppState{
		targets: targets,
	}
}

func (s *AppState) GetTargets() []*Target {
	return slices.Collect(maps.Values(s.targets))
}

func (s *AppState) GetTarget(name string) *Target {
	return s.targets[name]
}

type Target struct {
	Host        string
	TargetState TargetState

	probeCount      int
	failCount       int
	RecentFailCount int

	Probe *Probe
}

type Probe struct {
	State          ProbeState
	SequenceID     int
	Latency        float64
	AverageLatency float64
	Loss           float64
}

func NewTarget(host string) *Target {
	return &Target{
		Host:            host,
		TargetState:     Unknown,
		probeCount:      0,
		failCount:       0,
		RecentFailCount: 0,
		Probe:           nil,
	}
}

func (t *Target) UpdateProbe(probe *Probe) {

	if t.Probe == nil || probe.SequenceID > t.Probe.SequenceID {
		t.Probe = probe
		t.probeCount = probe.SequenceID

		if probe.State == Failure {
			t.RecentFailCount++
			t.failCount++
		} else if probe.State == Success {
			t.RecentFailCount = 0
		}

	} else {
		/* Check if this probe and previous probe were a fail */
		if probe.State == Failure {
			t.failCount++
			if t.RecentFailCount != 0 {
				t.RecentFailCount++
			}
		}
	}

	t.updateTargetState()
}

func (t *Target) updateTargetState() {
	if t.RecentFailCount == 0 {
		t.TargetState = Online
	} else if t.RecentFailCount <= 3 {
		t.TargetState = Failing
	} else {
		t.TargetState = Offline
	}
	//log.Printf("%s, %+v", t.Host, t.TargetState)
}
