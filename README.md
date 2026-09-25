# Monitor Agent

Lightweight Linux resource monitoring agent written in Go, designed to run continuously on low-resource devices such as Raspberry Pi.

The agent collects system metrics locally and exposes them through a lightweight HTTP endpoint compatible with the Prometheus exposition format.

The project focuses on simplicity, low resource consumption, testability, and clean separation between metric collection, monitoring logic, storage, and HTTP exposure.

## Features

* CPU usage monitoring
* Memory usage monitoring
* Network traffic monitoring per interface
* CPU temperature monitoring on supported Linux systems
* Static host information
* Prometheus-compatible `/metrics` endpoint
* Health and readiness endpoints
* Collector availability tracking
* Graceful shutdown
* Periodic collection with configurable intervals
* Retry handling for temporarily unavailable collectors
* Thread-safe in-memory metric storage
* systemd service with a dedicated unprivileged user
* systemd security hardening
* Unit tests and benchmarks
* Raspberry Pi validation
* HTTP/resource load-testing utility

## Architecture

The application is organized into small packages with clear responsibilities:

```text
cmd/monitor-agent
    ├── collector
    ├── config
    ├── metrics
    └── monitor

collector ──> metrics
monitor   ──> collector
monitor   ──> metrics
config    ──> stdlib
metrics   ──> stdlib
```

### Packages

#### `cmd/monitor-agent`

Application entry point.

Responsible for:

* loading and validating configuration
* creating the metric store
* creating the collector status store
* registering HTTP endpoints
* starting monitoring goroutines
* starting the HTTP server
* handling OS shutdown signals
* performing graceful shutdown

#### `internal/collector`

Responsible for reading operating-system data and converting it into metrics.

Current collectors:

* CPU
* memory
* network
* temperature
* system information

Collectors use small reader functions and parsing functions where appropriate, making the system data sources independently testable.

#### `internal/monitor`

Responsible for the execution lifecycle of collectors.

It handles:

* periodic collection
* retries
* collector states
* collector errors
* readiness state

CPU monitoring has a specialized monitoring loop because CPU usage is calculated from two snapshots of cumulative CPU counters.

#### `internal/metrics`

Responsible for:

* metric representation
* metric validation
* thread-safe in-memory storage
* Prometheus-compatible formatting
* HTTP metrics exposure
* health checks
* readiness checks

#### `internal/config`

Contains runtime configuration and validation.

#### `tools/loadtest`

Contains a standalone utility used to exercise the HTTP API and observe resource consumption over time.

---

## Project structure

```text
monitor-agent/
├── .devcontainer/
│   └── devcontainer.json
├── .github/
│   └── dependabot.yml
├── cmd/
│   └── monitor-agent/
│       └── main.go
├── deploy/
│   └── monitor-agent.service
├── internal/
│   ├── collector/
│   │   ├── cpu.go
│   │   ├── cpu_test.go
│   │   ├── memory.go
│   │   ├── memory_test.go
│   │   ├── network.go
│   │   ├── network_test.go
│   │   ├── system_info.go
│   │   ├── system_info_test.go
│   │   ├── temperature.go
│   │   └── temperature_test.go
│   ├── config/
│   │   ├── config.go
│   │   └── config_test.go
│   ├── metrics/
│   │   ├── health.go
│   │   ├── http.go
│   │   ├── http_test.go
│   │   ├── metrics.go
│   │   ├── ready.go
│   │   ├── ready_test.go
│   │   ├── store.go
│   │   ├── store_test.go
│   │   ├── validation.go
│   │   └── validation_test.go
│   └── monitor/
│       ├── error.go
│       ├── error_test.go
│       ├── monitor.go
│       ├── status.go
│       ├── status_store.go
│       └── status_store_test.go
├── tools/
│   └── loadtest/
│       └── main.go
├── .gitignore
├── LICENSE
├── go.mod
└── README.md
```

The repository may contain locally generated binaries during development, but these are build artifacts and are excluded from version control.

---

## Collected metrics

### CPU

```text
cpu_usage_percent
```

Type:

```text
gauge
```

CPU usage is calculated from two `/proc/stat` snapshots.

The calculation uses:

* user
* nice
* system
* idle
* iowait
* IRQ
* softIRQ
* steal

Guest and guest_nice counters are excluded from the calculation to avoid double-counting CPU time.

---

### Memory

```text
memory_total_bytes
memory_available_bytes
memory_used_bytes
```

Type:

```text
gauge
```

Memory information is read from `/proc/meminfo`.

The collector uses `MemTotal` and `MemAvailable` and reports values in bytes.

---

### Network

