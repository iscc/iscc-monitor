# iscc-monitor

An independent Trust & Transparency service for the ISCC-Hub network: it follows
every hub's tlog-tiles transparency log, verifies signatures and RFC-6962
consistency, mirrors the logs, and publishes verifiable evidence so no party has
to trust the monitor itself.

## Development

Stack: **Go 1.24, `CGO_ENABLED=0`**, module `github.com/iscc/iscc-monitor` (ADR-0003). Quality gates
run via **mise** — `mise run check` (build + vet + test) must stay green; formatting via
`mise run fmt` / `gofmt -l .`.

The project is built by an autonomous **CID loop** (Continuous Iterative Development): the `/build`
skill runs one verified increment (`update-state → define-next → advance → review`), looped to the
target by `/goal` or a scheduled task. Specs are the source of truth (`.claude/prd`, `.claude/plans`,
`.claude/adr`); the loop's moving state lives in `.claude/context/`. See
`.claude/skills/build/SKILL.md` and `.claude/skills/build/AUTOMATION.md`.

## Running a local dev instance

The monitor is a single binary (`cmd/iscc-monitor`) configured entirely through environment variables
(`internal/config`):

- `ISCC_MONITOR_DB` (required) — path to the network's SQLite file (one file per network, ADR-0007).
- `ISCC_MONITOR_REALM` (required) — path to the realm-membership document (one hub domain per line).
  `internal/registry/testdata/realm.txt` is the testnet pilot (`sb0.iscc.id` + `sb1.amlet.id`).
- `ISCC_MONITOR_NORMAL` (optional, default `5m`) — clean-hub poll interval.
- `ISCC_MONITOR_FROZEN` (optional, default `1h`) — frozen-hub evidence-only re-poll interval; must be
  `>= NORMAL` (the freeze back-off, ADR-0006).
- `ISCC_MONITOR_ADDR` (optional, default `:9464`) — HTTP listen address.

Run it against the testnet realm with a short poll cadence:

```sh
go build -o /tmp/iscc-monitor ./cmd/iscc-monitor
ISCC_MONITOR_DB=/tmp/monitor-dev.db \
ISCC_MONITOR_REALM=internal/registry/testdata/realm.txt \
ISCC_MONITOR_NORMAL=30s \
ISCC_MONITOR_ADDR=0.0.0.0:41464 \
/tmp/iscc-monitor
```

**Dev container:** `.devcontainer/devcontainer.json` publishes `41464:41464` (`runArgs`), so binding to
`0.0.0.0:41464` (not `127.0.0.1`) makes the instance reachable from the host at `http://localhost:41464`
after a container rebuild.

After the first poll (~seconds) the HTTP surface is live. Most endpoints are JSON / text / Prometheus;
`GET /` now serves a minimal server-rendered HTML dashboard (no CSS/JS yet):

- `GET /` — server-rendered HTML dashboard listing every realm hub with its store-provable status
  (frozen / verified / inactive) and coverage window (`monitored_since` size + time, observed size).
- `GET /healthz` — liveness + store readiness.
- `GET /metrics` — Prometheus: hub status, last-observed, poll failures, violations.
- `GET /<domain>/log/` — server-rendered HTML log browser: the mirrored accepted checkpoint
  `(size, root)` for that hub plus links into its `entries`/proof routes (e.g. `/sb0.iscc.id/log/`).
- `GET /<domain>/log/checkpoint` — mirrored signed checkpoint (e.g. `/sb0.iscc.id/log/checkpoint`).
- `GET /<domain>/log/tile/...` — raw mirrored tlog-tiles BLOBs.
- `GET /<domain>/log/entries?index=<seq>` — single-leaf record bytes from the local mirror.
- `GET /<domain>/log/inclusion?iscc_id=<id>[&index=<n>]` — computed inclusion proof.
- `GET /<domain>/log/consistency?from=<n>` — computed consistency proof.
- `GET /<domain>/log/verify?iscc_id=<id>` — verify-for-me JSON verdict (hub status +
  accepted checkpoint `(size, root)` + Merkle-verified inclusion result).

## Language

**Split view**:
A hub presenting different, internally-consistent histories to different
audiences. Detectable in v1 only by *comparison* against the monitor (the monitor
is the anchor, not an autonomous detector); autonomous detection awaits monitor
gossip (M7).
_Avoid_: fork, equivocation (use "split view" as the canonical term)

**Self-consistency violation**:
Successive checkpoints the monitor observes from one hub that fail RFC-6962
consistency — i.e. the hub rewrote history *against this monitor*. This is what
the monitor detects **autonomously** and what triggers freeze + alert.
_Avoid_: inconsistency (too vague), fork

