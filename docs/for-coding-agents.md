---
title: For Coding Agents
description: Dense reference for AI agents that verify ISCC Monitor evidence, call its API or change the monitor itself
icon: lucide/bot
---

# For Coding Agents

Prescriptive reference for agents. Two audiences: agents that **use** a monitor instance (verify evidence, call the
API; sections marked *use*) and agents that **change** the monitor in `iscc/iscc-monitor` (marked *change*).
Full Markdown of this site: [`llms-full.txt`](https://monitor.iscc.codes/llms-full.txt).

## Terms

Use these words exactly. The codebase, the UI and these docs use them the same way.

| Term                     | Meaning                                                                                   | Do not say                 |
| ------------------------ | ----------------------------------------------------------------------------------------- | -------------------------- |
| Monitor instance         | One running deployment, such as `monitor.iscc.io`                                         | "the monitor" for one host |
| Hub status               | `verified`, `unresolvable`, `unverified`, `frozen` or `inactive`                          |                            |
| Self-consistency violation | Two checkpoints from one hub that fail RFC 6962 consistency; freezes the hub            | inconsistency, fork        |
| Split view               | A hub showing different histories to different audiences; found by comparison only       | fork, equivocation         |
| Comparison anchor        | The monitor's role as a signed reference to compare a client's `(size, root)` against     | witness                    |
| Bitcoin anchoring        | OpenTimestamps proof of a checkpoint root in Bitcoin; the only meaning of "anchor"        |                            |
| Coverage                 | `monitored_since` (size and time) to now; no guarantee before it                          |                            |
| Proof bundle             | Self-contained JSON evidence a client verifies without the monitor                        | receipt                    |
| verify-for-me            | `/<hub>/log/verify`: the monitor's own verdict, the weaker path                            |                            |
| Declaration / Deletion   | Log records with `note.$schema` `iscc-note-0.8.0` / `iscc-note-delete-0.8.0`              |                            |
| Redaction                | A hub's local suppression; writes no record, the monitor never sees it                     | deletion                   |

## Decision dispatch *(use)*

| Goal                                                     | Use                                                                         |
| -------------------------------------------------------- | --------------------------------------------------------------------------- |
| Prove an ISCC-ID is in its hub's log, trusting nobody    | `GET /inclusion/<iscc_id>.bundle`, then verify it yourself (recipe below)   |
| Show a person the evidence                               | `GET /inclusion/<iscc_id>`, the HTML certificate                            |
| Quick yes/no, trusting the instance                      | `GET /<hub>/log/verify?iscc_id=<id>`                                        |
| Check your own `(size, root)` for a split view           | Compare with `GET /<hub>/log/checkpoint`; prove growth with `consistency`   |
| Read a hub's log when the hub is offline                 | The tlog-tiles mirror: `/<hub>/log/checkpoint`, `/<hub>/log/tile/...`       |
| Get one record's bytes                                   | `GET /<hub>/log/entries?index=<n>`                                          |
| Bitcoin timestamp of a root                              | `GET /<hub>/log/checkpoint.ots[?size=<n>]`, then `ots verify -d <root hex>` |
| Machine-readable API description                         | `GET /openapi.json` or `/openapi.yaml` (OpenAPI 3.1)                        |
| Health, build, metrics                                   | `/healthz`, `/version`, `/metrics`                                          |

All routes are `GET`, need no authentication and send `Access-Control-Allow-Origin: *`. Every page:
[HTTP API](reference/http-api.md).

## Constraints *(use)*

| Rule                                                                                                          |
| ------------------------------------------------------------------------------------------------------------- |
| ISCC-ID inputs accept any case, surrounding whitespace and an `ISCC:` or `did:iscc:` prefix.                   |
| The ISCC-ID itself names its hub (12-bit `hub_id`); the instance resolves it through the realm's Hub-List.     |
| "Cannot certify" answers are `200`, not 4xx/5xx: the bundle route returns `{"error": ..., "iscc_id": ...}`, verify-for-me returns `"verified": false` with a `reason`. Check the body. |
| A bundle exists only after the instance verified the inclusion itself; it is never partial.                    |
| `anchor` in a bundle is optional; it is present only when a confirmed Bitcoin anchor covers the record.        |
| `key.revoked` (RFC 3339) is present only when the hub's did:web document marks the key revoked.                |
| `/<hub>/log/consistency?from=<n>` accepts any `n` up to the accepted size; a larger `n` is `400` (the instance has not seen that state yet). |
| `checkpoint.ots` answers `404` while the root is not yet anchored.                                             |
| A log's origin is `<hub domain>/log`, for example `iscc.id/log`.                                               |
| Guarantees start at coverage start; an instance cannot vouch for what happened before it watched.              |

## Recipe: verify a proof bundle *(use)*

The documented script does every step: [Verify a proof bundle](guides/verify-bundle.md)
(`uv run verify_bundle.py --resolve <file>`). To implement it yourself, with the members from
[Proof bundle format](reference/proof-bundle.md):

| Step | Check                                                                                                                           |
| ---- | ------------------------------------------------------------------------------------------------------------------------------- |
| 1    | `key.multibase` is base58btc (`z…`); decoded, it is `0xed 0x01` followed by the 32-byte Ed25519 public key.                      |
| 2    | `checkpoint` is a signed note: body lines origin, size, base64 root; a blank line; signature lines `— <name> <base64>`.            |
| 3    | The signature base64 decodes to a 4-byte key hash and a 64-byte signature. Key hash = `SHA-256(name + "\n" + 0x01 + pubkey)[:4]`; it equals `key.id` (hex). |
| 4    | The Ed25519 signature verifies over the note body, up to and including its last newline before the blank line.                   |
| 5    | Leaf hash = `SHA-256(0x00 + base64decode(record))`.                                                                              |
| 6    | Rebuild the root from the leaf, `inclusion.leafIndex`, `inclusion.treeSize` and `inclusion.inclusionProof` (RFC 9162 §2.1.3.2); it equals the checkpoint root and `treeSize` equals the checkpoint size. |
| 7    | Fetch `https://<hub.domain>/.well-known/did.json` yourself; `key.multibase` must be a `publicKeyMultibase` in it.                 |
| 8    | If `anchor` is present: steps 2–6 for its `checkpoint`, `inclusion` and `key`; then `ots verify -d <anchor root in hex>` on its base64-decoded `ots` (needs a Bitcoin node). |

Skipping step 7 leaves the monitor in the trust path.

## Architecture map *(change)*

Go 1.26, `CGO_ENABLED=0`, one binary, configured only by `ISCC_MONITOR_*` environment variables, one SQLite file per
network.

| Path                     | Contains                                                                                  |
| ------------------------ | ----------------------------------------------------------------------------------------- |
| `cmd/iscc-monitor`       | `main.go`: config, realm loading, hub registration, HTTP mux, OTS loop, realm refresh      |
| `cmd/wasm`               | In-browser inclusion verifier, built to `internal/web/verify.wasm`                        |
| `cmd/notices`            | Generates `THIRD_PARTY_NOTICES.txt`                                                        |
| `internal/config`        | Typed, validated startup configuration                                                    |
| `internal/registry`      | Hub-List YAML (or legacy domains file) parsing; `AtomicHubList` hub_id resolver            |
| `internal/logclient`     | Pure verification chain: checkpoint, did:web key, consistency, tiles, fsck, inclusion proofs |
| `internal/didweb`        | did:web key resolution, revocation windows                                                |
| `internal/follower`      | `Loop` and `PollHub`: the per-hub poll pipeline; freeze; OTS stamping (`OTSTick`)          |
| `internal/store`         | SQLite: `hubs`, `checkpoints`, `tiles`, `entry_bundles`, `follow_state`, `hub_keys`, `iscc_index`, `ots`, `violations`; `schema.sql` plus `user_version` migrations |
| `internal/proof/verify`  | Pure `VerifyInclusion`, shared by server and WASM                                         |
| `internal/proofserve`    | `/<hub>/log/inclusion`, `consistency`, `entries`, `verify`                                |
| `internal/tilesserve`    | `/<hub>/log/checkpoint`, `tile/...`, `checkpoint.ots`                                      |
| `internal/certificate`   | `/inclusion/<iscc_id>` and its `.bundle`; Trust Seal                                       |
| `internal/dashboard`, `dossier` | `/` and `/<hub>` pages                                                             |
| `internal/chrome`, `badge`, `web` | Shared masthead and footer, status badge, `/_ds/` static assets                  |
| `internal/openapi`, `docs` | `openapi.yaml` + `openapi.json`, `/docs` API reference page                             |
| `internal/index`         | ISCC-IDv1 decoding and `Canonical`                                                        |
| `internal/ots`, `otsclient` | OpenTimestamps proofs and calendar transport                                           |
| `internal/metrics`, `metricshttp`, `healthz`, `version`, `corsmw` | Operations routes and the CORS middleware          |
| `docs/`, `zensical.toml` | This site                                                                                 |
| `.claude/`               | PRD, ADRs, design specs, CID-loop state; not public docs                                  |

Runtime: `run()` starts the HTTP server, the OTS loop and the hourly realm refresh as goroutines; `follower.Loop` runs
in the foreground and polls each hub (`ISCC_MONITOR_NORMAL`, or `ISCC_MONITOR_FROZEN` for frozen hubs).

Poll pipeline (`follower.PollHub`): fetch checkpoint → `AcceptCheckpoint` verdict (signature under the did:web key)
→ only if `verified`: mirror the tiles → RFC 6962 consistency against the prior accepted checkpoint → on a
violation, freeze the hub, keep both signed checkpoints and alert once; otherwise record the checkpoint and coverage,
advance the follow cursor, rebuild the root from the mirror (fsck) and record the root for OpenTimestamps. A frozen
hub is re-polled for evidence only and never advances.

## Change playbook *(change)*

| If you change                          | Also                                                                                           |
| -------------------------------------- | ---------------------------------------------------------------------------------------------- |
| An `ISCC_MONITOR_*` key                | `docs/operate/configuration.md`; `internal/config` has a test that fails when the two differ    |
| A machine route                        | `internal/openapi/openapi.yaml` and `openapi.json` (tests check they agree and match the mux); `docs/reference/http-api.md` |
| An HTML route                          | `docs/reference/http-api.md`; keep it out of the OpenAPI document                              |
| The proof bundle                       | `docs/reference/proof-bundle.md`, `docs/guides/verify_bundle.py`, the example `docs/reference/MEIGKV5N6NTVF4AB.bundle.json` (the docs workflow runs the script on it) |
| `internal/proof/verify` or `cmd/wasm`  | `mise run build:wasm`, then `web.WasmVerifyHash`; a test pins the hash of the committed file    |
| A metric or a log line                 | `docs/operate/monitoring.md`                                                                   |
| The database schema                    | Append a migration step (`user_version`); never edit an applied one; `docs/operate/data.md`    |
| A dependency or vendored asset         | `mise run notices`; a new vendored asset also goes into `vendoredAssets` in `cmd/notices`       |
| Operator branding                      | `docs/operate/branding.md`                                                                     |
| Anything user-visible                  | `CHANGELOG.md`, written by hand for readers                                                    |
| The public API, CLI or configuration   | Update this page                                                                               |

Gates: `mise run check` (build, vet, test) and `gofmt -l .` must be clean; for docs, `mise run docs:build` and
`mise run docs:check` must end with `0 errors, 0 warnings`.

## Common mistakes

**NEVER** treat a `200` from the bundle or verify route as success. **ALWAYS** read `error`, `verified` and `reason`.

---

**NEVER** trust the `key` in a bundle on its own. **ALWAYS** check it against the hub's did:web document.

---

**NEVER** call the Comparison anchor an "anchor" alone or a "witness". **ALWAYS** reserve "anchor" for Bitcoin
anchoring.

---

**NEVER** claim the monitor detects split views on its own. **ALWAYS** say it detects self-consistency violations;
split views need a comparison against it.

---

**NEVER** advance accepted state for a hub that is not `verified`, or for a frozen hub. **ALWAYS** keep frozen hubs
evidence-only.

---

**NEVER** add a request to another origin to a page (CDN, fonts, analytics). **ALWAYS** serve assets from `/_ds/`.

---

**NEVER** commit to `main` or create tags from an automated loop. **ALWAYS** release through a PR from `develop`
with a merge commit, as in `CLAUDE.md`.
