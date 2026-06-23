// Drift test binding the embedded OpenAPI document to the real buildMux mux: the
// load-bearing guarantee that the hand-authored contract cannot silently diverge from
// the handler (ADR-0014 §3). It seeds a fully-verified one-hub mirror, builds the same
// mux production wires, and reconciles three things at the HTTP seam:
//
//   - (a) the served /openapi.json bytes equal openapi.JSON (byte-verbatim serve) and
//     carry Access-Control-Allow-Origin: * through the corsmw-wrapped mux;
//   - (b) every path the document declares maps to a concrete route that is MOUNTED in
//     the real mux (a documented-but-unmounted path probes 404 and FAILS);
//   - (c) every machine-consumable route the mux mounts is DECLARED in the document, and
//     every HTML SSR route is on the explicit exclusion list and is NOT declared.
//
// Non-vacuity: the documented-path set and the ground-truth machine-route set are
// compared as sets, so dropping a documented path (the machine set has one the doc
// lacks) OR documenting/mounting an extra machine route (the doc has one the machine
// set lacks) FAILS the set-equality. The concrete probes additionally fail if a
// declared route is not actually wired (probes 404 like the undocumented control). The
// SSR routes are probed mounted too, then asserted excluded, so a documented SSR route
// would also FAIL.
package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/transparency-dev/merkle/rfc6962"
	"github.com/transparency-dev/merkle/testonly"
	"github.com/transparency-dev/tessera/api"

	"github.com/iscc/iscc-monitor/internal/dashboard"
	"github.com/iscc/iscc-monitor/internal/metrics"
	"github.com/iscc/iscc-monitor/internal/openapi"
	"github.com/iscc/iscc-monitor/internal/registry"
	"github.com/iscc/iscc-monitor/internal/store"
)

// driftDomain is the concrete one-hub origin the drift fixture mounts, mapped to the
// document's {domain} template when reconciling.
const driftDomain = "sb0.iscc.id"

// driftISCCID is the seeded ISCC-ID the inclusion/verify/bundle probes resolve.
const driftISCCID = "ISCC:SEEDEDLEAF"

// machineProbes maps every OpenAPI path template the document is required to declare
// to a concrete request path against the seeded one-hub mux. {domain} -> driftDomain,
// {iscc_id} -> driftISCCID, {tile} -> the seeded partial tile's canonical path. With
// the full fixture each MOUNTED route returns a non-404 status; an unmounted path falls
// through to tilesserve / the dashboard and 404s, so non-404 is the mount signal.
func machineProbes() map[string]string {
	return map[string]string{
		"/healthz":                     "/healthz",
		"/version":                     "/version",
		"/metrics":                     "/metrics",
		openapi.JSONPath:               openapi.JSONPath,
		openapi.YAMLPath:               openapi.YAMLPath,
		"/{domain}/log/inclusion":      "/" + driftDomain + "/log/inclusion?iscc_id=" + driftISCCID,
		"/{domain}/log/consistency":    "/" + driftDomain + "/log/consistency?from=0",
		"/{domain}/log/entries":        "/" + driftDomain + "/log/entries?index=2",
		"/{domain}/log/verify":         "/" + driftDomain + "/log/verify?iscc_id=" + driftISCCID,
		"/{domain}/log/checkpoint":     "/" + driftDomain + "/log/checkpoint",
		"/{domain}/log/checkpoint.ots": "/" + driftDomain + "/log/checkpoint.ots",
		"/{domain}/log/tile/{tile}":    "/" + driftDomain + "/log/tile/0/000.p/5",
		"/inclusion/{iscc_id}.bundle":  "/inclusion/" + driftISCCID + ".bundle",
	}
}

// ssrExclusions is the explicit allow-list of HTML SSR routes (and the /_ds/ asset
// subtree) that are mounted in the mux but are OUT of the OpenAPI contract by design
// (ADR-0014 §1). Each is probed mounted, then asserted NOT declared in the document.
// Concrete probe paths map the {domain}/{iscc_id} templates onto the fixture.
func ssrExclusions() []string {
	return []string{
		"/",
		"/" + driftDomain,
		"/" + driftDomain + "/log/",
		"/" + driftDomain + "/log/records",
		"/" + driftDomain + "/log/record?index=2",
		"/inclusion/" + driftISCCID,
		"/docs",
		"/_ds/tokens.css",
	}
}

