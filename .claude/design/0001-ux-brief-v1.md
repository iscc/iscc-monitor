# UX Design Brief — iscc-monitor v1 web surfaces

> For: UX designer · From: product/eng · Source of truth: `.claude/prd/0001-iscc-monitor-v1.md`
> and `CONTEXT.md` (vocabulary). Where this brief and an ADR/PRD disagree, the PRD/ADR wins —
> flag it, don't silently diverge.

## TL;DR

Design the web-facing surfaces of a **trust & transparency monitor** for a network of content-timestamping
servers ("hubs"). The whole point of the product is that **users should not have to trust the monitor**.
Your central design problem is therefore communicating an unusual idea to non-technical people: *"this page
is showing you evidence you can re-check yourself, not a verdict you have to believe."* Get that right and
the rest follows.

Deliver **three alternative design directions, each organized around a different axis** (defined at the end),
plus a recommendation. Wireframe/flow fidelity is fine — we want to compare structural and trust-communication
choices, not pixels.

---

## 1. What the product is (in one minute)

A network ("realm") has many **hubs**. Each hub publishes an append-only, cryptographically-verifiable log of
content declarations (think: a tamper-evident ledger of "this content existed and was registered at this time").
Hubs are supposed to never rewrite history, but **nobody independently watches them**.

The **monitor** is that independent watcher. One deployed monitor ("a **monitor instance**", e.g. `monitor.iscc.id`)
continuously:

- **follows** every hub, **verifies** each hub's cryptographic signatures and checks that the hub never rewrote
  its history (RFC-6962 consistency);
- **freezes** a hub the moment it catches it misbehaving, preserving the evidence permanently;
- **mirrors** every hub's data, so it stays an availability backstop and can serve proofs even if a hub goes offline;
- **Bitcoin-anchors** the data it verified (a trustless timestamp via OpenTimestamps);
- and publishes all of this for people to **re-verify themselves**.

The people who most need this — **declarers, auditors, litigators, businesses, legal staff** — are *not*
command-line users. They need to *see* the trust situation in a browser, and verify a specific claim, without
installing anything and **without having to trust the monitor either**.

---

## 2. The one idea everything hangs on: verifiable cache, not trusted oracle

This is the soul of the product. Please internalize it before designing.

- The monitor is a **verifiable cache**, *not* a **trusted oracle**. Reading a green badge it rendered is NOT the
  same as the claim being true.
- There are two tiers of trust in every screen, and the UI must keep them visually distinct:
  1. **"The monitor reports…"** — server-rendered state. Convenient, but you're trusting the monitor's display.
  2. **"Your own browser verified…"** — the user's browser re-ran the cryptographic math locally (via a WASM
     verifier). This is the authoritative tier and the product's killer feature.
