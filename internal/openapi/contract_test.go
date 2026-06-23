// Per-operation contract golden: the structural guard the path-only drift test
// (TestOpenAPIDrift / TestOpenAPIDocsAgree) is blind to. That gate reconciles only
// the path SET, so a wrong query param, a wrong 200 media type, or a missing
// response code rides along green (three such defects shipped and passed every gate
// — see learnings/openapi.md). This file pins, table-driven and against BOTH the
// YAML source of truth AND its JSON twin, the ground-truth facts the slice-4 doc
// edits establish, each grounded in the real handler:
//
//   - /{domain}/log/verify get.parameters has iscc_id + domain but NOT index
//     (serveVerify reads only iscc_id and always uses seqs[0]);
//   - /{domain}/log/inclusion get.parameters DOES have index (serveInclusion reads
//     it via selectSeq — proving the test distinguishes the two phantom-vs-real
//     index params and is not a vacuous "no index anywhere" check);
//   - /{domain}/log/checkpoint responses.200 media type is application/octet-stream
//     (tilesserve writeBlob serves that, not text/plain);
//   - /healthz responses has BOTH 200 and 503 (a non-nil store ping returns 503).
//
// It also bans a fenced ```mermaid block in either served doc body: that body is the
// only trigger for the vendored Stoplight Elements bundle's unpkg-mermaid lazy-load,
// so the ban closes the latent no-CDN gap review filed. Each assertion is
// mutation-non-vacuous: reverting any one slice-4 edit (or adding a mermaid fence)
// fails a row.
package openapi

import (
	"testing"

	yaml "gopkg.in/yaml.v3"
)

// contractDoc is the minimal slice of an OpenAPI document this golden walks: just
// the per-path GET operation's parameter names and response-code keys. yaml.v3
// reads JSON too, so the same decode covers both committed artifacts.
type contractDoc struct {
	Paths map[string]struct {
		Get struct {
			Parameters []struct {
				Name string `yaml:"name"`
				Ref  string `yaml:"$ref"`
			} `yaml:"parameters"`
			Responses map[string]any `yaml:"responses"`
		} `yaml:"get"`
	} `yaml:"paths"`
}

// content200 pulls the response-200 content-media-type key set for one path's GET
// operation out of the raw document (the typed contractDoc above intentionally
// keeps responses as `any` so this stays a focused, separate walk).
type mediaDoc struct {
	Paths map[string]struct {
		Get struct {
			Responses map[string]struct {
				Content map[string]any `yaml:"content"`
			} `yaml:"responses"`
		} `yaml:"get"`
	} `yaml:"paths"`
}

// parseContract decodes an embedded doc into the parameter/response golden shape.
func parseContract(t *testing.T, doc []byte) contractDoc {
	t.Helper()
	var parsed contractDoc
	if err := yaml.Unmarshal(doc, &parsed); err != nil {
		t.Fatalf("contract doc does not parse: %v", err)
	}
	return parsed
}

// parseMedia decodes an embedded doc into the response-media-type golden shape.
func parseMedia(t *testing.T, doc []byte) mediaDoc {
	t.Helper()
	var parsed mediaDoc
	if err := yaml.Unmarshal(doc, &parsed); err != nil {
		t.Fatalf("media doc does not parse: %v", err)
	}
	return parsed
}

// paramNames returns the parameter names declared on a path's GET operation,
// substituting the referenced component name for a $ref (e.g. the shared Domain
// parameter is "#/components/parameters/Domain" -> "domain"). A path absent from the
// document yields a nil slice (and a failing membership check below).
func paramNames(c contractDoc, path string) []string {
	op, ok := c.Paths[path]
	if !ok {
		return nil
	}
	names := make([]string, 0, len(op.Get.Parameters))
	for _, p := range op.Get.Parameters {
		switch {
		case p.Name != "":
			names = append(names, p.Name)
		case p.Ref == "#/components/parameters/Domain":
			names = append(names, "domain")
		case p.Ref != "":
			names = append(names, p.Ref)
		}
	}
	return names
}

// has reports membership in a string slice.
func has(names []string, want string) bool {
	for _, n := range names {
		if n == want {
			return true
		}
	}
	return false
}

