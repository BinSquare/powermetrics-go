# powermetrics-go

Go library that wraps the macOS powermetrics to monitor system performance metrics including CPU/GPU power, frequency, and process activity.

Powermetrics always outputs the Apple Neural Engine (ANE) data as 0, so I can't verify if my parsing actually works or not. I welcome contributions if I'm wrong.

- Note: Temperature values may be 0 on Apple Silicon Macs (M1/M2/M3/M4), they seem to report thermal pressure "levels" instead of temp for some reason.

## Sample Output

The following outputs shows what each type of metric looks like based on the sample log:

### CPU Cluster Information

```
E-Cluster Online: 100%
E-Cluster HW active frequency: 1293 MHz
E-Cluster HW active residency: 100.00% (1020 MHz:  75% 1404 MHz: 3.5% 1788 MHz: 5.1% 2112 MHz: 5.0% 2352 MHz: 5.0% 2532 MHz: 2.5% 2592 MHz: 3.9%)
E-Cluster idle residency:   0.00%
E-Cluster down residency:   0.00%
```

### CPU Residency Metrics

```
CPU 0 frequency: 1338 MHz
CPU 0 active residency:  55.11% (1020 MHz:  39% 1404 MHz: 2.2% 1788 MHz: 3.2% 2112 MHz: 3.2% 2352 MHz: 3.4% 2532 MHz: 1.7% 2592 MHz: 2.3%)
CPU 0 idle residency:  44.89%
CPU 0 down residency:   0.00%
```

### GPU Residency Metrics

```
GPU HW active frequency: 338 MHz
GPU HW active residency:   1.63% (338 MHz: 1.6% 618 MHz:   0% 796 MHz:   0% 924 MHz:   0% 952 MHz:   0% 1056 MHz:   0% 1062 MHz:   0% 1182 MHz:   0% 1182 MHz:   0% 1312 MHz:   0% 1242 MHz:   0% 1380 MHz:   0% 1326 MHz:   0% 1470 MHz:   0% 1578 MHz:   0%)
GPU SW requested state: (P1 : 100% P2 :   0% P3 :   0% P4 :   0% P5 :   0% P6 :   0% P7 :   0% P8 :   0% P9 :   0% P10 :   0% P11 :   0% P12 :   0% P13 :   0% P14 :   0% P15 :   0%)
GPU SW state: (SW_P1 : 1.6% SW_P2 :   0% SW_P3 :   0% SW_P4 :   0% SW_P5 :   0% SW_P6 :   0% SW_P7 :   0% SW_P8 :   0% SW_P9 :   0% SW_P10 :   0% SW_P11 :   0% SW_P12 :   0% SW_P13 :   0% SW_P14 :   0% SW_P15 :   0%)
GPU idle residency:  98.37%
GPU Power: 28 mW
```

### Battery Metrics

```
Battery: percent_charge: 36
```

### Network Metrics

```
out: 57.75 packets/s, 4586.65 bytes/s
in:  86.02 packets/s, 113827.21 bytes/s
```

### Disk Metrics

```
read: 8.56 ops/s 45.67 KBytes/s
write: 73.88 ops/s 2070.85 KBytes/s
```

### Interrupt Metrics

```
CPU 0:
	Total IRQ: 2977.12 interrupts/sec
	|-> IPI: 2232.79 interrupts/sec
	|-> TIMER: 547.20 interrupts/sec
CPU 1:
	Total IRQ: 2685.60 interrupts/sec
	|-> IPI: 2072.89 interrupts/sec
	|-> TIMER: 504.58 interrupts/sec
```

## SDK Usage Examples

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/BinSquare/powermetrics-go"
)

