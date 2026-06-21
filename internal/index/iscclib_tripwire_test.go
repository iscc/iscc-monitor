// Tripwire test for the ADR-0011 ISCC-IDv1 migration trigger.
//
// ADR-0011 adopts github.com/iscc/iscc-lib/packages/go as the canonical Go ISCC
// codec but carves out ISCC-IDv1: iscc-lib cannot decode it today, so the in-repo
// port internal/index.Decode stays the interim ISCC-IDv1 codec. This test is the
// one honest, load-bearing use of the dependency in the monitor — it keeps the
// require live and asserts iscc-lib's current behaviour so the migration trigger
// is executable, not just documented.
//
// iscc-lib's decodeHeader hard-rejects any header Version > 0
// (fmt.Errorf("iscc: invalid Version: %d", ...)). An ISCC-IDv1 has Version = 1, so
// IsccDecode rejects every real ISCC-IDv1 with "invalid Version: 1". The subject
// here, "MAIGHFECJMOPMIAB", is the exact golden vector iscc_test.go decodes via the
// in-repo port — so this tripwire proves iscc-lib rejects precisely what the port
// handles.
//
// When upstream iscc/iscc-lib#43 lands ISCC-IDv1 support, IsccDecode will succeed
// and this assertion flips RED. That is the signal to the next CID loop iteration:
// migrate internal/index.Decode to the iscc-lib call and delete the port
// (the ADR-0011 migration trigger).
//
// The iscc-lib import is confined to this _test.go file; the WASM build excludes
// test files, so the production decoder leaf (iscc.go) stays iscc-lib-free and
// WASM-buildable (ADR-0011's explicit purity guard).
package index

import (
	"strings"
	"testing"

	iscc "github.com/iscc/iscc-lib/packages/go"
)

// isccIDv1GoldenVector is the realm-0 golden ISCC-IDv1 (Version = 1) that
// iscc_test.go decodes via the in-repo port. iscc-lib must reject it today.
const isccIDv1GoldenVector = "MAIGHFECJMOPMIAB"

func TestISCCLibRejectsISCCIDv1Today(t *testing.T) {
	// Tripwire: iscc-lib currently rejects every ISCC-IDv1 (Version = 1) with an
	// "invalid Version" error. When iscc/iscc-lib#43 lands ISCC-IDv1 support this
	// assertion flips RED — migrate internal/index.Decode to iscc-lib and delete
	// the port (ADR-0011 migration trigger).
	res, err := iscc.IsccDecode(isccIDv1GoldenVector)
	if err == nil {
		t.Fatalf("iscc-lib now decodes ISCC-IDv1 %q (got %+v, want rejection): "+
			"iscc/iscc-lib#43 has landed — migrate internal/index.Decode to "+
			"iscc-lib and delete the port (ADR-0011 migration trigger)",
			isccIDv1GoldenVector, res)
	}
	if !strings.Contains(err.Error(), "invalid Version") {
		t.Fatalf("iscc-lib rejected ISCC-IDv1 %q with an unexpected error %q, "+
			"want one mentioning %q", isccIDv1GoldenVector, err, "invalid Version")
	}
}
