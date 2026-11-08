-- schema.sql — core M1 data model for one network's SQLite database.
--
-- This DDL is embedded into internal/store and applied verbatim on every Open.
-- Every statement is idempotent (CREATE … IF NOT EXISTS) so opening an existing
-- database is a no-op and restart-survival is trivially true. There is NO
-- `network` column anywhere: each network lives in its own file (mainnet.db /
-- testnet.db) per ADR-0007, so the file IS the network namespace.
--
-- Time convention: all timestamps are stored as INTEGER unix-seconds (the
-- simplest form for `observed_at`-style ordering and comparisons). The one
-- exception is iscc_index.note_timestamp — the record's own RFC-3339 string
-- (note.timestamp), stored VERBATIM as TEXT and never parsed (ADR-0008: index the
-- raw value, never interpret). Booleans are INTEGER 0/1. Tree roots, raw
-- record/checkpoint bytes, ISCC-IDs, and OTS proof bytes are BLOB. The column lists
-- match the build plan's "SQLite schema (core tables)" block exactly.

-- hubs — the followed logs and the monitor's verdict on each (status glossary in
-- CLAUDE.md). monitored_since_{size,time} record the coverage start (ADR-0001).
CREATE TABLE IF NOT EXISTS hubs (
    hub_id                INTEGER PRIMARY KEY,
    domain                TEXT NOT NULL,
    origin                TEXT NOT NULL,
    base_url              TEXT NOT NULL,
    active                INTEGER NOT NULL DEFAULT 1,
    status                TEXT,
    monitored_since_size  INTEGER,
    monitored_since_time  INTEGER,
    first_seen            INTEGER,
    last_seen             INTEGER
);

-- hub_keys — the did:web key cache (ADR-0009). The DID document is the source of
-- truth; these rows are a resolved cache. pubkey_raw is the 32-byte Ed25519 key,
-- pubkey_z its z6Mk… multibase form, key_id the BE-uint32 signed-note keyhash.
CREATE TABLE IF NOT EXISTS hub_keys (
    hub_id      INTEGER NOT NULL REFERENCES hubs(hub_id),
    key_id      INTEGER NOT NULL,
    pubkey_raw  BLOB NOT NULL,
    pubkey_z    TEXT,
    revoked_at  INTEGER,
    resolved_at INTEGER
);

-- checkpoints — every observed hub-signed checkpoint (irreplaceable evidence).
-- root is the RFC-6962 SHA-256 tree head, raw the full signed-note bytes.
-- consistent/root_rebuilt record the follower's verification verdict. The UNIQUE
-- key dedupes a re-observed (size, root) while still recording the first sighting.
CREATE TABLE IF NOT EXISTS checkpoints (
    id           INTEGER PRIMARY KEY,
    hub_id       INTEGER NOT NULL REFERENCES hubs(hub_id),
    tree_size    INTEGER NOT NULL,
    root         BLOB NOT NULL,
    raw          BLOB NOT NULL,
    observed_at  INTEGER,
    consistent   INTEGER,
    root_rebuilt INTEGER,
    UNIQUE(hub_id, tree_size, root)
);

-- violations — self-consistency violations (ADR-0006). kind ∈ {fork,shrink,
-- equivocation}; raw_a/raw_b are the two contradictory checkpoint bytes and
-- proof_json the supporting consistency proof. Irreplaceable evidence.
CREATE TABLE IF NOT EXISTS violations (
    id          INTEGER PRIMARY KEY,
    hub_id      INTEGER NOT NULL REFERENCES hubs(hub_id),
    kind        TEXT NOT NULL,
    detected_at INTEGER,
    raw_a       BLOB,
    raw_b       BLOB,
    proof_json  TEXT
);