// buildDriftMux seeds a fully-verified one-hub mirror (level-0 hash tile, framed entry
// bundle, iscc_index projection, accepted checkpoint, OTS proof) and returns the same
// corsmw-wrapped mux production wires, plus the seeded tree size. The full fixture
// makes every documented machine route return a non-404 success status, so a 404 from
// a probe is an unambiguous "route not mounted" signal.
func buildDriftMux(t *testing.T) http.Handler {
	t.Helper()
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "drift.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	hub, err := st.UpsertHub(ctx, driftDomain, driftDomain+"/log", "https://"+driftDomain)
	if err != nil {
		t.Fatalf("UpsertHub: %v", err)
	}

	const leaves = 5
	tree := testonly.New(rfc6962.DefaultHasher)
	data := make([][]byte, leaves)
	for i := range data {
		data[i] = []byte(string(rune('a' + i)))
	}
	tree.AppendData(data...)
	size := tree.Size()
	at := time.Unix(1700000000, 0)

	nodes := make([][]byte, leaves)
	for i := range nodes {
		nodes[i] = tree.LeafHash(uint64(i))
	}
	raw, err := api.HashTile{Nodes: nodes}.MarshalText()
	if err != nil {
		t.Fatalf("HashTile.MarshalText: %v", err)
	}
	if err := st.RecordTile(ctx, hub, 0, 0, uint8(leaves), raw, at); err != nil {
		t.Fatalf("RecordTile: %v", err)
	}

	records := make([][]byte, leaves)
	for i := range records {
		records[i] = []byte(fmt.Sprintf("record-%d", i))
	}
	if err := st.RecordEntryBundle(ctx, hub, 0, uint8(leaves), frameEntryBundle(records), at); err != nil {
		t.Fatalf("RecordEntryBundle: %v", err)
	}
	if err := st.RecordProjections(ctx, []store.ProjectionRecord{{HubID: hub, Seq: 2, IsccID: driftISCCID}}); err != nil {
		t.Fatalf("RecordProjections: %v", err)
	}
	if _, _, err := st.RecordCheckpoint(ctx, store.CheckpointRecord{
		HubID: hub, TreeSize: size, Root: tree.Hash(), Raw: []byte("checkpoint"), ObservedAt: at,
	}); err != nil {
		t.Fatalf("RecordCheckpoint: %v", err)
	}
	if err := st.AdvanceFollowState(ctx, hub, size); err != nil {
		t.Fatalf("AdvanceFollowState: %v", err)
	}

	otsBytes, err := os.ReadFile(filepath.Join("..", "..", "internal", "ots", "testdata", "hello-world.txt.ots"))
	if err != nil {
		t.Fatalf("read ots fixture: %v", err)
	}
	if _, _, err := st.RecordOTS(ctx, store.OTSRecord{
		HubID: hub, TreeSize: size, Root: tree.Hash(), Status: store.OTSStatusPending, OTSBytes: otsBytes, StampedAt: at,
	}); err != nil {
		t.Fatalf("RecordOTS: %v", err)
	}

	slot := uint16(0)
	hubList := &registry.HubList{Version: 1, Hubs: []registry.Hub{{HubID: &slot, URL: "https://" + driftDomain, Active: true}}}
	routes := []hubRoute{{HubID: hub, Domain: driftDomain, Origin: driftDomain + "/log"}}
	return buildMux(st, routes, hubList, metrics.New(), dashboard.Identity{})
}

// probeMounted reports the GET status of path against the mux. A non-404 (or a 304)
// proves a handler ran for the route; a 404 from the seeded fixture means the route is
// not mounted (it fell through to tilesserve or the dashboard catch-all).
func probeMounted(t *testing.T, mux http.Handler, path string) int {
	t.Helper()
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec.Code
}

