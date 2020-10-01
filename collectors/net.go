package collectors

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ovh/noderig/core"
	"github.com/shirou/gopsutil/net"
	log "github.com/sirupsen/logrus"
)

// Net collects network related metrics
type Net struct {
	interfaces []string
	mutex      sync.RWMutex
	sensision  bytes.Buffer
	level      uint8
	period     uint
}

// NewNet returns an initialized Net collector.
func NewNet(period uint, level uint8, opts interface{}) *Net {

	var ifaces []string

	if opts != nil {
		if options, ok := opts.(map[string]interface{}); ok {
			if val, ok := options["interfaces"]; ok {
				if ifs, ok := val.([]interface{}); ok {
					for _, v := range ifs {
						if s, ok := v.(string); ok {
							ifaces = append(ifaces, s)
						}
					}
				} else if ifs, ok := val.([]string); ok {
					ifaces = append(ifaces, ifs...)
				}
			}
		}
	}

	c := &Net{
		level:      level,
		period:     period,
		interfaces: ifaces,
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
func (c *Net) Metrics() *bytes.Buffer {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var res bytes.Buffer
	res.Write(c.sensision.Bytes())
	return &res
}

func (c *Net) scrape() error {
	counters, err := net.IOCounters(true)
	if err != nil {
		return err
	}

	var in, out uint64
	for _, cnt := range counters {
		if cnt.Name == "lo" {
			continue
		} else if c.interfaces != nil && !stringInSlice(cnt.Name, c.interfaces) {
			continue
		}
		in += cnt.BytesRecv
		out += cnt.BytesSent
	}
	in /= uint64(c.period / 1000)
	out /= uint64(c.period / 1000)

	// protect consistency
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.sensision.Reset()

	class := "os.net.bytes"
	now := time.Now().UnixNano() / 1000

	if c.level == 1 {
		gts := core.GetSeriesOutput(now, class, fmt.Sprintf("{%v}", core.ToLabels("direction", "in")), in)
		c.sensision.WriteString(gts)

		gts = core.GetSeriesOutput(now, class, fmt.Sprintf("{%v}", core.ToLabels("direction", "out")), out)
		c.sensision.WriteString(gts)
	}

	if c.level > 1 {
		for _, cnt := range counters {
			if cnt.Name == "lo" {
				continue
			} else if c.interfaces != nil && !stringInSlice(cnt.Name, c.interfaces) {
				continue
			}

			gts := core.GetSeriesOutput(now, class,
				fmt.Sprintf("{%v,%v}", core.ToLabels("iface", cnt.Name), core.ToLabels("direction", "in")), cnt.BytesRecv)
			c.sensision.WriteString(gts)

			gts = core.GetSeriesOutput(now, class,
				fmt.Sprintf("{%v,%v}", core.ToLabels("iface", cnt.Name), core.ToLabels("direction", "out")), cnt.BytesSent)
			c.sensision.WriteString(gts)
		}
	}

	if c.level > 2 {
		for _, cnt := range counters {
			if cnt.Name == "lo" {
				continue
			} else if c.interfaces != nil && !stringInSlice(cnt.Name, c.interfaces) {
				continue
			}

			gts := core.GetSeriesOutput(now, "os.net.packets",
				fmt.Sprintf("{%v,%v}", core.ToLabels("iface", cnt.Name), core.ToLabels("direction", "in")), cnt.PacketsRecv)
			c.sensision.WriteString(gts)

			gts = core.GetSeriesOutput(now, "os.net.packets",
				fmt.Sprintf("{%v,%v}", core.ToLabels("iface", cnt.Name), core.ToLabels("direction", "out")), cnt.PacketsSent)
			c.sensision.WriteString(gts)

			gts = core.GetSeriesOutput(now, "os.net.errs",
				fmt.Sprintf("{%v,%v}", core.ToLabels("iface", cnt.Name), core.ToLabels("direction", "in")), cnt.Errin)
			c.sensision.WriteString(gts)

			gts = core.GetSeriesOutput(now, "os.net.errs",
				fmt.Sprintf("{%v,%v}", core.ToLabels("iface", cnt.Name), core.ToLabels("direction", "out")), cnt.Errout)
			c.sensision.WriteString(gts)

			gts = core.GetSeriesOutput(now, "os.net.dropped",
				fmt.Sprintf("{%v,%v}", core.ToLabels("iface", cnt.Name), core.ToLabels("direction", "in")), cnt.Dropin)
			c.sensision.WriteString(gts)

			gts = core.GetSeriesOutput(now, "os.net.dropped",
				fmt.Sprintf("{%v,%v}", core.ToLabels("iface", cnt.Name), core.ToLabels("direction", "out")), cnt.Dropout)
			c.sensision.WriteString(gts)
		}
	}

	return nil
}

func stringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}

		if strings.HasPrefix(strings.TrimSpace(v), "~") {

			matched, _ := regexp.MatchString(strings.Replace(strings.TrimSpace(v), "~", "", 1), str)

			if matched {
				return true
			}
		}

	}
	return false
}
