package store

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseValid(t *testing.T) {
	r, err := Parse([]byte(`
[vars]
db = "postgres"

[cmd.sb]
run = "git status -sb"

[cmd."docker.pg"]
run = "docker start {{db}}"
desc = "start pg"

[cmd."docker.all"]
steps = ["docker.pg", "sb"]
`))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := r.Names(), []string{"docker.all", "docker.pg", "sb"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("names %v", got)
	}
	if r.Vars["db"] != "postgres" || r.Cmd["docker.pg"].Desc != "start pg" {
		t.Fatalf("bad decode: %+v", r)
	}
}

func TestParseErrors(t *testing.T) {
	cases := map[string]string{
		"bad toml":        "[cmd.a\nrun=1",
		"both":            "[cmd.a]\nrun = \"x\"\nsteps = [\"b\"]\n[cmd.b]\nrun=\"y\"",
		"neither":         "[cmd.a]\ndesc = \"x\"",
		"bad name":        "[cmd.\"A B\"]\nrun = \"x\"",
		"unknown step":    "[cmd.a]\nsteps = [\"nope\"]",
		"cycle":           "[cmd.a]\nsteps = [\"b\"]\n[cmd.b]\nsteps = [\"a\"]",
		"bad shell":       "[cmd.a]\nrun = \"x\"\nshell = \"pwsh\"",
		"unquoted dotted": "[cmd.docker.pg]\nrun = \"x\"",
	}
	want := map[string]string{
		"bad toml": "byway edit", "both": "exactly one of run or steps", "neither": "exactly one of run or steps",
		"bad name": "invalid name", "unknown step": `unknown step "nope"`, "cycle": "cycle",
		"bad shell": `unsupported shell "pwsh"`, "unquoted dotted": "need quotes",
	}
	for name, src := range cases {
		_, err := Parse([]byte(src))
		if err == nil || !strings.Contains(err.Error(), want[name]) {
			t.Errorf("%s: err = %v, want containing %q", name, err, want[name])
		}
	}
}

func TestLoadMissingIsEmpty(t *testing.T) {
	r, err := Load(filepath.Join(t.TempDir(), "none.toml"))
	if err != nil || len(r.Cmd) != 0 || r.Vars == nil {
		t.Fatalf("r=%+v err=%v", r, err)
	}
}

func TestPath(t *testing.T) {
	t.Setenv("BYWAY_FILE", "/x/y.toml")
	if Path() != "/x/y.toml" {
		t.Fatal(Path())
	}
	t.Setenv("BYWAY_FILE", "")
	t.Setenv("XDG_CONFIG_HOME", "/cfg")
	if Path() != "/cfg/byway/byway.toml" {
		t.Fatal(Path())
	}
}
