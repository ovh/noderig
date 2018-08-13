package collectors

import (
	"bytes"
	"fmt"
	"path"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/shirou/gopsutil/disk"
)

// Disk collects disk related metrics
type Disk struct {
	mutex        sync.RWMutex
	sensision    bytes.Buffer
	level        uint8
	period       uint
	allowedDisks map[string]struct{}
}

// NewDisk returns an initialized Disk collector.
func NewDisk(period uint, level uint8, opts interface{}) *Disk {

	allowedDisks := map[string]struct{}{}
	if opts != nil {
		if options, ok := opts.(map[string]interface{}); ok {
			if val, ok := options["names"]; ok {
				if diskNames, ok := val.([]interface{}); ok {
					for _, v := range diskNames {
						if diskName, ok := v.(string); ok {
							allowedDisks[diskName] = struct{}{}
						}
					}
				}
			}
		}
	}

	c := &Disk{
		level:        level,
		period:       period,
		allowedDisks: allowedDisks,
	}

	if level > 0 {
		tick := time.Tick(time.Duration(period) * time.Millisecond)
		go func() {
			for range tick {
				if err := c.scrape(); err != nil {
					log.Error(err)
				}
			}
		}()
	}

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

	parts, err := disk.Partitions(false)
	if err != nil {
		return err
	}

	dev := make(map[string]disk.UsageStat)
	for _, p := range parts {
		if _, ok := dev[p.Device]; ok {
			continue
		}
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

	for diskPath, usage := range dev {
		if len(c.allowedDisks) > 0 {
			_, diskName := path.Split(diskPath) // return "sda1" from "/dev/sda1"
			if _, allowed := c.allowedDisks[diskName]; !allowed {
				log.Debug("Disk " + diskPath + " is blacklisted, skip it")
				continue
			}
		}
		gts := fmt.Sprintf("%v{disk=%v}{mount=%v} %v\n", now, diskPath, usage.Path, usage.UsedPercent)
		c.sensision.WriteString(gts)
	}

	if c.level > 1 {
		for diskPath, usage := range dev {
			if len(c.allowedDisks) > 0 {
				_, diskName := path.Split(diskPath) // return "sda1" from "/dev/sda1"
				if _, allowed := c.allowedDisks[diskName]; !allowed {
					log.Debug("Disk " + diskPath + " is blacklisted, skip it")
					continue
				}
			}

			gts := fmt.Sprintf("%v.used{disk=%v}{mount=%v} %v\n", now, diskPath, usage.Path, usage.Used)
			c.sensision.WriteString(gts)
			gts = fmt.Sprintf("%v.total{disk=%v}{mount=%v} %v\n", now, diskPath, usage.Path, usage.Total)
			c.sensision.WriteString(gts)
			gts = fmt.Sprintf("%v.inodes.used{disk=%v}{mount=%v} %v\n", now, diskPath, usage.Path, usage.InodesUsed)
			c.sensision.WriteString(gts)
			gts = fmt.Sprintf("%v.inodes.total{disk=%v}{mount=%v} %v\n", now, diskPath, usage.Path, usage.InodesTotal)
			c.sensision.WriteString(gts)
		}
	}

	if c.level > 2 {
		for name, stats := range counters {
			if len(c.allowedDisks) > 0 {
				if _, allowed := c.allowedDisks[name]; !allowed {
					log.Debug("Disk name " + name + " is blacklisted, skip it")
					continue
				}
			}
			gts := fmt.Sprintf("%v.bytes.read{name=%v} %v\n", now, name, stats.ReadBytes)
			c.sensision.WriteString(gts)
			gts = fmt.Sprintf("%v.bytes.write{name=%v} %v\n", now, name, stats.WriteBytes)
			c.sensision.WriteString(gts)

			if c.level > 3 {
				gts = fmt.Sprintf("%v.io.read{name=%v} %v\n", now, name, stats.ReadCount)
				c.sensision.WriteString(gts)
				gts = fmt.Sprintf("%v.io.write{name=%v} %v\n", now, name, stats.WriteCount)
				c.sensision.WriteString(gts)

				if c.level > 4 {
					gts = fmt.Sprintf("%v.io.read.ms{name=%v} %v\n", now, name, stats.ReadTime)
					c.sensision.WriteString(gts)
					gts = fmt.Sprintf("%v.io.write.ms{name=%v} %v\n", now, name, stats.WriteTime)
					c.sensision.WriteString(gts)
					gts = fmt.Sprintf("%v.io{name=%v} %v\n", now, name, stats.IopsInProgress)
					c.sensision.WriteString(gts)
					gts = fmt.Sprintf("%v.io.ms{name=%v} %v\n", now, name, stats.IoTime)
					c.sensision.WriteString(gts)
					gts = fmt.Sprintf("%v.io.weighted.ms{name=%v} %v\n", now, name, stats.WeightedIO)
					c.sensision.WriteString(gts)
				}
			}
		}
	}

	return nil
}
