package collectors

import (
	"bytes"
	"sync"
	"time"

	"github.com/ovh/noderig/core"
	"github.com/shirou/gopsutil/mem"
	log "github.com/sirupsen/logrus"
)

// Memory collects memory related metrics
type Memory struct {
	mutex     sync.RWMutex
	sensision bytes.Buffer
	level     uint8
}

// NewMemory returns an initialized Memory collector.
func NewMemory(period uint, level uint8) *Memory {
	c := &Memory{
		level: level,
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
func (c *Memory) Metrics() *bytes.Buffer {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var res bytes.Buffer
	res.Write(c.sensision.Bytes())
	return &res
}

func (c *Memory) scrape() error {
	virt, err := mem.VirtualMemory()
	if err != nil {
		return err
	}

	swap, err := mem.SwapMemory()
	if err != nil {
		return err
	}

	// protect consistency
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Delete previous metrics
	c.sensision.Reset()

	memClass := "os.mem"
	swapClass := "os.swap"

	now := time.Now().UnixNano() / 1000

	gts := core.GetSeriesOutput(now, memClass, "{}", virt.UsedPercent)
	c.sensision.WriteString(gts)

	gts = core.GetSeriesOutput(now, swapClass, "{}", swap.UsedPercent)
	c.sensision.WriteString(gts)

	if c.level > 1 {
		gts := core.GetSeriesOutput(now, memClass+".used", "{}", virt.Used)
		c.sensision.WriteString(gts)
		gts = core.GetSeriesOutput(now, memClass+".total", "{}", virt.Total)
		c.sensision.WriteString(gts)
		gts = core.GetSeriesOutput(now, swapClass+".used", "{}", swap.Used)
		c.sensision.WriteString(gts)
		gts = core.GetSeriesOutput(now, swapClass+".total", "{}", swap.Total)
		c.sensision.WriteString(gts)
	}
	if c.level > 2 {
		gts := core.GetSeriesOutput(now, memClass+".free", "{}", virt.Free)
		c.sensision.WriteString(gts)
		gts = core.GetSeriesOutput(now, memClass+".buffers", "{}", virt.Buffers)
		c.sensision.WriteString(gts)
		gts = core.GetSeriesOutput(now, memClass+".cached", "{}", virt.Cached)
		c.sensision.WriteString(gts)
	}

	return nil
}