```text
network_receive_bytes_total
network_transmit_bytes_total
```

Type:

```text
counter
```

Network statistics are read from `/proc/net/dev`.

Each metric contains the network interface as a label:

```text
network_receive_bytes_total{interface="wlan0"} ...
network_transmit_bytes_total{interface="wlan0"} ...
```

---

### Temperature

```text
temperature_celsius
```

Type:

```text
gauge
```

On Raspberry Pi systems, the collector reads the CPU temperature from:

```text
/sys/class/thermal/thermal_zone0/temp
```

The kernel value is expressed in millidegrees Celsius and converted to degrees Celsius.

Temperature collection is optional because not every Linux environment exposes a compatible thermal sensor.

When the sensor is unavailable, the collector reports a distinct `not_supported` state rather than treating the condition as a generic application failure.

---

### System information

```text
system_info
```

Type:

```text
gauge
```

The metric contains static host information through labels:

```text
system_info{
    hostname="rpi3b",
    os="linux",
    arch="arm",
    cpus="4"
} 1
```

The current implementation reports:

* hostname
* operating system
* architecture
* CPU count

---

## HTTP API

The agent listens on:

```text
:8080
```

### `GET /metrics`

Returns the currently stored metrics using the Prometheus text exposition format.

Example:

```text
# HELP cpu_usage_percent CPU usage percentage
# TYPE cpu_usage_percent gauge
cpu_usage_percent 15.86

# HELP memory_available_bytes Available memory
# TYPE memory_available_bytes gauge
memory_available_bytes 714096640

# HELP network_receive_bytes_total Network received bytes
# TYPE network_receive_bytes_total counter
network_receive_bytes_total{interface="wlan0"} 35909895
```

Metric names and samples are emitted deterministically by sorting metric names and label sets.

---

### `GET /health`

Returns:

```text
OK
```

This endpoint indicates that the HTTP server is responding.

---

### `GET /ready`

Returns:

```text
READY
```

when all required collectors are available.

If a required collector is not available, the endpoint returns:

```text
NOT READY
```

with HTTP status:

```text
503 Service Unavailable
```

The temperature collector is currently optional, so the absence of a supported temperature sensor does not prevent the agent from becoming ready.

---

## Collector states

Collectors can have the following states:

```text
unknown
available
unavailable
not_supported
```

Required collectors:

* CPU
* memory
* network

Optional collector:

* temperature

Readiness is determined from the state of required collectors.

---

## Configuration

The current configuration is defined in `internal/config`.

Default intervals are:

| Collector   |   Interval |
| ----------- | ---------: |
| CPU         |  3 seconds |
| Memory      | 30 seconds |
| Network     | 30 seconds |
| Temperature | 30 seconds |
| Retry       |   1 second |

All intervals must be greater than zero.

The current version does not expose external configuration through command-line flags, environment variables, or configuration files.

This keeps the initial agent implementation intentionally small. External configuration can be added later if deployment requirements justify it.

---

## Requirements

The project requires:

* Go 1.26 or newer
* Linux for full collector functionality

The agent can be developed and tested inside the provided Dev Container.

Some Linux-specific collectors depend on kernel interfaces such as:

```text
/proc/stat
/proc/meminfo
/proc/net/dev
/sys/class/thermal/thermal_zone0/temp
```

Therefore, behavior can differ between the development container and the Raspberry Pi.

For example, the development container may not expose a thermal sensor, while the Raspberry Pi does.

---

## Development

Build the agent:

```bash
go build ./cmd/monitor-agent
```

Build the load-testing utility:

```bash
go build ./tools/loadtest
```

Run the agent directly:

```bash
go run ./cmd/monitor-agent
```

The agent starts the HTTP server on port `8080`.

You can then query:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
curl http://localhost:8080/metrics
```

---

## Testing

Run the complete test suite:

```bash
go test ./...
```

Run tests with the race detector:

```bash
go test -race ./...
```

Run static analysis:

```bash
go vet ./...
```

The project includes unit tests for:

* CPU parsing and calculation
* memory parsing
* network parsing
* temperature parsing
* system information
* metric validation
* metric storage
* HTTP handlers
* readiness
* collector status
* collector error handling
* configuration validation

---

## Benchmarks

Benchmarks are included for the main performance-sensitive components.

Example results obtained during development:

```text
BenchmarkCalculateCPUUsage-11
82,211,955
13.86 ns/op
0 B/op
0 allocs/op

BenchmarkGetMemoryInfo-11
926,244
1,714 ns/op
4,768 B/op
18 allocs/op

BenchmarkGetNetworkStats-11
561,974
2,012 ns/op
5,584 B/op
12 allocs/op

