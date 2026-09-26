// Package tmpl resolves {{…}} placeholders in a command template.
package tmpl

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/gabrielegenovesi/byway/internal/shell"
)

// Input is everything a template can draw values from.
type Input struct {
	Args  []string          // positional arguments
	Flags map[string]string // --k=v given on the command line
	Vars  map[string]string // [vars] from the registry
	Env   func(string) (string, bool)
	Shell string // quoting rules; "" means sh
}

// groups: 1 raw marker, 2 key, 3 ":default". Keys start with [A-Za-z0-9@]
// so Go/docker templates like {{.Name}} never match.
var ph = regexp.MustCompile(`\{\{(!?)([A-Za-z0-9@][A-Za-z0-9_.-]*)(:[^{}]*)?\}\}`)

// Resolve substitutes every placeholder in t. Positional args not consumed
// by {{N}} are appended (quoted) unless t contains {{@}}.
func Resolve(t string, in Input) (string, error) {
	if in.Env == nil {
		in.Env = os.LookupEnv
	}
	matches := ph.FindAllStringSubmatchIndex(t, -1)
	maxN, hasAll := 0, false
	for _, m := range matches {
		key := t[m[4]:m[5]]
		if key == "@" {
			hasAll = true
		} else if n, err := strconv.Atoi(key); err == nil && n > maxN {
			maxN = n
		}
	}
	q := func(s string, raw bool) string {
		if raw {
			return s
		}
		return shell.Quote(s, in.Shell)
	}
	rest := []string{}
	if len(in.Args) > maxN {
		rest = in.Args[maxN:]
	}

	var b strings.Builder
	last := 0
	for _, m := range matches {
		b.WriteString(t[last:m[0]])
		last = m[1]
		raw := m[3] > m[2]
		key := t[m[4]:m[5]]
		hasDef := m[6] != -1
		def := ""
		if hasDef {
			def = t[m[6]+1 : m[7]]
		}
		var v string
		switch n, numErr := strconv.Atoi(key); {
		case key == "@":
			parts := make([]string, len(rest))
			for i, a := range rest {
				parts[i] = q(a, raw)
			}
			b.WriteString(strings.Join(parts, " "))
			continue
		case numErr == nil:
			if n < 1 {
				return "", fmt.Errorf("invalid placeholder {{%s}}", key)
			}
			switch {
			case n <= len(in.Args):
				v = in.Args[n-1]
			case hasDef:
				v = def
			default:
				return "", fmt.Errorf("missing argument %d", n)
			}
		case strings.HasPrefix(key, "env."):
			var ok bool
			if v, ok = in.Env(key[4:]); !ok {
				if !hasDef {
					return "", fmt.Errorf("missing environment variable %s", key[4:])
				}
				v = def
			}
		default:
			var ok bool
			if v, ok = in.Flags[key]; !ok {
				if v, ok = in.Vars[key]; !ok {
					if !hasDef {
						return "", fmt.Errorf("missing variable %s (pass --%s=value)", key, key)
					}
					v = def
				}
			}
		}
		b.WriteString(q(v, raw))
	}
	b.WriteString(t[last:])
	if !hasAll {
		for _, a := range rest {
			b.WriteString(" " + q(a, false))
		}
	}
	return b.String(), nil
}

// Refs lists named variables referenced by t, first occurrence order.
func Refs(t string) []string {
	var out []string
	seen := map[string]bool{}
	for _, m := range ph.FindAllStringSubmatch(t, -1) {
		key := m[2]
		if _, err := strconv.Atoi(key); err == nil || key == "@" || strings.HasPrefix(key, "env.") || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}
	return out
}
