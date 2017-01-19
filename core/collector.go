package core

import (
	"bytes"
)

// Collector interface
type Collector interface {
	Metrics() *bytes.Buffer
}
