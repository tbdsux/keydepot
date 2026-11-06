// Package dotenv reads and writes .env files.
package dotenv

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Var is one NAME=value line.
type Var struct {
	Name, Value string
}

// Parse reads every variable in order, keeping repeated names, so importing
// loses nothing. Quoted values may span lines.
func Parse(text string) ([]Var, error) {
	text = strings.TrimPrefix(strings.ReplaceAll(text, "\r\n", "\n"), "\uFEFF")
	lines := strings.Split(text, "\n")

	var vars []Var

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		start := i + 1
		line = strings.TrimPrefix(line, "export ")

		name, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("Line %d has no “=”.", start)
		}

		name = strings.TrimSpace(name)
		if !valid(name) {
			return nil, fmt.Errorf("Line %d: “%s” isn't a valid variable name.", start, name)
		}

		value = strings.TrimSpace(value)

		if value == "" || value[0] != '"' && value[0] != '\'' {
			// An unquoted value ends where a comment starts.
			if j := strings.Index(" "+value, " #"); j >= 0 {
				value = strings.TrimSpace(value[:j])
			}

			vars = append(vars, Var{Name: name, Value: value})

			continue
		}

		quote, rest := value[0], value[1:]

		for {
			v, after, closed := unquote(rest, quote)
			if closed {
				after = strings.TrimSpace(after)
				if after != "" && after[0] != '#' {
					return nil, fmt.Errorf("Line %d has text after the closing quote.", i+1)
				}

				vars = append(vars, Var{Name: name, Value: v})

				break
			}

			if i+1 >= len(lines) {
				return nil, fmt.Errorf("Line %d opens a quote that never closes.", start)
			}

			i++
			rest += "\n" + lines[i]
		}
	}

	return vars, nil
}

// unquote reads up to the closing quote. Single quotes are literal; double
// quotes take the escapes Format writes.
func unquote(s string, quote byte) (value, after string, closed bool) {
	if quote == '\'' {
		v, after, closed := strings.Cut(s, "'")
		return v, after, closed
	}

	var b strings.Builder

	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '"':
			return b.String(), s[i+1:], true

		case s[i] == '\\' && i+1 < len(s):
			i++

			switch s[i] {
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case '"', '\\':
				b.WriteByte(s[i])
			default:
				b.WriteByte('\\')
				b.WriteByte(s[i])
			}

		default:
			b.WriteByte(s[i])
		}
	}

	return "", "", false
}

func valid(name string) bool {
	if name == "" || name[0] >= '0' && name[0] <= '9' {
		return false
	}

	for _, r := range name {
		if !(r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return false
		}
	}

	return true
}

// Name turns a key's name into a variable name: one that already is one is
// kept, others become upper snake case, so "Deploy key" is DEPLOY_KEY.
func Name(s string) string {
	s = strings.TrimSpace(s)
	if valid(s) {
		return s
	}

	var b strings.Builder

	for _, r := range strings.ToUpper(s) {
		if r == '_' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else if b.Len() > 0 && !strings.HasSuffix(b.String(), "_") {
			b.WriteByte('_')
		}
	}

	name := strings.TrimSuffix(b.String(), "_")
	if name == "" || name[0] >= '0' && name[0] <= '9' {
		name = "_" + name
	}

	return name
}

// Format writes one line per variable, quoting values so Parse, and the
// common .env loaders, read them back unchanged.
func Format(vars []Var) string {
	var b strings.Builder

	for _, v := range vars {
		b.WriteString(Name(v.Name) + "=" + quote(v.Value) + "\n")
	}

	return b.String()
}

func quote(value string) string {
	plain := value != ""

	for _, r := range value {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("_-./:@+,=%", r)) {
			plain = false
			break
		}
	}

	switch {
	case plain:
		return value

	case !strings.ContainsAny(value, "'\n\r"):
		return "'" + value + "'"
	}

	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\r", `\r`, "\t", `\t`)

	return `"` + r.Replace(value) + `"`
}

// maxFileSize matches the largest key the store keeps.
const maxFileSize = 1 << 20

// ReadFiles joins the files' text, each under a comment naming its path, so
// one Parse reads them all.
func ReadFiles(paths []string) (string, error) {
	var b strings.Builder

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return "", err
		}

		if info.Size() > maxFileSize {
			return "", fmt.Errorf("“%s” is over 1 MiB, which is too big for a .env file.", filepath.Base(path))
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}

		b.WriteString("# " + path + "\n" + strings.TrimRight(string(data), "\n") + "\n")
	}

	return b.String(), nil
}
