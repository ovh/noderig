package collectors

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/shirou/gopsutil/load"
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

	now := fmt.Sprintf("%v//", time.Now().UnixNano()/1000)

	gts := fmt.Sprintf("%v os.load1{} %v\n", now, avg.Load1)
	c.sensision.WriteString(gts)

	if c.level > 1 {
		gts := fmt.Sprintf("%v os.load5{} %v\n", now, avg.Load5)
		c.sensision.WriteString(gts)

		gts = fmt.Sprintf("%v os.load15{} %v\n", now, avg.Load15)
		c.sensision.WriteString(gts)
	}

	return nil
}