- The most important non-technical user story (PRD #7) is literally: *"I want my own browser to re-verify the
  claim, not just read a badge the server rendered."* Make that act **visible, legible, and reassuring** to
  someone who has no idea what a Merkle proof is.

A design that makes the monitor's green checkmark look like the final word has **failed the brief**, no matter
how polished.

---

## 3. Audiences (design for these, in priority order)

| Persona | Technical? | Core goal on the site | What they fear |
|---|---|---|---|
| **Declarer** (registered some content) | No | "Prove my declaration is really, permanently in the log." | The hub disappears or quietly drops their record. |
| **Non-technical business / legal user** | No | "At a glance, is this network healthy and is *this* hub trustworthy right now?" | Trusting a number a server just made up. |
| **Auditor / litigator** | Mixed | "Get durable, self-contained evidence that stands up without trusting anyone." | Evidence that depends on a party that can change its story. |
| **Skeptical / technical client** | Yes | "Check *my own* record of the log against the monitor's, and re-verify locally." | A hub showing different histories to different people (a **split view**). |
| **Monitor / hub operator** | Yes | "Is my instance healthy? Is my hub being seen correctly?" | A real problem hidden behind a misleadingly-green UI. |

Primary design target: **the two non-technical personas**, without dumbing the experience down for the auditor who
needs depth. Progressive disclosure is your friend.

---

## 4. Surfaces & screens to design

There are **three product surfaces**. Two live on each monitor instance; one is a separate, federation-wide app.

### A. Trust dashboard *(server-rendered, per instance, e.g. `monitor.iscc.id`)*
The front door. Network-wide health + per-hub trust status, glanceable, honest.

### B. Log browser *(server-rendered, per instance)*
Browse the records inside a given hub's log — declarations and deletions — and inspect any single record.

### C. Verifier app *(separate origin: `monitor.iscc.codes`, static/GitHub Pages, monitor-agnostic)*
The trust-critical, in-browser re-verifier. It is **deliberately hosted on a different domain** that no monitor
controls, so a malicious monitor can't tamper with the checker. It runs the verification math **locally in the
user's browser (WASM)** and works against *any* monitor instance via a `?monitor=<url>` parameter. One audited
checker, used across the whole federation.

> Relationship: the verifier app is a **progressive enhancement** layered on the server-rendered dashboard. The
> dashboard must be useful and readable with **no JavaScript at all**; in-browser re-verification is the upgrade.

#### Concrete screens / states we need from you
1. **Dashboard — network overview**: all hubs in the realm, each with status, coverage, last activity.
2. **Hub detail**: one hub's full trust picture — status, coverage window, latest checkpoint, Bitcoin-anchor
   status, history of observations, link into its log.
3. **Frozen-hub state**: the alarming case — a hub caught rewriting history. This must look *categorically*
   different from a benign error. (See §5 — "frozen" ≠ "unresolvable".)
4. **Log browser — record list** (paginated/searchable) for a hub.
5. **Single record / declaration view**, including the case where a declaration was later **deleted** (the original
   is never removed; the deletion is an additional record — show the full per-id history honestly).
6. **Inclusion result / proof view**: "is my specific ISCC-ID in the log?" → the answer, the evidence, the
   Bitcoin timestamp, and a **download the proof bundle** action.
7. **Verify-in-your-browser flow**: the moment local re-verification happens — idle → verifying → verified
   (or mismatch). Including the **"check my own (size, root)"** input for the skeptical client detecting a split view.
8. **Verifier app standalone**: how it looks when opened directly (with/without a `?monitor=` target), how the
   user understands they're on an *independent* checker, and how it shows *which* instance it's auditing.
9. **Empty / loading / error / degraded states** for all of the above (a hub offline, did-doc unresolvable, OTS
   not yet anchored, monitoring just started so coverage is thin, etc.).

---

## 5. Invariants — non-negotiable, must hold in **every** direction

These come from the spec and ADRs. Treat them as hard constraints, not suggestions.

1. **Two trust tiers stay distinct.** Never let "the monitor reports X" look identical to "your browser verified X."
   (§2.)

2. **Honest coverage, always.** Every hub has a **coverage window** — `monitored_since` (a size + a time). All
   guarantees hold **only from coverage start onward**. The monitor *cannot* vouch for anything before it started
   watching. The UI must **show the coverage window** and must **never imply pre-coverage guarantees**. (A claim
   timestamped before coverage started should visibly say so.)

3. **The five hub statuses are distinct and must read distinctly** — especially that a *caught-misbehaving* hub
   never looks like a *misconfigured* one:

   | Status | Plain meaning | Tone |
   |---|---|---|
   | **verified** | Signature valid, history consistent, data rebuilt. All good. | calm / positive |
   | **unresolvable** | Monitor can't fetch/parse the hub's identity document. Likely a config/availability issue. | caution / "needs attention" |
   | **unverified** | The hub is signing with a key its *own* identity document doesn't list — internally broken. | caution / "something's wrong with the hub" |
   | **frozen** | The monitor **caught the hub rewriting its own history** (a self-consistency violation). Serious. | alarm / "do not trust new state" |
   | **inactive** | Removed/paused in the realm registry. Not a fault. | neutral / muted |

   `frozen` is the headline event of the whole product — design it to be unmistakable and *non-dismissable* (it
   requires manual operator review to clear; it must never look like a transient blip that'll self-heal).

   Note: **all** hubs keep being mirrored and served regardless of status — the monitor reports problems, it never
   goes dark on a hub. So even a `frozen` or `unresolvable` hub still has browsable data and a coverage window.

4. **Color is never the only signal.** With five statuses including a critical one, status must be conveyed by
   **icon + label + (optionally) shape/position**, not hue alone. Accessibility-grade contrast and colorblind-safe.
   Assume these states will be screenshotted into legal/audit contexts in grayscale.

5. **The verifier app's independence must be felt.** The user should be able to tell they're using a checker that
   the monitor they're auditing does **not** control, and see **which** instance it is pointed at. Don't bury the
   `monitor.iscc.codes` ↔ `monitor.iscc.id` distinction.

6. **Two kinds of "anchor" — never conflate them.**
   - **Bitcoin anchoring** = a trustless timestamp on the Bitcoin chain (via OpenTimestamps). The word "anchor"
     in time/Bitcoin contexts means *only* this. The authoritative check is the user running `ots verify` /
     downloading the `.ots` — the monitor's own "anchored" claim is calendar-asserted, not authoritative. Show
     it as "pending → confirmed on Bitcoin" honestly; "pending" is the normal early state, not an error.
   - **Comparison anchor** = the monitor's role as a reference a client compares its own `(size, root)` against
     to catch a split view. Different concept. If both appear on one screen, they must be visibly different things.

7. **Vocabulary discipline.** Use the canonical terms from `CONTEXT.md`. In particular:
   - Say **"split view"** — avoid "fork", "equivocation".
   - Say **"self-consistency violation"** for what triggers a freeze — avoid bare "inconsistency".
   - Say **"coverage"** for the monitored window.
   - **"anchoring"** is Bitcoin-only (see above).
   - A **deletion** is a *new record*, never a removal/takedown; the original declaration is always preserved.
   - Don't use "witness" anywhere (reserved for a future deferred role).
   Provide microcopy that a non-technical reader understands *and* that an auditor recognizes as precise. Where a
   precise term is unavoidable (e.g. "checkpoint", "inclusion proof"), pair it with a plain-language gloss or tooltip.

8. **Server-rendered baseline.** The dashboard and log browser must work as plain server-rendered HTML/CSS with
   **no JS**. WASM re-verification is layered on top. Don't propose anything that *hard-requires* a heavy SPA
   framework for these two surfaces. The verifier app may be fully client-side (it's a static site).

9. **Multiple instances, no central brand authority.** Anyone can run a monitor instance on their own domain. The
   instance dashboard's chrome should make the **operator/instance identity** legible (whose monitor is this?)
   without implying the data is more authoritative because of who's hosting it.

10. **Truthful empties & pending.** "No anchor yet", "coverage just started", "hub offline, serving mirror" are
    normal, expected states — design them as informative, not as failures or scary errors.

---

## 6. What data each key screen must surface

So you're not guessing at content. (Field names are illustrative; copy is yours to craft.)

- **Hub (dashboard card / detail):** human name + domain · status (one of the five) · coverage window
  (`monitored_since` size + time) · latest checkpoint (tree size + observed time) · Bitcoin-anchor status
  (pending/confirmed + block height when confirmed) · last-seen / activity · link to its log browser. If
  `frozen`: the violation kind (a split view / a shrink / a rewrite), when detected, and that evidence is preserved.

- **Inclusion / proof result:** the ISCC-ID being checked · found? at which position · the hub-signed checkpoint
  it's proven against · the inclusion proof itself · the record's raw bytes · the hub's key · the Bitcoin timestamp
  (if any) · **any other records for the same id** (e.g. a later deletion — show the full history) · a
  **"download proof bundle"** action · a **"verify this in your browser"** action.

- **Single record:** its ISCC-ID · its type (declaration / deletion / *or an unrecognized future type* — the system
  indexes types it doesn't recognize, so your record view must degrade gracefully for an unknown type) · which hub
  · position in the log · link to prove inclusion.

- **Verifier app:** which monitor instance it's pointed at (`?monitor=`) · the input it's verifying (an id, or a
  pasted/loaded proof bundle, or a user-supplied `(size, root)`) · the live verification state · the verdict, with
  an explicit "this was computed **here, in your browser**" framing · what to do on a **mismatch** (the scary,
  important path — a mismatch may mean a split view; guide the user, don't just throw a red error).

---

## 7. Tone & brand notes

There is **no existing brand or visual identity** — you have latitude, within these guardrails:

- **Trustworthy, sober, precise.** This is infrastructure for evidence and legal/audit contexts, closer to a status
  page / certificate-transparency tool / financial ledger than to a consumer app. Avoid playful, avoid hype.
- **Honest over reassuring.** The product's credibility comes from *showing problems plainly*. It must be just as
  good at saying "this hub is frozen / not covered / not yet anchored" as at showing green. Resist any visual
  language that makes everything-is-fine the default resting impression.
- **Screenshot-ready as evidence.** People will paste these screens into reports and legal filings. Favor clear
  labels, visible timestamps, legible-in-grayscale, and unambiguous state.
- Responsive (desktop-first is acceptable for the dashboard; the verifier app and inclusion result should be usable
  on mobile, since a declarer may check a claim from a phone).

---

## 8. Out of scope (don't design these)

So you don't over-build:

- **Cosigning / multi-monitor gossip / cross-monitor split-view comparison UI** — deferred to a later milestone.
  (v1 split-view detection is the monitor catching a hub against *itself*; comparing two *monitors* is future.)
- **Resolution/projection views** — e.g. "who currently owns this id" or "current gateway URL". The data model
  enables them but v1 does not build them.
- **Account/login/settings, write actions, configuration UIs.** v1 surfaces are read-only and public. No auth.
- **Operator deployment/config screens** (it's a single binary + container; configured outside the browser).
- **Authoritative Bitcoin verification in-page** — the page shows calendar-asserted anchor status and lets the user
  download the `.ots`; the authoritative check is the user running `ots verify` themselves.

---

## 9. Deliverable: three directions along three different axes

Produce **three alternative directions**, each anchored on a **different organizing axis**, so we compare genuinely
different structural bets — not three palettes of the same layout. **All three must satisfy every invariant in §5**;
they differ in *organizing principle and emphasis*, not in honesty.

### Direction 1 — Axis: **mental model = "operational status board"**
Treat it like a familiar uptime/status page. Optimize for the non-technical business user's *glance*: the whole
realm's health visible at once, per-hub status front and center, depth a click away. Bet: familiarity and low
cognitive load win adoption.

### Direction 2 — Axis: **primary persona & depth = "evidence ledger / forensic"**
Optimize for the auditor/litigator and the declarer who needs *durable proof of a specific claim*. Foreground the
single-claim journey: search an id → see its evidence → download the proof bundle → see the Bitcoin timestamp.
Document-like, citation-friendly, serious. Bet: the product's real value is per-claim evidence, so lead with it.

### Direction 3 — Axis: **interaction paradigm = "verify-it-yourself first"**
Make the **in-browser re-verification the protagonist**, not a footnote. The hero interaction is "the monitor
claims X → press to have *your own browser* confirm X," with the local computation made tangible and the
two-trust-tier distinction (§2) as the central visual motif. Bet: the differentiator is *don't-trust-us*, so the UI
should dramatize the moment the user stops trusting and starts verifying.

For each direction provide: the dashboard overview, hub detail (incl. a **frozen** hub), the inclusion/proof result,
and the verify-in-browser moment — enough to judge the trust-communication approach. Then give a short **recommendation**
with rationale and any hybrid you'd actually ship.

---

## 10. Reference vocabulary (read `CONTEXT.md` for the full list)

`hub` · `monitor instance` · `verifier app` · `coverage` / `monitored_since` · `hub status` (verified /
unresolvable / unverified / frozen / inactive) · `self-consistency violation` · `split view` · `verifiable cache`
(not "trusted oracle") · `proof bundle` · `Bitcoin anchoring` vs `comparison anchor` · `mirror` · `declaration` vs
`deletion` vs `redaction` · `checkpoint` · `inclusion proof`. **Use these exact terms; the "_Avoid_" notes in
`CONTEXT.md` are binding.**
