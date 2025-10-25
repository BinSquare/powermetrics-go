package powermetrics

import (
	"fmt"
	"time"
)

const defaultPowermetricsPath = "/usr/bin/powermetrics"

var defaultPowermetricsArgs = []string{
	"--samplers", "default",
	"--show-process-gpu",
	"-i", "1000",
}

// Config holds configuration for the powermetrics collector.
type Config struct {
	PowermetricsPath string
	PowermetricsArgs []string
	SampleWindow     time.Duration
}

func normalizeConfig(cfg Config) Config {
	normalized := cfg

	if normalized.PowermetricsPath == "" {
		normalized.PowermetricsPath = defaultPowermetricsPath
	}

	args := normalized.PowermetricsArgs
	if len(args) == 0 {
		args = append([]string{}, defaultPowermetricsArgs...)
	} else {
		args = append([]string{}, args...)
	}

	window := normalized.SampleWindow
	if window <= 0 {
		window = time.Second
	}

	args = ensureIntervalArgument(args, window)

	normalized.PowermetricsArgs = args
	normalized.SampleWindow = window

	return normalized
}

func ensureIntervalArgument(args []string, window time.Duration) []string {
	interval := fmt.Sprintf("%d", window.Milliseconds())
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-i" {
			args[i+1] = interval
			return args
		}
	}
	return append(args, "-i", interval)
}