func main() {
    ctx := context.Background()

    // Start collecting metrics (requires sudo)
    metricsChan, err := powermetrics.RunDefault(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // Process metrics
    for metrics := range metricsChan {
        if metrics.SystemSample != nil {
            fmt.Printf("CPU Power: %.2f W | GPU Power: %.2f W | Battery: %.2f%%\n",
                metrics.SystemSample.CPUPowerWatts,
                metrics.SystemSample.GPUPowerWatts,
                metrics.SystemSample.BatteryPercent)
        }
    }
}
```

### Accessing Specific Values

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/BinSquare/powermetrics-go"
)

func main() {
    ctx := context.Background()

    metricsChan, err := powermetrics.RunDefault(ctx)
    if err != nil {
        log.Fatal(err)
    }

    for metrics := range metricsChan {
        if metrics.SystemSample != nil {
            // Access specific values directly
            battery := metrics.SystemSample.BatteryPercent
            cpuPower := metrics.SystemSample.CPUPowerWatts
            gpuPower := metrics.SystemSample.GPUPowerWatts
            cpuFreq := metrics.SystemSample.CPUFrequencyMHz

            fmt.Printf("Battery: %.2f%%, CPU Power: %.2fW, CPU Freq: %.0fMHz\n",
                battery, cpuPower, cpuFreq)
        }
    }
}
```

### Receiving Errors and Custom Samplers

```go
stream, err := powermetrics.RunWithConfigStream(ctx, powermetrics.Config{
    SampleWindow: 500 * time.Millisecond,
    PowermetricsArgs: []string{
        "--samplers", "tasks,battery,network,disk,interrupts,cpu_power,gpu_power,ane_power,thermal",
        "--show-process-gpu",
    },
})
if err != nil {
    log.Fatal(err)
}

for {
    select {
    case metrics, ok := <-stream.Metrics:
        if !ok {
            return
        }
        fmt.Printf("CPU Power: %.2fW\n", metrics.SystemSample.CPUPowerWatts)
    case err, ok := <-stream.Errors:
        if ok && err != nil {
            log.Println("powermetrics warning:", err)
        } else if !ok {
            return
        }
    }
}
```

### Parsing Saved Logs or Custom Readers

```go
f, err := os.Open("powermetrics_sample.log")
if err != nil {
    log.Fatal(err)
}
defer f.Close()

stream := powermetrics.RunReader(ctx, powermetrics.Config{}, f)
for metrics := range stream.Metrics {
    fmt.Printf("Battery: %.1f%%\n", metrics.SystemSample.BatteryPercent)
}

for err := range stream.Errors {
    log.Println("parser error:", err)
}
```

## Requirements

- macOS (powermetrics is macOS-specific)
- Root privileges (sudo required to run powermetrics)

## Installation

```bash
go get github.com/BinSquare/powermetrics-go
```

## Usage

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/BinSquare/powermetrics-go"
)

