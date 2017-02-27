# Noderig - Export OS stats as Sensision Metrics

[![Build Status](https://travis-ci.org/runabove/noderig.svg?branch=master)](https://travis-ci.org/runabove/noderig)

Noderig collect OS metrics and expose them through a Sensision HTTP endpoint. Each collector is easily configurable, thanks to a simple level cursor

Noderig metrics:
- CPU
- Memory
- Load
- Disk
- Net
- External collectors

## Status

Noderig is currently under development. Feel free to comment or contribute!

## Building

Noderig is pretty easy to build.

- Clone the repository
- Install glide, follow instructions here https://glide.sh/
- Download dependencies `glide install`
- Build and run `go run noderig.go`

## Usage

```
noderig [flags]

Flags:
      --config string     config file to use
  -l  --listen string     listen address (default "127.0.0.1:9100")
  -v  --verbose           verbose output
      --period uint       default collection period (default 1000)
      --cpu uint8         cpu metrics level (default 1)
      --disk uint8        disk metrics level (default 1)
      --mem uint8         memory metrics level (default 1)
      --net uint8         network metrics level (default 1)
      --load uint8        load metrics level (default 1)
  -c  --collectors string external collectors directory (default "./collectors")
  -k  --keep-for uint     keep collectors data for the given number of fetch (default 3)
```

## Collectors
Noderig have some built-in collectors.

### CPU
<table>
<tr><td>0</td><td></td><td>disabled metrics</td></tr>
<tr><td>1</td><td>os.cpu{}</td><td>combined percentage of cpu usage</td></tr>
<tr><td rowspan="3">2</td><td>os.cpu.iowait{}</td><td>combined percentage of cpu iowait</td></tr>
<tr><td>os.cpu.user{}</td><td>combined percentage of cpu user</td></tr>
<tr><td>os.cpu.systems{}</td><td>combined percentage of cpu systems</td></tr>
<tr><td rowspan="3">3</td><td>os.cpu.iowait{chore:n}</td><td>chore percentage of cpu iowait</td></tr>
<tr><td>os.cpu.user{chore:n}</td><td>chore percentage of cpu user</td></tr>
<tr><td>os.cpu.systems{chore:n}</td><td>chore percentage of cpu systems</td></tr>
</table>

### Memory
<table>
<tr><td>0</td><td></td><td>disabled metrics</td></tr>
<tr><td rowspan="2">1</td><td>os.mem{}</td><td>percentage of memory used</td></tr>
<tr><td>os.swap{}</td><td>percentage of swap used</td></tr>
<tr><td rowspan="4">2</td><td>os.mem.used{}</td><td>used memory (bytes)</td></tr>
<tr><td>os.mem.total{}</td><td>total memory (bytes)</td></tr>
<tr><td>os.swap.used{}</td><td>used swap (bytes)</td></tr>
<tr><td>os.swap.total{}</td><td>total swap (bytes)</td></tr>
</table>

### Load
<table>
<tr><td>0</td><td></td><td>disabled metrics</td></tr>
<tr><td>1</td><td>os.load1{}</td><td>load 1</td></tr>
<tr><td rowspan="2">2</td><td>os.load5{}</td><td>load 5</td></tr>
<tr><td>os.load15{}</td><td>load 15</td></tr>
</table>

### Disk
<table>
<tr><td>0</td><td></td><td>disabled metrics</td></tr>
<tr><td>1</td><td>os.disk.fs{disk:/sda1}</td><td>disk used percent</td></tr>
<tr><td rowspan="2">2</td><td>os.disk.fs.used{disk:/sda1}</td><td>disk used capacity (bytes)</td></tr>
<tr><td>os.disk.fs.total{disk:/sda1}</td><td>disk total capacity (bytes)</td></tr>
<tr><td rowspan="2">3</td><td>os.disk.fs.bytes.read{disk:/sda1}</td><td>disk read count (bytes)</td></tr>
<tr><td>os.disk.fs.bytes.write{disk:/sda1}</td><td>disk write count (bytes)</td></tr>
<tr><td rowspan="2">4</td><td>os.disk.fs.io.read{disk:/sda1}</td><td>disk io read count (bytes)</td></tr>
<tr><td>os.disk.fs.io.write{disk:/sda1}</td><td>disk io write count (bytes)</td></tr>
<tr><td rowspan="5">5</td><td>os.disk.fs.io.read.ms{disk:/sda1}</td><td>disk io read time (ms)</td></tr>
<tr><td>os.disk.fs.io.write.ms{disk:/sda1}</td><td>disk io write time (ms)</td></tr>
<tr><td>os.disk.fs.io{disk:/sda1}</td><td>disk io in progress (count)</td></tr>
<tr><td>os.disk.fs.io.ms{disk:/sda1}</td><td>disk io time (ms)</td></tr>
<tr><td>os.disk.fs.io.weighted.ms{disk:/sda1}</td><td>disk io weighted time (ms)</td></tr>
</table>

### Net
<table>
<tr><td>0</td><td></td><td>disabled metrics</td></tr>
<tr><td rowspan="2">1</td><td>os.net.bytes{direction:in}</td><td>in bytes count (bytes)</td></tr>
<tr><td>os.net.bytes{direction:out}</td><td>out bytes count (bytes)</td></tr>
<tr><td rowspan="2">2</td><td>os.net.bytes{direction:in, iface:eth0}</td><td>iface in bytes count (bytes)</td></tr>
<tr><td>os.net.bytes{direction:out, iface:eth0}</td><td>iface out bytes count (bytes)</td></tr>
<tr><td rowspan="6">3</td><td>os.net.packets{direction:in, iface:eth0}</td><td>iface in packet count (packets)</td></tr>
<tr><td>os.net.packets{direction:out, iface:eth0}</td><td>iface out packet count (packets)</td></tr>
<tr><td>os.net.errs{direction:in, iface:eth0}</td><td>iface in error count (errors)</td></tr>
<tr><td>os.net.errs{direction:out, iface:eth0}</td><td>iface out error count (errors)</td></tr>
<tr><td>os.net.dropped{direction:in, iface:eth0}</td><td>iface in drop count (drops)</td></tr>
<tr><td>os.net.dropped{direction:out, iface:eth0}</td><td>iface out drop count (drops)</td></tr>
</table>

## Configuration

Noderig can read a simple default [config file](config.yaml).

Configuration is load and override in the following order:

- /etc/noderig/config.yaml
- ~/noderig/config.yaml
- ./config.yaml
- config filepath from command line

### Definitions

Config is composed of three main parts and some config fields:

#### Collectors

Noderig have some built-in collectors. They could be configured by a log level.
You can also defined custom collectors, in an scollector way. (see: http://bosun.org/scollector/external-collectors)

```yaml
cpu: 1  # CPU collector level     (Optional, default: 1)
mem: 1  # Memory collector level  (Optional, default: 1)
load: 1 # Load collector level    (Optional, default: 1)
disk: 1 # Disk collector level    (Optional, default: 1)
net: 1  # Network collector level (Optional, default: 1)
```

#### Parameters

Noderig can be customized through some parameters.

```yaml
period: 1000             # Duration within all the sources should be scraped in ms (Optional, default: 1000)
listen: none             # Listen address, set to none to disable http endpoint    (Optional, default: 127.0.0.1:9100)
collectors: /opt/noderig # Custom collectors directory                             (Optional, default: none)
```


#### Collectors Options

Some collectors can accept optional parameters.

```yaml
net-opts: 
  interfaces:            # Give a filtering list of interfaces for which you want metrics
    - eth0
    - eth1
```

## Sample metrics

```
1484828198557102// os.cpu{} 2.5202020226869237
1484828198560976// os.mem{} 24.328345730457112
1484828198560976// os.swap{} 0
1484828198557435// os.load1{} 0.63
1484828198561366// os.net.bytes{direction:in} 858
1484828198561366// os.net.bytes{direction:out} 778
1484828197570759// os.disk.fs{disk:/dev/sda1} 4.967614357908193
```

## Contributing

Instructions on how to contribute to Noderig are available on the [Contributing] page.

## Get in touch

- Twitter: [@notd33d33](https://twitter.com/notd33d33)

[contributing]: CONTRIBUTING.md
