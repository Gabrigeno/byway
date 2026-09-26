package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const reg = `
[vars]
db = "postgres"

[cmd.hi]
run = "echo hi"

[cmd.new]
run = "echo checkout {{1}}"
desc = "new branch"

[cmd."docker.pg"]
run = "echo start {{db}}"

[cmd.dps]
run = "echo ps"

[cmd.fail]
run = "exit 3"

[cmd.cat]
run = "cat"

[cmd.all]
steps = ["hi", "fail", "hi"]
`

func setup(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "byway.toml")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BYWAY_FILE", p)
	t.Setenv("BYWAY_INIT", "1") // never offer setup in tests
	return p
}

func runCLI(stdin string, args ...string) (string, string, int) {
	var out, errb bytes.Buffer
	code := Main(args, strings.NewReader(stdin), &out, &errb)
	return out.String(), errb.String(), code
}

func TestRun(t *testing.T) {
	setup(t, reg)
	cases := []struct {
		args []string
		out  string
		code int
	}{
		{[]string{"hi"}, "hi\n", 0},
		{[]string{"new", "feat-x"}, "checkout feat-x\n", 0},
		{[]string{"docker", "pg"}, "start postgres\n", 0},
		{[]string{"docker.pg", "--db=pg16"}, "start pg16\n", 0},
		{[]string{"dps", "--format=json", "-a"}, "ps --format=json -a\n", 0},
		{[]string{"fail"}, "", 3},
		{[]string{"all"}, "hi\n", 3},
		{[]string{"-n", "new", "a b"}, "echo checkout 'a b'\n", 0},
		{[]string{"show", "new", "x"}, "echo checkout x\n", 0},
		{[]string{"run", "hi"}, "hi\n", 0},
	}
	for _, c := range cases {
		out, errs, code := runCLI("", c.args...)
		if out != c.out || code != c.code {
			t.Errorf("%v: out=%q code=%d stderr=%q; want out=%q code=%d", c.args, out, code, errs, c.out, c.code)
		}
	}
}

func TestRunStdin(t *testing.T) {
	setup(t, reg)
	if out, _, _ := runCLI("piped", "cat"); out != "piped" {
		t.Fatalf("out=%q", out)
	}
}

func TestErrors(t *testing.T) {
	setup(t, reg)
	cases := map[string][]string{
		`"hii" not found. Did you mean: hi?`:        {"hii"},
		"byway: new: missing argument 1":             {"new"},
		"byway: all: takes no arguments":             {"all", "x"},
		`byway: "zzzzzz" not found (see: byway ls)`: {"zzzzzz"},
	}
	for want, args := range cases {
		_, errs, code := runCLI("", args...)
		if code != 1 || !strings.Contains(errs, want) {
			t.Errorf("%v: code=%d stderr=%q, want %q", args, code, errs, want)
		}
	}
}

func TestLs(t *testing.T) {
	setup(t, reg)
	out, _, _ := runCLI("", "ls", "docker")
	if !strings.Contains(out, "docker.pg") || strings.Contains(out, "hi") {
		t.Fatalf("ls docker = %q", out)
	}
	out, _, _ = runCLI("", "ls")
	if !strings.Contains(out, "new") || !strings.Contains(out, "new branch") {
		t.Fatalf("ls = %q", out)
	}
	out, _, _ = runCLI("", "__names")
	if !strings.Contains(out, "new\tnew branch\n") {
		t.Fatalf("__names = %q", out)
	}
}
