package integration

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"

	"github.com/Gabrigeno/byway/internal/store"
)

type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) { s.mu.Lock(); defer s.mu.Unlock(); return s.b.Write(p) }
func (s *syncBuf) String() string              { s.mu.Lock(); defer s.mu.Unlock(); return s.b.String() }

func buildBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	c := exec.Command("go", "build", "-o", filepath.Join(dir, "byway"), "../cmd/byway")
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	return dir
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// shellCmd returns an interactive shell whose startup loads byway's integration.
func shellCmd(t *testing.T, sh, home string) *exec.Cmd {
	switch sh {
	case "bash":
		rc := filepath.Join(home, ".bashrc")
		must(t, os.WriteFile(rc, []byte(`eval "$(command byway init bash)"`+"\n"), 0o644))
		return exec.Command("bash", "--noprofile", "--rcfile", rc, "-i")
	case "zsh":
		must(t, os.WriteFile(filepath.Join(home, ".zshrc"), []byte(`eval "$(command byway init zsh)"`+"\n"), 0o644))
		return exec.Command("zsh", "-i")
	default: // fish
		dir := filepath.Join(home, ".config", "fish")
		must(t, os.MkdirAll(dir, 0o755))
		must(t, os.WriteFile(filepath.Join(dir, "config.fish"), []byte("command byway init fish | source\n"), 0o644))
		return exec.Command("fish", "-i")
	}
}

func TestSaveFromRealShells(t *testing.T) {
	bindir := buildBinary(t)
	for _, sh := range []string{"bash", "zsh", "fish"} {
		t.Run(sh, func(t *testing.T) {
			if _, err := exec.LookPath(sh); err != nil {
				t.Skipf("%s not installed", sh)
			}
			home := t.TempDir()
			reg := filepath.Join(home, "byway.toml")
			env := append(os.Environ(),
				"HOME="+home, "ZDOTDIR="+home, "BYWAY_FILE="+reg, "TERM=dumb",
				"PATH="+bindir+string(os.PathListSeparator)+os.Getenv("PATH"),
				"XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
				"XDG_DATA_HOME="+filepath.Join(home, ".local", "share"),
				"XDG_STATE_HOME="+filepath.Join(home, ".local", "state"),
				"HISTFILE="+filepath.Join(home, ".hist"))
			c := shellCmd(t, sh, home)
			c.Env = env
			f, err := pty.Start(c)
			must(t, err)
			defer f.Close()
			var buf syncBuf
			go io.Copy(&buf, f)

			for _, line := range []string{"echo hello-byway", "byway save hi", "exit"} {
				time.Sleep(time.Second) // let the shell draw its prompt
				f.Write([]byte(line + "\r"))
			}
			done := make(chan error, 1)
			go func() { done <- c.Wait() }()
			select {
			case <-done:
			case <-time.After(20 * time.Second):
				c.Process.Kill()
				t.Fatalf("%s hung; session:\n%s", sh, buf.String())
			}

			r, err := store.Load(reg)
			if err != nil || r.Cmd["hi"].Run != "echo hello-byway" {
				t.Fatalf("saved %q (err %v); session:\n%s", r.Cmd["hi"].Run, err, buf.String())
			}
			run := exec.Command(filepath.Join(bindir, "byway"), "hi")
			run.Env = env
			if out, err := run.Output(); err != nil || string(out) != "hello-byway\n" {
				t.Fatalf("byway hi = %q, %v", out, err)
			}
		})
	}
}
