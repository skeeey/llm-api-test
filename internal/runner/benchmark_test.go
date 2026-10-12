package runner

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"llm-api-test/internal/registry"
)

func TestProgressLine(t *testing.T) {
	got := progressLine(3*time.Second, 1, 4)
	want := "[benchmark] elapsed 3s, 1/4 requests completed"
	if got != want {
		t.Errorf("progressLine = %q, want %q", got, want)
	}
}

// RunBenchmark derives the input-token rate from the run's wall clock and
// keeps one sample per request for the JSON report.
func TestRunBenchmarkInputRateAndSamples(t *testing.T) {
	bc := fakeBenchmarkCase{}
	r := RunBenchmark(context.Background(), bc, "m", staticPrompt("pong"), 2, 3, nil)
	if r.InputTokens != 60 { // 6 requests x 10 prompt tokens
		t.Errorf("InputTokens = %d, want 60", r.InputTokens)
	}
	if r.InputTokensPerSec <= 0 {
		t.Errorf("InputTokensPerSec = %v, want > 0", r.InputTokensPerSec)
	}
	if len(r.Samples) != 6 {
		t.Errorf("Samples = %d, want 6", len(r.Samples))
	}
}

// The throughput text report must surface the usage-derived figures the
// vendor comparison needs: input tok/s and observed cache reads.
func TestFormatBenchmarkReportRates(t *testing.T) {
	r := BenchmarkReport{
		CaseID: "chat:benchmark", Mode: "throughput", Stream: true, TotalRequests: 9,
		InputTokens: 9000, CachedTokens: 7200, CacheHitRate: 0.8, InputTokensPerSec: 1234.5,
	}
	out := FormatBenchmarkReport(r)
	if !strings.Contains(out, "Input:") || !strings.Contains(out, "1234.5 tok/s") {
		t.Errorf("report missing input rate:\n%s", out)
	}
	if !strings.Contains(out, "Cache:") || !strings.Contains(out, "7200/9000") {
		t.Errorf("report missing cache line:\n%s", out)
	}
}

// staticPrompt adapts a fixed prompt to the per-request prompt source.
func staticPrompt(s string) func() string { return func() string { return s } }

// RunBenchmark asks the prompt source once per request: a salted run gives
// every request its own prefix while plain runs reuse one shared prompt.
func TestRunBenchmarkPromptSourcePerRequest(t *testing.T) {
	bc := &promptRecordingCase{}
	n := 0
	promptOf := func() string { n++; return fmt.Sprintf("prompt-%d", n) }
	RunBenchmark(context.Background(), bc, "m", promptOf, 2, 3, nil)
	if got := bc.count(); got != 6 {
		t.Errorf("requests reaching the case = %d, want 6", got)
	}
	if got := bc.distinct(); got != 6 {
		t.Errorf("distinct prompts = %d, want 6", got)
	}
}

// promptRecordingCase records every prompt its Run receives.
type promptRecordingCase struct {
	mu      sync.Mutex
	prompts []string
}

func (*promptRecordingCase) ID() string   { return "fake:prompt" }
func (*promptRecordingCase) Desc() string { return "fake" }

func (c *promptRecordingCase) Run(ctx context.Context, model, prompt string) *registry.Metrics {
	c.mu.Lock()
	c.prompts = append(c.prompts, prompt)
	c.mu.Unlock()
	return &registry.Metrics{PromptTokens: 10, CompletionTokens: 5}
}

func (c *promptRecordingCase) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.prompts)
}

func (c *promptRecordingCase) distinct() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	seen := map[string]bool{}
	for _, p := range c.prompts {
		seen[p] = true
	}
	return len(seen)
}
