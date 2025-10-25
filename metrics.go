package powermetrics

import "github.com/binsquare/benchtop/powermetrics-go/internal"

// Metrics represents a single powermetrics sample.
type Metrics struct {
	SystemSample      *internal.SystemSample
	GPUProcessSamples []internal.GPUProcessSample
	Errors            []string
	Clusters          []ClusterInfo
}

// ClusterInfo captures summary information about a CPU cluster.
type ClusterInfo struct {
	Name          string
	Type          string // "Performance" or "Efficiency"
	OnlinePercent float64
	HWActiveFreq  float64
}
