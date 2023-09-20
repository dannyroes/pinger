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
	if s.End.IsZero() {
		return time.Now().Sub(s.Start).Round(time.Second)
	} else {
		return s.End.Sub(s.Start).Round(time.Second)
	}
}

func (s *Status) TimeRange() string {
	if s.End.IsZero() {
		return fmt.Sprintf("(%s)", s.Start.Format(time.DateTime))

	}
	return fmt.Sprintf("(%s - %s)", s.Start.Format(time.DateTime), s.End.Format(time.DateTime))
}

var statusHistory []*Status
var currentStatus *Status
var successCount = 0
var failCount = 0
var m = sync.Mutex{}

func MonitorUptime(host string) {
	fmt.Printf("ping %s\n", host)
	pinger, err := probing.NewPinger(host)
	if err != nil {
		panic(err)
	}

	m.Lock()
	statusHistory = make([]*Status, 1)
	currentStatus = &Status{
		State: StatusStart,
		Start: time.Now(),
	}
	statusHistory[0] = currentStatus
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

				fmt.Println("got response")
				if rtt < time.Second {
					m.Lock()
					failCount = 0
					successCount++
					processState()
					tick = time.NewTimer(timeout)
					m.Unlock()
				} else {
					fmt.Printf("ignoring response rtt %v", rtt)
				}

			case <-tick.C:
				fmt.Println("missed response")
				m.Lock()
				failCount++
				successCount = 0
				processState()
				tick = time.NewTimer(timeout)
				m.Unlock()
			}
		}
	}()
}

func processState() {
	if successCount >= 5 && currentStatus.State != StatusUp {
		fmt.Println("Status is now UP")
		currentStatus.End = time.Now().Add(-1 * time.Second)
		currentStatus = &Status{
			Start: time.Now(),
			State: StatusUp,
		}
		statusHistory = append([]*Status{currentStatus}, statusHistory...)
	} else if failCount >= 5 && currentStatus.State != StatusDown {
		fmt.Println("Status is now DOWN")
		currentStatus.End = time.Now().Add(-1 * time.Second)
		currentStatus = &Status{
			Start: time.Now(),
			State: StatusDown,
		}
		statusHistory = append([]*Status{currentStatus}, statusHistory...)
	}
}

func GetState() []*Status {
	m.Lock()
	s := make([]*Status, len(statusHistory))
	for i, v := range statusHistory {
		s[i] = v
	}
	m.Unlock()
	return s
}
