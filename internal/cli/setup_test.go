package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOfferNeeded(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	zsh := map[string]string{"SHELL": "/bin/zsh"}
	if sh, ok := offerNeeded("byway", env(zsh), home, true); !ok || sh != "zsh" {
		t.Fatalf("want offer for zsh, got %q %v", sh, ok)
	}
	if _, ok := offerNeeded("byway", env(zsh), home, false); ok {
		t.Fatal("offered without tty")
	}
	if _, ok := offerNeeded("byway", env(map[string]string{"SHELL": "/bin/zsh", "BYWAY_INIT": "1"}), home, true); ok {
		t.Fatal("offered with BYWAY_INIT")
	}
	for _, c := range []string{"init", "setup", "__names"} {
		if _, ok := offerNeeded(c, env(zsh), home, true); ok {
			t.Fatalf("offered on %s", c)
		}
	}
	os.WriteFile(filepath.Join(home, ".zshrc"), []byte("eval \"$(command byway init zsh)\"  # byway\n"), 0o644)
	if _, ok := offerNeeded("byway", env(zsh), home, true); ok {
		t.Fatal("offered when already installed")
	}
}

func TestInitAndSetupCommands(t *testing.T) {
	setup(t, "")
	out, _, code := runCLI("", "init", "fish")
	if code != 0 || !strings.Contains(out, "BYWAY_LAST_CMD") {
		t.Fatalf("init fish: %d %q", code, out)
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	if _, errs, code := runCLI("", "setup", "--shell", "bash"); code != 0 {
		t.Fatal(errs)
	}
	if b, _ := os.ReadFile(filepath.Join(home, ".bashrc")); !strings.Contains(string(b), "byway init bash") {
		t.Fatalf("bashrc = %q", b)
	}
	if _, errs, code := runCLI("", "setup", "--shell", "bash", "--uninstall"); code != 0 {
		t.Fatal(errs)
	}
	if b, _ := os.ReadFile(filepath.Join(home, ".bashrc")); strings.Contains(string(b), "byway") {
		t.Fatalf("bashrc after uninstall = %q", b)
	}
}