BenchmarkGetTemperature-11
5,568,524
212.7 ns/op
552 B/op
3 allocs/op

BenchmarkMetricsStoreSet-11
11,380,582
102.1 ns/op
40 B/op
2 allocs/op

BenchmarkMetricsHandler-11
197,253
5,957 ns/op
11,051 B/op
114 allocs/op
```

These numbers are environment-dependent and are included as development measurements rather than fixed performance guarantees.

No micro-optimization was introduced based on these results. The current implementation is already lightweight enough for the intended Raspberry Pi use case.

---

## Raspberry Pi

The agent has been tested on a Raspberry Pi 3B+ with 1 GB of RAM.

Example system information:

```text
system_info{
    arch="arm",
    cpus="4",
    hostname="rpi3b",
    os="linux"
} 1
```

Example temperature:

```text
temperature_celsius 56.382
```

Example memory:

```text
memory_total_bytes 947048448
memory_available_bytes 714096640
memory_used_bytes 232951808
```

Example network metrics:

```text
network_receive_bytes_total{interface="lo"} 293639
network_receive_bytes_total{interface="wlan0"} 35909895

network_transmit_bytes_total{interface="lo"} 293639
network_transmit_bytes_total{interface="wlan0"} 1440111
```

The agent successfully exposed `/health`, `/ready`, and `/metrics` on the Raspberry Pi during validation.

---

## systemd deployment

A systemd service definition is included at:

```text
deploy/monitor-agent.service
```

The service runs the agent using a dedicated unprivileged system user:

```text
monitor-agent
```

Example user creation:

```bash
sudo useradd \
    --system \
    --no-create-home \
    --shell /usr/sbin/nologin \
    monitor-agent
```

Build the binary:

```bash
go build -o monitor-agent ./cmd/monitor-agent
```

Install it:

```bash
sudo install \
    -o root \
    -g root \
    -m 755 \
    monitor-agent \
    /usr/local/bin/monitor-agent
```

Install the service:

```bash
sudo install \
    -o root \
    -g root \
    -m 644 \
    deploy/monitor-agent.service \
    /etc/systemd/system/monitor-agent.service
```

Reload systemd:

```bash
sudo systemctl daemon-reload
```

Enable the service:

```bash
sudo systemctl enable monitor-agent
```

Start it:

```bash
sudo systemctl start monitor-agent
```

Check the service:

```bash
sudo systemctl status monitor-agent
```

Check whether it is enabled:

```bash
systemctl is-enabled monitor-agent
```

View logs:

```bash
journalctl -u monitor-agent
```

---

## systemd security

The service definition includes several systemd hardening options:

```text
User=monitor-agent
Group=monitor-agent

NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictSUIDSGID=true

CapabilityBoundingSet=
AmbientCapabilities=
```

The service therefore runs without root privileges and with a restricted system environment.

The agent does not currently require Linux capabilities to collect its supported metrics.

---

## Graceful shutdown

The agent handles:

* `SIGINT`
* `SIGTERM`

Shutdown is coordinated through Go's `context` mechanism.

When a shutdown signal is received:

1. The root context is cancelled.
2. Collector monitoring loops stop.
3. The HTTP server begins graceful shutdown.
4. The server waits for active requests up to the configured shutdown timeout.
5. Monitoring goroutines finish.
6. The process exits.

Example log output:

```text
shutdown signal received
shutdown complete
```

This behavior was also verified through systemd restarts on the Raspberry Pi.

---

## Load testing

The repository includes a small load-testing utility:

```text
tools/loadtest
```

It periodically calls:

```text
/metrics
/health
/ready
```

and records process resource information.

The default test duration is 30 minutes.

Example:

```bash
go run ./tools/loadtest
```

The utility can be used to observe:

* request count
* successful requests
* failed requests
* response latency
* response size
* requests per minute
* process RSS
* cgroup memory
* CPU consumption

---

## Raspberry Pi load-test results

A 30-minute load test was performed on the Raspberry Pi using:

```text
metrics: 15s
health: 30s
ready: 30s
resource sampling: 60s
```

Results:

```text
[metrics]
requests:       120
successes:      120
failures:       0
bytes:          145666
avg latency:    2.273361ms
max latency:    7.088183ms
requests/min:   4.00

[health]
requests:       60
successes:      60
failures:       0
bytes:          180
avg latency:    2.232523ms
max latency:    10.62342ms
requests/min:   2.00

[ready]
requests:       60
successes:      60
failures:       0
bytes:          360
avg latency:    2.19014ms
max latency:    4.834485ms
requests/min:   2.00

