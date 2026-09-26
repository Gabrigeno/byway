package tmpl

import (
	"reflect"
	"strings"
	"testing"
)

func TestResolve(t *testing.T) {
	env := func(k string) (string, bool) {
		if k == "HOME" {
			return "/home/u", true
		}
		return "", false
	}
	cases := []struct {
		name, tmpl string
		in         Input
		want, err  string
	}{
		{"literal", "git status -sb", Input{}, "git status -sb", ""},
		{"positional", "git checkout -b {{1}}", Input{Args: []string{"feat-x"}}, "git checkout -b feat-x", ""},
		{"escaped", "echo {{1}}", Input{Args: []string{"a; rm -rf ~"}}, "echo 'a; rm -rf ~'", ""},
		{"raw", "echo {{!1}}", Input{Args: []string{"$HOME"}}, "echo $HOME", ""},
		{"missing", "echo {{2}}", Input{Args: []string{"a"}}, "", "missing argument 2"},
		{"default", "git merge {{1:main}}", Input{}, "git merge main", ""},
		{"empty default", "echo x{{1:}}", Input{}, "echo x''", ""},
		{"all", "docker {{1}} {{@}}", Input{Args: []string{"ps", "-a", "b c"}}, "docker ps -a 'b c'", ""},
		{"extra appended", "docker compose ps", Input{Args: []string{"-a"}}, "docker compose ps -a", ""},
		{"var", "docker start {{db}}", Input{Vars: map[string]string{"db": "postgres"}}, "docker start postgres", ""},
		{"flag beats var", "docker start {{db}}", Input{Vars: map[string]string{"db": "pg"}, Flags: map[string]string{"db": "pg16"}}, "docker start pg16", ""},
		{"var default", "docker start {{db:pg}}", Input{}, "docker start pg", ""},
		{"missing var", "echo {{db}}", Input{}, "", "missing variable db"},
		{"env", "ls {{env.HOME}}", Input{Env: env}, "ls /home/u", ""},
		{"missing env", "ls {{env.NOPE}}", Input{Env: env}, "", "missing environment variable NOPE"},
		{"go template literal", `docker ps --format "table {{.Name}}\t{{.Status}}"`, Input{}, `docker ps --format "table {{.Name}}\t{{.Status}}"`, ""},
		{"fish quoting", "echo {{1}}", Input{Args: []string{"it's"}, Shell: "fish"}, `echo 'it\'s'`, ""},
		{"zero", "echo {{0}}", Input{}, "", "invalid placeholder {{0}}"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Resolve(c.tmpl, c.in)
			if c.err != "" {
				if err == nil || !strings.Contains(err.Error(), c.err) {
					t.Fatalf("err = %v, want containing %q", err, c.err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestRefs(t *testing.T) {
	got := Refs("{{1}} {{db}} {{env.X}} {{!name:d}} {{@}} {{db}} {{.Name}}")
	if want := []string{"db", "name"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
