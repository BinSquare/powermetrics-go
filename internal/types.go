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
	ANEPowerWatts      float64
	DRAMPowerWatts     float64
	BatteryPercent     float64
}

// CPUResidencyData represents frequency residency percentages for a CPU
type CPUResidencyData map[float64]float64  // frequency -> percentage

// CPUResidencyMetrics captures detailed CPU residency information
type CPUResidencyMetrics struct {
	CPUID              int
	ActiveResidency    CPUResidencyData
	IdleResidency      float64
	DownResidency      float64
	Frequency          float64
}

// ClusterResidencyMetrics captures detailed cluster residency information
type ClusterResidencyMetrics struct {
	Name           string
	Type           string // "Performance" or "Efficiency"
	OnlinePercent  float64
	HWActiveFreq   float64
	HWActiveResidency float64
	HWActiveFreqResidency map[float64]float64  // frequency -> percentage
	IdleResidency  float64
	DownResidency  float64
}

// GPUSoftwareStateData represents software state percentages
type GPUSoftwareStateData map[string]float64  // state -> percentage

// GPUResidencyMetrics captures detailed GPU residency information
type GPUResidencyMetrics struct {
	HWActiveResidency float64
	HWActiveFreqResidency map[float64]float64  // frequency -> percentage
	SWRequestedStates GPUSoftwareStateData
	SWStates         GPUSoftwareStateData
	IdleResidency    float64
	PowerMilliwatts  float64
}

// NetworkMetrics captures network activity statistics
type NetworkMetrics struct {
	InPacketsPerSec   float64
	InBytesPerSec     float64
	OutPacketsPerSec  float64
	OutBytesPerSec    float64
}

// DiskMetrics captures disk activity statistics
type DiskMetrics struct {
	ReadOpsPerSec      float64
	ReadBytesPerSec    float64
	WriteOpsPerSec     float64
	WriteBytesPerSec   float64
}

// InterruptMetrics captures interrupt distribution per CPU
type InterruptMetrics struct {
	CPUID      int
	TotalIRQ   float64
	IPI        float64
	TIMER      float64
}

// GPUProcessSample captures per-process GPU metrics
type GPUProcessSample struct {
	PID          int
	Name         string
	BusyPercent  float64
	ActiveNanos  uint64
	FrequencyMHz float64
}