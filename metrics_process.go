package powermetrics

// ProcessSample captures per-process CPU metrics from the powermetrics "Running tasks" table.
type ProcessSample struct {
	PID               int
	Name              string
	CPUMsPerSec       float64
	UserPercent       float64
	DeadlinesLT2Ms    float64
	Deadlines2To5Ms   float64
	WakeupsInterrupts float64
	WakeupsPkgIdle    float64
}