-- tiles — the mirrored hash tiles as BLOBs (ADR-0005), keyed by (level, index,
-- width). is_full marks an immutable width==256 tile; partials (.p/<W>) are
-- re-fetched and overwritten every poll (ADR-0005 partial-tile discipline).
CREATE TABLE IF NOT EXISTS tiles (
    hub_id     INTEGER NOT NULL REFERENCES hubs(hub_id),
    level      INTEGER NOT NULL,
    tile_index INTEGER NOT NULL,
    width      INTEGER NOT NULL,
    data       BLOB NOT NULL,
    is_full    INTEGER NOT NULL DEFAULT 0,
    sha256     BLOB,
    updated_at INTEGER,
    PRIMARY KEY (hub_id, level, tile_index, width)
);

-- entry_bundles — the mirrored entry bundles as BLOBs (ADR-0005), keyed by
-- (bundle_index, width). Same partial-tile discipline as tiles.
CREATE TABLE IF NOT EXISTS entry_bundles (
    hub_id       INTEGER NOT NULL REFERENCES hubs(hub_id),
    bundle_index INTEGER NOT NULL,
    width        INTEGER NOT NULL,
    data         BLOB NOT NULL,
    is_full      INTEGER NOT NULL DEFAULT 0,
    sha256       BLOB,
    updated_at   INTEGER,
    PRIMARY KEY (hub_id, bundle_index, width)
);

-- iscc_index — the schema-agnostic record index (ADR-0008). The key is the
-- composite (hub_id, seq): seq is each hub's ABSOLUTE leaf index, so two hubs in a
-- multi-hub realm both index low leaves (both seq 0, 1, …) without colliding
-- (mirroring the composite PKs on tiles / entry_bundles). iscc_id → seq is
-- ONE-TO-MANY within a hub (declarations, deletions, future note types share an
-- id). note_schema stores the raw note.$schema string so unknown types are indexed
-- and proof-able without ever being interpreted. note_timestamp stores the raw,
-- optional note.timestamp RFC-3339 string verbatim (NULL when the record carries
-- none) — the one RFC-3339-TEXT exception to the unix-seconds time convention
-- above, never parsed.
CREATE TABLE IF NOT EXISTS iscc_index (
    hub_id         INTEGER NOT NULL REFERENCES hubs(hub_id),
    seq            INTEGER NOT NULL,
    iscc_id        BLOB,
    iscc_id_str    TEXT,
    note_schema    TEXT,
    note_timestamp TEXT,
    record_sha256  BLOB,
    PRIMARY KEY (hub_id, seq)
);

-- Lookups go iscc_id → []seq, so index the (non-unique) iscc_id column.
CREATE INDEX IF NOT EXISTS iscc_index_by_iscc_id ON iscc_index (iscc_id);

-- follow_state — the per-hub poll cursor and freeze flag, survived across
-- restart so a frozen hub stays frozen (ADR-0006, no auto-unfreeze) and polling
-- resumes from last_size. One row per hub.
CREATE TABLE IF NOT EXISTS follow_state (
    hub_id     INTEGER PRIMARY KEY REFERENCES hubs(hub_id),
    last_size  INTEGER,
    frozen     INTEGER NOT NULL DEFAULT 0,
    last_error TEXT
);

-- ots — OpenTimestamps proofs for distinct observed roots (ADR-0004), one per
-- (hub, tree_size, root). status tracks pending → Bitcoin-confirmed; ots_bytes
-- is the serialized proof. Irreplaceable evidence. No cosigs table in v1 (M7).
CREATE TABLE IF NOT EXISTS ots (
    id            INTEGER PRIMARY KEY,
    hub_id        INTEGER NOT NULL REFERENCES hubs(hub_id),
    tree_size     INTEGER NOT NULL,
    root          BLOB NOT NULL,
    status        TEXT,
    ots_bytes     BLOB,
    calendar_urls TEXT,
    stamped_at    INTEGER,
    upgraded_at   INTEGER,
    btc_height    INTEGER,
    attempts      INTEGER NOT NULL DEFAULT 0,
    next_retry    INTEGER,
    UNIQUE(hub_id, tree_size, root)
);
