package data

import (
	"fmt"
	"time"
)

const (
	StatusUp    = "UP"
	StatusDown  = "DOWN"
	StatusStart = "START"
)

type Status struct {
	Start time.Time
	End   time.Time
	State string
}

func (s Status) String() string {
	timeRange := ""
	var dur time.Duration
	if s.End.IsZero() {
		timeRange = fmt.Sprintf("(%s)", s.Start.Format(time.DateTime))
		dur = time.Now().Sub(s.Start).Round(time.Second)
	} else {
		timeRange = fmt.Sprintf("(%s - %s)", s.Start.Format(time.DateTime), s.End.Format(time.DateTime))
		dur = s.End.Sub(s.Start).Round(time.Second)
	}

	return fmt.Sprintf("%s for %v %s", s.State, dur, timeRange)
}
