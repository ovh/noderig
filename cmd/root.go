package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"net/http"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ovh/noderig/collectors"
	"github.com/ovh/noderig/core"
)

var cs []core.Collector
var csMutex = sync.Mutex{}

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
	RootCmd.Flags().StringSlice("net-opts.interfaces", make([]string, 0), "give a filtering list of network interfaces to collect metrics on")
	RootCmd.Flags().StringSlice("disk-opts.names", make([]string, 0), "give a filtering list of disks names to collect metrics on")
	RootCmd.Flags().Uint64("period", 1000, "default collection period")
	RootCmd.Flags().StringP("collectors", "c", "./collectors", "external collectors directory")
	RootCmd.Flags().Uint64P("keep-for", "k", 3, "keep collectors data for the given number of fetch")
	RootCmd.Flags().String("format", "sensision", "the output global format of noderig")
	RootCmd.Flags().String("separator", ".", "the class separator string")

	err := viper.BindPFlags(RootCmd.PersistentFlags())
	if err != nil {
		log.WithError(err).Fatal("failed to init command line")
	}
	err = viper.BindPFlags(RootCmd.Flags())
	if err != nil {
		log.WithError(err).Fatal("failed to init command line")
	}
}

// Load config - initialize defaults and read config.
func initConfig() {
	if viper.GetBool("verbose") {
		log.SetLevel(log.DebugLevel)
	}

	// Defaults
	viper.SetDefault("flushPeriod", 10000)
	viper.SetDefault("keep-metrics", false)

	// Bind environment variables
	viper.SetEnvPrefix("noderig")
	viper.AutomaticEnv()

	// Load user defined config
	cfgFile := viper.GetString("config")
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		err := viper.ReadInConfig()
		if err != nil {
			log.Panicf("Fatal error in config file: %v \n", err)
		}
	} else {
		// Set config search path
		viper.AddConfigPath("/etc/noderig/")
		viper.AddConfigPath("$HOME/.noderig")
		viper.AddConfigPath(".")

		// Load default config
		viper.SetConfigName("config")
		if err := viper.MergeInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				log.Debug("No config file found")
			} else {
				log.Panicf("Fatal error in config file: %v \n", err)
			}
		}
	}

	if viper.IsSet("format") {
		format := viper.GetString("format")

		if format != "sensision" {
			core.Format = format
		}
	}

	if viper.IsSet("separator") {
		separator := viper.GetString("separator")

		if separator != "separator" {
			core.Separator = separator
		}
	}

	if viper.IsSet("labels") {
		labelsSet := viper.GetStringMapString("labels")

		if len(labelsSet) > 0 {

			labelsSplices := make([]string, 0)

			for key, value := range labelsSet {
				labelsSplices = append(labelsSplices, core.ToLabels(key, value))
			}

			core.DefaultLabels = strings.Join(labelsSplices, ",")
		}
	}

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Info("Config file changed, reload...")

		csMutex.Lock()
		defer csMutex.Unlock()
		cs = getCollectors()

		log.Infof("Reloaded - %d", len(cs))
	})
}

// RootCmd launch the aggregator agent.
var RootCmd = &cobra.Command{
	Use:   "noderig",
	Short: "Noderig expose node stats as Sensision metrics",
	Run:   rootFn,
}

func setFlagToViperMap(viperMap, flag, mapkey string) {
	if viper.Get(viperMap) == nil && viper.Get(flag) != nil {
		viper.Set(viperMap, map[string]interface{}{mapkey: viper.GetStringSlice(flag)})
	} else {
		if options, ok := viper.Get(viperMap).(map[string]interface{}); ok {
			if viper.Get(flag) != nil && options[mapkey] == nil {
				options[mapkey] = viper.GetStringSlice(flag)
				viper.Set(viperMap, options)
			}
		}
	}
}

func rootFn(cmd *cobra.Command, args []string) {
	log.Info("Noderig starting")
	log.Infof("External collectors will be loaded from: '%s'", viper.GetString("collectors"))

	setFlagToViperMap("net-opts", "net-opts.interfaces", "interfaces")
	setFlagToViperMap("disk-opts", "disk-opts.names", "names")

	cs = getCollectors()

	log.Infof("Noderig started - %v", len(cs))

	for _, k := range viper.AllKeys() {
		log.Debugf("Configuration %s = %+v", k, viper.Get(k))
	}

	// Setup http
	http.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		csMutex.Lock()
		defer csMutex.Unlock()
		for _, c := range cs {
			_, err := w.Write(c.Metrics().Bytes())
			if err != nil {
				log.WithError(err).Error("cannot write metric into file")
			}
		}
	}))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(`<html>
	             <head><title>Noderig</title></head>
	             <body>
	             <h1>Noderig</h1>
	             <p><a href="/metrics">Metrics</a></p>
	             <p><a href="https://github.com/ovh/noderig">Github</a></p>
	             </body>
	             </html>`))
		if err != nil {
			log.WithError(err).Error("cannot send body to client")
		}
	})
	log.Info("Http started")

	if viper.IsSet("flushPath") {
		flushPath := viper.GetString("flushPath")
		ticker := time.NewTicker(time.Duration(viper.GetInt("flushPeriod")) * time.Millisecond)
		go func() {
			for range ticker.C {
				path := fmt.Sprintf("%v%v", flushPath, time.Now().Unix())
				log.Debugf("Flush to file: %v%v", path, ".tmp")
				file, err := os.Create(path + ".tmp")
				if err != nil {
					log.Errorf("Flush failed: %v", err)
				}

				csMutex.Lock()
				for _, c := range cs {
					_, err := file.Write(c.Metrics().Bytes())
					if err != nil {
						log.WithError(err).Error("Cannot write metric into file")
					}
				}
				csMutex.Unlock()

				if err := file.Close(); err != nil {
					log.WithError(err).Error("Cannot close flush file")
				}

				// Move tmp file to metrics one
				log.Debugf("Move to file: %v%v", path, ".metrics")
				err = os.Rename(path+".tmp", path+".metrics")
				if err != nil {
					log.WithError(err).Error("Cannot rotate metrics file")
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

		quit := make(chan os.Signal, 2)

		signal.Notify(quit, syscall.SIGTERM)
		signal.Notify(quit, syscall.SIGINT)

		<-quit

	}
}

func getCollectors() []core.Collector {
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
			return cs
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
				disk := collectors.NewCollector(
					path.Join(dir.Name(), file.Name()),
					uint(interval),
					uint(viper.GetInt("keep-for")),
					viper.GetBool("keep-metrics"))
				cs = append(cs, disk)
			}
		}
	}

	return cs
}
