package main

import (
	"strings"
	"testing"
)

func TestParseTOMLLockShape(t *testing.T) {
	got, err := parseTOML([]byte(`
# request vs pin
schema = "2" # lock format

[components.do-skills]
repo = "https://github.com/edheltzel/Do-Skills.git"
ref = "master"
commit = "78e2c15bfbc484d74a2d38ef3fa727a99272bf25"
tree = "69d5316f75fc267dc4bf21b526ff333df4388830"
`))
	if err != nil {
		t.Fatal(err)
	}
	if tomlString(got, "schema") != "2" {
		t.Fatalf("schema=%q", tomlString(got, "schema"))
	}
	c := tomlChild(tomlChild(got, "components"), "do-skills")
	if tomlString(c, "ref") != "master" || tomlString(c, "commit") != "78e2c15bfbc484d74a2d38ef3fa727a99272bf25" {
		t.Fatalf("pin fields: %+v", c)
	}
	if tomlString(c, "tree") != "69d5316f75fc267dc4bf21b526ff333df4388830" {
		t.Fatalf("tree=%q", tomlString(c, "tree"))
	}
}

func TestParseTOMLQuotedKeyAndEscapes(t *testing.T) {
	got, err := parseTOML([]byte(`"a-b" = "say \"hi\"\tnow"`))
	if err != nil {
		t.Fatal(err)
	}
	if tomlString(got, "a-b") != "say \"hi\"\tnow" {
		t.Fatalf("got %q", tomlString(got, "a-b"))
	}
}

func TestParseTOMLBadInput(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{`schema = 2`, "quoted string"},
		{`schema = "2`, "unterminated string"},
		{`[components.do-skills`, "unclosed table name"},
		{`= "x"`, "invalid key"},
		{`schema "2"`, "expected '='"},
		{`schema = "2" extra`, "trailing text"},
		{"[components]\n[components]\n", "duplicate table"},
		{"x = \"a\"\nx = \"b\"\n", "duplicate key"},
		{`x = "\q"`, "unknown escape"},
		{`x = []`, "quoted string"},
		{`x = true`, "quoted string"},
		{`[components.do-skills] leftover`, "trailing text"},
		{"x = \"a\"\n[x]\n", "not a table"},
	}
	for _, tc := range cases {
		_, err := parseTOML([]byte(tc.in))
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("parseTOML(%q) err=%v, want %q", tc.in, err, tc.want)
		}
	}
}
