// Tests for the pure config loader: a minimal {DB, realm} input loads with the
// documented default intervals, a full input round-trips every field, and the
// validations fail closed — a missing required path, an unparseable interval, a
// non-positive interval, and a Frozen < Normal cross-check each return a non-nil
// wrapped error naming the offending key and the zero Config.
package config

import (
	"strings"
	"testing"
	"time"
)

// fromMap builds the injected get closure from a map, mirroring the shape of the
// os.LookupEnv-backed closure the binary passes: a present key returns (value,
// true); an absent key returns ("", false).
func fromMap(m map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		v, ok := m[key]
		return v, ok
	}
}

func TestLoadGolden(t *testing.T) {
	// A full input round-trips every field verbatim.
	get := fromMap(map[string]string{
		keyDB:     "/var/lib/iscc-monitor/testnet.db",
		keyRealm:  "/etc/iscc-monitor/realm.txt",
		keyNormal: "30s",
		keyFrozen: "10m",
	})
	got, err := Load(get)
	if err != nil {
		t.Fatalf("Load(full) error: %v", err)
	}
	want := Config{
		DBPath:    "/var/lib/iscc-monitor/testnet.db",
		RealmPath: "/etc/iscc-monitor/realm.txt",
		Normal:    30 * time.Second,
		Frozen:    10 * time.Minute,
	}
	if got != want {
		t.Errorf("Load(full) = %#v, want %#v", got, want)
	}
}

func TestLoadDefaults(t *testing.T) {
	// A minimal input with only the two required paths loads with the documented
	// default intervals (Normal=5m, Frozen=1h).
	get := fromMap(map[string]string{
		keyDB:    "/data/mainnet.db",
		keyRealm: "/data/realm.txt",
	})
	got, err := Load(get)
	if err != nil {
		t.Fatalf("Load(minimal) error: %v", err)
	}
	want := Config{
		DBPath:    "/data/mainnet.db",
		RealmPath: "/data/realm.txt",
		Normal:    defaultNormal,
		Frozen:    defaultFrozen,
	}
	if got != want {
		t.Errorf("Load(minimal) = %#v, want %#v", got, want)
	}
	if !(want.Frozen >= want.Normal) {
		t.Fatalf("default intervals violate Frozen >= Normal: Normal=%s Frozen=%s", want.Normal, want.Frozen)
	}
}

func TestLoad(t *testing.T) {
	cases := []struct {
		name string
		in   map[string]string
		want Config
	}{
		{
			name: "only normal overridden, frozen defaults",
			in:   map[string]string{keyDB: "/db", keyRealm: "/realm", keyNormal: "1m"},
			want: Config{DBPath: "/db", RealmPath: "/realm", Normal: time.Minute, Frozen: defaultFrozen},
		},
		{
			name: "only frozen overridden, normal defaults",
			in:   map[string]string{keyDB: "/db", keyRealm: "/realm", keyFrozen: "2h"},
			want: Config{DBPath: "/db", RealmPath: "/realm", Normal: defaultNormal, Frozen: 2 * time.Hour},
		},
		{
			name: "frozen equal to normal is accepted",
			in:   map[string]string{keyDB: "/db", keyRealm: "/realm", keyNormal: "15m", keyFrozen: "15m"},
			want: Config{DBPath: "/db", RealmPath: "/realm", Normal: 15 * time.Minute, Frozen: 15 * time.Minute},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Load(fromMap(tc.in))
			if err != nil {
				t.Fatalf("Load(%v) error: %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("Load(%v) = %#v, want %#v", tc.in, got, tc.want)
			}
		})
	}
}

func TestLoadErrors(t *testing.T) {
	cases := []struct {
		name    string
		in      map[string]string
		wantKey string
	}{
		{
			name:    "missing db path",
			in:      map[string]string{keyRealm: "/realm"},
			wantKey: keyDB,
		},
		{
			name:    "empty db path",
			in:      map[string]string{keyDB: "", keyRealm: "/realm"},
			wantKey: keyDB,
		},
		{
			name:    "missing realm path",
			in:      map[string]string{keyDB: "/db"},
			wantKey: keyRealm,
		},
		{
			name:    "empty realm path",
			in:      map[string]string{keyDB: "/db", keyRealm: ""},
			wantKey: keyRealm,
		},
		{
			name:    "unparseable normal interval",
			in:      map[string]string{keyDB: "/db", keyRealm: "/realm", keyNormal: "soon"},
			wantKey: keyNormal,
		},
		{
			name:    "unparseable frozen interval",
			in:      map[string]string{keyDB: "/db", keyRealm: "/realm", keyFrozen: "later"},
			wantKey: keyFrozen,
		},
		{
			name:    "non-positive normal interval",
			in:      map[string]string{keyDB: "/db", keyRealm: "/realm", keyNormal: "0s"},
			wantKey: keyNormal,
		},
		{
			name:    "negative frozen interval",
			in:      map[string]string{keyDB: "/db", keyRealm: "/realm", keyFrozen: "-1m"},
			wantKey: keyFrozen,
		},
		{
			name:    "frozen shorter than normal",
			in:      map[string]string{keyDB: "/db", keyRealm: "/realm", keyNormal: "10m", keyFrozen: "1m"},
			wantKey: keyFrozen,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Load(fromMap(tc.in))
			if err == nil {
				t.Fatalf("Load(%v) = %#v, want error", tc.in, got)
			}
			if got != (Config{}) {
				t.Errorf("Load(%v) returned %#v alongside error, want zero Config", tc.in, got)
			}
			if !strings.Contains(err.Error(), tc.wantKey) {
				t.Errorf("Load(%v) error %q does not name the offending key %q", tc.in, err, tc.wantKey)
			}
		})
	}
}