// TestContractVerifyHasNoIndexParam pins slice-4 edit 1: /{domain}/log/verify must
// declare iscc_id + domain but NOT index — serveVerify reads only iscc_id and always
// uses seqs[0]. Re-adding the phantom `index` param to either artifact fails this.
func TestContractVerifyHasNoIndexParam(t *testing.T) {
	for _, a := range artifacts() {
		c := parseContract(t, a.doc)
		names := paramNames(c, "/{domain}/log/verify")
		if !has(names, "iscc_id") {
			t.Errorf("[%s] /verify params %v missing iscc_id", a.name, names)
		}
		if !has(names, "domain") {
			t.Errorf("[%s] /verify params %v missing domain", a.name, names)
		}
		if has(names, "index") {
			t.Errorf("[%s] /verify params %v contains the phantom index (serveVerify never reads it)", a.name, names)
		}
	}
}

// TestContractInclusionHasIndexParam pins that /{domain}/log/inclusion DOES declare
// index — serveInclusion legitimately reads it via selectSeq. This makes the
// verify-no-index check above non-vacuous: the golden distinguishes the phantom
// index from the real one rather than banning index everywhere. Dropping the real
// inclusion index param fails this row.
func TestContractInclusionHasIndexParam(t *testing.T) {
	for _, a := range artifacts() {
		c := parseContract(t, a.doc)
		names := paramNames(c, "/{domain}/log/inclusion")
		if !has(names, "index") {
			t.Errorf("[%s] /inclusion params %v missing the legitimate index param (serveInclusion reads it)", a.name, names)
		}
	}
}

// TestContractCheckpointMediaType pins slice-4 edit 2: /{domain}/log/checkpoint's
// 200 media type is application/octet-stream — it is a tilesserve BLOB route
// (writeBlob), not text/plain. Restoring text/plain to either artifact fails this.
func TestContractCheckpointMediaType(t *testing.T) {
	for _, a := range artifacts() {
		m := parseMedia(t, a.doc)
		op, ok := m.Paths["/{domain}/log/checkpoint"]
		if !ok {
			t.Fatalf("[%s] /{domain}/log/checkpoint not declared", a.name)
		}
		content := op.Get.Responses["200"].Content
		if _, ok := content["application/octet-stream"]; !ok {
			t.Errorf("[%s] /checkpoint 200 media types %v lack application/octet-stream (it is a tilesserve BLOB route)", a.name, mediaKeys(content))
		}
		if _, ok := content["text/plain"]; ok {
			t.Errorf("[%s] /checkpoint 200 still advertises text/plain (writeBlob serves octet-stream)", a.name)
		}
	}
}

// TestContractHealthzHas503 pins slice-4 edit 3: /healthz declares BOTH 200 and 503
// — a non-nil store ping returns 503 {"status":"unavailable"}. Dropping the 503 from
// either artifact fails this row.
func TestContractHealthzHas503(t *testing.T) {
	for _, a := range artifacts() {
		c := parseContract(t, a.doc)
		op, ok := c.Paths["/healthz"]
		if !ok {
			t.Fatalf("[%s] /healthz not declared", a.name)
		}
		if _, ok := op.Get.Responses["200"]; !ok {
			t.Errorf("[%s] /healthz responses missing 200", a.name)
		}
		if _, ok := op.Get.Responses["503"]; !ok {
			t.Errorf("[%s] /healthz responses missing 503 (a non-nil store ping returns 503)", a.name)
		}
	}
}

// TestNoMermaidInContract is the durable no-CDN guard review filed: the served
// OpenAPI doc body is the ONLY trigger for the vendored Stoplight Elements bundle's
// unpkg-mermaid lazy-load, so a fenced ```mermaid block in any description would let
// /docs fetch mermaid from unpkg, puncturing the hard no-CDN invariant. Banning it in
// both committed artifacts closes that latent gap. Adding a ```mermaid fence to any
// description fails this; reverting the guard makes it pass.
func TestNoMermaidInContract(t *testing.T) {
	for _, a := range artifacts() {
		if containsFold(a.doc, "```mermaid") {
			t.Errorf("[%s] OpenAPI doc body contains a fenced ```mermaid block (would trigger Elements' unpkg-mermaid fetch — no-CDN break)", a.name)
		}
	}
}

// artifact pairs a committed document's bytes with its name for table assertions
// that must hold in BOTH the YAML source and its JSON twin (so the twin cannot
// silently rot — mirroring TestVerifyForMeFlaggedWeaker).
type artifact struct {
	name string
	doc  []byte
}

// artifacts returns the two committed documents to assert each contract fact against.
func artifacts() []artifact {
	return []artifact{{"YAML", YAML}, {"JSON", JSON}}
}

// mediaKeys returns the media-type keys of a content map for a failure message.
func mediaKeys(content map[string]any) []string {
	keys := make([]string, 0, len(content))
	for k := range content {
		keys = append(keys, k)
	}
	return keys
}
