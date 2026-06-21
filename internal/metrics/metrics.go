// Package metrics is the monitor's pure, stdlib-only metrics leaf: an in-memory
// registry of the alert-worthy series the follower mutates, rendered in
// Prometheus text-exposition format. It is the first slice of the /metrics gap
// (plan §9, PRD §observability); the HTTP /metrics handler and the follower
// wiring are deliberately separate later slices.
//
// The leaf is net-free, db-free, log-free, and clock-free: it imports only
// stdlib formatting/sorting/sync primitives and never reads time.Now(). Any
// timestamp value (last_observed_at) is supplied by the caller, so tests stay
// deterministic. The metric names and label values are the contract a later
// wiring slice fills: the kind label reuses the store.Violation.Kind /
// logclient.ViolationKind strings ("shrink"/"fork"/"equivocation") verbatim and
// the status label reuses the glossary hub-status set, so no synonyms are
// invented at the metrics seam.
//
// Concurrency: the maps are guarded by a sync.RWMutex. The follower is the
// single writer per DB, but a later /metrics HTTP read path renders concurrently
// with follower writes, so correctness must not rely on the caller being
// single-threaded — reads take the read lock, mutations the write lock.
package metrics

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Metric name constants. snake_case with an iscc_monitor_ prefix and a _total
// suffix on counters, per Prometheus convention.
const (
	nameViolations     = "iscc_monitor_violations_total"
	namePollFailures   = "iscc_monitor_poll_failures_total"
	nameHubStatus      = "iscc_monitor_hub_status"
	nameLastObservedAt = "iscc_monitor_last_observed_at"
)

// violationKey identifies one violations_total sample: a (hub_id, kind) pair.
type violationKey struct {
	hubID int64
	kind  string
}

// statusKey identifies one hub_status sample: a (hub_id, status) pair.
type statusKey struct {
	hubID  int64
	status string
}

// Registry holds the monitor's alert-worthy metric series in memory and renders
// them in Prometheus text-exposition format. The zero Registry is not ready for
// use; construct one with New. It is safe for concurrent use.
type Registry struct {
	mu             sync.RWMutex
	violations     map[violationKey]uint64
	pollFailures   map[int64]uint64
	hubStatus      map[statusKey]uint64
	lastObservedAt map[int64]int64
}

// New returns an empty Registry ready for use.
func New() *Registry {
	return &Registry{
		violations:     make(map[violationKey]uint64),
		pollFailures:   make(map[int64]uint64),
		hubStatus:      make(map[statusKey]uint64),
		lastObservedAt: make(map[int64]int64),
	}
}

// IncViolation increments the violations counter for one hub and trigger kind.
// kind is the store.Violation.Kind / logclient.ViolationKind string
// ("shrink"/"fork"/"equivocation"); the registry does not validate it, so the
// caller must pass the canonical value.
func (r *Registry) IncViolation(hubID int64, kind string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.violations[violationKey{hubID: hubID, kind: kind}]++
}

// IncPollFailure increments the poll-failure counter for one hub. It is the
// transport/garbled-body fault counter the follower's PollHub error path feeds.
func (r *Registry) IncPollFailure(hubID int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pollFailures[hubID]++
}

// SetHubStatus records the hub's current status as the active one: it sets the
// sample for (hubID, status) to 1 and clears every other status sample for the
// same hub to 0, so exactly one status reads 1 per hub. status is the glossary
// hub-status string (verified/unresolvable/unverified/frozen/inactive).
func (r *Registry) SetHubStatus(hubID int64, status string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k := range r.hubStatus {
		if k.hubID == hubID {
			r.hubStatus[k] = 0
		}
	}
	r.hubStatus[statusKey{hubID: hubID, status: status}] = 1
}

// Status returns the hub's current active status (the single status whose gauge
// sample reads 1) and ok reporting whether any status is recorded for the hub.
// SetHubStatus keeps at most one active status per hub (it zeroes the others), so
// the first sample reading 1 is unambiguous; a hub never written, or one whose
// only samples read 0, yields ("", false). It is the read accessor the dashboard
// consumes to overlay the in-memory glossary verdict onto the store-provable
// status; it does not touch the Prometheus render path.
func (r *Registry) Status(hubID int64) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for k, v := range r.hubStatus {
		if k.hubID == hubID && v == 1 {
			return k.status, true
		}
	}
	return "", false
}

// SetLastObservedAt records the unix-seconds timestamp of the hub's most recent
// observation. The value is supplied by the caller (the leaf never reads the
// clock) so the series stays deterministic under test.
func (r *Registry) SetLastObservedAt(hubID int64, unixSeconds int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastObservedAt[hubID] = unixSeconds
}