**Coverage**:
The window — `monitored_since` (size + time) to now — over which the monitor has
observed a hub. All guarantees hold only *from coverage start onward*: a
late-starting monitor recovers full data and forward guarantees by backfilling
tiles, but cannot retroactively detect equivocation from before any monitor
watched (spec §13 "cold start"). The dashboard must show the coverage window and
never imply pre-coverage guarantees.

**Comparison anchor**:
The monitor's role of publishing independently-signed records of what each hub
showed it, so a client can check *its own* `(size, root)` against the monitor's
mirrored tree and thereby detect a split view. Distinct from *Bitcoin anchoring*.
_Avoid_: witness (reserved for the deferred C2SP witness-cosigner role, M7)

**Bitcoin anchoring**:
Timestamping a checkpoint root to the Bitcoin chain via OpenTimestamps. The word
"anchoring" refers **only** to this; never to the comparison-anchor role above.

**Hub status**:
The monitor's verdict on a hub, one of: **verified** (signature valid under the
hub's did:web key, consistent, root rebuilt) · **unresolvable** (can't fetch/parse
the hub's `did.json`) · **unverified** (signature doesn't match the hub's own
did:web key = internally-broken hub) · **frozen** (self-consistency violation;
serving stops advancing, evidence preserved) · **inactive** (removed/paused in the
realm registry). Followed and mirrored regardless of status; only `verified`
checkpoints advance accepted state.

**did:web key resolution**:
The hub's signing key comes from its did:web document
(`did:web:<domain>` → `/.well-known/did.json`); the domain owner manages keys and
rotation/revocation there (CID 1.0 `verificationMethod.revoked` validity windows).
The realm registry advertises domains only, no keys. Domain ownership *is* the
identity; domain-compromise detection is out of scope.
_Avoid_: Hub-List key, valid_from (superseded — see ADR-0009)

**Verifiable cache**:
What the monitor is, as opposed to a *trusted oracle*. Fetching data from the
monitor is not the same as trusting it: clients re-verify whatever it serves
against the hub's signature and Merkle math. Only equivocation detection treats
the monitor as a (semi-trusted) reference.

**Proof bundle**:
A self-contained package — `{hub-signed checkpoint, inclusion/consistency proof,
record bytes, hub key, ots?}` — that a client verifies on its own, removing the
monitor from the trust path. Shared by the in-browser verifier, the CLI, and
verify-for-me.
_Avoid_: receipt (that is the hub's IsccReceipt, a different artifact)

**verify-for-me**:
The REST surface where the monitor returns a verdict the caller *trusts*. The
explicitly weaker, non-authoritative path; the authoritative path is the client
verifying a proof bundle itself.

**Monitor instance**:
A running deployment of the monitor (data API + per-instance server-rendered
dashboard) at its own domain — e.g. the Foundation's `monitor.iscc.id`, or a
third party's. The federation is many instances; anyone can run one.
_Avoid_: "the monitor" when you mean a specific instance

**Verifier app**:
The monitor-agnostic in-browser verifier hosted at `monitor.iscc.codes` (GitHub
Pages from the repo), separate from any instance. Takes `?monitor=<url>` and
verifies that instance's data client-side; one audited artifact serves the whole
federation. `.codes` = code, `.id` = instance.

**Mirror**:
The monitor's complete copy of a hub's hash tiles and entry bundles — stored as
BLOBs in SQLite (not a filesystem tree), served at canonical tlog-tiles paths, and
`fsck`-verifiable via a SQLite-backed fetcher. Makes the monitor an *Aggregator*
(ISCC-Log §2.3): an availability backstop that keeps a hub's log verifiable even
if the hub goes offline.

**Irreplaceable evidence**:
The monitor records that can never be regenerated: observed checkpoints, split-view
pairs, cosigs, OTS proofs. (A split-view pair is two contradictory hub-signed
checkpoints a hub will never re-serve.) Contrast with the *Mirror* and
`iscc_index`, which are rebuildable by re-fetch + fsck. Backups exist to protect
irreplaceable evidence.

**Declaration**:
A log record with `note.$schema = iscc-note-0.8.0` that commits an ISCC-ID. The
default subject of an inclusion proof.

**Deletion**:
A *new* log record (`note.$schema = iscc-note-delete-0.8.0`) carrying an existing
`iscc_id`. It marks the declaration redacted *in derived views* but never removes
the committed declaration record. One of potentially many records sharing an id.
_Avoid_: removal, takedown

**Redaction**:
A hub's *local* policy choice to suppress a declaration from its own responses. It
writes **no** log record and **does not propagate** — so the monitor never sees it
and always preserves the original declaration. Distinct from a *Deletion*.
_Avoid_: deletion (a redaction is not a log record)

**Projection**:
A derived, schema-aware view folded from log records (e.g. current owner, current
gateway URL, deletion status). Never part of the verifiable structure; the
verification/evidence path is schema-agnostic. Adding a projection for a new note
type is additive.
_Avoid_: index (the `iscc_index` is schema-agnostic; a projection interprets)
