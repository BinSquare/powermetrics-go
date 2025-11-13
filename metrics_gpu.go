package powermetrics

// GPUSoftwareStateData represents software state residency percentages.
type GPUSoftwareStateData map[string]float64

// GPUResidencyMetrics captures detailed GPU residency information.
type GPUResidencyMetrics struct {
	HWActiveResidency     float64
	HWActiveFreqResidency map[float64]float64
	SWRequestedStates     GPUSoftwareStateData
	SWStates              GPUSoftwareStateData
	IdleResidency         float64
	PowerMilliwatts       float64
}

// GPUProcessSample captures per-process GPU metrics.
type GPUProcessSample struct {
	PID          int
	Name         string
	BusyPercent  float64
	ActiveNanos  uint64
	FrequencyMHz float64
}
