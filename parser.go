package powermetrics

import (
	"bufio"
	"context"
	"io"
	"os/exec"

	"github.com/binsquare/benchtop/powermetrics-go/internal"
)

// Parser handles invoking powermetrics and parsing its output.
type Parser struct {
	config       Config
	system       internal.SystemSample
	frequencyMHz float64
	clusterInfo  map[string]*ClusterInfo
}

// NewParser creates a parser using the provided configuration, filling in defaults as required.
func NewParser(cfg Config) *Parser {
	normalized := normalizeConfig(cfg)

	return &Parser{
		config:      normalized,
		clusterInfo: make(map[string]*ClusterInfo),
	}
}

// Run executes powermetrics and returns a channel that emits parsed metrics until the context is cancelled.
func (p *Parser) Run(ctx context.Context) (<-chan Metrics, error) {
	cmd := exec.CommandContext(ctx, p.config.PowermetricsPath, p.config.PowermetricsArgs...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	out := make(chan Metrics, 128)
	go p.run(ctx, cmd, stdout, out)

	return out, nil
}

func (p *Parser) run(ctx context.Context, cmd *exec.Cmd, stdout io.Reader, out chan<- Metrics) {
	defer close(out)

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			_ = cmd.Process.Kill()
			return
		default:
		}

		line := scanner.Text()
		metrics, err := p.ParseLine(line)
		if err != nil {
			continue
		}

		if metrics != nil {
			out <- *metrics
		}
	}
}

// RunWithConfig executes powermetrics with the given configuration and returns a channel of metrics.
func RunWithConfig(ctx context.Context, config Config) (<-chan Metrics, error) {
	parser := NewParser(config)
	return parser.Run(ctx)
}

// RunDefault executes powermetrics with default configuration.
func RunDefault(ctx context.Context) (<-chan Metrics, error) {
	parser := NewParser(Config{})
	return parser.Run(ctx)
}
