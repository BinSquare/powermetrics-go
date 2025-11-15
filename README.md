# powermetrics-go

Go library that wraps the macOS powermetrics to monitor system performance metrics including CPU/GPU power, frequency, and process activity.

You can use this library to build tools that use or display sys information.

- Powermetrics always outputs the Apple Neural Engine (ANE) data as 0 for me. Please contribute if I'm mis-parsing outputs!
- Temperature on Apple Silicon Macs (M1/M2/M3/M4) report thermal pressure "levels". Different versions of powermetrics may not follow same parsing.

## Requirements

- only works for macos if that wasn't obvious
- root (sudo required to run powermetrics under the hood)

## Installation

```bash
go get github.com/BinSquare/powermetrics-go
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

### Custom Configuration

```go
config := powermetrics.Config{
    SampleWindow:     500 * time.Millisecond,
    PowermetricsArgs: []string{"--samplers", "cpu_power,gpu_power", "-i", "500"},
}

parser := powermetrics.NewParser(config)
metricsChan, err := parser.Run(ctx)
```

## Notes on execution.

The `powermetrics` command requires root privileges to access system performance counters. This means you must run your application with `sudo`:

```bash
sudo go run your_program.go
```

Or if building an executable:

```bash
sudo ./your_program
```

### API

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

### Bundled an example in /example folder.

```bash
# Build the CLI tool from the cli directory
cd examples/cli
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

````bash
# Default output every second
sudo ./powermetrics-cli

# JSON output every 500ms
sudo ./powermetrics-cli -interval 500ms -json

# Only system metrics in JSON
sudo ./powermetrics-cli -system -json

# Show debug information
sudo ./powermetrics-cli -debug

# Output example (Apple Silicon Macs may show N/A for temperature values): # CPU Power: 1.23 W, GPU Power: 0.45 W, ANE Power: 0.12 W, CPU Freq: 2447 MHz, GPU Freq: 338 MHz, CPU Temp: N/A, GPU Temp: N/A, ANE Busy: 0.00% ```

## Samples

### CPU Cluster Information

````

E-Cluster Online: 100%
E-Cluster HW active frequency: 1293 MHz
E-Cluster HW active residency: 100.00% (1020 MHz: 75% 1404 MHz: 3.5% 1788 MHz: 5.1% 2112 MHz: 5.0% 2352 MHz: 5.0% 2532 MHz: 2.5% 2592 MHz: 3.9%)
E-Cluster idle residency: 0.00%
E-Cluster down residency: 0.00%

```

### GPU Residency Metrics

```

GPU HW active frequency: 338 MHz
GPU HW active residency: 1.63% (338 MHz: 1.6% 618 MHz: 0% 796 MHz: 0% 924 MHz: 0% 952 MHz: 0% 1056 MHz: 0% 1062 MHz: 0% 1182 MHz: 0% 1182 MHz: 0% 1312 MHz: 0% 1242 MHz: 0% 1380 MHz: 0% 1326 MHz: 0% 1470 MHz: 0% 1578 MHz: 0%)
GPU SW requested state: (P1 : 100% P2 : 0% P3 : 0% P4 : 0% P5 : 0% P6 : 0% P7 : 0% P8 : 0% P9 : 0% P10 : 0% P11 : 0% P12 : 0% P13 : 0% P14 : 0% P15 : 0%)
GPU SW state: (SW_P1 : 1.6% SW_P2 : 0% SW_P3 : 0% SW_P4 : 0% SW_P5 : 0% SW_P6 : 0% SW_P7 : 0% SW_P8 : 0% SW_P9 : 0% SW_P10 : 0% SW_P11 : 0% SW_P12 : 0% SW_P13 : 0% SW_P14 : 0% SW_P15 : 0%)
GPU idle residency: 98.37%
GPU Power: 28 mW

```

### Disk Metrics

```

read: 8.56 ops/s 45.67 KBytes/s
write: 73.88 ops/s 2070.85 KBytes/s

```

## License

Apache 2.0 license.

```

```

```
