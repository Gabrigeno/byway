// Package capture turns a shell-history entry into a saved command.
package capture

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var selfRe = regexp.MustCompile(`^(command\s+)?byway\s+save(\s|$)`)

// PickLast returns the newest entry in raw (oldest first) that is not the
// "byway save" call itself.
// ponytail: multi-line history entries arrive split; the last line wins.
func PickLast(raw string) (string, error) {
	lines := strings.Split(raw, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		l := strings.TrimSpace(lines[i])
		if l != "" && !selfRe.MatchString(l) {
			return l, nil
		}
	}
	return "", errors.New("no previous command found in shell history")
}

var keyRe = regexp.MustCompile(`^([1-9][0-9]*|[a-z0-9_-]+)$`)

// Parametrize replaces whole-word occurrences of each value with {{KEY}}.
func Parametrize(cmd string, subs []string) (string, error) {
	for _, s := range subs {
		k, v, ok := strings.Cut(s, "=")
		if !ok || v == "" || !keyRe.MatchString(k) || k == "0" {
			return "", fmt.Errorf("--arg %q: want N=value or name=value", s)
		}
		qv := regexp.QuoteMeta(v)
		re := regexp.MustCompile(`(^|[\s=/:,('"])(?:"` + qv + `"|'` + qv + `'|` + qv + `)($|[\s/:,;&|).'"])`)
		if !re.MatchString(cmd) {
			return "", fmt.Errorf("--arg %s: %q not found as a whole word in: %s", k, v, cmd)
		}
		// repeat: adjacent matches share a boundary char and need a second pass
		for {
			next := re.ReplaceAllString(cmd, "${1}{{"+k+"}}${2}")
			if next == cmd {
				break
			}
			cmd = next
		}
	}
	return cmd, nil
}
