# powermetrics-go

A Go library for parsing macOS powermetrics output to monitor system performance metrics including CPU/GPU power, frequency, temperature*, and process activity.
* Note: Temperature values may be 0 on Apple Silicon Macs (M1/M2/M3) as these systems report thermal pressure levels rather than direct temperature values in powermetrics output

## Overview

This library provides a Go interface for parsing the output from macOS's `powermetrics` command-line tool, which collects system performance data including power consumption, frequency, and thermal metrics.

## Features

- Parse system-wide metrics (CPU/GPU power, frequency, temperature*) 
  *Note: Temperature values may be 0 on Apple Silicon Macs (M1/M2/M3) as these systems report thermal pressure levels rather than direct temperature values in powermetrics output
- Parse per-process GPU activity
- Parse CPU cluster information
- Parse detailed CPU residency metrics with frequency breakdowns
- Parse GPU software/hardware state distributions
- Parse network activity metrics (packets/bytes in/out per second)
- Parse disk activity metrics (I/O operations and throughput)
- Parse battery charge percentage
- Parse interrupt distribution per CPU
- Support for ANE (Apple Neural Engine) power and busy metrics
- Support for both watts (W) and milliwatts (mW) power values (with automatic conversion)
- Configurable sampling intervals
- Structured data output for programmatic consumption

## Sample Output

The following sample output shows what each type of metric looks like based on the sample log:

### System-wide Metrics
```
CPU Power: 954 mW
GPU Power: 28 mW
ANE Power: 0 mW
Combined Power (CPU + GPU + ANE): 983 mW
```

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
- `Parser`: Handles invoking powermetrics and parsing output
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

### Functions

- `RunDefault(ctx)`: Run with default configuration
- `RunWithConfig(ctx, config)`: Run with custom configuration
- `NewParser(config)`: Create a new parser with configuration

## Command Line Interface

A command-line interface is included for easy debugging and direct usage.

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

## Apple Silicon Compatibility

This library is fully compatible with Apple Silicon Macs (M1/M2/M3). Note the following differences in behavior:

- Temperature values may be 0 as Apple Silicon Macs report thermal pressure levels rather than direct temperature values
- Power values are often reported in milliwatts (mW) and are automatically converted to watts (W) 
- ANE (Apple Neural Engine) metrics are supported and reported

## License

Apache 2.0 license.