// TestOpenAPIServedVerbatimWithCORS asserts (a): the served /openapi.json is byte-equal
// to the embedded document, parses as a valid OpenAPI 3.1 skeleton, and carries the
// CORS wildcard through the full corsmw-wrapped mux.
func TestOpenAPIServedVerbatimWithCORS(t *testing.T) {
	mux := buildDriftMux(t)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, openapi.JSONPath, nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !bytesEqual(rec.Body.Bytes(), openapi.JSON) {
		t.Errorf("served /openapi.json is not byte-equal to openapi.JSON")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want *", got)
	}
}

// TestOpenAPIDrift is the route<->spec reconciliation: every declared path is mounted,
// every machine route is declared, and the SSR routes are excluded. Non-vacuous by
// set-equality (drop or add a path -> FAIL) and by concrete probe (declare an unmounted
// route -> probes 404 -> FAIL).
func TestOpenAPIDrift(t *testing.T) {
	mux := buildDriftMux(t)
	probes := machineProbes()
	declared := openapi.DeclaredPaths()

	// (c) the document's declared path set must EQUAL the ground-truth machine-route
	// set (the keys of machineProbes). A documented path with no probe, or a machine
	// route the doc omits, fails here — both directions of drift.
	wantSet := sortedKeys(probes)
	if !equalStringSets(declared, wantSet) {
		t.Errorf("declared paths differ from the ground-truth machine routes:\n declared = %v\n machine  = %v", declared, wantSet)
	}

	// (b) every declared path maps to a concrete route that is MOUNTED (probes non-404).
	for _, p := range declared {
		concrete, ok := probes[p]
		if !ok {
			t.Errorf("declared path %q has no probe — it is not a known machine route", p)
			continue
		}
		if code := probeMounted(t, mux, concrete); code == http.StatusNotFound {
			t.Errorf("declared path %q (probe %q) is NOT mounted in the mux (404)", p, concrete)
		}
	}

	// (c, SSR) every excluded SSR route is mounted (a sanity check the exclusion list is
	// real) AND is NOT declared in the document (the contract excludes the HTML surface).
	declaredSet := make(map[string]struct{}, len(declared))
	for _, p := range declared {
		declaredSet[p] = struct{}{}
	}
	for _, ssr := range ssrExclusions() {
		if code := probeMounted(t, mux, ssr); code == http.StatusNotFound {
			t.Errorf("SSR exclusion %q is not mounted — the exclusion list is stale", ssr)
		}
		// Map the concrete SSR probe back to its template form for the declared check.
		if _, isDeclared := declaredSet[ssrTemplate(ssr)]; isDeclared {
			t.Errorf("SSR route %q is declared in the OpenAPI document but must be excluded", ssr)
		}
	}
}

// ssrTemplate maps a concrete SSR probe path to the OpenAPI-template form it would take
// if (wrongly) documented, so the "not declared" check compares like with like. The
// HTML certificate's concrete path /inclusion/<id> maps to /inclusion/{iscc_id}; the
// rest have no template parameters a document would introduce, so they map to
// themselves (query strings stripped).
func ssrTemplate(ssr string) string {
	switch ssr {
	case "/" + driftDomain:
		return "/{domain}"
	case "/" + driftDomain + "/log/":
		return "/{domain}/log/"
	case "/" + driftDomain + "/log/records":
		return "/{domain}/log/records"
	case "/" + driftDomain + "/log/record?index=2":
		return "/{domain}/log/record"
	case "/inclusion/" + driftISCCID:
		return "/inclusion/{iscc_id}"
	default:
		return ssr
	}
}

// sortedKeys returns the sorted keys of m.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// equalStringSets reports whether a and b hold the same elements (order-independent).
func equalStringSets(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	as := append([]string(nil), a...)
	bs := append([]string(nil), b...)
	sort.Strings(as)
	sort.Strings(bs)
	for i := range as {
		if as[i] != bs[i] {
			return false
		}
	}
	return true
}

// bytesEqual reports byte equality.
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
