package shell

import (
	"os/exec"
	"testing"
)

func TestQuoteRoundTrip(t *testing.T) {
	inputs := []string{"plain", "", "with space", "it's", `back\slash`, `"dq"`, "$HOME",
		"a;rm -rf ~", "ünï", "new\nline", "*", "{{1}}", "=cmd", "%self"}
	for _, sh := range []string{"sh", "bash", "zsh", "fish"} {
		bin, err := exec.LookPath(sh)
		if err != nil {
			t.Logf("skip %s: not installed", sh)
			continue
		}
		for _, in := range inputs {
			out, err := exec.Command(bin, "-c", "printf '%s' "+Quote(in, sh)).Output()
			if err != nil {
				t.Fatalf("%s %q: %v", sh, in, err)
			}
			if string(out) != in {
				t.Errorf("%s: Quote(%q) round-tripped to %q", sh, in, out)
			}
		}
	}
}

func TestQuoteLeavesSafeWordsBare(t *testing.T) {
	for _, s := range []string{"feat-x", "main", "a/b.c", "v1.2:3", "-a"} {
		if got := Quote(s, "sh"); got != s {
			t.Errorf("Quote(%q) = %q, want bare", s, got)
		}
	}
}
