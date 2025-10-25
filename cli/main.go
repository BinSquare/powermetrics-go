package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/BinSquare/powermetrics-go"
)

func main() {
	var (
		interval    = flag.Duration("interval", 1*time.Second, "sampling interval (e.g., 500ms, 1s, 2s)")
		jsonOutput  = flag.Bool("json", false, "output metrics in JSON format")
		onlySystem  = flag.Bool("system", false, "only show system metrics, skip process metrics")
		onlyProcess = flag.Bool("process", false, "only show process metrics, skip system metrics")
		help        = flag.Bool("help", false, "show help message")
		debug       = flag.Bool("debug", false, "show debug information")
	)

	flag.Parse()

	if *help {
		fmt.Println("powermetrics-go CLI tool")
		fmt.Println("Usage: sudo ./powermetrics-go [options]")
		fmt.Println("")
		fmt.Println("Options:")
		flag.PrintDefaults()
		return
	}

	if *debug {
		fmt.Println("Debug: Starting powermetrics collection")
		fmt.Printf("Debug: Interval: %v\n", *interval)
		fmt.Printf("Debug: JSON Output: %t\n", *jsonOutput)
		fmt.Printf("Debug: System only: %t\n", *onlySystem)
		fmt.Printf("Debug: Process only: %t\n", *onlyProcess)
	}

	// Create config with custom interval
	config := powermetrics.Config{
		SampleWindow:     *interval,
		PowermetricsArgs: []string{"--samplers", "cpu_power,gpu_power,thermal", "-i", fmt.Sprintf("%d", interval.Milliseconds())},
	}

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		fmt.Println("\nReceived signal, stopping...")
		cancel()
	}()

	// Start collecting metrics (requires sudo)
	if *debug {
		fmt.Println("Debug: Starting powermetrics parser")
	}
	parser := powermetrics.NewParser(config)
	metricsChan, err := parser.Run(ctx)
	if err != nil {
		log.Fatal("Failed to start powermetrics: ", err)
	}

	if *debug {
		fmt.Println("Debug: Successfully started metrics collection")
		fmt.Println("Debug: Waiting for metrics...")
	}

	// Process metrics
	for metrics := range metricsChan {
		if *debug {
			fmt.Println("Debug: Received metrics")
		}

		if *onlyProcess && len(metrics.GPUProcessSamples) > 0 {
			if *jsonOutput {
				data, _ := json.Marshal(metrics.GPUProcessSamples)
				fmt.Println(string(data))
			} else {
				fmt.Printf("GPU Processes: %d\n", len(metrics.GPUProcessSamples))
				for _, proc := range metrics.GPUProcessSamples {
					fmt.Printf("  PID: %d, Name: %s, Busy: %.2f%%, Active: %d ns\n", 
						proc.PID, proc.Name, proc.BusyPercent, proc.ActiveNanos)
				}
			}
		} else if *onlySystem && metrics.SystemSample != nil {
			if *jsonOutput {
				data, _ := json.Marshal(metrics.SystemSample)
				fmt.Println(string(data))
			} else {
				fmt.Printf("CPU Power: %.2f W, GPU Power: %.2f W, CPU Temp: %.2f°C, GPU Temp: %.2f°C\n",
					metrics.SystemSample.CPUPowerWatts, metrics.SystemSample.GPUPowerWatts,
					metrics.SystemSample.CPUTemperatureC, metrics.SystemSample.GPUTemperatureC)
			}
		} else if !*onlyProcess && !*onlySystem {
			// Show all metrics
			output := make(map[string]interface{})

			if metrics.SystemSample != nil {
				if *jsonOutput {
					output["system"] = metrics.SystemSample
				} else {
					fmt.Printf("CPU Power: %.2f W, GPU Power: %.2f W, CPU Freq: %.0f MHz, GPU Freq: %.0f MHz, CPU Temp: %.2f°C, GPU Temp: %.2f°C, ANE Busy: %.2f%%\n",
						metrics.SystemSample.CPUPowerWatts, metrics.SystemSample.GPUPowerWatts,
						metrics.SystemSample.CPUFrequencyMHz, metrics.SystemSample.GPUFrequencyMHz,
						metrics.SystemSample.CPUTemperatureC, metrics.SystemSample.GPUTemperatureC,
						metrics.SystemSample.ANEBusyPercent)
				}
			}

			if len(metrics.GPUProcessSamples) > 0 {
				if *jsonOutput {
					output["processes"] = metrics.GPUProcessSamples
				} else {
					fmt.Printf("GPU Processes: %d\n", len(metrics.GPUProcessSamples))
					for _, proc := range metrics.GPUProcessSamples {
						fmt.Printf("  PID: %d, Name: %s, Busy: %.2f%%, Active: %d ns\n", 
							proc.PID, proc.Name, proc.BusyPercent, proc.ActiveNanos)
					}
				}
			}

			if len(metrics.Clusters) > 0 {
				if *jsonOutput {
					output["clusters"] = metrics.Clusters
				} else {
					fmt.Printf("CPU Clusters: %d\n", len(metrics.Clusters))
					for _, cluster := range metrics.Clusters {
						fmt.Printf("  Name: %s, Type: %s, Online: %.2f%%, Freq: %.0f MHz\n",
							cluster.Name, cluster.Type, cluster.OnlinePercent, cluster.HWActiveFreq)
					}
				}
			}

			if *jsonOutput {
				data, _ := json.Marshal(output)
				fmt.Println(string(data))
			}
		}

		if *debug {
			fmt.Println("Debug: Metrics processed, continuing...")
		}
	}

	if *debug {
		fmt.Println("Debug: Exiting")
	}
}