package powermetrics

import (
	"reflect"
	"regexp"
	"testing"
	"time"
)

func TestNormalizeConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    Config
		expected Config
	}{
		{
			name:  "default config",
			input: Config{},
			expected: Config{
				PowermetricsPath: "/usr/bin/powermetrics",
				PowermetricsArgs: []string{"--samplers", "default", "--show-process-gpu", "-i", "1000"},
				SampleWindow:     time.Second,
			},
		},
		{
			name: "custom path and args",
			input: Config{
				PowermetricsPath: "/custom/path",
				PowermetricsArgs: []string{"--samplers", "cpu_power", "-i", "500"},
				SampleWindow:     2 * time.Second,
			},
			expected: Config{
				PowermetricsPath: "/custom/path",
				PowermetricsArgs: []string{"--samplers", "cpu_power", "-i", "2000"},
				SampleWindow:     2 * time.Second,
			},
		},
		{
			name: "custom args without interval",
			input: Config{
				PowermetricsArgs: []string{"--samplers", "cpu_power"},
				SampleWindow:     500 * time.Millisecond,
			},
			expected: Config{
				PowermetricsPath: "/usr/bin/powermetrics",
				PowermetricsArgs: []string{"--samplers", "cpu_power", "-i", "500"},
				SampleWindow:     500 * time.Millisecond,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			result := normalizeConfig(tt.input)
			if result.PowermetricsPath != tt.expected.PowermetricsPath {
				t.Errorf("PowermetricsPath: got %s, want %s", result.PowermetricsPath, tt.expected.PowermetricsPath)
			}
			if !reflect.DeepEqual(result.PowermetricsArgs, tt.expected.PowermetricsArgs) {
				t.Errorf("PowermetricsArgs: got %v, want %v", result.PowermetricsArgs, tt.expected.PowermetricsArgs)
			}
			if result.SampleWindow != tt.expected.SampleWindow {
				t.Errorf("SampleWindow: got %v, want %v", result.SampleWindow, tt.expected.SampleWindow)
			}
		})
	}
}

func TestConvertToNanoseconds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    float64
		unit     string
		expected uint64
	}{
		{"microseconds", 1.0, "us", 1000},
		{"milliseconds", 1.0, "ms", 1000000},
		{"seconds", 1.0, "s", 1000000000},
		{"uppercase units", 1.0, "US", 1000},
		{"invalid unit", 1.0, "invalid", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			result := convertToNanoseconds(tt.value, tt.unit)
			if result != tt.expected {
				t.Errorf("convertToNanoseconds(%.1f, %q) = %d, want %d", tt.value, tt.unit, result, tt.expected)
			}
		})
	}
}

func TestClampPercent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"normal value", 50.0, 50.0},
		{"negative value", -10.0, 0.0},
		{"over 100", 150.0, 100.0},
		{"zero", 0.0, 0.0},
		{"exactly 100", 100.0, 100.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			result := clampPercent(tt.input)
			if result != tt.expected {
				t.Errorf("clampPercent(%.1f) = %.1f, want %.1f", tt.input, result, tt.expected)
			}
		})
	}
}

func TestHasAll(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		str      string
		tokens   []string
		expected bool
	}{
		{"all present", "cpu power frequency", []string{"cpu", "power"}, true},
		{"not all present", "cpu frequency", []string{"cpu", "power"}, false},
		{"empty tokens", "cpu power", []string{}, true},
		{"substring - still matches", "cpu-power freq", []string{"cpu", "power"}, true}, // hasAll matches substrings
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			result := hasAll(tt.str, tt.tokens...)
			if result != tt.expected {
				t.Errorf("hasAll(%q, %v) = %t, want %t", tt.str, tt.tokens, result, tt.expected)
			}
		})
	}
}

func TestHasNone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		str      string
		tokens   []string
		expected bool
	}{
		{"none present", "cpu power frequency", []string{"gpu", "ane"}, true},
		{"one present", "cpu gpu power", []string{"gpu", "ane"}, false},
		{"all present", "gpu ane cpu", []string{"gpu", "ane"}, false},
		{"empty tokens", "cpu power", []string{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			result := hasNone(tt.str, tt.tokens...)
			if result != tt.expected {
				t.Errorf("hasNone(%q, %v) = %t, want %t", tt.str, tt.tokens, result, tt.expected)
			}
		})
	}
}

func TestParseTrailingValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		line     string
		suffix   string
		expected float64
		found    bool
	}{
		{"simple watts", "CPU Power: 15.5 W", "w", 15.5, true},
		{"with parentheses", "CPU Power: 15.5 W (100%)", "w", 15.5, true},
		{"with colon", "GPU Busy: 45.2%", "%", 45.2, true},
		{"no match", "CPU Frequency: 2.4 GHz", "w", 0, false},
		{"non-numeric", "CPU Power: N/A W", "w", 0, false},
		{"multiple numbers", "Total: 10.0 W out of 100.0 W", "w", 100.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			result, found := parseTrailingValue(tt.line, tt.suffix)
			if found != tt.found {
				t.Errorf("parseTrailingValue(%q, %q): found = %t, want %t", tt.line, tt.suffix, found, tt.found)
			}
			if found && result != tt.expected {
				t.Errorf("parseTrailingValue(%q, %q) = %f, want %f", tt.line, tt.suffix, result, tt.expected)
			}
		})
	}
}

func TestParseLeadingValueAfterColon(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		line     string
		suffix   string
		expected float64
		found    bool
	}{
		{"simple colon", "GPU HW active residency: 45.2%", "%", 45.2, true},
		{"with parentheses", "GPU HW active residency: 45.2% (something)", "%", 45.2, true},
		{"no colon", "GPU Busy: 45.2%", "%", 45.2, true},
		{"no match", "CPU Power: 15.5 W", "%", 0, false},
		{"no colon no match", "No colon here and no percent", "%", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			result, found := parseLeadingValueAfterColon(tt.line, tt.suffix)
			if found != tt.found {
				t.Errorf("parseLeadingValueAfterColon(%q, %q): found = %t, want %t", tt.line, tt.suffix, found, tt.found)
			}
			if found && result != tt.expected {
				t.Errorf("parseLeadingValueAfterColon(%q, %q) = %f, want %f", tt.line, tt.suffix, result, tt.expected)
			}
		})
	}
}

func TestDeriveBusyPercent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		activeNs     uint64
		explicitPercent string
		window       time.Duration
		expected     float64
	}{
		{"with explicit percent", 0, "85.5", time.Second, 85.5},
		{"computed from nanoseconds", 500000000, "", time.Second, 50.0}, // 0.5s out of 1s = 50%
		{"computed with clamping", 1500000000, "", time.Second, 100.0}, // 1.5s out of 1s = 150% -> 100%
		{"zero active", 0, "", time.Second, 0.0},
		{"zero window", 500000000, "", 0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			result := deriveBusyPercent(tt.activeNs, tt.explicitPercent, tt.window)
			if result != tt.expected {
				t.Errorf("deriveBusyPercent(%d, %q, %v) = %f, want %f", 
					tt.activeNs, tt.explicitPercent, tt.window, result, tt.expected)
			}
		})
	}
}

func TestParser_ParseLineSystemMetrics(t *testing.T) {
	t.Parallel()

	parser := NewParser(Config{})

	tests := []struct {
		name     string
		line     string
		hasSystem bool
		cpuPower float64
		gpuPower float64
	}{
		{
			"cpu power",
			"CPU Power: 15.5 W",
			true,
			15.5,
			0,
		},
		{
			"gpu power",
			"GPU Power: 3.2 W",
			true,
			0,
			3.2,
		},
		{
			"cpu frequency",
			"CPU Frequency: 2400 MHz",
			true,
			0,
			0,
		},
		{
			"gpu busy with percentage",
			"GPU Busy: 45.2%",
			true,
			0,
			0,
		},
		{
			"comment line",
			"-- Sample comment --",
			false,
			0,
			0,
		},
		{
			"empty line",
			"",
			false,
			0,
			0,
		},
		{
			"irrelevant line",
			"System uptime: 10 days",
			false,
			0,
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			metrics, err := parser.ParseLine(tt.line)
			if err != nil {
				t.Fatalf("ParseLine(%q) returned error: %v", tt.line, err)
			}
			
			if (metrics != nil) != tt.hasSystem {
				t.Fatalf("ParseLine(%q) returned metrics=%t, want %t", tt.line, metrics != nil, tt.hasSystem)
			}
			
			if metrics != nil && tt.hasSystem {
				if metrics.SystemSample != nil {
					if tt.cpuPower > 0 && metrics.SystemSample.CPUPowerWatts != tt.cpuPower {
						t.Errorf("Expected CPU Power %f, got %f", tt.cpuPower, metrics.SystemSample.CPUPowerWatts)
					}
					if tt.gpuPower > 0 && metrics.SystemSample.GPUPowerWatts != tt.gpuPower {
						t.Errorf("Expected GPU Power %f, got %f", tt.gpuPower, metrics.SystemSample.GPUPowerWatts)
					}
				}
			}
		})
	}
}

