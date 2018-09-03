package collectors

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mitchellh/go-ps"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Process collector
type Process struct {
	whitelist []string
	mutex     sync.RWMutex
	sensision bytes.Buffer
	level     uint8
}

// NewProcess collector
func NewProcess(period uint, level uint8, opts *viper.Viper) *Process {
	p := &Process{
		level: level,
	}

	if opts != nil {
		p.whitelist = opts.GetStringSlice("whitelist")
	}
	log.Debugf("Process whitelist: %+v", p.whitelist)

	if p.level == 0 || len(p.whitelist) == 0 {
		return p
	}

	log.Debugf("Process whitelist: %+v", p.whitelist)

	tick := time.Tick(time.Duration(period) * time.Millisecond)
	go func() {
		for range tick {
			if err := p.scrape(); err != nil {
				log.Error(err)
			}
		}
	}()

	return p
}

// Metrics delivers metrics.
func (p *Process) Metrics() *bytes.Buffer {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	var res bytes.Buffer
	res.Write(p.sensision.Bytes())
	return &res
}

func (p *Process) scrape() error {
	processes, err := ps.Processes()
	if err != nil {
		return err
	}
	now := time.Now()

	res := map[string]bool{}

	for _, processName := range p.whitelist {
		res[processName] = false
		for _, process := range processes {
			if strings.Contains(process.Executable(), processName) {
				res[processName] = true
				continue
			}
		}
	}

	// protect consistency
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.sensision.Reset()

	for pcs, ok := range res {
		if _, err := p.sensision.WriteString(processMetric(now, pcs, ok)); err != nil {
			return err
		}
	}

	return nil
}

func processMetric(t time.Time, processName string, up bool) string {
	if up {
		return fmt.Sprintf("%d// os.process.up{name=%s} true\n", t.UnixNano()/1000, processName)
	}
	return fmt.Sprintf("%d// os.process.up{name=%s} false\n", t.UnixNano()/1000, processName)
}