func main() {
    ctx := context.Background()

    // Start collecting metrics (requires sudo)
    metricsChan, err := powermetrics.RunDefault(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // Process metrics
    for metrics := range metricsChan {
        if metrics.SystemSample != nil {
            fmt.Printf("CPU Power: %.2f W\n", metrics.SystemSample.CPUPowerWatts)
            fmt.Printf("GPU Power: %.2f W\n", metrics.SystemSample.GPUPowerWatts)
        }

        if len(metrics.GPUProcessSamples) > 0 {
            fmt.Printf("GPU Processes: %d\n", len(metrics.GPUProcessSamples))
        }
    }
}
```

### Custom Configuration

```go
config := powermetrics.Config{
    SampleWindow:     500 * time.Millisecond,
    PowermetricsArgs: []string{"--samplers", "cpu_power,gpu_power", "-i", "500"},
}

parser := powermetrics.NewParser(config)
metricsChan, err := parser.Run(ctx)
```

## Important: Sudo Requirement

The `powermetrics` command requires root privileges to access system performance counters. This means you must run your application with `sudo`:

```bash
sudo go run your_program.go
```

Or if building an executable:

```bash
sudo ./your_program
```

This is a system-level requirement of powermetrics, not this library.

## API

### Types

- `Config`: Configuration for the powermetrics collector
- `Metrics`: Represents a single powermetrics sample
- `ClusterInfo`: CPU cluster information
- `Stream`: Bundles a metrics channel with an errors channel
- `Parser`: Handles invoking powermetrics and parsing output (now exposes methods for `RunWithErrors` and `RunWithReader`)
- `SystemSample`: Contains system metrics including CPU/GPU/ANE power, frequencies, temperatures, and busy percentages
  - `CPUPowerWatts`: CPU power consumption in watts
  - `GPUPowerWatts`: GPU power consumption in watts
  - `ANEPowerWatts`: Apple Neural Engine power consumption in watts
  - `CPUFrequencyMHz`: CPU frequency in MHz
  - `GPUFrequencyMHz`: GPU frequency in MHz
  - `CPUTemperatureC`: CPU temperature in Celsius (may be 0 on Apple Silicon Macs)
  - `GPUTemperatureC`: GPU temperature in Celsius (may be 0 on Apple Silicon Macs)
  - `ANEBusyPercent`: ANE utilization percentage
  - `GPUBusyPercent`: GPU utilization percentage
  - `DRAMPowerWatts`: DRAM power consumption in watts
  - `BatteryPercent`: Battery charge percentage
- `CPUResidencyMetrics`: Contains detailed CPU residency information per core
  - `CPUID`: CPU identifier
  - `ActiveResidency`: Frequency to percentage map of time spent at each frequency
  - `IdleResidency`: Percentage of time CPU was idle
  - `DownResidency`: Percentage of time CPU was down/clock-gated
  - `Frequency`: Current frequency of the CPU
- `GPUResidencyMetrics`: Contains detailed GPU residency information
  - `HWActiveResidency`: Percentage of time GPU hardware was active
  - `HWActiveFreqResidency`: Map of frequency to percentage for GPU hardware active time
  - `SWRequestedStates`: GPU software requested state distribution (P1-P15)
  - `SWStates`: Current GPU software state distribution (P1-P15)
  - `IdleResidency`: Percentage of time GPU was idle
  - `PowerMilliwatts`: GPU power consumption in milliwatts
- `NetworkMetrics`: Contains network activity statistics
  - `InPacketsPerSec`: Incoming packets per second
  - `InBytesPerSec`: Incoming bytes per second
  - `OutPacketsPerSec`: Outgoing packets per second
  - `OutBytesPerSec`: Outgoing bytes per second
- `DiskMetrics`: Contains disk activity statistics
  - `ReadOpsPerSec`: Read operations per second
  - `ReadBytesPerSec`: Read bytes per second
  - `WriteOpsPerSec`: Write operations per second
  - `WriteBytesPerSec`: Write bytes per second
- `InterruptMetrics`: Contains interrupt distribution per CPU
  - `CPUID`: CPU identifier
  - `TotalIRQ`: Total interrupts per second
  - `IPI`: Inter-processor interrupts per second
  - `TIMER`: Timer interrupts per second

### Building the CLI

```bash
# Build the CLI tool from the cli directory
cd cli
go build -o powermetrics-cli

# Run with sudo (required for powermetrics)
sudo ./powermetrics-cli
```

### CLI Options

```bash
sudo ./powermetrics-cli -help
```

Available options:

- `-interval`: Sampling interval (default 1s, e.g., 500ms, 1s, 2s)
- `-json`: Output metrics in JSON format
- `-system`: Only show system metrics
- `-process`: Only show process metrics
- `-cpu-residency`: Only show CPU residency metrics with detailed frequency breakdowns
- `-gpu-residency`: Only show GPU residency metrics with software/hardware state distributions
- `-network`: Only show network metrics (packets/bytes in/out per second)
- `-disk`: Only show disk metrics (I/O operations and throughput)
- `-battery`: Only show battery charge percentage
- `-interrupts`: Only show interrupt metrics per CPU
- `-debug`: Show debug information
- `-help`: Show help message

### CLI Examples

```bash
# Default output every second
sudo ./powermetrics-cli

# JSON output every 500ms
sudo ./powermetrics-cli -interval 500ms -json

# Only system metrics in JSON
sudo ./powermetrics-cli -system -json

# Show debug information
sudo ./powermetrics-cli -debug

# Output example (Apple Silicon Macs may show N/A for temperature values):
# CPU Power: 1.23 W, GPU Power: 0.45 W, ANE Power: 0.12 W, CPU Freq: 2447 MHz, GPU Freq: 338 MHz, CPU Temp: N/A, GPU Temp: N/A, ANE Busy: 0.00%
```

## License

Apache 2.0 license.
