package collectors

import (
	"bytes"
	"fmt"
	"sync"
	"time"

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

	now := fmt.Sprintf("%v//", time.Now().UnixNano()/1000)

	if c.level == 1 {
		gts := fmt.Sprintf("%v os.net.bytes{direction=in} %v\n", now, in)
		c.sensision.WriteString(gts)

		gts = fmt.Sprintf("%v os.net.bytes{direction=out} %v\n", now, out)
		c.sensision.WriteString(gts)
	}

	if c.level > 1 {
		for _, cnt := range counters {
			if cnt.Name == "lo" {
				continue
			} else if c.interfaces != nil && !stringInSlice(cnt.Name, c.interfaces) {
				continue
			}
			gts := fmt.Sprintf("%v os.net.bytes{iface=%v,direction=in} %v\n", now, cnt.Name, cnt.BytesRecv)
			c.sensision.WriteString(gts)

			gts = fmt.Sprintf("%v os.net.bytes{iface=%v,direction=out} %v\n", now, cnt.Name, cnt.BytesSent)
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
			gts := fmt.Sprintf("%v os.net.packets{iface=%v,direction=in} %v\n", now, cnt.Name, cnt.PacketsRecv)
			c.sensision.WriteString(gts)

			gts = fmt.Sprintf("%v os.net.packets{iface=%v,direction=out} %v\n", now, cnt.Name, cnt.PacketsSent)
			c.sensision.WriteString(gts)

			gts = fmt.Sprintf("%v os.net.errs{iface=%v,direction=in} %v\n", now, cnt.Name, cnt.Errin)
			c.sensision.WriteString(gts)

			gts = fmt.Sprintf("%v os.net.errs{iface=%v,direction=out} %v\n", now, cnt.Name, cnt.Errout)
			c.sensision.WriteString(gts)

			gts = fmt.Sprintf("%v os.net.dropped{iface=%v,direction=in} %v\n", now, cnt.Name, cnt.Dropin)
			c.sensision.WriteString(gts)

			gts = fmt.Sprintf("%v os.net.dropped{iface=%v,direction=out} %v\n", now, cnt.Name, cnt.Dropout)
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
	}
	return false
}
