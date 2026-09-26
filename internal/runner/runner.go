// Package runner executes resolved command lines.
package runner

import (
	"errors"
	"io"
	"os/exec"
)

type Step struct {
	Shell string // "" means sh
	Line  string
}

// Run executes steps in order, inheriting stdio, and returns the first
// non-zero exit code.
func Run(steps []Step, in io.Reader, out, errw io.Writer) (int, error) {
	for _, s := range steps {
		bin := s.Shell
		if bin == "" {
			bin = "sh"
		}
		c := exec.Command(bin, "-c", s.Line)
		c.Stdin, c.Stdout, c.Stderr = in, out, errw
		err := c.Run()
		var ee *exec.ExitError
		switch {
		case errors.As(err, &ee):
			return ee.ExitCode(), nil
		case err != nil:
			return 127, err
		}
	}
	return 0, nil
}
