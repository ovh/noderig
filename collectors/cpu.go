package collectors

import (
	"bytes"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/ovh/noderig/core"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/host"
	log "github.com/sirupsen/logrus"
)

// CPU collects cpu related metrics
type CPU struct {
	times []cpu.TimesStat

	mutex     sync.RWMutex
	sensision bytes.Buffer
	level     uint8
	modules   []string
}

// NewCPU returns an initialized CPU collector.
func NewCPU(period uint, level uint8, modules []string) *CPU {
	c := &CPU{
		level:   level,
		modules: modules,
	}

	if level == 0 {
		return c
	}

	tick := time.NewTicker(time.Duration(period) * time.Millisecond)
	go func() {
		for range tick.C {
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
	nices := make([]float64, len(times))
	irqs := make([]float64, len(times))
	for i, t := range times {
		dt := t.Total() - c.times[i].Total()
		idles[i] = (t.Idle - c.times[i].Idle) / dt
		systems[i] = (t.System - c.times[i].System) / dt
		users[i] = (t.User - c.times[i].User) / dt
		iowaits[i] = (t.Iowait - c.times[i].Iowait) / dt
		nices[i] = (t.Nice - c.times[i].Nice) / dt
		irqs[i] = (t.Irq - c.times[i].Irq) / dt
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

	class := "os.cpu"

	now := time.Now().UnixNano() / 1000
	gts := core.GetSeriesOutput(now, class, "{}", global)
	c.sensision.WriteString(gts)

	if c.level == 2 {
		iowait := 0.0
		for _, v := range iowaits {
			iowait += v
		}
		iowait = iowait / float64(len(iowaits)) * 100
		gts := core.GetSeriesOutput(now, class+".iowait", "{}", iowait)
		c.sensision.WriteString(gts)

		user := 0.0
		for _, v := range users {
			user += v
		}
		user = user / float64(len(users)) * 100
		gts = core.GetSeriesOutput(now, class+".user", "{}", user)
		c.sensision.WriteString(gts)

		system := 0.0
		for _, v := range systems {
			system += v
		}
		system = system / float64(len(systems)) * 100
		gts = core.GetSeriesOutput(now, class+".systems", "{}", system)
		c.sensision.WriteString(gts)

		nice := 0.0
		for _, v := range nices {
			nice += v
		}
		nice = nice / float64(len(nices)) * 100
		gts = core.GetSeriesOutput(now, class+".nice", "{}", nice)
		c.sensision.WriteString(gts)

		irq := 0.0
		for _, v := range irqs {
			irq += v
		}
		irq = irq / float64(len(irqs)) * 100
		gts = core.GetSeriesOutput(now, class+".irq", "{}", irq)
		c.sensision.WriteString(gts)
	}

	if c.level == 3 {
		for i, v := range iowaits {
			gts := core.GetSeriesOutput(now,
				fmt.Sprintf("%v.iowait", class),
				fmt.Sprintf("{%v}", core.ToLabels("chore", i)), v*100)
			c.sensision.WriteString(gts)
		}

		for i, v := range users {
			gts := core.GetSeriesOutput(now, fmt.Sprintf("%v.user", class),
				fmt.Sprintf("{%v}", core.ToLabels("chore", i)), v*100)
			c.sensision.WriteString(gts)
		}

		for i, v := range systems {
			gts := core.GetSeriesOutput(now, fmt.Sprintf("%v.systems", class),
				fmt.Sprintf("{%v}", core.ToLabels("chore", i)), v*100)
			c.sensision.WriteString(gts)
		}

		for i, v := range nices {
			gts := core.GetSeriesOutput(now, fmt.Sprintf("%v.nice", class),
				fmt.Sprintf("{%v}", core.ToLabels("chore", i)), v*100)
			c.sensision.WriteString(gts)
		}

		for i, v := range irqs {
			gts := core.GetSeriesOutput(now, fmt.Sprintf("%v.irq", class),
				fmt.Sprintf("{%v}", core.ToLabels("chore", i)), v*100)
			c.sensision.WriteString(gts)
		}
	}

	for _, m := range c.modules {
		switch m {
		case "temperature":
			temps, err := host.SensorsTemperatures()
			if err != nil {
				return err
			}

			platform, _, _, err := host.PlatformInformation()
			if err != nil {
				return err
			}

			// Get CPU temperature
			re := regexp.MustCompile("^coretemp_packageid([0-9+])_input$")
			if platform == "darwin" {
				re = regexp.MustCompile("^TC([0-9+])P$")
			}

			for _, temp := range temps {
				submatches := re.FindStringSubmatch(temp.SensorKey)
				if len(submatches) > 0 {

					gts := core.GetSeriesOutput(now, fmt.Sprintf("%v.temperature", class),
						fmt.Sprintf("{%v}", core.ToLabels("id", submatches[1])), temp.Temperature)
					c.sensision.WriteString(gts)
				}
			}

			if c.level >= 2 {
				// Get per core temp
				re := regexp.MustCompile("^coretemp_core([0-9+])_input$")
				if platform == "darwin" {
					re = regexp.MustCompile("^TC([0-9+])C$")
				}

				for _, temp := range temps {
					submatches := re.FindStringSubmatch(temp.SensorKey)
					if len(submatches) > 0 {
						gts := core.GetSeriesOutput(now, fmt.Sprintf("%v.temperature", class),
							fmt.Sprintf("{%v}", core.ToLabels("core", submatches[1])), temp.Temperature)
						c.sensision.WriteString(gts)
					}
				}
			}
		default:
			log.Warnf("[CPU] module '%s' not found", m)
		}
	}

	return nil
}
