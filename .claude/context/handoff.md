## 2026-06-22 — Ship the canonical `deploy/realm-testnet.txt` and bake it into the image

**Done:** Created the canonical, mountable/bakeable testnet realm document at
`deploy/realm-testnet.txt` (domains-only, ADR-0009, the two real testnet hubs), repointed the
Dockerfile bake `COPY` from the Go testdata fixture to this canonical file, added a non-vacuous golden
test that `registry.Parse` accepts the real `deploy/` file as exactly the two ordered entries, and
documented the canonical path in CLAUDE.md's env table. The deploy contract no longer reaches into Go
testdata.

**Files changed:**
- `deploy/realm-testnet.txt` (new): canonical realm doc — header comment naming it the
  mountable/bakeable doc baked at `/etc/iscc-monitor/realm.txt`, then `sb0.iscc.id` + `sb1.amlet.id`.
- `internal/registry/deploy_test.go` (new test, not counted): `TestParseCanonicalDeployRealm` reads
  `../../deploy/realm-testnet.txt` and `reflect.DeepEqual`-asserts the two ordered `Entry` values;
  mirrors `TestParseGolden`.
- `Dockerfile`: bake `COPY` source changed to `deploy/realm-testnet.txt` (baked path
  `/etc/iscc-monitor/realm.txt` unchanged); adjacent comment updated — no longer calls the canonical
  file "a later M-Deploy slice", now describes it as the canonical mount/bake doc.
- `CLAUDE.md` (doc): env table for `ISCC_MONITOR_REALM` now names `deploy/realm-testnet.txt` as the
  canonical doc baked at `/etc/iscc-monitor/realm.txt`; reframes testdata as the in-tree test fixture.

**Verification:** `mise run check` → green (all 28 packages `ok`; `go build`/`go vet` clean; `gofmt -l .`
empty outside `cauldron/`). Per-criterion:
- [x] `mise run check` green.
- [x] `go test -run TestParseCanonicalDeployRealm ./internal/registry` → `ok` (reads the real file,
  asserts two ordered entries).
- [x] `deploy/realm-testnet.txt` exists under repo-root `deploy/`, domains-only, `Parse` accepts it →
  exactly `sb0.iscc.id`, `sb1.amlet.id`.
- [x] `grep -q 'COPY deploy/realm-testnet.txt /etc/iscc-monitor/realm.txt' Dockerfile` succeeds;
  `grep 'COPY .*testdata/realm.txt' Dockerfile` → no match.
- [x] CLAUDE.md env table names `deploy/realm-testnet.txt` baked at `/etc/iscc-monitor/realm.txt`.

**Next:** The operability/deployment doc is the next M-Deploy slice — it can fold in the remaining
three `critical` infra asks (persistence volume path, backup unit for irreplaceable evidence, egress /
`/metrics` exposure, migration policy). Then the root `README.md` (`target.md` "Done When" requires it
before DONE). The `normal` `workflow_dispatch` ref-guard issue stays a fold-in candidate whenever a
workflow file is next touched.

**Notes:**
- The canonical `deploy/realm-testnet.txt` is content-equivalent but NOT byte-identical to
  `internal/registry/testdata/realm.txt`: the testdata fixture has a blank line between the two hubs and
  a different header comment; the canonical file has no inter-hub blank and a deploy-oriented header.
  This is intentional and harmless — `Parse` drops blank lines and `#` comments, so both map to the
  identical two ordered `Entry` values (both golden tests confirm). `next.md` Scope said "same content
  as testdata"; I read that as the same *parsed* membership (two domains, ADR-0009 domains-only), not a
  byte-for-byte copy, since the canonical file needs its own canonical-doc header comment. Flagging for
  review in case byte-identity was intended — trivially adjustable, but parse-equivalence is what the
  deploy contract and the tests require.
- Docker is CI-only on this host (per prior reviews) — the Dockerfile `COPY` change is a one-line
  source-path swap to an existing tracked file and cannot be `docker build`-verified locally; the CI
  `docker` job is the real oracle (bakes the realm, boots, polls `/healthz`). The baked path is
  unchanged, so that smoke keeps passing.
- `internal/registry/testdata/realm.txt` left untouched (Not-In-Scope honored) — it stays the in-package
  test fixture; the two files are independent artifacts with no import relationship.
