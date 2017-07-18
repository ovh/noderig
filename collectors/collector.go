package collectors

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/shirou/gopsutil/cpu"
)

// Collector collects external metrics
type Collector struct {
	times []cpu.TimesStat

	mutex     sync.RWMutex
	sensision bytes.Buffer
	fetched   []bytes.Buffer
	path      string
}

// NewCollector returns an initialized external collector.
func NewCollector(path string, period uint, keep uint) *Collector {
	c := &Collector{
		path:    path,
		fetched: make([]bytes.Buffer, keep),
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
func (c *Collector) Metrics() *bytes.Buffer {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var top bytes.Buffer
	top.Grow(c.sensision.Len())
	top.Write(c.sensision.Bytes())

	for i := len(c.fetched) - 1; i > 0; i-- {
		c.fetched[i] = c.fetched[i-1]
	}
	c.fetched[0] = top

	c.sensision.Reset()

	var res bytes.Buffer
	for i := 0; i < len(c.fetched); i++ {
		res.Write(c.fetched[i].Bytes())
	}
	return &res
}

// DataPoint is an opentsdb data point
type dataPoint struct {
	Metric    string            `json:"metric"`
	Timestamp int64             `json:"timestamp"`
	Value     interface{}       `json:"value"`
	Tags      map[string]string `json:"tags"`
}

// opentsdb metadata data point
type metasend struct {
	Metric string            `json:",omitempty"`
	Tags   map[string]string `json:",omitempty"`
	Name   string            `json:",omitempty"`
	Value  interface{}
	Time   *time.Time `json:",omitempty"`
}

func (c *Collector) scrape() (cmdError error) {
	cmd := exec.Command(c.path)

	pr, pw := io.Pipe()
	s := bufio.NewScanner(pr)
	cmd.Stdout = pw
	er, ew := io.Pipe()
	cmd.Stderr = ew

	if err := cmd.Start(); err != nil {
		return err
	}

	// Wait for close
	go func() {
		cmdError = cmd.Wait()
		pw.Close()
		ew.Close()
	}()

	// Stderr handler
	go func() {
		es := bufio.NewScanner(er)
		for es.Scan() {
			line := strings.TrimSpace(es.Text())
			log.Errorf("%v: %v", c.path, line)
		}
	}()

	// Stdout handler
scan:
	for s.Scan() {
		t := strings.TrimSpace(s.Text())
		if len(t) == 0 {
			continue
		}

		var dp dataPoint
		if t[0] != '{' {
			sp := strings.Fields(t)
			if len(sp) < 3 {
				log.Warnf("%v: invalid data point - %v", c.path, sp)
				continue
			}
			ts, err := strconv.ParseInt(sp[1], 10, 64)
			if err != nil {
				log.Warnf("%v: invalid timestamp - %v", c.path, t)
				continue
			}

			var val interface{}
			val, err = strconv.ParseInt(sp[2], 10, 64)
			if err != nil {
				val, err = strconv.ParseFloat(sp[2], 64)
				if err != nil {
					log.Warnf("%v: invalid value - %v", c.path, t)
					continue
				}
			}

			dp = dataPoint{
				Metric:    sp[0],
				Timestamp: ts,
				Value:     val,
				Tags:      make(map[string]string),
			}
			for _, tag := range sp[3:] {
				for _, v := range strings.Split(tag, ",") {
					sp := strings.SplitN(v, "=", 2)
					if len(sp) != 2 {
						log.Warnf("%v: invalid tag - %v", c.path, t)
						continue scan
					}
					dp.Tags[c.sanitize(sp[0])] = c.sanitize(sp[1])
				}
			}
		} else {
			if err := json.Unmarshal([]byte(t), &dp); err != nil {
				// Maybe meta json
				var m metasend
				if err := json.Unmarshal([]byte(t), &m); err == nil {
					continue // skip metadata
				}
				log.Warnf("%v: invalid data point - %v", c.path, t)
				continue
			}
		}

		// add metric
		var labels string
		for k, v := range dp.Tags {
			labels += k + "=" + v + ","
		}
		labels = strings.TrimSuffix(labels, ",")

		c.mutex.Lock()
		gts := fmt.Sprintf("%v000000// %v{%v} ", dp.Timestamp, dp.Metric, labels)
		switch dp.Value.(type) {
		default:
			gts += fmt.Sprintf("%v\n", dp.Value)
		case string:
			gts += fmt.Sprintf("'%v'\n", dp.Value)
		}
		c.sensision.WriteString(gts)
		c.mutex.Unlock()
	}

	if err := s.Err(); err != nil {
		return err
	}
	return nil
}

func (c *Collector) sanitize(v string) string {
	s := strings.TrimSpace(v)
	s = strings.Replace(v, ",", "%2C", -1)
	s = strings.Replace(s, "}", "%7D", -1)
	return strings.Replace(s, "=", "%3D", -1)
}
