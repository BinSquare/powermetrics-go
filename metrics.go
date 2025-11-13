package powermetrics

// Metrics represents a single powermetrics sample.
type Metrics struct {
	SystemSample       *SystemSample
	GPUProcessSamples  []GPUProcessSample
	Clusters           []ClusterInfo
	CPUResidencies     []CPUResidencyMetrics
	ClusterResidencies []ClusterResidencyMetrics
	GPUResidency       *GPUResidencyMetrics
	Network            *NetworkMetrics
	Disk               *DiskMetrics
	Interrupts         []InterruptMetrics
}
