// Tests for the metrics leaf: the rendered Prometheus text block is byte-stable
// and deterministically sorted, the kind label reuses the real violations.kind
// strings, SetHubStatus keeps exactly one active status per hub, and the
// registry is safe under concurrent mutate-while-render.
package metrics

import (
	"strconv"
	"strings"
	"sync"
	"testing"
)

// TestRenderGolden asserts the full rendered block byte-equals an expected
// Prometheus text block with # HELP/# TYPE headers and deterministically sorted
// samples, exercising the two required counters plus the two gauges.
func TestRenderGolden(t *testing.T) {
	r := New()

	// Two hubs, multiple kinds, multiple increments — proves counting and the
	// per-(hub,kind) keying, with the real violations.kind strings.
	r.IncViolation(1, "fork")
	r.IncViolation(1, "shrink")
	r.IncViolation(1, "shrink")
	r.IncViolation(2, "equivocation")

	r.IncPollFailure(1)
	r.IncPollFailure(2)
	r.IncPollFailure(2)

	r.SetHubStatus(1, "verified")
	r.SetHubStatus(2, "frozen")

	r.SetLastObservedAt(1, 1700000000)
	r.SetLastObservedAt(2, 1700000050)

	want := strings.Join([]string{
		"# HELP iscc_monitor_violations_total Self-consistency violations detected per hub and trigger kind.",
		"# TYPE iscc_monitor_violations_total counter",
		`iscc_monitor_violations_total{hub_id="1",kind="fork"} 1`,
		`iscc_monitor_violations_total{hub_id="1",kind="shrink"} 2`,
		`iscc_monitor_violations_total{hub_id="2",kind="equivocation"} 1`,
		"# HELP iscc_monitor_poll_failures_total Poll faults per hub (transport or garbled-body failures).",
		"# TYPE iscc_monitor_poll_failures_total counter",
		`iscc_monitor_poll_failures_total{hub_id="1"} 1`,
		`iscc_monitor_poll_failures_total{hub_id="2"} 2`,
		"# HELP iscc_monitor_hub_status Current hub status (1 for the active status, 0 otherwise).",
		"# TYPE iscc_monitor_hub_status gauge",
		`iscc_monitor_hub_status{hub_id="1",status="verified"} 1`,
		`iscc_monitor_hub_status{hub_id="2",status="frozen"} 1`,
		"# HELP iscc_monitor_last_observed_at Unix-seconds timestamp of the hub's most recent observation.",
		"# TYPE iscc_monitor_last_observed_at gauge",
		`iscc_monitor_last_observed_at{hub_id="1"} 1700000000`,
		`iscc_monitor_last_observed_at{hub_id="2"} 1700000050`,
		"",
	}, "\n")

	got := r.String()
	if got != want {
		t.Errorf("rendered block mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// TestRenderMinimalRequired asserts the two MUST-have families render exactly as
// expected for a registry with a single violations_total{kind="fork"} and a
// single poll_failures_total increment — the minimal contract from next.md.
func TestRenderMinimalRequired(t *testing.T) {
	r := New()
	r.IncViolation(7, "fork")
	r.IncPollFailure(7)

	got := r.String()

	wantViolation := `iscc_monitor_violations_total{hub_id="7",kind="fork"} 1`
	if !strings.Contains(got, wantViolation) {
		t.Errorf("missing violation sample %q in:\n%s", wantViolation, got)
	}
	wantPoll := `iscc_monitor_poll_failures_total{hub_id="7"} 1`
	if !strings.Contains(got, wantPoll) {
		t.Errorf("missing poll-failure sample %q in:\n%s", wantPoll, got)
	}
	if !strings.Contains(got, "# TYPE iscc_monitor_violations_total counter") {
		t.Errorf("missing violations # TYPE line in:\n%s", got)
	}
	if !strings.Contains(got, "# TYPE iscc_monitor_poll_failures_total counter") {
		t.Errorf("missing poll-failures # TYPE line in:\n%s", got)
	}
}

// TestRenderDeterministic asserts repeated renders of the same registry produce
// byte-identical output despite random map iteration order.
func TestRenderDeterministic(t *testing.T) {
	r := New()
	for _, kind := range []string{"shrink", "fork", "equivocation"} {
		for hub := int64(1); hub <= 5; hub++ {
			r.IncViolation(hub, kind)
			r.IncPollFailure(hub)
			r.SetHubStatus(hub, "verified")
			r.SetLastObservedAt(hub, 1700000000+hub)
		}
	}
	first := r.String()
	for i := 0; i < 50; i++ {
		if got := r.String(); got != first {
			t.Fatalf("render %d differs from first render:\n%s", i, got)
		}
	}
}

// TestSetHubStatusSingleActive asserts SetHubStatus clears the prior status for a
// hub so exactly one status reads 1; the superseded status drops to 0.
func TestSetHubStatusSingleActive(t *testing.T) {
	r := New()
	r.SetHubStatus(3, "verified")
	r.SetHubStatus(3, "frozen")

	got := r.String()
	if !strings.Contains(got, `iscc_monitor_hub_status{hub_id="3",status="frozen"} 1`) {
		t.Errorf("active status frozen should read 1 in:\n%s", got)
	}
	if !strings.Contains(got, `iscc_monitor_hub_status{hub_id="3",status="verified"} 0`) {
		t.Errorf("superseded status verified should read 0 in:\n%s", got)
	}
}

// TestEmptyRegistryRendersHeadersOnly asserts a fresh registry renders every
// family's # HELP/# TYPE header with no sample lines and no panic.
func TestEmptyRegistryRendersHeadersOnly(t *testing.T) {
	got := New().String()
	want := strings.Join([]string{
		"# HELP iscc_monitor_violations_total Self-consistency violations detected per hub and trigger kind.",
		"# TYPE iscc_monitor_violations_total counter",
		"# HELP iscc_monitor_poll_failures_total Poll faults per hub (transport or garbled-body failures).",
		"# TYPE iscc_monitor_poll_failures_total counter",
		"# HELP iscc_monitor_hub_status Current hub status (1 for the active status, 0 otherwise).",
		"# TYPE iscc_monitor_hub_status gauge",
		"# HELP iscc_monitor_last_observed_at Unix-seconds timestamp of the hub's most recent observation.",
		"# TYPE iscc_monitor_last_observed_at gauge",
		"",
	}, "\n")
	if got != want {
		t.Errorf("empty registry mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// TestConcurrentMutateAndRender drives many goroutines mutating the registry
// while others render it, proving the RWMutex makes concurrent mutate-while-read
// safe (the leaf must not rely on the single-writer follower). It asserts the
// final counters are exact, so no increment was lost to a race.
func TestConcurrentMutateAndRender(t *testing.T) {
	r := New()
	const writers = 8
	const iters = 1000

	var wg sync.WaitGroup
	wg.Add(writers + 2)

	for i := 0; i < writers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				r.IncViolation(1, "fork")
				r.IncPollFailure(1)
			}
		}()
	}
	// Two concurrent readers render while the writers mutate.
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				_ = r.String()
			}
		}()
	}
	wg.Wait()

	total := strconv.FormatUint(uint64(writers*iters), 10)
	got := r.String()
	wantViolation := `iscc_monitor_violations_total{hub_id="1",kind="fork"} ` + total
	if !strings.Contains(got, wantViolation) {
		t.Errorf("expected %q after concurrent increments, got:\n%s", wantViolation, got)
	}
	wantPoll := `iscc_monitor_poll_failures_total{hub_id="1"} ` + total
	if !strings.Contains(got, wantPoll) {
		t.Errorf("expected %q after concurrent increments, got:\n%s", wantPoll, got)
	}
}
