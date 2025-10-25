# powermetrics-go

A Go library for parsing macOS powermetrics output to monitor system performance metrics including CPU/GPU power, frequency, temperature, and process activity.

## Overview

This library provides a Go interface for parsing the output from macOS's `powermetrics` command-line tool, which collects system performance data including power consumption, frequency, and thermal metrics.

## Features

- Parse system-wide metrics (CPU/GPU power, frequency, temperature)
- Parse per-process GPU activity
- Parse CPU cluster information
- Configurable sampling intervals
- Structured data output for programmatic consumption

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
```

## License

Apache 2.0 license.
