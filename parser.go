package powermetrics

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
)

// Parser handles invoking powermetrics and parsing its output.
type Parser struct {
	config             Config
	system             SystemSample
	frequencyMHz       float64
	processSamples     []ProcessSample
	clusterInfo        map[string]*ClusterInfo
	cpuResidencies     map[int]*CPUResidencyMetrics
	clusterResidencies map[string]*ClusterResidencyMetrics
	networkInfo        *NetworkMetrics
	diskInfo           *DiskMetrics
	interruptInfo      map[int]*InterruptMetrics
	gpuResidency       *GPUResidencyMetrics
}

// NewParser creates a parser using the provided configuration, filling in defaults as required.
func NewParser(cfg Config) *Parser {
	normalized := normalizeConfig(cfg)

	return &Parser{
		config:             normalized,
		clusterInfo:        make(map[string]*ClusterInfo),
		cpuResidencies:     make(map[int]*CPUResidencyMetrics),
		clusterResidencies: make(map[string]*ClusterResidencyMetrics),
		interruptInfo:      make(map[int]*InterruptMetrics),
		gpuResidency: &GPUResidencyMetrics{
			HWActiveFreqResidency: make(map[float64]float64),
			SWRequestedStates:     make(GPUSoftwareStateData),
			SWStates:              make(GPUSoftwareStateData),
		},
	}
}

// Stream represents a metrics stream paired with an error channel.
type Stream struct {
	Metrics <-chan Metrics
	Errors  <-chan error
}

type readerFactory func(context.Context) (io.Reader, func() error, error)

// Run executes powermetrics and returns a channel of metrics.
// Deprecated: prefer RunWithErrors to also receive runtime diagnostics.
func (p *Parser) Run(ctx context.Context) (<-chan Metrics, error) {
	stream, err := p.RunWithErrors(ctx)
	if err != nil {
		return nil, err
	}

	// Drain errors to avoid goroutine leaks while keeping backward compatibility.
	go func() {
		for range stream.Errors {
		}
	}()

	return stream.Metrics, nil
}

// RunWithErrors executes powermetrics and returns a Stream that includes both metrics and errors.
func (p *Parser) RunWithErrors(ctx context.Context) (*Stream, error) {
	return p.newStream(ctx, func(ctx context.Context) (io.Reader, func() error, error) {
		cmd := exec.CommandContext(ctx, p.config.PowermetricsPath, p.config.PowermetricsArgs...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, nil, err
		}

		if err := cmd.Start(); err != nil {
			return nil, nil, err
		}

		return stdout, cmd.Wait, nil
	})
}

// RunWithReader parses powermetrics output from an arbitrary io.Reader (e.g., a log file).
// The caller is responsible for closing the reader if needed.
func (p *Parser) RunWithReader(ctx context.Context, reader io.Reader) *Stream {
	if reader == nil {
		panic("powermetrics: reader cannot be nil")
	}
	stream, err := p.newStream(ctx, func(context.Context) (io.Reader, func() error, error) {
		return reader, nil, nil
	})
	if err != nil {
		panic(fmt.Sprintf("powermetrics: reader stream failed: %v", err))
	}
	return stream
}

func (p *Parser) newStream(ctx context.Context, factory readerFactory) (*Stream, error) {
	if factory == nil {
		return nil, fmt.Errorf("powermetrics: reader factory cannot be nil")
	}

	reader, wait, err := factory(ctx)
	if err != nil {
		return nil, err
	}
	if reader == nil {
		return nil, fmt.Errorf("powermetrics: reader factory returned nil reader")
	}

	return p.streamFromReader(ctx, reader, wait), nil
}

func (p *Parser) streamFromReader(ctx context.Context, reader io.Reader, wait func() error) *Stream {
	metricsCh := make(chan Metrics, 128)
	errCh := make(chan error, 16)

	go func() {
		defer close(metricsCh)
		defer close(errCh)

		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				if wait != nil {
					_ = wait()
				}
				return
			default:
			}

			line := scanner.Text()
			metrics, err := p.ParseLine(line)
			if err != nil {
				errCh <- fmt.Errorf("parse line: %w", err)
				continue
			}

			if metrics != nil {
				metricsCh <- *metrics
			}
		}

		if metrics := p.flushProcessSamples(); metrics != nil {
			metricsCh <- *metrics
		}

		if err := scanner.Err(); err != nil {
			errCh <- err
		}

		if wait != nil {
			if err := wait(); err != nil && ctx.Err() == nil {
				errCh <- err
			}
		}
	}()

	return &Stream{
		Metrics: metricsCh,
		Errors:  errCh,
	}
}

// RunWithConfig executes powermetrics with the given configuration and returns a channel of metrics.
func RunWithConfig(ctx context.Context, config Config) (<-chan Metrics, error) {
	parser := NewParser(config)
	return parser.Run(ctx)
}

// RunWithConfigStream executes powermetrics with the given configuration and exposes metrics alongside errors.
func RunWithConfigStream(ctx context.Context, config Config) (*Stream, error) {
	parser := NewParser(config)
	return parser.RunWithErrors(ctx)
}

// RunDefault executes powermetrics with default configuration.
func RunDefault(ctx context.Context) (<-chan Metrics, error) {
	parser := NewParser(Config{})
	return parser.Run(ctx)
}

// RunDefaultStream executes powermetrics with default configuration and returns metrics with an error channel.
func RunDefaultStream(ctx context.Context) (*Stream, error) {
	parser := NewParser(Config{})
	return parser.RunWithErrors(ctx)
}

// RunReader parses powermetrics output from an arbitrary reader using the provided configuration.
func RunReader(ctx context.Context, config Config, reader io.Reader) *Stream {
	parser := NewParser(config)
	return parser.RunWithReader(ctx, reader)
}
