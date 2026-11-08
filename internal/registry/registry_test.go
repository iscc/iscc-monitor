// Tests for the realm-membership parser: the golden domains-only document maps
// to ordered Entry values (comments and blank lines dropped, input order kept),
// URL-shaped lines are rejected naming the bad line, and an empty/all-comment
// document yields no entries and no error.
package registry

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseGolden(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "realm.txt"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	got, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse(realm.txt) error: %v", err)
	}
	want := []Entry{
		{Domain: "sb0.iscc.id", BaseURL: "https://sb0.iscc.id"},
		{Domain: "sb1.amlet.id", BaseURL: "https://sb1.amlet.id"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Parse(realm.txt) = %#v, want %#v", got, want)
	}
}

func TestParse(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []Entry
	}{
		{
			name: "single domain",
			in:   "sb0.iscc.id\n",
			want: []Entry{{Domain: "sb0.iscc.id", BaseURL: "https://sb0.iscc.id"}},
		},
		{
			name: "comments and blanks dropped, order preserved",
			in:   "# header\nsb0.iscc.id\n\n  # indented comment\nsb1.amlet.id\n",
			want: []Entry{
				{Domain: "sb0.iscc.id", BaseURL: "https://sb0.iscc.id"},
				{Domain: "sb1.amlet.id", BaseURL: "https://sb1.amlet.id"},
			},
		},
		{
			name: "surrounding whitespace trimmed",
			in:   "   sb0.iscc.id   \n",
			want: []Entry{{Domain: "sb0.iscc.id", BaseURL: "https://sb0.iscc.id"}},
		},
		{
			name: "host with port allowed",
			in:   "localhost:8443\n",
			want: []Entry{{Domain: "localhost:8443", BaseURL: "https://localhost:8443"}},
		},
		{
			name: "no trailing newline",
			in:   "sb0.iscc.id",
			want: []Entry{{Domain: "sb0.iscc.id", BaseURL: "https://sb0.iscc.id"}},
		},
		{
			name: "all comments and blanks -> empty",
			in:   "# only comments\n\n   \n# and blanks\n",
			want: nil,
		},
		{
			name: "empty document -> empty",
			in:   "",
			want: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse([]byte(tc.in))
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tc.in, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Parse(%q) = %#v, want %#v", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		badLine string
	}{
		{name: "scheme", in: "https://sb0.iscc.id\n", badLine: "https://sb0.iscc.id"},
		{name: "path", in: "sb0.iscc.id/log\n", badLine: "sb0.iscc.id/log"},
		{name: "scheme on second line", in: "sb0.iscc.id\nhttps://sb1.amlet.id\n", badLine: "https://sb1.amlet.id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse([]byte(tc.in))
			if err == nil {
				t.Fatalf("Parse(%q) = %#v, want error", tc.in, got)
			}
			if got != nil {
				t.Errorf("Parse(%q) returned entries %#v alongside error, want nil", tc.in, got)
			}
			if !strings.Contains(err.Error(), tc.badLine) {
				t.Errorf("Parse(%q) error %q does not name the bad line %q", tc.in, err, tc.badLine)
			}
		})
	}
}
