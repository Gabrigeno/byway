package capture

import (
	"strings"
	"testing"
)

func TestPickLast(t *testing.T) {
	cases := map[string]string{
		"echo hi\nbyway save x":            "echo hi",
		"\techo hi\n\tbyway save x -f":     "echo hi",
		"echo hi":                          "echo hi",
		"git st\n\n  command byway save y": "git st",
		"byway saved-thing":                "byway saved-thing",
	}
	for in, want := range cases {
		if got, err := PickLast(in); err != nil || got != want {
			t.Errorf("PickLast(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := PickLast("byway save x\n"); err == nil {
		t.Error("want error when nothing but byway save")
	}
}

func TestParametrize(t *testing.T) {
	cases := []struct {
		cmd  string
		subs []string
		want string
	}{
		{"git checkout -b feat-x", []string{"1=feat-x"}, "git checkout -b {{1}}"},
		{`git checkout -b "feat-x"`, []string{"1=feat-x"}, "git checkout -b {{1}}"},
		{"psql -d mydb -c 'drop database mydb'", []string{"db=mydb"}, "psql -d {{db}} -c 'drop database {{db}}'"},
		{"cp a a.bak", []string{"1=a"}, "cp {{1}} {{1}}.bak"},
		{"x --name=foo", []string{"1=foo"}, "x --name={{1}}"},
	}
	for _, c := range cases {
		got, err := Parametrize(c.cmd, c.subs)
		if err != nil || got != c.want {
			t.Errorf("Parametrize(%q,%v) = %q, %v; want %q", c.cmd, c.subs, got, err, c.want)
		}
	}
}

func TestParametrizeErrors(t *testing.T) {
	for sub, want := range map[string]string{"1=zzz": "not found", "nokey": "want N=value", "0=x": "want N=value", "=x": "want N=value"} {
		_, err := Parametrize("echo abc", []string{sub})
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: err = %v", sub, err)
		}
	}
}
