package powermetrics

import "github.com/BinSquare/powermetrics-go/internal"

// Metrics represents a single powermetrics sample.
type Metrics struct {
	SystemSample      *internal.SystemSample
	GPUProcessSamples []internal.GPUProcessSample
	Errors            []string
	Clusters          []ClusterInfo
	CPUResidencies    []internal.CPUResidencyMetrics
	ClusterResidencies []internal.ClusterResidencyMetrics
	GPUResidency      *internal.GPUResidencyMetrics
	Network           *internal.NetworkMetrics
	Disk              *internal.DiskMetrics
	Interrupts        []internal.InterruptMetrics
}

// ClusterInfo captures summary information about a CPU cluster.
type ClusterInfo struct {
	Name          string
	Type          string // "Performance" or "Efficiency"
	OnlinePercent float64
	HWActiveFreq  float64
}
