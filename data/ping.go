package data

import (
	"fmt"
	"sync"
	"time"

	probing "github.com/prometheus-community/pro-bing"
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

func (s *Status) String() string {
	return fmt.Sprintf("%s for %v %s", s.State, s.Duration(), s.TimeRange())
}

func (s *Status) Duration() time.Duration {
	if s.IsActive() {
		return time.Since(s.Start).Round(time.Second)
	} else {
		return s.End.Sub(s.Start).Round(time.Second)
	}
}

func (s *Status) TimeRange() string {
	if s.IsActive() {
		return fmt.Sprintf("(%s)", s.Start.Format(time.DateTime))

	}
	return fmt.Sprintf("(%s to %s)", s.Start.Format(time.DateTime), s.End.Format(time.DateTime))
}

func (s *Status) IsActive() bool {
	return s.End.IsZero()
}

func (s *Status) RelativeEnd() string {
	if s.IsActive() {
		return "now"
	}

	dur := time.Since(s.End)
	str := ""

	switch {
	case dur > 2*time.Hour:
		dur = dur.Round(time.Hour)
		str = fmt.Sprintf("%v hours", dur.Hours())
	case dur > 2*time.Minute:
		dur = dur.Round(time.Minute)
		str = fmt.Sprintf("%v minutes", dur.Minutes())
	default:
		dur = dur.Round(time.Second)
		str = fmt.Sprintf("%v seconds", dur.Seconds())
	}

	return fmt.Sprintf("%v ago", str)
}

var statusHistory []*Status
var currentStatus *Status
var successStart = time.Now()
var failStart = time.Now()
var m = sync.Mutex{}
var stateDuration = 10 * time.Second

func MonitorUptime(host string) {
	fmt.Printf("ping %s\n", host)
	pinger, err := probing.NewPinger(host)
	if err != nil {
		panic(err)
	}

	m.Lock()
	statusHistory = make([]*Status, 0)
	currentStatus = &Status{
		State: StatusStart,
		Start: time.Now(),
	}
	m.Unlock()

	c := make(chan time.Duration)

	pinger.OnRecv = func(pkt *probing.Packet) {
		c <- pkt.Rtt
	}

	go func() {
		err = pinger.Run()
		close(c)
		if err != nil {
			panic(err)
		}
	}()

	go func() {
		timeout := 1100 * time.Millisecond
		tick := time.NewTimer(timeout)
		for {
			select {
			case rtt, ok := <-c:
				if !ok {
					fmt.Println("done monitoring")
					return
				}

				if rtt < time.Second {
					m.Lock()
					failStart = time.Now()
					processState()
					tick = time.NewTimer(timeout)
					m.Unlock()
				} else {
					fmt.Printf("ignoring late response rtt %v", rtt)
				}

			case <-tick.C:
				m.Lock()
				successStart = time.Now()
				processState()
				tick = time.NewTimer(timeout)
				m.Unlock()
			}
		}
	}()
}

func processState() {
	if time.Since(successStart) > stateDuration && currentStatus.State != StatusUp {
		fmt.Println("Status is now UP")
		currentStatus.End = successStart
		currentStatus = &Status{
			Start: successStart.Add(time.Second),
			State: StatusUp,
		}
		statusHistory = append([]*Status{currentStatus}, statusHistory...)
	} else if time.Since(failStart) > stateDuration && currentStatus.State != StatusDown {
		fmt.Println("Status is now DOWN")
		currentStatus.End = failStart
		currentStatus = &Status{
			Start: failStart.Add(time.Second),
			State: StatusDown,
		}
		statusHistory = append([]*Status{currentStatus}, statusHistory...)
	}
}

func GetState() []*Status {
	m.Lock()
	s := make([]*Status, len(statusHistory))
	copy(s, statusHistory)
	m.Unlock()
	return s
}
