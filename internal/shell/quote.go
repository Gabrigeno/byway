// Package shell holds everything that depends on the user's shell:
// quoting, integration scripts and rc-file setup.
package shell

import (
	"regexp"
	"strings"
)

// safe words need no quoting in any supported shell ('=' and '%' excluded:
// zsh expands a leading '=' and fish historically expanded '%').
var safe = regexp.MustCompile(`^[A-Za-z0-9_./:@+,-]+$`)

// Quote returns s as a single literal word for shell sh.
func Quote(s, sh string) string {
	if safe.MatchString(s) {
		return s
	}
	if sh == "fish" {
		r := strings.NewReplacer(`\`, `\\`, `'`, `\'`)
		return "'" + r.Replace(s) + "'"
	}
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}
