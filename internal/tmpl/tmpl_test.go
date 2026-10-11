package tmpl

import (
	"os/exec"
	"testing"
)

func TestResolveInsideQuotes(t *testing.T) {
	evil := `x'"; echo PWNED; '"`
	for _, tmpl := range []string{"printf %s {{1}}", "printf %s '{{1}}'", `printf %s "{{1}}"`, `printf %s "a'{{1}}"`} {
		line, err := Resolve(tmpl, Input{Args: []string{evil}})
		if err != nil {
			t.Fatal(err)
		}
		out, err := exec.Command("sh", "-c", line).Output()
		if err != nil {
			t.Fatalf("%s: %v", line, err)
		}
		want := evil
		if tmpl == `printf %s "a'{{1}}"` {
			want = "a'" + evil
		}
		if string(out) != want {
			t.Errorf("%s -> %s: got %q, want %q", tmpl, line, out, want)
		}
	}
}
