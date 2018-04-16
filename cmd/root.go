package cmd

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"time"

	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ovh/noderig/collectors"
	"github.com/ovh/noderig/core"
)

// Aggregator init - define command line arguments.
func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringP("config", "", "", "config file to use")
	RootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")

	RootCmd.Flags().StringP("listen", "l", "127.0.0.1:9100", "listen address")
	RootCmd.Flags().Uint8("cpu", 1, "cpu metrics level")
	RootCmd.Flags().Uint8("load", 1, "load metrics level")
	RootCmd.Flags().Uint8("mem", 1, "memory metrics level")
	RootCmd.Flags().Uint8("disk", 1, "disk metrics level")
	RootCmd.Flags().Uint8("net", 1, "network metrics level")
	RootCmd.Flags().Uint64("period", 1000, "default collection period")
	RootCmd.Flags().StringP("collectors", "c", "./collectors", "external collectors directory")
	RootCmd.Flags().Uint64P("keep-for", "k", 3, "keep collectors data for the given number of fetch")

	viper.BindPFlags(RootCmd.PersistentFlags())
	viper.BindPFlags(RootCmd.Flags())
}

// Load config - initialize defaults and read config.
func initConfig() {
	if viper.GetBool("verbose") {
		log.SetLevel(log.DebugLevel)
	}

	// Defaults
	viper.SetDefault("flushPeriod", 10000)

	// Bind environment variables
	viper.SetEnvPrefix("noderig")
	viper.AutomaticEnv()

	// Set config search path
	viper.AddConfigPath("/etc/noderig/")
	viper.AddConfigPath("$HOME/.noderig")
	viper.AddConfigPath(".")

	// Load config
	viper.SetConfigName("config")
	if err := viper.MergeInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Debug("No config file found")
		} else {
			log.Panicf("Fatal error in config file: %v \n", err)
		}
	}

	// Load user defined config
	cfgFile := viper.GetString("config")
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		err := viper.ReadInConfig()
		if err != nil {
			log.Panicf("Fatal error in config file: %v \n", err)
		}
	}
}

// RootCmd launch the aggregator agent.
var RootCmd = &cobra.Command{
	Use:   "noderig",
	Short: "Noderig expose node stats as Sensision metrics",
	Run: func(cmd *cobra.Command, args []string) {
		log.Info("Noderig starting")

		// Build collectors
		var cs []core.Collector

		cpu := collectors.NewCPU(uint(viper.GetInt("period")), uint8(viper.GetInt("cpu")), viper.GetStringSlice("cpu-mods"))
		cs = append(cs, cpu)

		mem := collectors.NewMemory(uint(viper.GetInt("period")), uint8(viper.GetInt("mem")))
		cs = append(cs, mem)

		load := collectors.NewLoad(uint(viper.GetInt("period")), uint8(viper.GetInt("load")))
		cs = append(cs, load)

		net := collectors.NewNet(uint(viper.GetInt("period")), uint8(viper.GetInt("net")), viper.Get("net-opts"))
		cs = append(cs, net)

		disk := collectors.NewDisk(uint(viper.GetInt("period")), uint8(viper.GetInt("disk")), viper.Get("disk-opts"))
		cs = append(cs, disk)

		// Load external collectors
		cpath := viper.GetString("collectors")
		cdir, err := os.Open(cpath)
		if err == nil {
			idirs, err := cdir.Readdir(0)
			if err != nil {
				log.Error(err)
				return
			}
			for _, idir := range idirs {
				idirname := idir.Name()
				i, err := strconv.Atoi(idirname)
				if err != nil {
					if idirname != "etc" && idirname != "lib" {
						log.Warn("Bad collector folder: ", idirname)
					}
					continue
				}

				interval := i * 1000
				if i <= 0 {
					interval = viper.GetInt("period")
				}

				dir, err := os.Open(path.Join(cpath, idirname))
				if err != nil {
					log.Error(err)
					continue
				}

				files, err := dir.Readdir(0)
				if err != nil {
					log.Error(err)
					continue
				}

				for _, file := range files {
					disk := collectors.NewCollector(path.Join(dir.Name(), file.Name()), uint(interval), uint(viper.GetInt("keep-for")))
					cs = append(cs, disk)
				}
			}
		}

		log.Infof("Noderig started - %v", len(cs))

		// Setup http
		http.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			for _, c := range cs {
				w.Write(c.Metrics().Bytes())
			}
		}))
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`<html>
	             <head><title>Noderig</title></head>
	             <body>
	             <h1>Noderig</h1>
	             <p><a href="/metrics">Metrics</a></p>
	             <p><a href="https://github.com/ovh/noderig">Github</a></p>
	             </body>
	             </html>`))

		})
		log.Info("Http started")

		if viper.IsSet("flushPath") {
			flushPath := viper.GetString("flushPath")
			ticker := time.NewTicker(time.Duration(viper.GetInt("flushPeriod")) * time.Millisecond)
			go func() {
				for {
					select {
					case <-ticker.C:
						path := fmt.Sprintf("%v%v", flushPath, time.Now().Unix())
						log.Debugf("Flush to file: %v%v", path, ".tmp")
						file, err := os.Create(path + ".tmp")
						if err != nil {
							log.Errorf("Flush failed: %v", err)
						}

						for _, c := range cs {
							file.Write(c.Metrics().Bytes())
						}

						file.Close()

						// Move tmp file to metrics one
						log.Debugf("Move to file: %v%v", path, ".metrics")
						os.Rename(path+".tmp", path+".metrics")
					}
				}
			}()
			log.Info("Flush routine started")
		}

		log.Info("Started")

		if viper.GetString("listen") != "none" {
			log.Infof("Listen %s", viper.GetString("listen"))
			log.Fatal(http.ListenAndServe(viper.GetString("listen"), nil))
		} else {
			select {}
		}
	},
}
