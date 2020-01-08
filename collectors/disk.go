package collectors

import (
	"bytes"
	"fmt"
	"path"
	"sync"
	"time"

	"github.com/ovh/noderig/core"
	"github.com/shirou/gopsutil/disk"
	log "github.com/sirupsen/logrus"
)

// Disk collects disk related metrics
type Disk struct {
	mutex        sync.RWMutex
	sensision    bytes.Buffer
	level        uint8
	period       uint
	allowedDisks []string
}

// NewDisk returns an initialized Disk collector.
func NewDisk(period uint, level uint8, opts interface{}) *Disk {

	var allowedDisks []string
	if opts != nil {
		if options, ok := opts.(map[string]interface{}); ok {
			if val, ok := options["names"]; ok {
				if diskNames, ok := val.([]interface{}); ok {
					for _, v := range diskNames {
						if diskName, ok := v.(string); ok {
							allowedDisks = append(allowedDisks, diskName)
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
		tick := time.NewTicker(time.Duration(period) * time.Millisecond)
		go func() {
			for range tick.C {
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

	now := time.Now().UnixNano() / 1000
	class := "os.disk.fs"

	for diskPath, usage := range dev {
		if len(c.allowedDisks) > 0 {
			_, diskName := path.Split(diskPath) // return "sda1" from "/dev/sda1"
			if !stringInSlice(diskName, c.allowedDisks) {
				log.Debug("Disk " + diskPath + " is blacklisted, skip it")
				continue
			}
		}

		gts := core.GetSeriesOutputAttributes(now, class,
			fmt.Sprintf("{%v}", core.ToLabels("disk", diskPath)), fmt.Sprintf("{mount=%v}", usage.Path), usage.UsedPercent)
		c.sensision.WriteString(gts)
	}

	if c.level > 1 {
		for diskPath, usage := range dev {
			if len(c.allowedDisks) > 0 {
				_, diskName := path.Split(diskPath) // return "sda1" from "/dev/sda1"
				if !stringInSlice(diskName, c.allowedDisks) {
					log.Debug("Disk " + diskPath + " is blacklisted, skip it")
					continue
				}
			}

			gts := core.GetSeriesOutputAttributes(now, class+".used",
				fmt.Sprintf("{%v}", core.ToLabels("disk", diskPath)), fmt.Sprintf("{mount=%v}", usage.Path), usage.Used)
			c.sensision.WriteString(gts)
			gts = core.GetSeriesOutputAttributes(now, class+".total",
				fmt.Sprintf("{%v}", core.ToLabels("disk", diskPath)),
				fmt.Sprintf("{mount=%v}", usage.Path), usage.Total)
			c.sensision.WriteString(gts)
			gts = core.GetSeriesOutputAttributes(now, class+".inodes.used",
				fmt.Sprintf("{%v}", core.ToLabels("disk", diskPath)),
				fmt.Sprintf("{mount=%v}", usage.Path), usage.InodesUsed)
			c.sensision.WriteString(gts)
			gts = core.GetSeriesOutputAttributes(now, class+".inodes.total",
				fmt.Sprintf("{%v}", core.ToLabels("disk", diskPath)),
				fmt.Sprintf("{mount=%v}", usage.Path), usage.InodesTotal)
			c.sensision.WriteString(gts)
		}
	}

	if c.level > 2 {
		for name, stats := range counters {
			if len(c.allowedDisks) > 0 {
				if !stringInSlice(name, c.allowedDisks) {
					log.Debug("Disk name " + name + " is blacklisted, skip it")
					continue
				}
			}
			gts := core.GetSeriesOutput(now, class+".bytes.read",
				fmt.Sprintf("{%v}", core.ToLabels("name", name)), stats.ReadBytes)
			c.sensision.WriteString(gts)
			gts = core.GetSeriesOutput(now, class+".bytes.write",
				fmt.Sprintf("{%v}", core.ToLabels("name", name)), stats.WriteBytes)
			c.sensision.WriteString(gts)

			if c.level > 3 {
				gts = core.GetSeriesOutput(now, class+".io.read",
					fmt.Sprintf("{%v}", core.ToLabels("name", name)), stats.ReadCount)
				c.sensision.WriteString(gts)
				gts = core.GetSeriesOutput(now, class+".io.write",
					fmt.Sprintf("{%v}", core.ToLabels("name", name)), stats.WriteCount)
				c.sensision.WriteString(gts)

				if c.level > 4 {
					gts = core.GetSeriesOutput(now, class+".io.read.ms",
						fmt.Sprintf("{%v}", core.ToLabels("name", name)), stats.ReadTime)
					c.sensision.WriteString(gts)
					gts = core.GetSeriesOutput(now, class+".io.write.ms",
						fmt.Sprintf("{%v}", core.ToLabels("name", name)), stats.WriteTime)
					c.sensision.WriteString(gts)
					gts = core.GetSeriesOutput(now, class+".io",
						fmt.Sprintf("{%v}", core.ToLabels("name", name)), stats.IopsInProgress)
					c.sensision.WriteString(gts)
					gts = core.GetSeriesOutput(now, class+".io.ms",
						fmt.Sprintf("{%v}", core.ToLabels("name", name)), stats.IoTime)
					c.sensision.WriteString(gts)
					gts = core.GetSeriesOutput(now, class+".io.weighted.ms",
						fmt.Sprintf("{%v}", core.ToLabels("name", name)), stats.WeightedIO)
					c.sensision.WriteString(gts)
				}
			}
		}
	}

	return nil
}
