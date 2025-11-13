package powermetrics

// SystemSample captures system-level metrics reported by powermetrics.
type SystemSample struct {
	CPUPowerWatts   float64
	CPUFrequencyMHz float64
	GPUBusyPercent  float64
	GPUPowerWatts   float64
	GPUFrequencyMHz float64
	GPUTemperatureC float64
	CPUTemperatureC float64
	ANEBusyPercent  float64
	ANEPowerWatts   float64
	DRAMPowerWatts  float64
	BatteryPercent  float64
}
