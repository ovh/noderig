package core

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"
)

var (

	// Format is the collector ouptut format for Noderig
	Format = "sensition"

	// Separator pattern for each classnames
	Separator = "."

	// DefaultLabels add default labels
	DefaultLabels = ""
)

// Collector interface
type Collector interface {
	Metrics() *bytes.Buffer
}

// GetSeriesOutput series output format rendering
func GetSeriesOutput(tick int64, class string, labels string, value interface{}) string {
	return GetSeriesOutputAttributes(tick, class, labels, "", value)
}

// GetSeriesOutputAttributes series output format rendering with Attributes
func GetSeriesOutputAttributes(tick int64, class string, labels string, attributes string, value interface{}) string {

	if Separator != "." {
		class = strings.Replace(class, ".", Separator, -1)
	}

	if DefaultLabels != "" {
		labels = strings.Replace(labels, "{", "", 1)
		labels = strings.TrimSpace(labels)

		prefix := ","
		if strings.HasPrefix(labels, "}") {
			prefix = ""
		}
		labels = "{" + DefaultLabels + prefix + labels
	}

	switch Format {
	case "sensition":
		return toSensitionFormat(tick, class, labels, attributes, value)
	case "prometheus":
		return toPrometheusFormat(tick, class, labels, value)
	default:
		return toSensitionFormat(tick, class, labels, attributes, value)
	}
}

func toSensitionFormat(tick int64, class string, labels string, attributes string, value interface{}) string {
	gtsValue := ""
	switch v := value.(type) {
	case string:
		gtsValue += fmt.Sprintf("'%v'", url.PathEscape(v))
	default:
		gtsValue += fmt.Sprintf("%v", v)
	}

	if len(attributes) > 0 {
		return fmt.Sprintf("%v// %v%v%v %v\n", tick, class, labels, attributes, gtsValue)
	}
	return fmt.Sprintf("%v// %v%v %v\n", tick, class, labels, gtsValue)
}

func toPrometheusFormat(tick int64, class string, labels string, value interface{}) string {
	gtsValue := ""
	switch value.(type) {
	case string:
		return ""
	default:
		gtsValue += fmt.Sprintf("%v", value)
	}
	return fmt.Sprintf("%v%v %v %v\n", class, labels, gtsValue, tick/1000)
}

// ToLabels ensure the correct output format for a key/value set for series labels
func ToLabels(key string, value interface{}) string {

	switch Format {
	case "sensition":
		return fmt.Sprintf("%v=%v", key, value)
	case "prometheus":
		return fmt.Sprintf("%v=\"%v\"", key, value)
	default:
		return fmt.Sprintf("%v=%v", key, value)
	}
}
