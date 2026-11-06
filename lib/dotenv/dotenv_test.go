package dotenv

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	text := "\uFEFF# comment\r\n" +
		"export TOKEN=abc123 # trailing\n" +
		"EMPTY=\n" +
		"BLANK= # only a comment\n" +
		"HASH=a#b\n" +
		"SINGLE='lit\\n $x'\n" +
		"DOUBLE=\"a\\nb \\\"q\\\"\"\n" +
		"MULTI=\"-----BEGIN KEY-----\nabc\n-----END KEY-----\"\n" +
		"TOKEN=again\n"

	got, err := Parse(text)
	if err != nil {
		t.Fatal(err)
	}

	want := []Var{
		{Name: "TOKEN", Value: "abc123"},
		{Name: "EMPTY", Value: ""},
		{Name: "BLANK", Value: ""},
		{Name: "HASH", Value: "a#b"},
		{Name: "SINGLE", Value: `lit\n $x`},
		{Name: "DOUBLE", Value: "a\nb \"q\""},
		{Name: "MULTI", Value: "-----BEGIN KEY-----\nabc\n-----END KEY-----"},
		{Name: "TOKEN", Value: "again"},
	}

	if !slices.Equal(got, want) {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestParseErrors(t *testing.T) {
	for text, want := range map[string]string{
		"A=1\nnope":           "Line 2 has no",
		"1A=x":                "isn't a valid variable name",
		"A B=x":               "isn't a valid variable name",
		"A=\"open\nstill":     "Line 1 opens a quote",
		"A='x' y":             "text after the closing quote",
		"A=\"x\\\"\nB=2\n\"z": "text after the closing quote",
	} {
		if _, err := Parse(text); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: got %v, want %q", text, err, want)
		}
	}
}

func TestFormatRoundTrip(t *testing.T) {
	vars := []Var{
		{Name: "PLAIN", Value: "postgres://u@h:5432/db"},
		{Name: "SPACES", Value: "two words $HOME"},
		{Name: "QUOTE", Value: `it's "x" \n`},
		{Name: "KEY", Value: "-----BEGIN KEY-----\r\nabc\n-----END KEY-----\n"},
		{Name: "TAB", Value: "a\tb"},
	}

	got, err := Parse(Format(vars))
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(got, vars) {
		t.Errorf("got %q\nwant %q", got, vars)
	}
}

func TestName(t *testing.T) {
	for in, want := range map[string]string{
		"API_KEY":               "API_KEY",
		"lowerOK":               "lowerOK",
		"Production deploy key": "PRODUCTION_DEPLOY_KEY",
		" -GitHub (CI) token- ": "GITHUB_CI_TOKEN",
		"2fa seed":              "_2FA_SEED",
		"日本":                    "_",
	} {
		if got := Name(in); got != want {
			t.Errorf("Name(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestReadFiles(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.env"), filepath.Join(dir, ".env")
	os.WriteFile(a, []byte("A=1\n\n"), 0o600)
	os.WriteFile(b, []byte("B='2'"), 0o600)

	text, err := ReadFiles([]string{a, b})
	if err != nil {
		t.Fatal(err)
	}

	got, err := Parse(text)
	if err != nil || !slices.Equal(got, []Var{{Name: "A", Value: "1"}, {Name: "B", Value: "2"}}) {
		t.Errorf("got %q, %v from %q", got, err, text)
	}

	os.WriteFile(b, make([]byte, maxFileSize+1), 0o600)

	if _, err := ReadFiles([]string{b}); err == nil {
		t.Error("read a file over the limit")
	}
}
