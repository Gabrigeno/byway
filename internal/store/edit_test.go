package store

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, s string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "byway.toml")
	if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

const base = "# my registry\n[vars]\nx = \"1\"  # keep\n\n[cmd.sb]\nrun = \"git status -sb\"\n"

func TestPutAppendsPreservingComments(t *testing.T) {
	p := write(t, base)
	if err := Put(p, "docker.pg", Command{Run: "docker start pg", Desc: "start postgres"}, false); err != nil {
		t.Fatal(err)
	}
	want := base + "\n[cmd.\"docker.pg\"]\nrun = \"docker start pg\"\ndesc = \"start postgres\"\n"
	if got := read(t, p); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	if st, _ := os.Stat(p); st.Mode().Perm() != 0o600 {
		t.Fatalf("perm changed to %v", st.Mode().Perm())
	}
}

func TestPutRefusesExisting(t *testing.T) {
	p := write(t, base)
	if err := Put(p, "sb", Command{Run: "x"}, false); !errors.Is(err, ErrExists) {
		t.Fatalf("err = %v", err)
	}
}

func TestPutOverwriteReplacesInPlace(t *testing.T) {
	src := "[cmd.a]\nrun = \"echo a\"\n\n# b doc\n[cmd.b]\nrun = \"echo b\"\n"
	p := write(t, src)
	if err := Put(p, "a", Command{Run: "echo A"}, true); err != nil {
		t.Fatal(err)
	}
	want := "[cmd.a]\nrun = \"echo A\"\n\n# b doc\n[cmd.b]\nrun = \"echo b\"\n"
	if got := read(t, p); got != want {
		t.Fatalf("got:\n%s", got)
	}
}

func TestRemoveKeepsNeighbourComments(t *testing.T) {
	p := write(t, "[vars]\n\n[cmd.a]\nrun = \"echo a\"\n\n# b doc\n[cmd.b]\nrun = \"echo b\"\n")
	if err := Remove(p, "a"); err != nil {
		t.Fatal(err)
	}
	if got, want := read(t, p), "[vars]\n\n# b doc\n[cmd.b]\nrun = \"echo b\"\n"; got != want {
		t.Fatalf("got:\n%q", got)
	}
	if err := Remove(p, "a"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestPutLeavesInvalidFileUntouched(t *testing.T) {
	p := write(t, "[cmd.a\n")
	if err := Put(p, "b", Command{Run: "x"}, false); err == nil {
		t.Fatal("expected error")
	}
	if read(t, p) != "[cmd.a\n" {
		t.Fatal("file was modified")
	}
}

func TestPutCreatesMissingFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "deep", "byway.toml")
	if err := Put(p, "sb", Command{Run: "git status -sb"}, false); err != nil {
		t.Fatal(err)
	}
	if got := read(t, p); got != "[cmd.sb]\nrun = \"git status -sb\"\n" {
		t.Fatalf("got %q", got)
	}
}

func TestPutFollowsSymlink(t *testing.T) {
	target := write(t, base)
	link := filepath.Join(t.TempDir(), "byway.toml")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := Put(link, "new", Command{Run: "echo hi"}, false); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Lstat(link); fi.Mode()&os.ModeSymlink == 0 {
		t.Fatal("symlink replaced by regular file")
	}
	if r, _ := Load(target); r.Cmd["new"].Run != "echo hi" {
		t.Fatal("target not updated")
	}
}

func TestPutRoundTripsTrickyStrings(t *testing.T) {
	run := "docker ps --format \"table {{.Name}}\\t{{.Status}}\" | grep 'it''s' \\\n  && echo ünï"
	p := write(t, "")
	if err := Put(p, "tricky", Command{Run: run}, false); err != nil {
		t.Fatal(err)
	}
	r, err := Load(p)
	if err != nil || r.Cmd["tricky"].Run != run {
		t.Fatalf("got %q err %v", r.Cmd["tricky"].Run, err)
	}
}

func TestLine(t *testing.T) {
	p := write(t, base)
	if n, err := Line(p, "sb"); err != nil || n != 5 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}
