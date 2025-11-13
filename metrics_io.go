package powermetrics

// NetworkMetrics captures network activity statistics.
type NetworkMetrics struct {
	InPacketsPerSec  float64
	InBytesPerSec    float64
	OutPacketsPerSec float64
	OutBytesPerSec   float64
}

// DiskMetrics captures disk activity statistics.
type DiskMetrics struct {
	ReadOpsPerSec    float64
	ReadBytesPerSec  float64
	WriteOpsPerSec   float64
	WriteBytesPerSec float64
}
