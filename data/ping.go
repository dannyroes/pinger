package data

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

const (
	StatusUp    = "UP"
	StatusDown  = "DOWN"
	StatusStart = "START"
)

var (
	OutputCadence = 30 * time.Second
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
	slog.Debug("Starting ping monitor", "host", host)
	pinger, err := probing.NewPinger(host)
	if err != nil {
		panic(err)
	}

	m.Lock()
	if len(statusHistory) == 0 {
		statusHistory = make([]*Status, 0)
		currentStatus = &Status{
			State: StatusStart,
			Start: time.Now(),
		}
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
					slog.Debug("Ending monitor")
					return
				}

				if rtt < time.Second {
					m.Lock()
					failStart = time.Now()
					processState()
					tick = time.NewTimer(timeout)
					m.Unlock()
				} else {
					slog.Debug("Ignoring late response", "rtt", rtt)
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
		slog.Info("Status is now UP")
		currentStatus.End = successStart
		currentStatus = &Status{
			Start: successStart.Add(time.Second),
			State: StatusUp,
		}
		statusHistory = append([]*Status{currentStatus}, statusHistory...)
	} else if time.Since(failStart) > stateDuration && currentStatus.State != StatusDown {
		slog.Info("Status is now DOWN")
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

func OutputState(ctx context.Context, file string) error {
	// Confirm we can actually access the given file
	h, err := os.Create(file)
	if err != nil {
		return err
	}
	h.Close()
	go func() {
		for {
			select {
			case <-time.After(OutputCadence):
				m.Lock()
				out, err := json.Marshal(statusHistory)
				m.Unlock()
				if err != nil {
					slog.Error("Couldn't generate output", "error", err)
					return
				}

				if err = writeOutput(file, out); err != nil {
					slog.Error("Couldn't write output", "error", err)
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func writeOutput(file string, out []byte) error {
	h, err := os.Create(file)
	if err != nil {
		return err
	}

	defer h.Close()
	_, err = h.Write(out)
	if err != nil {
		return err
	}
	return nil
}

func InputState(file string) error {
	r, err := os.Open(file)
	if err != nil {
		return err
	}

	defer r.Close()

	var status []*Status
	d := json.NewDecoder(r)
	err = d.Decode(&status)
	if err != nil {
		slog.Error("Couldn't parse input", "error", err)
		return err
	}

	m.Lock()
	statusHistory = status
	currentStatus = statusHistory[0]
	m.Unlock()

	return nil
}
