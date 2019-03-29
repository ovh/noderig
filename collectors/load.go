package collectors

import (
	"bytes"
	"sync"
	"time"

	"github.com/ovh/noderig/core"
	"github.com/shirou/gopsutil/load"
	log "github.com/sirupsen/logrus"
)

// Load collects load related metrics
type Load struct {
	mutex     sync.RWMutex
	sensision bytes.Buffer
	level     uint8
}

// NewLoad returns an initialized Load collector.
func NewLoad(period uint, level uint8) *Load {
	c := &Load{
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
func (c *Load) Metrics() *bytes.Buffer {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var res bytes.Buffer
	res.Write(c.sensision.Bytes())
	return &res
}

func (c *Load) scrape() error {
	avg, err := load.Avg()
	if err != nil {
		return err
	}

	// protect consistency
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Delete previous metrics
	c.sensision.Reset()

	class := "os.load"

	now := time.Now().UnixNano() / 1000

	gts := core.GetSeriesOutput(now, class+"1", "{}", avg.Load1)
	c.sensision.WriteString(gts)

	if c.level > 1 {
		gts := core.GetSeriesOutput(now, class+"5", "{}", avg.Load5)
		c.sensision.WriteString(gts)

		gts = core.GetSeriesOutput(now, class+"15", "{}", avg.Load15)
		c.sensision.WriteString(gts)
	}

	return nil
}
