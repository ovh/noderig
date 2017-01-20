package collectors

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/shirou/gopsutil/disk"
)

// Disk collects disk related metrics
type Disk struct {
	counters map[string]disk.IOCountersStat

	mutex     sync.RWMutex
	sensision bytes.Buffer
	level     uint8
	period    uint
}

// NewDisk returns an initialized Disk collector.
func NewDisk(period uint, level uint8) *Disk {
	c := &Disk{
		level:  level,
		period: period,
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
func (c *Disk) Metrics() *bytes.Buffer {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var res bytes.Buffer
	res.Write(c.sensision.Bytes())
	return &res
}

func (c *Disk) scrape() error {
	counters, err := disk.IOCounters()
	if err != nil {
		return err
	}

	if c.counters == nil {
		c.counters = counters
		return nil
	}

	parts, err := disk.Partitions(false)
	if err != nil {
		return err
	}

	dev := make(map[string]disk.UsageStat)
	for _, p := range parts {
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}
		dev[p.Device] = *usage
	}

	// protect consistency
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.sensision.Reset()

	now := fmt.Sprintf("%v// os.disk.fs", time.Now().UnixNano()/1000)

	for path, usage := range dev {
		gts := fmt.Sprintf("%v{disk=%v} %v\n", now, path, usage.UsedPercent)
		c.sensision.WriteString(gts)
	}

	if c.level > 1 {
		for path, usage := range dev {
			gts := fmt.Sprintf("%v.used{disk=%v} %v\n", now, path, usage.Used)
			c.sensision.WriteString(gts)
			gts = fmt.Sprintf("%v.total{disk=%v} %v\n", now, path, usage.Total)
			c.sensision.WriteString(gts)

			if c.level > 2 {
				for name, stats := range counters {
					if strings.HasSuffix(path, name) {
						gts = fmt.Sprintf("%v.bytes.read{disk=%v} %v\n", now, path, (stats.ReadBytes - c.counters[name].ReadBytes) / uint64(c.period/1000))
						c.sensision.WriteString(gts)
						gts = fmt.Sprintf("%v.bytes.write{disk=%v} %v\n", now, path, (stats.WriteBytes - c.counters[name].WriteBytes) / uint64(c.period/1000))
						c.sensision.WriteString(gts)

						if c.level > 3 {
							gts = fmt.Sprintf("%v.io.read{disk=%v} %v\n", now, path, (stats.ReadCount - c.counters[name].ReadCount) / uint64(c.period/1000))
							c.sensision.WriteString(gts)
							gts = fmt.Sprintf("%v.io.write{disk=%v} %v\n", now, path, (stats.WriteCount - c.counters[name].WriteCount) / uint64(c.period/1000))
							c.sensision.WriteString(gts)
						}
					}
				}
			}
		}
	}

	c.counters = counters
	return nil
}
