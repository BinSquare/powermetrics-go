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
	cpuFreqResidencyRegex = regexp.MustCompile(`(\d+) MHz: +([\d.]+)%`)
	clusterFreqResidencyRegex = regexp.MustCompile(`(\d+) MHz: +([\d.]+)%`)
	clusterHWActiveResidencyRegex = regexp.MustCompile(`HW active residency: +([\d.]+)%`)
	cpuActiveResidencyRegex = regexp.MustCompile(`active residency: +([\d.]+)%`)
	cpuIdleResidencyRegex = regexp.MustCompile(`idle residency: +([\d.]+)%`)
	cpuDownResidencyRegex = regexp.MustCompile(`down residency: +([\d.]+)%`)
	batteryRegex = regexp.MustCompile(`Battery: percent_charge: ([\d.]+)`)
	networkRegex = regexp.MustCompile(`out: ([\d.]+) packets/s, ([\d.]+) bytes/s`)
	networkInRegex = regexp.MustCompile(`in: +([\d.]+) packets/s, ([\d.]+) bytes/s`)
	diskReadRegex = regexp.MustCompile(`read: ([\d.]+) ops/s ([\d.]+) KBytes/s`)
	diskWriteRegex = regexp.MustCompile(`write: ([\d.]+) ops/s ([\d.]+) KBytes/s`)
	interruptRegex = regexp.MustCompile(`CPU (\d+):`)
	interruptTotalRegex = regexp.MustCompile(`Total IRQ: ([\d.]+) interrupts/sec`)
	interruptIPITimerRegex = regexp.MustCompile(`\|-> (IPI|TIMER): ([\d.]+) interrupts/sec`)
	gpuFreqRegex = regexp.MustCompile(`GPU HW active frequency: ([\d.]+) MHz`)
	gpuHwActiveResidencyRegex = regexp.MustCompile(`GPU HW active residency: +([\d.]+)%`)
	gpuIdleResidencyRegex = regexp.MustCompile(`GPU idle residency: +([\d.]+)%`)
	gpuSWStateRegex = regexp.MustCompile(`GPU SW (?:requested state|state): \(([^)]+)\)`)
)

// ParseLine parses a single line of powermetrics output and returns the derived metrics.
func (p *Parser) ParseLine(line string) (*Metrics, error) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "--") {
		return nil, nil
	}

	line = trimmed

	// Handle sections
	if strings.Contains(line, "**** Processor usage ****") {
		// This indicates we're starting the processor usage section
		return nil, nil
	} else if strings.Contains(line, "**** Network activity ****") {
		// This indicates we're starting the network activity section
		return nil, nil
	} else if strings.Contains(line, "**** Disk activity ****") {
		// This indicates we're starting the disk activity section
		return nil, nil
	} else if strings.Contains(line, "****  Interrupt distribution ****") {
		// This indicates we're starting the interrupt distribution section
		return nil, nil
	} else if strings.Contains(line, "**** GPU usage ****") {
		// This indicates we're starting the GPU usage section
		return nil, nil
	} else if strings.Contains(line, "**** Battery and backlight usage ****") {
		// This indicates we're starting the battery section
		return nil, nil
	}

	// Check if existing values were nil before update to detect new data
	prevNetworkInfoWasNil := p.networkInfo == nil
	prevDiskInfoWasNil := p.diskInfo == nil
	prevSystem := p.system

	p.updateClusterInfo(line)
	p.updateCPUInfo(line)
	p.updateNetworkInfo(line)
	p.updateDiskInfo(line)
	p.updateInterruptInfo(line)
	p.updateGPUResidencyInfo(line)
	p.updateBatteryInfo(line)

	// Check if any values changed or new values were added to decide whether to return metrics
	systemChanged := p.system != prevSystem
	networkChanged := (p.networkInfo != nil) && (prevNetworkInfoWasNil || p.networkInfo.InPacketsPerSec > 0 || p.networkInfo.OutPacketsPerSec > 0)
	diskChanged := (p.diskInfo != nil) && (prevDiskInfoWasNil || p.diskInfo.ReadOpsPerSec > 0 || p.diskInfo.WriteOpsPerSec > 0)
	clusterChanged := len(p.clusterInfo) > 0 // If cluster info is added, mark as changed
	cpuResidencyChanged := len(p.cpuResidencies) > 0 // If CPU residency info is added, mark as changed
	clusterResidencyChanged := len(p.clusterResidencies) > 0 // If cluster residency info is added, mark as changed
	gpuResidencyChanged := p.gpuResidency != nil && (p.gpuResidency.HWActiveResidency > 0 || p.gpuResidency.IdleResidency > 0 || len(p.gpuResidency.HWActiveFreqResidency) > 0 || len(p.gpuResidency.SWStates) > 0)

	if metrics, err := p.parseGPUProcessLine(line); err != nil {
		return nil, err
	} else if metrics != nil {
		return metrics, nil
	}

	lower := strings.ToLower(line)
	systemMetrics := p.parseSystemMetrics(line, lower)
	
	// If any metrics-related data changed, return the full metrics structure
	if systemChanged || networkChanged || diskChanged || clusterChanged || 
	   cpuResidencyChanged || clusterResidencyChanged || gpuResidencyChanged {
		return p.buildMetrics(), nil
	}
	
	return systemMetrics, nil
}

