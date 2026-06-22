<!-- area: internal/config -->
<!-- indexed-as: config.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# `internal/config` — startup-value leaf

Read this when a step touches the area above. Durable cross-cutting rules live in
the index (`.claude/context/learnings.md`); the package-local mechanics are here.

## Config loader (`internal/config`)

- **`Load(get func(key string) (string, bool)) (Config, error)` is the pure startup-value leaf**
  (`config.go`, imports exactly `{fmt time}`). `get` is injected (the binary backs it with
  `os.LookupEnv` next step), so the package touches no env/fs/net and is 100%-covered by a map-backed
  fake. Keys are `ISCC_MONITOR_{DB,REALM,NORMAL,FROZEN}`; `DB`/`REALM` required (absent OR empty →
  named error), intervals default `Normal=5m`/`Frozen=1h`, parsed via `time.ParseDuration`. The keys
  are package constants the binary references symbolically and are not yet load-bearing against any
  external contract — renamable cheaply at wiring time (e.g. if a flag surface is preferred).
- **The load-bearing cross-check is `Frozen >= Normal`** — it ties config to `loop.go due()`'s
  back-off (a frozen hub polls on the *longer* `Frozen` interval, ADR-0006). Verified `Loop.Normal`/
  `Loop.Frozen` are `time.Duration` fields config feeds directly; `Frozen == Normal` is accepted,
  `Frozen < Normal` rejected. The `frozen < normal` error names BOTH keys; the test asserts only that
  `keyFrozen` appears, which holds.
- **A *present but* non-positive interval (`0s`, `-1m`) is rejected** (named error) — a deliberate
  addition beyond the literal `>0` spec wording; absent keys still take the positive defaults. Oracle
  gate correctly N/A (no proof/verify/didweb/merkle/fsck path; go.mod/go.sum byte-identical). Note:
  `required` does NOT trim whitespace, so a whitespace-only path (`" "`) passes config and would fail
  later at `os.ReadFile`/`store.Open` — acceptable (config validates presence, the binary owns I/O),
  but the binary should surface that fs error clearly.
- **settled: `ISCC_MONITOR_REALM` stays a `required` *config* key, but the IMAGE now supplies its value
  via `ENV`.** The `Dockerfile` both `COPY`s `deploy/realm-testnet.txt` → `/etc/iscc-monitor/realm.txt`
  AND sets `ENV ISCC_MONITOR_REALM=/etc/iscc-monitor/realm.txt` (`Dockerfile:51`), so a fresh container
  boots without the var being passed — and the CI `docker` smoke job proves it (it drops the `-e
  ISCC_MONITOR_REALM` line; a missing/misspelled `ENV` would make `config.Load` exit non-zero and time the
  smoke loop out). Two distinct contracts: `config.Load` still calls `required(get, keyRealm)` (a missing
  *config value* errors) — the loader knows nothing about a baked default; the image supplies the *value*.
  So a Compose / `docker run` snippet may legitimately OMIT the var (it is baked), but `ISCC_MONITOR_DB`
  stays un-defaulted (no safe image default — it must point at the operator's volume). Do NOT add a Go-side
  `RealmPath` default — that would invert the contract and break the map-backed config tests.
- **Masthead-identity keys (`ISCC_MONITOR_{INSTANCE,OPERATOR,REALM_NAME}`) are free-form display
  strings read through `optional(get, key, "")`** — no validation, empty default; each masthead handler
  applies its own static fail-safe on an empty field, so config must NOT carry a fallback copy (that
  would duplicate the handlers' placeholder). `RealmName` is deliberately the human realm NAME, distinct
  from the required `RealmPath` (`ISCC_MONITOR_REALM`, the realm-document filesystem path) — overloading
  the path var would leak a filename into the ledger subtitle. `main.go`'s `identity(cfg)` (not config)
  builds `dashboard.Identity` from these fields; config stays a `{fmt time}`-only leaf with no
  `dashboard` import (would invert the dep / risk a cycle).
