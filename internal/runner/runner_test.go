package runner

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunSequenceStopsOnFailure(t *testing.T) {
	var out bytes.Buffer
	code, err := Run([]Step{{Line: "echo one"}, {Line: "exit 3"}, {Line: "echo never"}}, strings.NewReader(""), &out, &out)
	if err != nil || code != 3 || out.String() != "one\n" {
		t.Fatalf("code=%d err=%v out=%q", code, err, out.String())
	}
}

func TestRunPassesStdin(t *testing.T) {
	var out bytes.Buffer
	if code, _ := Run([]Step{{Line: "cat"}}, strings.NewReader("piped"), &out, &out); code != 0 || out.String() != "piped" {
		t.Fatalf("code=%d out=%q", code, out.String())
	}
}

func TestRunMissingShell(t *testing.T) {
	code, err := Run([]Step{{Shell: "no-such-shell-xyz", Line: "true"}}, nil, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || code != 127 {
		t.Fatalf("code=%d err=%v", code, err)
	}
}
