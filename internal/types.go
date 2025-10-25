package internal

// SystemSample captures system-level metrics
type SystemSample struct {
	CPUPowerWatts      float64
	CPUFrequencyMHz    float64
	GPUBusyPercent     float64
	GPUPowerWatts      float64
	GPUFrequencyMHz    float64
	GPUTemperatureC    float64
	CPUTemperatureC    float64
	ANEBusyPercent     float64
	DRAMPowerWatts     float64
}

// GPUProcessSample captures per-process GPU metrics
type GPUProcessSample struct {
	PID          int
	Name         string
	BusyPercent  float64
	ActiveNanos  uint64
	FrequencyMHz float64
}