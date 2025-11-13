package powermetrics

// InterruptMetrics captures interrupt distribution per CPU.
type InterruptMetrics struct {
	CPUID    int
	TotalIRQ float64
	IPI      float64
	TIMER    float64
}
