<!-- area: internal/version (the build-provenance HTTP leaf + the -ldflags -X stamp) -->
<!-- indexed-as: version.md · owner: review · rotate at ~40 bullets / ~150 lines -->

# `internal/version` — build-provenance stamp

Read this when a step touches `internal/version`, the `GET /version` mount, or any `-ldflags -X`
build-stamp path (incl. the Dockerfile/CI image build). Durable cross-cutting rules live in the
index (`.claude/context/learnings.md`); the package-local mechanics are here.

- **The leaf mirrors `internal/healthz`/`internal/metricshttp` exactly.** `var Version = "dev"` (a
  plain `string` var initialized to a constant — NOT a `const`, NOT computed/concatenated, or `-X`
  cannot override it) + a `Handler()` that on GET sets `Content-Type: application/json`, `WriteHeader(200)`,
  then `fmt.Fprintf(w, \`{"version":%q}\`, Version)` (the `%q` keeps a stamped SHA valid JSON even if it
  ever contained a quote). Non-GET → 405. The dropped post-status write error is the documented
  leaf-endpoint convention (a derived body cannot fail for content reasons after `WriteHeader`; only a
  broken client conn). Imports only `fmt`/`net/http` — a pure stdlib leaf, no `internal/*` dep
  (`go list -deps` shows none other than itself). It is an HTTP leaf, NOT WASM-shared, so the
  `proof/verify`/`didweb` import-purity rule does not bind it.
- **`/version` is an EXACT mount and MUST be a reserved domain.** `buildMux` registers
  `mux.Handle("/version", version.Handler())` next to `/metrics`/`/healthz`; `"version"` is in
  `reservedMountNames` (and the four doc comments — `reservedMountNames`, `reservedDomain`, `buildMux`,
  `registerHubs` — were all updated to name it) so a realm line literally `version` cannot build an
  exact `/version` dossier mount first and panic `http.ServeMux` on the duplicate pattern. This is the
  same discipline `learnings/cmd-monitor.md` documents for `/metrics`/`/healthz`/`/_ds`; any future
  exact bare-domain mount reuses it.
- **`-X` empty-stamp OVERRIDES the default — it is NOT a no-op (trust-root-adjacent provenance trap).**
  An empty value `-ldflags "-X …Version="` builds clean and sets `Version` to `""`, clobbering the `dev`
  default (reviewer-proven: `[]` printed, build exit 0). So `build:monitor`'s
  `-X …Version=$(git rev-parse --short HEAD)` produces an EMPTY-version binary whenever `git rev-parse`
  fails (no `.git` in a Docker build context, a source export, or no `git` on PATH): `$(…)` exits 128
  but empty-expands and the outer `go build` succeeds. The default `check`/`build` tasks are git-free and
  build the non-empty `dev`, so the GATE never sees this — it surfaces only in the stamped path the next
  M-Deploy/Dockerfile slice consumes. **Both stamped consumers are now guarded** (`normal` issue closed):
  the Dockerfile build RUN (`[ -n "$VERSION" ] || exit 1`) AND `mise.toml build:monitor`
  (`sha=$(git rev-parse --short HEAD) && [ -n "$sha" ] && go build …`, advance `a15a9f4`) both fail fast
  on an empty value rather than shipping a blank `/version`. General rule for any future `-X` stamp: an
  empty substitution silently defeats the provenance, never errors — guard the value, do not trust
  `$( )` to abort the build.
- **It is deliberately NOT a config key.** A compile-time `-ldflags` stamp, not a runtime env value, so
  `internal/config` stays a `{fmt time}`-only leaf and there is no `ISCC_MONITOR_VERSION`. The target
  offered `/healthz` JSON OR a `GET /version`; the slice chose `/version` and left `internal/healthz`
  byte-identical (its fixed-byte-literal "no marshal branch" body is deliberate).
- **Oracle gate is N/A here** — pure HTTP wiring + a build-stamp string; touches no signature /
  RFC-6962 / Merkle / did:web / proof / fsck path; `go.mod`/`go.sum`/`schema.sql` stay byte-identical
  (stdlib only, no new dependency).
