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
		interval        = flag.Duration("interval", 1*time.Second, "sampling interval (e.g., 500ms, 1s, 2s)")
		jsonOutput      = flag.Bool("json", false, "output metrics in JSON format")
		onlySystem      = flag.Bool("system", false, "only show system metrics, skip process metrics")
		onlyProcess     = flag.Bool("process", false, "only show process metrics, skip system metrics")
		onlyCPUResidency = flag.Bool("cpu-residency", false, "only show CPU residency metrics")
		onlyGPUResidency = flag.Bool("gpu-residency", false, "only show GPU residency metrics")
		onlyNetwork     = flag.Bool("network", false, "only show network metrics")
		onlyDisk        = flag.Bool("disk", false, "only show disk metrics")
		onlyBattery     = flag.Bool("battery", false, "only show battery metrics")
		onlyInterrupts  = flag.Bool("interrupts", false, "only show interrupt metrics")
		help            = flag.Bool("help", false, "show help message")
		debug           = flag.Bool("debug", false, "show debug information")
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
		fmt.Printf("Debug: CPU Residency only: %t\n", *onlyCPUResidency)
		fmt.Printf("Debug: GPU Residency only: %t\n", *onlyGPUResidency)
		fmt.Printf("Debug: Network only: %t\n", *onlyNetwork)
		fmt.Printf("Debug: Disk only: %t\n", *onlyDisk)
		fmt.Printf("Debug: Battery only: %t\n", *onlyBattery)
		fmt.Printf("Debug: Interrupts only: %t\n", *onlyInterrupts)
	}

	// Create config with custom interval - using more reliable sampler configuration
	config := powermetrics.Config{
		SampleWindow:     *interval,
		PowermetricsArgs: []string{"--samplers", "tasks,battery,network,disk,interrupts,cpu_power,gpu_power,ane_power,thermal", "--show-process-gpu", "--show-initial-usage", "-i", fmt.Sprintf("%d", interval.Milliseconds())},
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

	// Process metrics - add rate limiting to respect the specified interval
	lastOutputTime := time.Now()
	
	for metrics := range metricsChan {
		if *debug {
			fmt.Println("Debug: Received metrics")
		}

		// Check if enough time has passed since the last output
		if time.Since(lastOutputTime) < *interval {
			continue // Skip this metric, wait for the interval to pass
		}
		
		// Update the last output time
		lastOutputTime = time.Now()

		if *onlyCPUResidency {
			if len(metrics.CPUResidencies) > 0 {
				if *jsonOutput {
					data, _ := json.Marshal(metrics.CPUResidencies)
					fmt.Println(string(data))
				} else {
					fmt.Printf("CPU Residencies: %d\n", len(metrics.CPUResidencies))
					for _, cpu := range metrics.CPUResidencies {
						fmt.Printf("  CPU %d: Freq %.0f MHz, Active: %.2f%%, Idle: %.2f%%, Down: %.2f%%\n",
							cpu.CPUID, cpu.Frequency, calculateTotalActive(cpu.ActiveResidency), cpu.IdleResidency, cpu.DownResidency)
						if len(cpu.ActiveResidency) > 0 {
							fmt.Printf("    Frequency Residency: ")
							for freq, percent := range cpu.ActiveResidency {
								fmt.Printf("%.0fMHz:%.2f%% ", freq, percent)
							}
							fmt.Printf("\n")
						}
					}
				}
			}
		} else if *onlyGPUResidency {
			if metrics.GPUResidency != nil {
				if *jsonOutput {
					data, _ := json.Marshal(metrics.GPUResidency)
					fmt.Println(string(data))
				} else {
					fmt.Printf("GPU Residency: HW Active: %.2f%%, Idle: %.2f%%, Power: %.2f mW\n",
						metrics.GPUResidency.HWActiveResidency, metrics.GPUResidency.IdleResidency, metrics.GPUResidency.PowerMilliwatts)
					if len(metrics.GPUResidency.HWActiveFreqResidency) > 0 {
						fmt.Printf("  Frequency Residency: ")
						for freq, percent := range metrics.GPUResidency.HWActiveFreqResidency {
							fmt.Printf("%.0fMHz:%.2f%% ", freq, percent)
						}
						fmt.Printf("\n")
					}
					if len(metrics.GPUResidency.SWRequestedStates) > 0 {
						fmt.Printf("  SW Requested States: ")
						for state, percent := range metrics.GPUResidency.SWRequestedStates {
							fmt.Printf("%s:%.2f%% ", state, percent)
						}
						fmt.Printf("\n")
					}
					if len(metrics.GPUResidency.SWStates) > 0 {
						fmt.Printf("  SW States: ")
						for state, percent := range metrics.GPUResidency.SWStates {
							fmt.Printf("%s:%.2f%% ", state, percent)
						}
						fmt.Printf("\n")
					}
				}
			}
		} else if *onlyNetwork {
			if metrics.Network != nil {
				if *jsonOutput {
					data, _ := json.Marshal(metrics.Network)
					fmt.Println(string(data))
				} else {
					fmt.Printf("Network: Out %d packets/s, %d bytes/s | In %d packets/s, %d bytes/s\n",
						int(metrics.Network.OutPacketsPerSec), int(metrics.Network.OutBytesPerSec),
						int(metrics.Network.InPacketsPerSec), int(metrics.Network.InBytesPerSec))
				}
			}
		} else if *onlyDisk {
			if metrics.Disk != nil {
				if *jsonOutput {
					data, _ := json.Marshal(metrics.Disk)
					fmt.Println(string(data))
				} else {
					fmt.Printf("Disk: Read %d ops/s, %d bytes/s | Write %d ops/s, %d bytes/s\n",
						int(metrics.Disk.ReadOpsPerSec), int(metrics.Disk.ReadBytesPerSec),
						int(metrics.Disk.WriteOpsPerSec), int(metrics.Disk.WriteBytesPerSec))
				}
			}
		} else if *onlyBattery {
			if metrics.SystemSample != nil && metrics.SystemSample.BatteryPercent > 0 {
				if *jsonOutput {
					data, _ := json.Marshal(map[string]float64{"battery_percent": metrics.SystemSample.BatteryPercent})
					fmt.Println(string(data))
				} else {
					fmt.Printf("Battery: %.2f%%\n", metrics.SystemSample.BatteryPercent)
				}
			}
		} else if *onlyInterrupts {
			if len(metrics.Interrupts) > 0 {
				if *jsonOutput {
					data, _ := json.Marshal(metrics.Interrupts)
					fmt.Println(string(data))
				} else {
					fmt.Printf("Interrupts: %d CPUs\n", len(metrics.Interrupts))
					for _, intr := range metrics.Interrupts {
						fmt.Printf("  CPU %d: Total IRQs %.2f/s, IPI %.2f/s, TIMER %.2f/s\n", 
							intr.CPUID, intr.TotalIRQ, intr.IPI, intr.TIMER)
					}
				}
			}
		} else if *onlyProcess {
			if len(metrics.GPUProcessSamples) > 0 {
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
			} else if *debug {
				fmt.Println("Debug: No GPU process samples available in this metrics update")
			}
		} else if *onlySystem && metrics.SystemSample != nil {
			if *jsonOutput {
				data, _ := json.Marshal(metrics.SystemSample)
				fmt.Println(string(data))
			} else {
				fmt.Printf("CPU Power: %.2f W, GPU Power: %.2f W, ANE Power: %.2f W, CPU Freq: %.0f MHz, GPU Freq: %.0f MHz, CPU Temp: %.2f°C, GPU Temp: %.2f°C, ANE Busy: %.2f%%, Battery: %.2f%%\n",
					metrics.SystemSample.CPUPowerWatts, metrics.SystemSample.GPUPowerWatts, metrics.SystemSample.ANEPowerWatts,
					metrics.SystemSample.CPUFrequencyMHz, metrics.SystemSample.GPUFrequencyMHz,
					metrics.SystemSample.CPUTemperatureC, metrics.SystemSample.GPUTemperatureC,
					metrics.SystemSample.ANEBusyPercent, metrics.SystemSample.BatteryPercent)
			}
		} else if !*onlyProcess && !*onlySystem && !*onlyCPUResidency && !*onlyGPUResidency && 
			!*onlyNetwork && !*onlyDisk && !*onlyBattery && !*onlyInterrupts {
			// Show all metrics
			output := make(map[string]interface{})

			if metrics.SystemSample != nil {
				if *jsonOutput {
					output["system"] = metrics.SystemSample
				} else {
					fmt.Printf("CPU Power: %.2f W, GPU Power: %.2f W, CPU Freq: %.0f MHz, GPU Freq: %.0f MHz, CPU Temp: %.2f°C, GPU Temp: %.2f°C, ANE Busy: %.2f%%, Battery: %.2f%%\n",
						metrics.SystemSample.CPUPowerWatts, metrics.SystemSample.GPUPowerWatts,
						metrics.SystemSample.CPUFrequencyMHz, metrics.SystemSample.GPUFrequencyMHz,
						metrics.SystemSample.CPUTemperatureC, metrics.SystemSample.GPUTemperatureC,
						metrics.SystemSample.ANEBusyPercent, metrics.SystemSample.BatteryPercent)
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

			if len(metrics.CPUResidencies) > 0 {
				if *jsonOutput {
					output["cpu_residencies"] = metrics.CPUResidencies
				} else {
					fmt.Printf("CPU Residencies: %d\n", len(metrics.CPUResidencies))
					for _, cpu := range metrics.CPUResidencies {
						fmt.Printf("  CPU %d: Freq %.0f MHz, Active: %.2f%%, Idle: %.2f%%, Down: %.2f%%\n",
							cpu.CPUID, cpu.Frequency, calculateTotalActive(cpu.ActiveResidency), cpu.IdleResidency, cpu.DownResidency)
					}
				}
			}

			if metrics.GPUResidency != nil {
				if *jsonOutput {
					output["gpu_residency"] = metrics.GPUResidency
				} else {
					fmt.Printf("GPU Residency: HW Active: %.2f%%, Idle: %.2f%%, Power: %.2f mW\n",
						metrics.GPUResidency.HWActiveResidency, metrics.GPUResidency.IdleResidency, metrics.GPUResidency.PowerMilliwatts)
				}
			}

			if metrics.Network != nil {
				if *jsonOutput {
					output["network"] = metrics.Network
				} else {
					fmt.Printf("Network: Out %d packets/s, %d bytes/s | In %d packets/s, %d bytes/s\n",
						int(metrics.Network.OutPacketsPerSec), int(metrics.Network.OutBytesPerSec),
						int(metrics.Network.InPacketsPerSec), int(metrics.Network.InBytesPerSec))
				}
			}

			if metrics.Disk != nil {
				if *jsonOutput {
					output["disk"] = metrics.Disk
				} else {
					fmt.Printf("Disk: Read %d ops/s, %d bytes/s | Write %d ops/s, %d bytes/s\n",
						int(metrics.Disk.ReadOpsPerSec), int(metrics.Disk.ReadBytesPerSec),
						int(metrics.Disk.WriteOpsPerSec), int(metrics.Disk.WriteBytesPerSec))
				}
			}

			if len(metrics.Interrupts) > 0 {
				if *jsonOutput {
					output["interrupts"] = metrics.Interrupts
				} else {
					fmt.Printf("Interrupts: %d CPUs\n", len(metrics.Interrupts))
					for _, intr := range metrics.Interrupts {
						fmt.Printf("  CPU %d: Total IRQs %.2f/s, IPI %.2f/s, TIMER %.2f/s\n", 
							intr.CPUID, intr.TotalIRQ, intr.IPI, intr.TIMER)
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

// Helper function to calculate total active residency from the frequency map
func calculateTotalActive(residencyMap map[float64]float64) float64 {
	total := 0.0
	for _, percent := range residencyMap {
		total += percent
	}
	return total
}