func TestParser_ParseLineGPUProcess(t *testing.T) {
	t.Parallel()

	parser := NewParser(Config{SampleWindow: time.Second})

	tests := []struct {
		name     string
		line     string
		hasGPUProcess bool
	}{
		{
			"simple process line",
			"pid 1234   Safari                     5.2ms  (85.5%)",
			true,
		},
		{
			"process with parentheses",
			"pid 5678   (chrome)                   2.1ms  (45.0%)",
			true,
		},
		{
			"process with microseconds",
			"pid 9999   Firefox                    1200us (70.0%)",
			true,
		},
		{
			"process with seconds",
			"pid 1011   TestApp                    0.5s   (90.0%)",
			true,
		},
		{
			"invalid line",
			"not a process line",
			false,
		},
		{
			"malformed pid",
			"pid invalid   App                    1.0ms  (50.0%)",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			metrics, err := parser.ParseLine(tt.line)
			if err != nil {
				t.Fatalf("ParseLine(%q) returned error: %v", tt.line, err)
			}
			
			if (metrics != nil) != tt.hasGPUProcess {
				t.Fatalf("ParseLine(%q) returned metrics=%t, want %t", tt.line, metrics != nil, tt.hasGPUProcess)
			}
			
			if metrics != nil && tt.hasGPUProcess {
				if len(metrics.GPUProcessSamples) == 0 {
					t.Fatalf("Expected GPU process samples, got none")
				}
			}
		})
	}
}

func TestParser_updateClusterInfo(t *testing.T) {
	t.Parallel()

	parser := NewParser(Config{})

	// Test cluster online regex
	line1 := "E-Cluster Online: 75.5%"
	_, err := parser.ParseLine(line1)
	if err != nil {
		t.Fatalf("ParseLine(%q) returned error: %v", line1, err)
	}
	
	if len(parser.clusterInfo) != 1 {
		t.Fatalf("Expected 1 cluster after parsing %q, got %d", line1, len(parser.clusterInfo))
	}
	
	cluster, exists := parser.clusterInfo["E-Cluster"]
	if !exists {
		t.Fatalf("Expected E-Cluster to exist after parsing %q", line1)
	}
	
	if cluster.OnlinePercent != 75.5 {
		t.Errorf("Expected OnlinePercent 75.5, got %f", cluster.OnlinePercent)
	}
	
	// Test cluster frequency regex
	line2 := "P1-Cluster HW active frequency: 2400.0 MHz"
	_, err = parser.ParseLine(line2)
	if err != nil {
		t.Fatalf("ParseLine(%q) returned error: %v", line2, err)
	}
	
	if len(parser.clusterInfo) != 2 {
		t.Fatalf("Expected 2 clusters after parsing %q, got %d", line2, len(parser.clusterInfo))
	}
	
	cluster2, exists := parser.clusterInfo["P1-Cluster"]
	if !exists {
		t.Fatalf("Expected P1-Cluster to exist after parsing %q", line2)
	}
	
	if cluster2.HWActiveFreq != 2400.0 {
		t.Errorf("Expected HWActiveFreq 2400.0, got %f", cluster2.HWActiveFreq)
	}
	
	// Verify cluster types are assigned correctly
	if cluster.Type != "Efficiency" {
		t.Errorf("Expected cluster type 'Efficiency' for E-Cluster, got %s", cluster.Type)
	}
	
	if cluster2.Type != "Performance" {
		t.Errorf("Expected cluster type 'Performance' for P1-Cluster, got %s", cluster2.Type)
	}
}

func TestEnsureIntervalArgument(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		window   time.Duration
		expected []string
	}{
		{
			"no interval argument",
			[]string{"--samplers", "cpu_power"},
			time.Second,
			[]string{"--samplers", "cpu_power", "-i", "1000"},
		},
		{
			"existing interval argument",
			[]string{"--samplers", "cpu_power", "-i", "500"},
			2 * time.Second,
			[]string{"--samplers", "cpu_power", "-i", "2000"},
		},
		{
			"interval at end",
			[]string{"-i", "100"},
			time.Millisecond,
			[]string{"-i", "1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			result := ensureIntervalArgument(tt.args, tt.window)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ensureIntervalArgument(%v, %v) = %v, want %v", tt.args, tt.window, result, tt.expected)
			}
		})
	}
}

func TestRegexCompilation(t *testing.T) {
	t.Parallel()

	// Test that all regexes compile properly
	regexes := []*regexp.Regexp{
		procLineRegex,
		numberExtractor,
		clusterOnlineRegex,
		clusterHWFreqRegex,
	}
	
	for i, re := range regexes {
		if re == nil {
			t.Errorf("Regex %d is nil, indicating compilation failure", i)
		}
	}
}