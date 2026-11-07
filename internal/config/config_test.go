// Tests for the pure config loader: a minimal {DB, realm} input loads with the
// documented default intervals and empty identity strings, a full input
// round-trips every field (including the three optional masthead-identity
// strings), a partially-set identity proves per-field independence, and the
// validations fail closed — a missing required path, an unparseable interval, a
// non-positive interval, a Frozen < Normal cross-check, and each malformed co-branding
// value each return a non-nil wrapped error naming the offending key and the zero
// Config. The co-branding keys round-trip, and footer links parse leniently around
// whitespace and empty pairs.
package config

import (
	"reflect"
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
		keyAddr:   "127.0.0.1:41234",

		keyOTSInterval: "30m",
		keyInstance:    "monitor.iscc.id",
		keyOperator:    "ISCC Foundation",
		keyRealmName:   "ISCC mainnet",

		keyOperatorName: "Example GmbH",
		keyOperatorURL:  "https://example.com/",
		keyOperatorLogo: "/etc/iscc-monitor/operator.svg",
		keyFooterLinks:  "Imprint=https://example.com/imprint|Privacy=https://example.com/privacy",
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
		Addr:      "127.0.0.1:41234",

		OTSInterval: 30 * time.Minute,
		Instance:    "monitor.iscc.id",
		Operator:    "ISCC Foundation",
		RealmName:   "ISCC mainnet",

		OperatorName: "Example GmbH",
		OperatorURL:  "https://example.com/",
		OperatorLogo: "/etc/iscc-monitor/operator.svg",
		FooterLinks: []Link{
			{Label: "Imprint", URL: "https://example.com/imprint"},
			{Label: "Privacy", URL: "https://example.com/privacy"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Load(full) = %#v, want %#v", got, want)
	}
}

func TestLoadDefaults(t *testing.T) {
	// A minimal input with only the two required paths loads with the documented
	// default intervals (Normal=5m, Frozen=1h, OTSInterval=1h).
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
		Addr:      defaultAddr,

		OTSInterval: defaultOTSInterval,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Load(minimal) = %#v, want %#v", got, want)
	}
	if got.OTSInterval != time.Hour {
		t.Errorf("Load(minimal) OTSInterval = %s, want default 1h", got.OTSInterval)
	}
	if got.Addr != ":9464" {
		t.Errorf("Load(minimal) Addr = %q, want default %q", got.Addr, ":9464")
	}
	if got.Instance != "" || got.Operator != "" || got.RealmName != "" {
		t.Errorf("Load(minimal) identity = {Instance:%q Operator:%q RealmName:%q}, want all empty when keys absent",
			got.Instance, got.Operator, got.RealmName)
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
			want: Config{DBPath: "/db", RealmPath: "/realm", Normal: time.Minute, Frozen: defaultFrozen, OTSInterval: defaultOTSInterval, Addr: defaultAddr},
		},
		{
			name: "only frozen overridden, normal defaults",
			in:   map[string]string{keyDB: "/db", keyRealm: "/realm", keyFrozen: "2h"},
			want: Config{DBPath: "/db", RealmPath: "/realm", Normal: defaultNormal, Frozen: 2 * time.Hour, OTSInterval: defaultOTSInterval, Addr: defaultAddr},
		},
		{
			name: "frozen equal to normal is accepted",
			in:   map[string]string{keyDB: "/db", keyRealm: "/realm", keyNormal: "15m", keyFrozen: "15m"},
			want: Config{DBPath: "/db", RealmPath: "/realm", Normal: 15 * time.Minute, Frozen: 15 * time.Minute, OTSInterval: defaultOTSInterval, Addr: defaultAddr},
		},
		{
			name: "addr overridden, intervals default",
			in:   map[string]string{keyDB: "/db", keyRealm: "/realm", keyAddr: ":40080"},
			want: Config{DBPath: "/db", RealmPath: "/realm", Normal: defaultNormal, Frozen: defaultFrozen, OTSInterval: defaultOTSInterval, Addr: ":40080"},
		},
		{
			name: "empty addr falls back to default",
			in:   map[string]string{keyDB: "/db", keyRealm: "/realm", keyAddr: ""},
			want: Config{DBPath: "/db", RealmPath: "/realm", Normal: defaultNormal, Frozen: defaultFrozen, OTSInterval: defaultOTSInterval, Addr: defaultAddr},
		},
		{
			name: "only instance identity set, operator and realm name stay empty",
			in:   map[string]string{keyDB: "/db", keyRealm: "/realm", keyInstance: "monitor.example"},
			want: Config{DBPath: "/db", RealmPath: "/realm", Normal: defaultNormal, Frozen: defaultFrozen, OTSInterval: defaultOTSInterval, Addr: defaultAddr, Instance: "monitor.example"},
		},
		{
			name: "only realm name set, distinct from realm path",
			in:   map[string]string{keyDB: "/db", keyRealm: "/realm", keyRealmName: "ISCC mainnet"},
			want: Config{DBPath: "/db", RealmPath: "/realm", Normal: defaultNormal, Frozen: defaultFrozen, OTSInterval: defaultOTSInterval, Addr: defaultAddr, RealmName: "ISCC mainnet"},
		},
		{
			name: "ots interval overridden",
			in:   map[string]string{keyDB: "/db", keyRealm: "/realm", keyOTSInterval: "15m"},
			want: Config{DBPath: "/db", RealmPath: "/realm", Normal: defaultNormal, Frozen: defaultFrozen, OTSInterval: 15 * time.Minute, Addr: defaultAddr},
		},
		{
			name: "operator name alone needs no url or logo",
			in:   map[string]string{keyDB: "/db", keyRealm: "/realm", keyOperatorName: "Example GmbH"},
			want: Config{DBPath: "/db", RealmPath: "/realm", Normal: defaultNormal, Frozen: defaultFrozen, OTSInterval: defaultOTSInterval, Addr: defaultAddr, OperatorName: "Example GmbH"},
		},
		{
			name: "footer links trim whitespace, skip empty pairs, accept mailto, keep = in a query",
			in: map[string]string{keyDB: "/db", keyRealm: "/realm",
				keyFooterLinks: " Contact = mailto:info@example.com | | Terms=https://example.com/t?lang=en |"},
			want: Config{DBPath: "/db", RealmPath: "/realm", Normal: defaultNormal, Frozen: defaultFrozen, OTSInterval: defaultOTSInterval, Addr: defaultAddr,
				FooterLinks: []Link{
					{Label: "Contact", URL: "mailto:info@example.com"},
					{Label: "Terms", URL: "https://example.com/t?lang=en"},
				}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Load(fromMap(tc.in))
			if err != nil {
				t.Fatalf("Load(%v) error: %v", tc.in, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
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
			name:    "unparseable ots interval",
			in:      map[string]string{keyDB: "/db", keyRealm: "/realm", keyOTSInterval: "hourly"},
			wantKey: keyOTSInterval,
		},
		{
			name:    "non-positive ots interval",
			in:      map[string]string{keyDB: "/db", keyRealm: "/realm", keyOTSInterval: "0s"},
			wantKey: keyOTSInterval,
		},
		{
			name:    "frozen shorter than normal",
			in:      map[string]string{keyDB: "/db", keyRealm: "/realm", keyNormal: "10m", keyFrozen: "1m"},
			wantKey: keyFrozen,
		},
		{
			name:    "operator url is not absolute",
			in:      map[string]string{keyDB: "/db", keyRealm: "/realm", keyOperatorName: "X", keyOperatorURL: "example.com"},
			wantKey: keyOperatorURL,
		},
		{
			name:    "operator url uses a script scheme",
			in:      map[string]string{keyDB: "/db", keyRealm: "/realm", keyOperatorName: "X", keyOperatorURL: "javascript:alert(1)"},
			wantKey: keyOperatorURL,
		},
		{
			name:    "operator url rejects mailto",
			in:      map[string]string{keyDB: "/db", keyRealm: "/realm", keyOperatorName: "X", keyOperatorURL: "mailto:a@example.com"},
			wantKey: keyOperatorURL,
		},
		{
			name:    "operator url without a name",
			in:      map[string]string{keyDB: "/db", keyRealm: "/realm", keyOperatorURL: "https://example.com"},
			wantKey: keyOperatorName,
		},
		{
			name:    "operator logo without a name",
			in:      map[string]string{keyDB: "/db", keyRealm: "/realm", keyOperatorLogo: "/logo.svg"},
			wantKey: keyOperatorName,
		},
		{
			name:    "footer link without =",
			in:      map[string]string{keyDB: "/db", keyRealm: "/realm", keyFooterLinks: "https://example.com/imprint"},
			wantKey: keyFooterLinks,
		},
		{
			name:    "footer link without a label",
			in:      map[string]string{keyDB: "/db", keyRealm: "/realm", keyFooterLinks: "=https://example.com/imprint"},
			wantKey: keyFooterLinks,
		},
		{
			name:    "footer link with a script scheme",
			in:      map[string]string{keyDB: "/db", keyRealm: "/realm", keyFooterLinks: "Imprint=https://example.com|Evil=javascript:alert(1)"},
			wantKey: keyFooterLinks,
		},
		{
			name:    "footer link with an empty mailto",
			in:      map[string]string{keyDB: "/db", keyRealm: "/realm", keyFooterLinks: "Contact=mailto:"},
			wantKey: keyFooterLinks,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Load(fromMap(tc.in))
			if err == nil {
				t.Fatalf("Load(%v) = %#v, want error", tc.in, got)
			}
			if !reflect.DeepEqual(got, Config{}) {
				t.Errorf("Load(%v) returned %#v alongside error, want zero Config", tc.in, got)
			}
			if !strings.Contains(err.Error(), tc.wantKey) {
				t.Errorf("Load(%v) error %q does not name the offending key %q", tc.in, err, tc.wantKey)
			}
		})
	}
}