func (p *Parser) buildMetrics() *Metrics {
	metrics := &Metrics{}

	if p.networkInfo != nil {
		metrics.Network = p.networkInfo
	}

	if p.diskInfo != nil {
		metrics.Disk = p.diskInfo
	}

	if clusters := p.clusterSnapshot(); len(clusters) > 0 {
		metrics.Clusters = clusters
	}

	// Add new metrics
	if len(p.cpuResidencies) > 0 {
		cpuResidencies := make([]internal.CPUResidencyMetrics, 0, len(p.cpuResidencies))
		for _, cpu := range p.cpuResidencies {
			cpuResidencies = append(cpuResidencies, *cpu)
		}
		metrics.CPUResidencies = cpuResidencies
	}

	if len(p.clusterResidencies) > 0 {
		clusterResidencies := make([]internal.ClusterResidencyMetrics, 0, len(p.clusterResidencies))
		for _, cluster := range p.clusterResidencies {
			clusterResidencies = append(clusterResidencies, *cluster)
		}
		metrics.ClusterResidencies = clusterResidencies
	}

	if p.gpuResidency != nil && (p.gpuResidency.HWActiveResidency > 0 || p.gpuResidency.IdleResidency > 0 || len(p.gpuResidency.HWActiveFreqResidency) > 0 || len(p.gpuResidency.SWStates) > 0) {
		metrics.GPUResidency = p.gpuResidency
	}

	if len(p.interruptInfo) > 0 {
		interrupts := make([]internal.InterruptMetrics, 0, len(p.interruptInfo))
		for _, interrupt := range p.interruptInfo {
			interrupts = append(interrupts, *interrupt)
		}
		metrics.Interrupts = interrupts
	}

	// Always include system metrics even if not updated from current line
	metrics.SystemSample = &p.system

	return metrics
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

	// Only return metrics if this specific line contributed to system metrics data
	if !updated {
		return nil
	}

	metrics := &Metrics{
		SystemSample: &p.system,
	}

	if clusters := p.clusterSnapshot(); len(clusters) > 0 {
		metrics.Clusters = clusters
	}

	// Add new metrics
	if len(p.cpuResidencies) > 0 {
		cpuResidencies := make([]internal.CPUResidencyMetrics, 0, len(p.cpuResidencies))
		for _, cpu := range p.cpuResidencies {
			cpuResidencies = append(cpuResidencies, *cpu)
		}
		metrics.CPUResidencies = cpuResidencies
	}

	if len(p.clusterResidencies) > 0 {
		clusterResidencies := make([]internal.ClusterResidencyMetrics, 0, len(p.clusterResidencies))
		for _, cluster := range p.clusterResidencies {
			clusterResidencies = append(clusterResidencies, *cluster)
		}
		metrics.ClusterResidencies = clusterResidencies
	}

	if p.gpuResidency != nil && (p.gpuResidency.HWActiveResidency > 0 || p.gpuResidency.IdleResidency > 0 || len(p.gpuResidency.HWActiveFreqResidency) > 0) {
		metrics.GPUResidency = p.gpuResidency
	}

	if p.networkInfo != nil {
		metrics.Network = p.networkInfo
	}

	if p.diskInfo != nil {
		metrics.Disk = p.diskInfo
	}

	if len(p.interruptInfo) > 0 {
		interrupts := make([]internal.InterruptMetrics, 0, len(p.interruptInfo))
		for _, interrupt := range p.interruptInfo {
			interrupts = append(interrupts, *interrupt)
		}
		metrics.Interrupts = interrupts
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

func (p *Parser) updateCPUInfo(line string) {
	// Check if the line is for a specific CPU frequency like "CPU 0 frequency: 1338 MHz"
	cpuFreqMatch := regexp.MustCompile(`CPU (\d+) frequency: ([\d.]+) MHz`).FindStringSubmatch(line)
	if cpuFreqMatch != nil {
		cpuID, _ := strconv.Atoi(cpuFreqMatch[1])
		freq, _ := strconv.ParseFloat(cpuFreqMatch[2], 64)
		cpu := p.ensureCPUResidency(cpuID)
		cpu.Frequency = freq
		return
	}

	// Check if the line is for a specific CPU interrupt line
	cpuMatch := interruptRegex.FindStringSubmatch(line)
	if cpuMatch != nil {
		cpuID, _ := strconv.Atoi(cpuMatch[1])
		p.ensureCPUResidency(cpuID)
		return
	}

	// Check for line like "CPU 0 active residency:  55.11% (1020 MHz:  39% 1404 MHz: 2.2%...)"
	cpuResidencyMatch := regexp.MustCompile(`CPU (\d+) active residency: +([\d.]+)%`).FindStringSubmatch(line)
	if cpuResidencyMatch != nil {
		cpuID, _ := strconv.Atoi(cpuResidencyMatch[1])
		cpu := p.ensureCPUResidency(cpuID)
		
		// Parse the frequency residency data from the parentheses
		openParenIdx := strings.Index(line, "(")
		if openParenIdx != -1 {
			freqDataStr := line[openParenIdx+1:]
			freqDataStr = strings.TrimRight(freqDataStr, ")")
			cpu.ActiveResidency = parseFreqResidency(freqDataStr)
		}
		return
	}

	// Check for idle residency
	idleMatch := regexp.MustCompile(`CPU (\d+) idle residency: +([\d.]+)%`).FindStringSubmatch(line)
	if idleMatch != nil {
		cpuID, _ := strconv.Atoi(idleMatch[1])
		idlePercent, _ := strconv.ParseFloat(idleMatch[2], 64)
		cpu := p.ensureCPUResidency(cpuID)
		cpu.IdleResidency = idlePercent
		return
	}

	// Check for down residency
	downMatch := regexp.MustCompile(`CPU (\d+) down residency: +([\d.]+)%`).FindStringSubmatch(line)
	if downMatch != nil {
		cpuID, _ := strconv.Atoi(downMatch[1])
		downPercent, _ := strconv.ParseFloat(downMatch[2], 64)
		cpu := p.ensureCPUResidency(cpuID)
		cpu.DownResidency = downPercent
		return
	}

	// Handle cluster residency information
	if strings.Contains(line, "-Cluster HW active residency:") {
		parts := strings.Split(line, ":")
		if len(parts) >= 2 {
			clusterName := strings.TrimSpace(strings.Split(line, " ")[0]) // Get the first part before the colon
			// Parse the percentage after "residency:"
			if val, ok := parseTrailingValue(line, "%"); ok {
				cluster := p.ensureClusterResidency(clusterName)
				cluster.HWActiveResidency = val
			}
			
			// Parse the frequency residency data in parentheses
			openParenIdx := strings.Index(line, "(")
			if openParenIdx != -1 {
				freqDataStr := line[openParenIdx+1:]
				freqDataStr = strings.TrimRight(freqDataStr, ")")
				cluster := p.ensureClusterResidency(clusterName)
				cluster.HWActiveFreqResidency = parseFreqResidency(freqDataStr)
			}
		}
		return
	}
}

func (p *Parser) ensureCPUResidency(cpuID int) *internal.CPUResidencyMetrics {
	if cpu, exists := p.cpuResidencies[cpuID]; exists {
		return cpu
	}

	cpu := &internal.CPUResidencyMetrics{
		CPUID: cpuID,
		ActiveResidency: make(internal.CPUResidencyData),
	}
	p.cpuResidencies[cpuID] = cpu
	return cpu
}

func (p *Parser) ensureClusterResidency(name string) *internal.ClusterResidencyMetrics {
	if cluster, exists := p.clusterResidencies[name]; exists {
		return cluster
	}

	cluster := &internal.ClusterResidencyMetrics{
		Name: name,
		HWActiveFreqResidency: make(map[float64]float64),
	}
	p.clusterResidencies[name] = cluster
	return cluster
}

func (p *Parser) updateNetworkInfo(line string) {
	// Parse outgoing network activity
	outMatches := networkRegex.FindStringSubmatch(line)
	if len(outMatches) >= 3 {
		if outPackets, err := strconv.ParseFloat(outMatches[1], 64); err == nil {
			if outBytes, err := strconv.ParseFloat(outMatches[2], 64); err == nil {
				if p.networkInfo == nil {
					p.networkInfo = &internal.NetworkMetrics{}
				}
				p.networkInfo.OutPacketsPerSec = outPackets
				p.networkInfo.OutBytesPerSec = outBytes
			}
		}
	}

	// Parse incoming network activity
	inMatches := networkInRegex.FindStringSubmatch(line)
	if len(inMatches) >= 3 {
		if inPackets, err := strconv.ParseFloat(inMatches[1], 64); err == nil {
			if inBytes, err := strconv.ParseFloat(inMatches[2], 64); err == nil {
				if p.networkInfo == nil {
					p.networkInfo = &internal.NetworkMetrics{}
				}
				p.networkInfo.InPacketsPerSec = inPackets
				p.networkInfo.InBytesPerSec = inBytes
			}
		}
	}
}

func (p *Parser) updateDiskInfo(line string) {
	// Parse read activity
	readMatches := diskReadRegex.FindStringSubmatch(line)
	if len(readMatches) >= 3 {
		if readOps, err := strconv.ParseFloat(readMatches[1], 64); err == nil {
			if readBytes, err := strconv.ParseFloat(readMatches[2], 64); err == nil {
				if p.diskInfo == nil {
					p.diskInfo = &internal.DiskMetrics{}
				}
				p.diskInfo.ReadOpsPerSec = readOps
				p.diskInfo.ReadBytesPerSec = readBytes * 1024 // Convert from KBytes to Bytes
			}
		}
	}

	// Parse write activity
	writeMatches := diskWriteRegex.FindStringSubmatch(line)
	if len(writeMatches) >= 3 {
		if writeOps, err := strconv.ParseFloat(writeMatches[1], 64); err == nil {
			if writeBytes, err := strconv.ParseFloat(writeMatches[2], 64); err == nil {
				if p.diskInfo == nil {
					p.diskInfo = &internal.DiskMetrics{}
				}
				p.diskInfo.WriteOpsPerSec = writeOps
				p.diskInfo.WriteBytesPerSec = writeBytes * 1024 // Convert from KBytes to Bytes
			}
		}
	}
}

func (p *Parser) updateInterruptInfo(line string) {
	// Check for CPU interrupt lines
	cpuMatch := interruptRegex.FindStringSubmatch(line)
	if cpuMatch != nil {
		cpuID, _ := strconv.Atoi(cpuMatch[1])
		p.ensureInterruptInfo(cpuID)
		return
	}

	// Check for total interrupts line
	totalMatch := interruptTotalRegex.FindStringSubmatch(line)
	if totalMatch != nil {
		// Find the most recently added interrupt that doesn't have this value set
		for _, interrupt := range p.interruptInfo {
			if interrupt.TotalIRQ == 0 { // If not yet set, assume this is for the most recently added CPU
				totalIRQ, _ := strconv.ParseFloat(totalMatch[1], 64)
				interrupt.TotalIRQ = totalIRQ
				break
			}
		}
		return
	}

	// Check for IPI and TIMER interrupt lines
	ipiTimerMatch := interruptIPITimerRegex.FindStringSubmatch(line)
	if ipiTimerMatch != nil {
		interruptType := ipiTimerMatch[1]
		value, _ := strconv.ParseFloat(ipiTimerMatch[2], 64)
		
		// Find the most recently added interrupt that doesn't have this value set
		for _, interrupt := range p.interruptInfo {
			if interruptType == "IPI" && interrupt.IPI == 0 {
				interrupt.IPI = value
				break
			} else if interruptType == "TIMER" && interrupt.TIMER == 0 {
				interrupt.TIMER = value
				break
			}
		}
	}
}

func (p *Parser) ensureInterruptInfo(cpuID int) *internal.InterruptMetrics {
	if interrupt, exists := p.interruptInfo[cpuID]; exists {
		return interrupt
	}

	interrupt := &internal.InterruptMetrics{
		CPUID: cpuID,
	}
	p.interruptInfo[cpuID] = interrupt
	return interrupt
}

func (p *Parser) updateGPUResidencyInfo(line string) {
	// Parse GPU HW active frequency
	if matches := gpuFreqRegex.FindStringSubmatch(line); matches != nil {
		freq, _ := strconv.ParseFloat(matches[1], 64)
		p.gpuResidency.PowerMilliwatts = freq // Temporary storage until we process the GPU power line
		return
	}

	// Parse GPU HW active residency
	if matches := gpuHwActiveResidencyRegex.FindStringSubmatch(line); matches != nil {
		residency, _ := strconv.ParseFloat(matches[1], 64)
		p.gpuResidency.HWActiveResidency = residency
		
		// Parse the frequency residency data in parentheses
		openParenIdx := strings.Index(line, "(")
		if openParenIdx != -1 {
			freqDataStr := line[openParenIdx+1:]
			freqDataStr = strings.TrimRight(freqDataStr, ")")
			p.gpuResidency.HWActiveFreqResidency = parseFreqResidency(freqDataStr)
		}
		return
	}

	// Parse GPU idle residency
	if matches := gpuIdleResidencyRegex.FindStringSubmatch(line); matches != nil {
		residency, _ := strconv.ParseFloat(matches[1], 64)
		p.gpuResidency.IdleResidency = residency
		return
	}

	// Parse GPU software states
	if matches := gpuSWStateRegex.FindStringSubmatch(line); matches != nil {
		stateStr := matches[1]
		states := parseGPUStates(stateStr)
		if strings.Contains(line, "requested state") {
			p.gpuResidency.SWRequestedStates = states
		} else {
			p.gpuResidency.SWStates = states
		}
		return
	}

	// Parse GPU power
	if hasAll(strings.ToLower(line), "gpu", "power") {
		var val float64
		var ok bool
		if val, ok = parseTrailingValue(line, "w"); ok {
			p.gpuResidency.PowerMilliwatts = val * 1000 // Convert to milliwatts
		} else if val, ok = parseTrailingValue(line, "mW"); ok {
			p.gpuResidency.PowerMilliwatts = val
		}
	}
}

func (p *Parser) updateBatteryInfo(line string) {
	if matches := batteryRegex.FindStringSubmatch(line); matches != nil {
		battery, _ := strconv.ParseFloat(matches[1], 64)
		p.system.BatteryPercent = battery
	}
}

func parseFreqResidency(freqDataStr string) internal.CPUResidencyData {
	residencies := make(internal.CPUResidencyData)
	
	// Find all matches of the frequency residency pattern
	matches := cpuFreqResidencyRegex.FindAllStringSubmatch(freqDataStr, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			freq, err := strconv.ParseFloat(match[1], 64)
			percent, err2 := strconv.ParseFloat(match[2], 64)
			if err == nil && err2 == nil {
				residencies[freq] = percent
			}
		}
	}
	
	return residencies
}

func parseGPUStates(stateStr string) internal.GPUSoftwareStateData {
	states := make(internal.GPUSoftwareStateData)
	
	// Split by space and process each "Pn : value%" pair
	pairs := strings.Split(stateStr, " ")
	for i := 0; i < len(pairs); i += 3 { // Each entry is "Pn", ":", "value%"
		if i+2 < len(pairs) {
			stateName := strings.Trim(pairs[i], ": ")
			if strings.HasPrefix(stateName, "P") {
				percentStr := strings.TrimRight(pairs[i+2], "%")
				if value, err := strconv.ParseFloat(percentStr, 64); err == nil {
					states[stateName] = value
				}
			}
		}
	}
	
	return states
}

// CalculateTotalActive calculates total active residency from the frequency map
func CalculateTotalActive(residencyMap map[float64]float64) float64 {
	total := 0.0
	for _, percent := range residencyMap {
		total += percent
	}
	return total
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
