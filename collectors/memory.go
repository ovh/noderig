package collectors

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/shirou/gopsutil/mem"
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

	ticker := time.NewTicker(time.Duration(period) * time.Millisecond)

	go func() {
		for {
			select {
			case <-ticker.C:
				if err := c.scrape(); err != nil {
					log.Error(err)
				}
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

	now := fmt.Sprintf("%v//", time.Now().UnixNano()/1000)

	gts := fmt.Sprintf("%v os.mem{} %v\n", now, virt.UsedPercent)
	c.sensision.WriteString(gts)

	gts = fmt.Sprintf("%v os.swap{} %v\n", now, swap.UsedPercent)
	c.sensision.WriteString(gts)

	if c.level > 1 {
		gts := fmt.Sprintf("%v os.mem.used{} %v\n", now, virt.Used)
		c.sensision.WriteString(gts)
		gts = fmt.Sprintf("%v os.mem.total{} %v\n", now, virt.Total)
		c.sensision.WriteString(gts)
		gts = fmt.Sprintf("%v os.swap.used{} %v\n", now, swap.Used)
		c.sensision.WriteString(gts)
		gts = fmt.Sprintf("%v os.swap.total{} %v\n", now, swap.Total)
		c.sensision.WriteString(gts)
	}

	return nil
}