[process resources]
samples:            30
initial RSS:        7504 KB
final RSS:          7728 KB
RSS change:         +224 KB
initial cgroup mem: 8936 KB
final cgroup mem:   9160 KB
memory change:      +224 KB
CPU accumulated:    4.997 s
CPU during test:    2.071 s
CPU average:        0.119%
```

During this test:

* all HTTP requests succeeded
* no HTTP failures were observed
* average `/metrics` latency was approximately 2.27 ms
* maximum `/metrics` latency was approximately 7.09 ms
* RSS remained around 7.5–7.7 MB during the sampled period
* average CPU usage was approximately 0.119%

These measurements are specific to the tested Raspberry Pi and workload and should not be interpreted as universal resource guarantees.

---

## Design decisions

### In-memory metric storage

Metrics are stored in memory rather than a database.

The agent is intended to be a local exporter rather than a long-term metrics database. Persistent storage belongs to the central monitoring system planned for a later stage.

### Prometheus exposition format

The agent exposes metrics using the Prometheus text format so that an external monitoring system can scrape the endpoint without requiring a custom protocol.

### Separate collector and monitor layers

Collectors are responsible for obtaining data.

The monitor layer is responsible for deciding when collectors run, handling retries, and maintaining collector state.

This separation keeps system-specific parsing logic independent from lifecycle management.

### Optional temperature collector

Temperature sensors are not consistently available across Linux environments.

The collector therefore distinguishes between:

```text
unavailable
```

and:

```text
not_supported
```

This allows the agent to run normally on systems without a compatible thermal sensor.

### No central scheduler

Each collector owns its own interval.

This keeps the monitoring loop simple and allows different collectors to operate at different frequencies.

### Thread-safe metric storage

Collectors run concurrently, while the HTTP server may read metrics at the same time.

`MetricsStore` therefore uses a read/write mutex to protect its internal state.

### Deterministic metric output

Metric names and label sets are sorted before being exposed.

This makes output stable and easier to inspect, test, compare, and debug.

### Graceful lifecycle management

The application uses Go contexts, `signal.NotifyContext`, `http.Server.Shutdown`, and a `sync.WaitGroup` to coordinate startup and shutdown.

---

## Current scope

The current version intentionally focuses on a single-node monitoring agent.

It currently provides:

* local system metric collection
* in-memory storage
* HTTP metric exposure
* health/readiness endpoints
* collector state tracking
* systemd deployment
* lightweight resource usage
* tests and benchmarks

It does **not** currently provide:

* persistent storage
* a central server
* dashboards
* historical metric queries
* authentication
* TLS configuration
* remote configuration
* alert management
* multi-node aggregation

These features belong to the next stage of the project rather than the current agent.

---

## Future work

Possible future components include:

### Central monitoring server

A separate service can receive or scrape metrics from multiple agents and provide:

* persistent storage
* historical queries
* aggregation
* dashboards
* alerting

### Remote agents

Multiple Raspberry Pis or Linux hosts could run the same lightweight agent and report to the central service.

### Configuration

Future versions could support:

* environment variables
* command-line flags
* configuration files

### Additional collectors

Potential future metrics include:

* disk usage
* filesystem statistics
* load average
* process statistics
* uptime
* additional hardware sensors

Additional collectors should only be added when they provide useful monitoring information without compromising the lightweight nature of the agent.

---

## Development environment

The repository includes a Dev Container configuration:

```text
.devcontainer/devcontainer.json
```

This provides a reproducible Linux development environment for working on the project from a development machine.

Because some collectors depend on Linux kernel interfaces and hardware-specific files, running the agent inside the Dev Container does not reproduce every Raspberry Pi feature.

In particular, temperature collection may be unavailable inside the container while remaining available on the physical Raspberry Pi.

---

## Dependency management

The project currently uses only the Go standard library.

The core agent does not require external Go dependencies.

The repository also includes Dependabot configuration under:

```text
.github/dependabot.yml
```

---

## License

This project is licensed under the MIT License.

See the [`LICENSE`](LICENSE) file for the complete license text.

---

## Status

The initial monitoring-agent scope is implemented and validated on a real Raspberry Pi 3B+.

The current implementation has:

* working CPU monitoring
* working memory monitoring
* working network monitoring
* temperature monitoring on supported hardware
* system information
* Prometheus-compatible metrics
* health and readiness endpoints
* collector status management
* graceful shutdown
* systemd deployment
* service hardening
* unit tests
* benchmarks
* HTTP/resource load testing
* Raspberry Pi validation

The next major stage is the development of the central monitoring server with persistence and visualization.
