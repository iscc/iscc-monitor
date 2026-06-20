# Issues

Lightweight backlog `define-next` can prioritize. Append entries; `review` deletes resolved ones.

**Format** — one entry per issue:

```
## <short title>
- **Priority:** critical | normal | low
- **Source:** [human] | [review] | [advance]
- **What / where / how to verify:** <the problem, its location, and the check that proves it fixed>
- **Spec:** <optional — target.md or an ADR section this is rooted in>
```

**Priority semantics:** `critical` preempts everything; `normal` is weighed against the state→target
gap; **`low` is skipped by the loop** (reserved for human-directed work). The `Source` tag records who
filed it and does **not** affect priority.

---

## `parseTime` fails open on a malformed (non-empty, unparseable) validity timestamp
- **Priority:** normal
- **Source:** [review]
- **What / where / how to verify:** `internal/didweb/resolve.go::parseTime` returns the zero
  `time.Time` for BOTH an absent field and a non-empty-but-unparseable RFC-3339 string (e.g. a garbled
  `revoked`/`validUntil`). `DIDKey.ValidAt` reads zero as "no constraint", so a malformed revocation
  timestamp silently fails *open* — the key stays valid. Deliberately deferred from the `ValidAt` step
  (changing it alters parsing semantics + the existing golden test). Decide at the `hub_keys`/fixture
  step: a hub serving a malformed validity window should arguably degrade to not-`verified` rather than
  be treated as unconstrained. Verify-fixed: a did.json with `"revoked": "not-a-date"` must NOT yield a
  `verified` verdict (add a fixture/test asserting the malformed-timestamp path is fail-closed, or an
  explicit documented decision that fail-open is intended with a parse-error surfaced to the caller).
- **Spec:** ADR-0009 (validity windows are the domain owner's rotation/revocation mechanism; a
  signature outside the window must not be `verified`).
