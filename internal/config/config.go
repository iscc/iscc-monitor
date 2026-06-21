// Package config turns a flat key/value lookup into the validated, typed startup
// values the iscc-monitor binary consumes: the SQLite database path, the
// realm-document path, and the two poll intervals the follower loop runs on.
//
// It is a pure leaf: it parses and validates values already in hand and performs
// no I/O. The raw lookup is injected as a get(key) closure (the binary backs it
// with os.LookupEnv next step), so the package never touches the environment,
// the filesystem, or the network and is fully unit-testable. Reading the realm
// document and opening the database belong to the binary; config only carries the
// realm-document path as a string and the database path as a string.
//
// Configuration keys (all read through get):
//
//	ISCC_MONITOR_DB     required — path to the network's SQLite file (ADR-0007:
//	                    one database file per network, so this is a single path).
//	ISCC_MONITOR_REALM  required — path to the realm-membership document on disk.
//	ISCC_MONITOR_NORMAL optional — clean-hub poll interval (default 5m).
//	ISCC_MONITOR_FROZEN optional — frozen-hub backed-off poll interval, the
//	                    evidence-only re-poll cadence (ADR-0006); default 1h.
//	ISCC_MONITOR_ADDR   optional — listen address for the /metrics HTTP server;
//	                    default :9464.
//
// Intervals are parsed with time.ParseDuration. The load-bearing cross-check is
// Frozen >= Normal: the follower loop encodes its back-off by polling a frozen
// hub on the longer Frozen interval (loop.go due()), so a Frozen < Normal config
// is rejected rather than silently accepted.
package config

import (
	"fmt"
	"time"
)

// Configuration key names read through the injected get closure.
const (
	keyDB     = "ISCC_MONITOR_DB"
	keyRealm  = "ISCC_MONITOR_REALM"
	keyNormal = "ISCC_MONITOR_NORMAL"
	keyFrozen = "ISCC_MONITOR_FROZEN"
	keyAddr   = "ISCC_MONITOR_ADDR"
)

// Default poll intervals applied when the corresponding key is absent. Normal is
// the clean-hub cadence; Frozen is the longer evidence-only re-poll cadence for a
// frozen hub (ADR-0006). The defaults satisfy Frozen >= Normal.
const (
	defaultNormal = 5 * time.Minute
	defaultFrozen = time.Hour
)

// defaultAddr is the listen address for the /metrics HTTP server when
// ISCC_MONITOR_ADDR is absent: an exotic non-standard port per the project port
// convention, fixed so the same deployment reuses it across restarts.
const defaultAddr = ":9464"

// Config is the validated, typed startup configuration for the monitor binary.
// DBPath is the single network's SQLite file (ADR-0007), RealmPath the on-disk
// realm-membership document, Normal/Frozen the follower loop's poll intervals
// (Frozen >= Normal encodes the freeze back-off, ADR-0006), and Addr the listen
// address for the /metrics HTTP server (default :9464).
type Config struct {
	DBPath    string
	RealmPath string
	Normal    time.Duration
	Frozen    time.Duration
	Addr      string
}

// Load reads the configuration from the injected get closure, applies the
// documented interval defaults, validates, and returns a typed Config.
//
// get(key) returns a value and whether the key was present (the closure the
// binary builds from os.LookupEnv has this shape). DBPath and RealmPath are
// required: an absent or empty value yields a wrapped error naming the missing
// key. The intervals default to Normal=5m and Frozen=1h when absent, are parsed
// with time.ParseDuration (an unparseable value is wrapped and named, never
// swallowed), and must each be positive. The cross-check Frozen >= Normal is
// enforced so the loop's freeze back-off (loop.go due()) actually backs a frozen
// hub off; a Frozen < Normal config is rejected.
//
// It fails closed: on any validation failure it returns the zero Config alongside
// a wrapped error so the binary surfaces a precise startup failure rather than a
// half-built config.
func Load(get func(key string) (string, bool)) (Config, error) {
	dbPath, err := required(get, keyDB)
	if err != nil {
		return Config{}, err
	}
	realmPath, err := required(get, keyRealm)
	if err != nil {
		return Config{}, err
	}
	normal, err := duration(get, keyNormal, defaultNormal)
	if err != nil {
		return Config{}, err
	}
	frozen, err := duration(get, keyFrozen, defaultFrozen)
	if err != nil {
		return Config{}, err
	}
	if frozen < normal {
		return Config{}, fmt.Errorf("config: %q (%s) must be >= %q (%s): freeze cadence backs off, never speeds up", keyFrozen, frozen, keyNormal, normal)
	}
	addr := optional(get, keyAddr, defaultAddr)
	return Config{DBPath: dbPath, RealmPath: realmPath, Normal: normal, Frozen: frozen, Addr: addr}, nil
}

// required returns the value for key, failing closed with a wrapped error naming
// the key when it is absent or empty (whitespace is not trimmed — a path is taken
// verbatim).
func required(get func(key string) (string, bool), key string) (string, error) {
	value, ok := get(key)
	if !ok || value == "" {
		return "", fmt.Errorf("config: required key %q is missing", key)
	}
	return value, nil
}

// optional returns the value for key, falling back to fallback when the key is
// absent or empty. A bad value is not validated here (the listen address surfaces
// at ListenAndServe), matching the optional-key pattern's no-validation-beyond-
// default contract.
func optional(get func(key string) (string, bool), key, fallback string) string {
	value, ok := get(key)
	if !ok || value == "" {
		return fallback
	}
	return value
}

// duration returns the interval for key, falling back to fallback when the key is
// absent. A present value is parsed with time.ParseDuration (an error is wrapped
// and names the key) and must be positive; a non-positive interval is rejected so
// the loop never polls on a zero or negative cadence.
func duration(get func(key string) (string, bool), key string, fallback time.Duration) (time.Duration, error) {
	value, ok := get(key)
	if !ok || value == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("config: key %q: %w", key, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("config: key %q (%s) must be a positive interval", key, d)
	}
	return d, nil
}
