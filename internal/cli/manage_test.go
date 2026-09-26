package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabrielegenovesi/byway/internal/store"
)

func cmdOf(t *testing.T, p, name string) store.Command {
	t.Helper()
	r, err := store.Load(p)
	if err != nil {
		t.Fatal(err)
	}
	return r.Cmd[name]
}

func TestSave(t *testing.T) {
	p := setup(t, "")
	t.Setenv("BYWAY_LAST_CMD", "\tgit checkout -b feat-x\n\tbyway save new --arg 1=feat-x")
	if _, errs, code := runCLI("", "save", "new", "-d", "new branch", "--arg", "1=feat-x"); code != 0 {
		t.Fatalf("code=%d %s", code, errs)
	}
	if c := cmdOf(t, p, "new"); c.Run != "git checkout -b {{1}}" || c.Desc != "new branch" {
		t.Fatalf("%+v", c)
	}
	if _, errs, code := runCLI("", "save", "new"); code != 1 || !strings.Contains(errs, "-f") {
		t.Fatalf("overwrite without -f: code=%d %s", code, errs)
	}
}

func TestSaveWithoutIntegration(t *testing.T) {
	setup(t, "")
	t.Setenv("BYWAY_LAST_CMD", "")
	if _, errs, code := runCLI("", "save", "x"); code != 1 || !strings.Contains(errs, "byway setup") {
		t.Fatalf("code=%d %s", code, errs)
	}
}

func TestAdd(t *testing.T) {
	p := setup(t, "")
	if _, errs, code := runCLI("", "add", "fmt", "--", "docker", "ps", "--format", "table {{.Name}}"); code != 0 {
		t.Fatal(errs)
	}
	if c := cmdOf(t, p, "fmt"); c.Run != "docker ps --format 'table {{.Name}}'" {
		t.Fatalf("%q", c.Run)
	}
	if _, errs, code := runCLI("echo from stdin\n", "add", "piped"); code != 0 {
		t.Fatal(errs)
	}
	if c := cmdOf(t, p, "piped"); c.Run != "echo from stdin" {
		t.Fatalf("%q", c.Run)
	}
}

func TestReservedNames(t *testing.T) {
	setup(t, "")
	for _, n := range []string{"ls", "save.x", "help"} {
		if _, errs, code := runCLI("", "add", n, "--", "true"); code != 1 || !strings.Contains(errs, "subcommand") {
			t.Errorf("%s: code=%d %s", n, code, errs)
		}
	}
}

func TestRm(t *testing.T) {
	p := setup(t, "[cmd.a]\nrun = \"true\"\n")
	if _, errs, code := runCLI("", "rm", "a"); code != 1 || !strings.Contains(errs, "-y") {
		t.Fatalf("non-tty without -y: code=%d %s", code, errs)
	}
	if _, errs, code := runCLI("", "rm", "a", "-y"); code != 0 {
		t.Fatal(errs)
	}
	if r, _ := store.Load(p); len(r.Cmd) != 0 {
		t.Fatal("not removed")
	}
	if _, errs, _ := runCLI("", "rm", "a", "-y"); !strings.Contains(errs, "not found") {
		t.Fatal(errs)
	}
}

func TestEdit(t *testing.T) {
	p := setup(t, "[vars]\n\n[cmd.a]\nrun = \"true\"\n")
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args")
	editor := filepath.Join(dir, "ed")
	os.WriteFile(editor, []byte("#!/bin/sh\necho \"$@\" > "+argsFile+"\n"), 0o755)
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", editor)
	if _, errs, code := runCLI("", "edit", "a"); code != 0 {
		t.Fatal(errs)
	}
	got, _ := os.ReadFile(argsFile)
	if strings.TrimSpace(string(got)) != "+3 "+p {
		t.Fatalf("editor args %q", got)
	}
}
