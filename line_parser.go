package powermetrics

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/BinSquare/powermetrics-go/internal"
)

var (
	procLineRegex      = regexp.MustCompile(`^pid\s+(\d+)\s+(.+?)\s+([0-9]+(?:\.[0-9]+)?)\s*(us|ms|s)(?:\s+\(([0-9]+(?:\.[0-9]+)?)\s*%\))?(?:\s+.*)?$`)
	numberExtractor    = regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)`)
	clusterOnlineRegex = regexp.MustCompile(`([A-Z0-9-]+)-Cluster Online: ([\d.]+)%`)
	clusterHWFreqRegex = regexp.MustCompile(`([A-Z0-9-]+)-Cluster HW active frequency: ([\d.]+) MHz`)
)

// ParseLine parses a single line of powermetrics output and returns the derived metrics.
func (p *Parser) ParseLine(line string) (*Metrics, error) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "--") {
		return nil, nil
	}

	line = trimmed

	p.updateClusterInfo(line)

	if metrics, err := p.parseGPUProcessLine(line); err != nil {
		return nil, err
	} else if metrics != nil {
		return metrics, nil
	}

	lower := strings.ToLower(line)
	return p.parseSystemMetrics(line, lower), nil
}

func (p *Parser) parseGPUProcessLine(line string) (*Metrics, error) {
	matches := procLineRegex.FindStringSubmatch(line)
	if matches == nil {
		return nil, nil
	}

	pid, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil, nil
	}

	rawName := strings.TrimSpace(matches[2])
	valueStr := matches[3]
	unit := matches[4]
	percentStr := matches[5]

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return nil, nil
	}

	activeNs := convertToNanoseconds(value, unit)
	busy := deriveBusyPercent(activeNs, percentStr, p.config.SampleWindow)

	sample := internal.GPUProcessSample{
		PID:          pid,
		Name:         strings.Trim(rawName, "()"),
		BusyPercent:  busy,
		ActiveNanos:  activeNs,
		FrequencyMHz: p.frequencyMHz,
	}

	return &Metrics{
		GPUProcessSamples: []internal.GPUProcessSample{sample},
	}, nil
}

func (p *Parser) parseSystemMetrics(line, lower string) *Metrics {
	updated := false

	if hasAll(lower, "cpu", "power") && hasNone(lower, "gpu") {
		// Try to parse watts first
		if val, ok := parseTrailingValue(line, "w"); ok {
			p.system.CPUPowerWatts = val
			updated = true
		} else if val, ok := parseTrailingValue(line, "mW"); ok {
			// If watts not found, try milliwatts and convert to watts
			p.system.CPUPowerWatts = val / 1000.0
			updated = true
		}
	}

	if hasAll(lower, "cpu", "frequency") && hasNone(lower, "gpu") {
		if val, ok := parseTrailingValue(line, "mhz"); ok {
			p.system.CPUFrequencyMHz = val
			updated = true
		}
	}

	if hasAll(lower, "gpu", "busy") {
		if val, ok := parseTrailingValue(line, "%"); ok {
			p.system.GPUBusyPercent = val
			updated = true
		}
	}

	if hasAll(lower, "gpu", "hw active residency") {
		if val, ok := parseLeadingValueAfterColon(line, "%"); ok {
			p.system.GPUBusyPercent = val
			updated = true
		}
	}

	if hasAll(lower, "gpu", "idle residency") {
		if val, ok := parseLeadingValueAfterColon(line, "%"); ok {
			if p.system.GPUBusyPercent == 0 {
				p.system.GPUBusyPercent = clampPercent(100 - val)
			}
			updated = true
		}
	}

	if hasAll(lower, "ane", "busy") {
		if val, ok := parseTrailingValue(line, "%"); ok {
			p.system.ANEBusyPercent = val
			updated = true
		}
	}
	
	if hasAll(lower, "ane", "power") {
		// Try to parse watts first
		if val, ok := parseTrailingValue(line, "w"); ok {
			p.system.ANEPowerWatts = val
			updated = true
		} else if val, ok := parseTrailingValue(line, "mW"); ok {
			// If watts not found, try milliwatts and convert to watts
			p.system.ANEPowerWatts = val / 1000.0
			updated = true
		}
	}

	if hasAll(lower, "gpu", "power") {
		// Try to parse watts first
		if val, ok := parseTrailingValue(line, "w"); ok {
			p.system.GPUPowerWatts = val
			updated = true
		} else if val, ok := parseTrailingValue(line, "mW"); ok {
			// If watts not found, try milliwatts and convert to watts
			p.system.GPUPowerWatts = val / 1000.0
			updated = true
		}
	}

	if hasAll(lower, "dram", "power") {
		if val, ok := parseTrailingValue(line, "w"); ok {
			p.system.DRAMPowerWatts = val
			updated = true
		}
	}

	if hasAll(lower, "gpu", "frequency") {
		if val, ok := parseTrailingValue(line, "mhz"); ok {
			p.frequencyMHz = val
			p.system.GPUFrequencyMHz = val
			updated = true
		}
	}

	if hasAll(lower, "gpu", "temperature") {
		if val, ok := parseTrailingValue(line, "c"); ok {
			p.system.GPUTemperatureC = val
			updated = true
		}
	}

	if hasAll(lower, "cpu", "temperature") {
		if val, ok := parseTrailingValue(line, "c"); ok {
			p.system.CPUTemperatureC = val
			updated = true
		}
	}
	
	// Additional temperature patterns for different Mac systems
	if hasAll(lower, "gpu", "die", "temp") || hasAll(lower, "gpu", "junction", "temp") {
		if val, ok := parseTrailingValue(line, "c"); ok {
			p.system.GPUTemperatureC = val
			updated = true
		}
	}
	
	if hasAll(lower, "cpu", "die", "temp") || hasAll(lower, "cpu", "junction", "temp") || hasAll(lower, "package", "temp") {
		if val, ok := parseTrailingValue(line, "c"); ok {
			p.system.CPUTemperatureC = val
			updated = true
		}
	}
	
	// Look for temperature values that might not have explicit CPU/GPU labels
	if hasAll(lower, "temperature") && (hasAny(lower, "die", "junction", "package") || strings.Contains(lower, "sensor")) {
		if val, ok := parseTrailingValue(line, "c"); ok {
			// If we already have a CPU temp, assign to GPU, otherwise CPU
			if p.system.CPUTemperatureC == 0 {
				p.system.CPUTemperatureC = val
			} else if p.system.GPUTemperatureC == 0 {
				p.system.GPUTemperatureC = val
			}
			updated = true
		}
	}

	// Additional temperature patterns that may appear in different Mac systems
	if hasAll(lower, "junction", "temperature") && hasAny(lower, "cpu", "gpu") {
		if val, ok := parseTrailingValue(line, "c"); ok {
			if hasAny(lower, "cpu", "package") {
				p.system.CPUTemperatureC = val
			} else if hasAny(lower, "gpu") {
				p.system.GPUTemperatureC = val
			}
			updated = true
		}
	}

	// Check for "die temperature" patterns
	if hasAll(lower, "die", "temperature") {
		if val, ok := parseTrailingValue(line, "c"); ok {
			if hasAny(lower, "cpu", "package") {
				p.system.CPUTemperatureC = val
			} else if hasAny(lower, "gpu") {
				p.system.GPUTemperatureC = val
			}
			updated = true
		}
	}

	// Check for temperature values that have "T" prefix
	if strings.Contains(lower, "temperature") && strings.Contains(lower, "c") {
		if val, ok := parseTrailingValue(line, "c"); ok {
			// If we can't determine CPU vs GPU, set both but prefer based on content
			if hasAny(lower, "cpu", "package", "processor") {
				p.system.CPUTemperatureC = val
			} else if hasAny(lower, "gpu", "graphics") {
				p.system.GPUTemperatureC = val
			} else {
				// Set both if uncertain
				p.system.CPUTemperatureC = val
				p.system.GPUTemperatureC = val
			}
			updated = true
		}
	}

	if !updated {
		return nil
	}

	metrics := &Metrics{
		SystemSample: &p.system,
	}

	if clusters := p.clusterSnapshot(); len(clusters) > 0 {
		metrics.Clusters = clusters
	}

	return metrics
}

func (p *Parser) updateClusterInfo(line string) {
	if matches := clusterOnlineRegex.FindStringSubmatch(line); matches != nil {
		name := matches[1] + "-Cluster"
		onlinePercent, _ := strconv.ParseFloat(matches[2], 64)

		cluster := p.ensureCluster(name)
		cluster.OnlinePercent = onlinePercent
		return
	}

	if matches := clusterHWFreqRegex.FindStringSubmatch(line); matches != nil {
		name := matches[1] + "-Cluster"
		freqMHz, _ := strconv.ParseFloat(matches[2], 64)

		cluster := p.ensureCluster(name)
		cluster.HWActiveFreq = freqMHz
	}
}

func (p *Parser) ensureCluster(name string) *ClusterInfo {
	if cluster, exists := p.clusterInfo[name]; exists {
		return cluster
	}

	clusterType := "Performance"
	if strings.HasPrefix(strings.ToUpper(name), "E-") {
		clusterType = "Efficiency"
	}

	cluster := &ClusterInfo{
		Name: name,
		Type: clusterType,
	}
	p.clusterInfo[name] = cluster
	return cluster
}

func (p *Parser) clusterSnapshot() []ClusterInfo {
	if len(p.clusterInfo) == 0 {
		return nil
	}

	clusters := make([]ClusterInfo, 0, len(p.clusterInfo))
	for _, cluster := range p.clusterInfo {
		clusters = append(clusters, *cluster)
	}

	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].Name < clusters[j].Name
	})

	return clusters
}

func deriveBusyPercent(activeNs uint64, explicitPercent string, window time.Duration) float64 {
	if explicitPercent != "" {
		if parsed, err := strconv.ParseFloat(explicitPercent, 64); err == nil {
			return clampPercent(parsed)
		}
	}

	if window <= 0 || activeNs == 0 {
		return 0
	}

	computed := (float64(activeNs) / float64(window.Nanoseconds())) * 100
	return clampPercent(computed)
}

func clampPercent(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func hasAll(str string, tokens ...string) bool {
	for _, token := range tokens {
		if !strings.Contains(str, token) {
			return false
		}
	}
	return true
}

func hasNone(str string, tokens ...string) bool {
	for _, token := range tokens {
		if strings.Contains(str, token) {
			return false
		}
	}
	return true
}

func hasAny(str string, tokens ...string) bool {
	for _, token := range tokens {
		if strings.Contains(str, token) {
			return true
		}
	}
	return false
}

func convertToNanoseconds(value float64, unit string) uint64 {
	switch strings.ToLower(unit) {
	case "us":
		return uint64(value * 1e3)
	case "ms":
		return uint64(value * 1e6)
	case "s":
		return uint64(value * 1e9)
	default:
		return 0
	}
}

func parseTrailingValue(line, suffix string) (float64, bool) {
	idx := strings.LastIndex(strings.ToLower(line), strings.ToLower(suffix))
	if idx == -1 {
		return 0, false
	}

	segment := line[:idx]
	if parenIdx := strings.Index(segment, "("); parenIdx != -1 {
		segment = segment[:parenIdx]
	}
	if colonIdx := strings.LastIndex(segment, ":"); colonIdx != -1 {
		segment = segment[colonIdx+1:]
	}

	matches := numberExtractor.FindAllString(segment, -1)
	if len(matches) == 0 {
		return 0, false
	}

	val, err := strconv.ParseFloat(matches[len(matches)-1], 64)
	if err != nil {
		return 0, false
	}

	return val, true
}

func parseLeadingValueAfterColon(line, suffix string) (float64, bool) {
	colonIdx := strings.Index(line, ":")
	segment := line
	if colonIdx != -1 {
		segment = line[colonIdx+1:]
	}
	if parenIdx := strings.Index(segment, "("); parenIdx != -1 {
		segment = segment[:parenIdx]
	}

	idx := strings.Index(strings.ToLower(segment), strings.ToLower(suffix))
	if idx == -1 {
		return 0, false
	}

	segment = segment[:idx]
	matches := numberExtractor.FindAllString(segment, -1)
	if len(matches) == 0 {
		return 0, false
	}

	val, err := strconv.ParseFloat(matches[0], 64)
	if err != nil {
		return 0, false
	}

	return val, true
}
