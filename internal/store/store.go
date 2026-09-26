// Package store reads and edits the byway registry file.
package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/BurntSushi/toml"
)

type Command struct {
	Run   string   `toml:"run,omitempty"`
	Steps []string `toml:"steps,omitempty"`
	Desc  string   `toml:"desc,omitempty"`
	Shell string   `toml:"shell,omitempty"`
}

type Registry struct {
	Vars map[string]string  `toml:"vars"`
	Cmd  map[string]Command `toml:"cmd"`
}

var nameRe = regexp.MustCompile(`^[a-z0-9_-]+(\.[a-z0-9_-]+)*$`)

func ValidName(n string) bool { return nameRe.MatchString(n) }

// Path returns the registry location: $BYWAY_FILE, then XDG, then ~/.config.
func Path() string {
	if p := os.Getenv("BYWAY_FILE"); p != "" {
		return p
	}
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "byway", "byway.toml")
}

// Load reads path; a missing file is an empty registry.
func Load(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Parse(nil)
	}
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

func Parse(data []byte) (*Registry, error) {
	var r Registry
	md, err := toml.Decode(string(data), &r)
	if err != nil {
		return nil, fmt.Errorf("invalid registry: %w (fix it with: byway edit)", err)
	}
	if und := md.Undecoded(); len(und) > 0 {
		return nil, fmt.Errorf("invalid registry: unknown key %q (namespaced names need quotes: [cmd.\"a.b\"])", und[0].String())
	}
	if r.Vars == nil {
		r.Vars = map[string]string{}
	}
	if r.Cmd == nil {
		r.Cmd = map[string]Command{}
	}
	return &r, r.validate()
}

func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.Cmd))
	for n := range r.Cmd {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func (r *Registry) validate() error {
	for _, name := range r.Names() {
		c := r.Cmd[name]
		if !ValidName(name) {
			return fmt.Errorf("command %q: invalid name (use a-z 0-9 _ - and . for namespaces)", name)
		}
		if (c.Run == "") == (len(c.Steps) == 0) {
			return fmt.Errorf("command %q: set exactly one of run or steps", name)
		}
		switch c.Shell {
		case "", "sh", "bash", "zsh", "fish":
		default:
			return fmt.Errorf("command %q: unsupported shell %q", name, c.Shell)
		}
		for _, s := range c.Steps {
			if _, ok := r.Cmd[s]; !ok {
				return fmt.Errorf("command %q: unknown step %q", name, s)
			}
		}
	}
	state := map[string]int{} // 1 visiting, 2 done
	var visit func(string) error
	visit = func(n string) error {
		switch state[n] {
		case 1:
			return fmt.Errorf("command %q: steps form a cycle", n)
		case 2:
			return nil
		}
		state[n] = 1
		for _, s := range r.Cmd[n].Steps {
			if err := visit(s); err != nil {
				return err
			}
		}
		state[n] = 2
		return nil
	}
	for _, n := range r.Names() {
		if err := visit(n); err != nil {
			return err
		}
	}
	return nil
}