// WriteText renders the registry in Prometheus text-exposition format. Each
// metric family emits a # HELP and # TYPE header followed by its sample lines;
// families are written in a fixed order and samples within a family are sorted
// deterministically (by label set) so the output is byte-stable across calls
// regardless of map iteration order.
func (r *Registry) WriteText(w io.Writer) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if err := r.writeViolations(w); err != nil {
		return err
	}
	if err := r.writePollFailures(w); err != nil {
		return err
	}
	if err := r.writeHubStatus(w); err != nil {
		return err
	}
	return r.writeLastObservedAt(w)
}

// String renders the registry to a string via WriteText. It is a convenience for
// tests and callers that want the rendered block directly.
func (r *Registry) String() string {
	var b strings.Builder
	// strings.Builder.Write never returns an error, so WriteText cannot fail here.
	_ = r.WriteText(&b)
	return b.String()
}

// writeViolations renders the violations_total counter family. Caller holds the
// read lock.
func (r *Registry) writeViolations(w io.Writer) error {
	if err := writeHeader(w, nameViolations, "counter",
		"Self-consistency violations detected per hub and trigger kind."); err != nil {
		return err
	}
	lines := make([]string, 0, len(r.violations))
	for k, v := range r.violations {
		labels := formatLabels(
			label{"hub_id", strconv.FormatInt(k.hubID, 10)},
			label{"kind", k.kind},
		)
		lines = append(lines, fmt.Sprintf("%s%s %d", nameViolations, labels, v))
	}
	return writeSorted(w, lines)
}

// writePollFailures renders the poll_failures_total counter family. Caller holds
// the read lock.
func (r *Registry) writePollFailures(w io.Writer) error {
	if err := writeHeader(w, namePollFailures, "counter",
		"Poll faults per hub (transport or garbled-body failures)."); err != nil {
		return err
	}
	lines := make([]string, 0, len(r.pollFailures))
	for hubID, v := range r.pollFailures {
		labels := formatLabels(label{"hub_id", strconv.FormatInt(hubID, 10)})
		lines = append(lines, fmt.Sprintf("%s%s %d", namePollFailures, labels, v))
	}
	return writeSorted(w, lines)
}

// writeHubStatus renders the hub_status gauge family. Caller holds the read lock.
func (r *Registry) writeHubStatus(w io.Writer) error {
	if err := writeHeader(w, nameHubStatus, "gauge",
		"Current hub status (1 for the active status, 0 otherwise)."); err != nil {
		return err
	}
	lines := make([]string, 0, len(r.hubStatus))
	for k, v := range r.hubStatus {
		labels := formatLabels(
			label{"hub_id", strconv.FormatInt(k.hubID, 10)},
			label{"status", k.status},
		)
		lines = append(lines, fmt.Sprintf("%s%s %d", nameHubStatus, labels, v))
	}
	return writeSorted(w, lines)
}

// writeLastObservedAt renders the last_observed_at gauge family. Caller holds the
// read lock.
func (r *Registry) writeLastObservedAt(w io.Writer) error {
	if err := writeHeader(w, nameLastObservedAt, "gauge",
		"Unix-seconds timestamp of the hub's most recent observation."); err != nil {
		return err
	}
	lines := make([]string, 0, len(r.lastObservedAt))
	for hubID, v := range r.lastObservedAt {
		labels := formatLabels(label{"hub_id", strconv.FormatInt(hubID, 10)})
		lines = append(lines, fmt.Sprintf("%s%s %d", nameLastObservedAt, labels, v))
	}
	return writeSorted(w, lines)
}

// label is one Prometheus label name/value pair.
type label struct {
	name  string
	value string
}

// formatLabels renders an ordered label set as a Prometheus label block, e.g.
// {hub_id="1",kind="fork"}. An empty set renders as the empty string (a bare
// metric name with no braces). Label values are escaped per the text format.
func formatLabels(labels ...label) string {
	if len(labels) == 0 {
		return ""
	}
	parts := make([]string, len(labels))
	for i, l := range labels {
		parts[i] = l.name + `="` + escapeLabelValue(l.value) + `"`
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// escapeLabelValue escapes a label value per the Prometheus text format: a
// backslash, a double-quote, and a line feed are backslash-escaped, then the
// caller wraps the result in double quotes. The kind/status/hub_id values this
// leaf renders never contain these characters, but escaping keeps the renderer
// total and the output well-formed for any future label.
func escapeLabelValue(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `"`, `\"`)
	v = strings.ReplaceAll(v, "\n", `\n`)
	return v
}

// writeHeader writes the # HELP and # TYPE lines for a metric family.
func writeHeader(w io.Writer, name, typ, help string) error {
	_, err := fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s %s\n", name, help, name, typ)
	return err
}

// writeSorted sorts the rendered sample lines lexically and writes them, one per
// line, so the family's output is byte-stable regardless of map order.
func writeSorted(w io.Writer, lines []string) error {
	sort.Strings(lines)
	for _, line := range lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}
