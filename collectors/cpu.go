package collectors

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/shirou/gopsutil/cpu"
)

// CPU collects cpu related metrics
type CPU struct {
	times []cpu.TimesStat

	mutex     sync.RWMutex
	sensision bytes.Buffer
	level     uint8
}

// NewCPU returns an initialized CPU collector.
func NewCPU(period uint, level uint8) *CPU {
	c := &CPU{
		level: level,
	}

	if level == 0 {
		return c
	}

	tick := time.Tick(time.Duration(period) * time.Millisecond)
	go func() {
		for range tick {
			if err := c.scrape(); err != nil {
				log.Error(err)
			}
		}
	}()

	return c
}

// Metrics delivers metrics.
func (c *CPU) Metrics() *bytes.Buffer {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var res bytes.Buffer
	res.Write(c.sensision.Bytes())
	return &res
}

// https://github.com/Leo-G/DevopsWiki/wiki/How-Linux-CPU-Usage-Time-and-Percentage-is-calculated
func (c *CPU) scrape() error {
	times, err := cpu.Times(true)
	if err != nil {
		return err
	}

	if len(c.times) == 0 { // init
		c.times = times
		return nil
	}

	idles := make([]float64, len(times))
	systems := make([]float64, len(times))
	users := make([]float64, len(times))
	iowaits := make([]float64, len(times))
	for i, t := range times {
		dt := t.Total() - c.times[i].Total()
		idles[i] = (t.Idle - c.times[i].Idle) / dt
		systems[i] = (t.System - c.times[i].System) / dt
		users[i] = (t.User - c.times[i].User) / dt
		iowaits[i] = (t.Iowait - c.times[i].Iowait) / dt
	}

	global := 0.0
	for _, v := range idles {
		global += v
	}
	global = (1.0 - global/float64(len(idles))) * 100

	c.times = times

	// protect consistency
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Delete previous metrics
	c.sensision.Reset()

	class := fmt.Sprintf("%v// os.cpu", time.Now().UnixNano()/1000)

	gts := fmt.Sprintf("%v{} %v\n", class, global)
	c.sensision.WriteString(gts)

	if c.level == 2 {
		iowait := 0.0
		for _, v := range iowaits {
			iowait += v
		}
		iowait = iowait / float64(len(iowaits)) * 100
		gts := fmt.Sprintf("%v.iowait{} %v\n", class, iowait)
		c.sensision.WriteString(gts)

		user := 0.0
		for _, v := range users {
			user += v
		}
		user = user / float64(len(users)) * 100
		gts = fmt.Sprintf("%v.user{} %v\n", class, user)
		c.sensision.WriteString(gts)

		system := 0.0
		for _, v := range systems {
			system += v
		}
		system = system / float64(len(systems)) * 100
		gts = fmt.Sprintf("%v.systems{} %v\n", class, system)
		c.sensision.WriteString(gts)
	}

	if c.level == 3 {
		for i, v := range iowaits {
			gts := fmt.Sprintf("%v.iowait{chore=%v} %v\n", class, i, v*100)
			c.sensision.WriteString(gts)
		}

		for i, v := range users {
			gts := fmt.Sprintf("%v.user{chore=%v} %v\n", class, i, v*100)
			c.sensision.WriteString(gts)
		}

		for i, v := range systems {
			gts := fmt.Sprintf("%v.systems{chore=%v} %v\n", class, i, v*100)
			c.sensision.WriteString(gts)
		}
	}

	return nil
}
