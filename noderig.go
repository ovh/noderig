// Noderig expose node metrics as Sensision metrics.
//
// Usage
//
// 		noderig  [flags]
// Flags:
//       --config string   config file to use
//       --help            display help
//   -v, --verbose         verbose output
//   -l, --listen          listen addresse
package main

import (
	log "github.com/sirupsen/logrus"

	"github.com/ovh/noderig/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		log.Panicf("%v", err)
	}
}
