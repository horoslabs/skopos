package scope

import (
	"strings"
	"testing"
)

func mustParse(t *testing.T, s string) *Scope {
	t.Helper()
	sc, err := Parse(strings.NewReader(s))
	if err != nil {
		t.Fatal(err)
	}
	return sc
}

func TestContains(t *testing.T) {
	sc := mustParse(t, "# comentario\n192.0.2.0/28\n198.51.100.7\nexample.com\n*.corp.example.org\n")
	cases := map[string]bool{
		"192.0.2.10":              true,
		"192.0.2.16":              false,
		"198.51.100.7":            true,
		"198.51.100.8":            false,
		"::ffff:192.0.2.1":        true,
		"example.com":             true,
		"EXAMPLE.com.":            true,
		"www.example.com":         false, // sin comodín, solo el dominio exacto
		"a.corp.example.org":      true,
		"corp.example.org":        false, // el comodín no incluye el apex
		"evilcorp.example.org":    false,
		"x.evil-corp.example.org": false,
		"":                        false,
	}
	for in, want := range cases {
		if got := sc.Contains(in); got != want {
			t.Errorf("Contains(%q) = %v, quería %v", in, got, want)
		}
	}
}

func TestParseErrors(t *testing.T) {
	for _, in := range []string{"", "# solo comentarios\n", "no es un dominio", "10.0.0.0/99", "*.com", "-bad.example.com"} {
		if _, err := Parse(strings.NewReader(in)); err == nil {
			t.Errorf("Parse(%q) debería fallar", in)
		}
	}
}
