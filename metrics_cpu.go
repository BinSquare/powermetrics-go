package powermetrics

// CPUResidencyData represents frequency residency percentages for a CPU.
type CPUResidencyData map[float64]float64

// CPUResidencyMetrics captures detailed CPU residency information.
type CPUResidencyMetrics struct {
	CPUID           int
	ActiveResidency CPUResidencyData
	IdleResidency   float64
	DownResidency   float64
	Frequency       float64
}

// ClusterInfo captures summary information about a CPU cluster.
type ClusterInfo struct {
	Name          string
	Type          string // "Performance" or "Efficiency"
	OnlinePercent float64
	HWActiveFreq  float64
}

// ClusterResidencyMetrics captures detailed cluster residency information.
type ClusterResidencyMetrics struct {
	Name                  string
	Type                  string
	OnlinePercent         float64
	HWActiveFreq          float64
	HWActiveResidency     float64
	HWActiveFreqResidency map[float64]float64
	IdleResidency         float64
	DownResidency         float64
}
