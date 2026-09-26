package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitScript(t *testing.T) {
	for _, sh := range []string{"fish", "zsh", "bash"} {
		s, err := InitScript(sh)
		if err != nil || !strings.Contains(s, "BYWAY_LAST_CMD") {
			t.Errorf("%s: %v", sh, err)
		}
	}
	if _, err := InitScript("pwsh"); err == nil {
		t.Error("want error for pwsh")
	}
}

func TestDetectShell(t *testing.T) {
	for in, want := range map[string]string{"/bin/zsh": "zsh", "/opt/homebrew/bin/fish": "fish", "/bin/bash": "bash", "/bin/tcsh": "", "": ""} {
		if got := DetectShell(in); got != want {
			t.Errorf("DetectShell(%q) = %q", in, got)
		}
	}
}

func TestSetupZshIdempotentAndUninstall(t *testing.T) {
	home := t.TempDir()
	rc := filepath.Join(home, ".zshrc")
	os.WriteFile(rc, []byte("export A=1"), 0o644) // no trailing newline on purpose
	for i := 0; i < 2; i++ {
		if _, err := Setup("zsh", home); err != nil {
			t.Fatal(err)
		}
	}
	got, _ := os.ReadFile(rc)
	want := "export A=1\neval \"$(command byway init zsh)\"  # byway\n"
	if string(got) != want {
		t.Fatalf("rc = %q", got)
	}
	if !Installed("zsh", home) {
		t.Fatal("Installed = false")
	}
	if _, err := Uninstall("zsh", home); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(rc); string(got) != "export A=1\n" {
		t.Fatalf("after uninstall rc = %q", got)
	}
}

func TestSetupFish(t *testing.T) {
	home := t.TempDir()
	p, err := Setup("fish", home)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(p); string(got) != "command byway init fish | source\n" {
		t.Fatalf("%q", got)
	}
	if !Installed("fish", home) {
		t.Fatal("not installed")
	}
	Uninstall("fish", home)
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatal("conf.d file still there")
	}
}

func TestDecline(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if Declined() {
		t.Fatal("declined before Decline")
	}
	if err := Decline(); err != nil || !Declined() {
		t.Fatal(err)
	}
}
