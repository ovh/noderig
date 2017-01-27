package collectors

import (
	"bytes"
	"fmt"
	"reflect"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/shirou/gopsutil/net"
)

// Net collects network related metrics
type Net struct {
	counters   []net.IOCountersStat
	interfaces []string
	mutex      sync.RWMutex
	sensision  bytes.Buffer
	level      uint8
	period     uint
}

// NewNet returns an initialized Net collector.
func NewNet(period uint, level uint8, opts interface{}) *Net {

	var options map[string]interface{}
	var ifaces []string

	if opts != nil {
		options = opts.(map[string]interface{})
		if val, ok := options["interfaces"]; ok {
			if reflect.TypeOf(val).Kind() == reflect.Slice {
				ifs := val.([]interface{})
				for _, v := range ifs {
					ifaces = append(ifaces, v.(string))
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

	if len(c.counters) == 0 { // init
		c.counters = counters
		return nil
	}

	var in, out uint64
	for i, cnt := range counters {
		if cnt.Name == "lo" {
			continue
		} else if c.interfaces != nil && !stringInSlice(cnt.Name, c.interfaces) {
			continue
		}
		in += cnt.BytesRecv - c.counters[i].BytesRecv
		out += cnt.BytesSent - c.counters[i].BytesSent
	}
	in = in / uint64(c.period/1000)
	out = out / uint64(c.period/1000)

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
		for i, cnt := range counters {
			if cnt.Name == "lo" {
				continue
			} else if c.interfaces != nil && !stringInSlice(cnt.Name, c.interfaces) {
				continue
			}
			gts := fmt.Sprintf("%v os.net.bytes{iface=%v,direction=in} %v\n", now, cnt.Name, (cnt.BytesRecv-c.counters[i].BytesRecv)/uint64(c.period/1000))
			c.sensision.WriteString(gts)

			gts = fmt.Sprintf("%v os.net.bytes{iface=%v,direction=out} %v\n", now, cnt.Name, (cnt.BytesSent-c.counters[i].BytesSent)/uint64(c.period/1000))
			c.sensision.WriteString(gts)
		}
	}

	if c.level > 2 {
		for i, cnt := range counters {
			if cnt.Name == "lo" {
				continue
			} else if c.interfaces != nil && !stringInSlice(cnt.Name, c.interfaces) {
				continue
			}
			gts := fmt.Sprintf("%v os.net.packets{iface=%v,direction=in} %v\n", now, cnt.Name, (cnt.PacketsRecv-c.counters[i].PacketsRecv)/uint64(c.period/1000))
			c.sensision.WriteString(gts)

			gts = fmt.Sprintf("%v os.net.packets{iface=%v,direction=out} %v\n", now, cnt.Name, (cnt.PacketsSent-c.counters[i].PacketsSent)/uint64(c.period/1000))
			c.sensision.WriteString(gts)

			gts = fmt.Sprintf("%v os.net.errs{iface=%v,direction=in} %v\n", now, cnt.Name, (cnt.Errin-c.counters[i].Errin)/uint64(c.period/1000))
			c.sensision.WriteString(gts)

			gts = fmt.Sprintf("%v os.net.errs{iface=%v,direction=out} %v\n", now, cnt.Name, (cnt.Errout-c.counters[i].Errout)/uint64(c.period/1000))
			c.sensision.WriteString(gts)

			gts = fmt.Sprintf("%v os.net.dropped{iface=%v,direction=in} %v\n", now, cnt.Name, (cnt.Dropin-c.counters[i].Dropin)/uint64(c.period/1000))
			c.sensision.WriteString(gts)

			gts = fmt.Sprintf("%v os.net.dropped{iface=%v,direction=out} %v\n", now, cnt.Name, (cnt.Dropout-c.counters[i].Dropout)/uint64(c.period/1000))
			c.sensision.WriteString(gts)
		}
	}

	c.counters = counters

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
