package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
)

var (
	ErrExists   = errors.New("command already exists (use -f to overwrite)")
	ErrNotFound = errors.New("command not found")
)

// Put adds or (with overwrite) replaces the block for name, editing the file
// as text so user comments and layout survive.
func Put(path, name string, c Command, overwrite bool) error {
	if !ValidName(name) {
		return fmt.Errorf("invalid name %q (use a-z 0-9 _ - and . for namespaces)", name)
	}
	block, err := renderBlock(name, c)
	if err != nil {
		return err
	}
	return edit(path, func(lines []string) ([]string, error) {
		if s, e, ok := findBlock(lines, name); ok {
			if !overwrite {
				return nil, ErrExists
			}
			return splice(lines, s, e, block), nil
		}
		for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
			lines = lines[:len(lines)-1]
		}
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		return append(lines, block...), nil
	})
}

func Remove(path, name string) error {
	return edit(path, func(lines []string) ([]string, error) {
		s, e, ok := findBlock(lines, name)
		if !ok {
			return nil, ErrNotFound
		}
		lines = splice(lines, s, e, nil)
		if s > 0 && s < len(lines) && strings.TrimSpace(lines[s-1]) == "" && strings.TrimSpace(lines[s]) == "" {
			lines = append(lines[:s], lines[s+1:]...)
		}
		return lines, nil
	})
}

// Line returns the 1-based line of name's [cmd.…] header.
func Line(path, name string) (int, error) {
	lines, err := readLines(resolve(path))
	if err != nil {
		return 0, err
	}
	s, _, ok := findBlock(lines, name)
	if !ok {
		return 0, ErrNotFound
	}
	return s + 1, nil
}

// edit runs fn on the file's lines under lock; the result must still parse.
func edit(path string, fn func([]string) ([]string, error)) error {
	path = resolve(path)
	return withLock(filepath.Dir(path), func() error {
		lines, err := readLines(path)
		if err != nil {
			return err
		}
		if _, err := Parse([]byte(strings.Join(lines, "\n"))); err != nil {
			return err
		}
		if lines, err = fn(lines); err != nil {
			return err
		}
		for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
			lines = lines[:len(lines)-1]
		}
		data := strings.Join(lines, "\n") + "\n"
		if _, err := Parse([]byte(data)); err != nil {
			return fmt.Errorf("refusing to write: %w", err)
		}
		return writeAtomic(path, []byte(data))
	})
}

// resolve follows a symlinked registry (dotfiles) so writes hit the target.
func resolve(path string) string {
	if p, err := filepath.EvalSymlinks(path); err == nil {
		return p
	}
	return path
}

func readLines(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) || len(data) == 0 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimRight(string(data), "\n"), "\n"), nil
}

func renderBlock(name string, c Command) ([]string, error) {
	hdr := "[cmd." + name + "]"
	if strings.Contains(name, ".") {
		hdr = `[cmd."` + name + `"]`
	}
	body, err := toml.Marshal(c)
	if err != nil {
		return nil, err
	}
	return append([]string{hdr}, strings.Split(strings.TrimRight(string(body), "\n"), "\n")...), nil
}

// header returns the table name of a "[…]" line with spaces removed.
func header(l string) (string, bool) {
	t := strings.TrimSpace(l)
	if !strings.HasPrefix(t, "[") {
		return "", false
	}
	i := strings.Index(t, "]")
	if i < 0 {
		return "", false
	}
	return strings.ReplaceAll(strings.Trim(t[:i], "["), " ", ""), true
}

// lines belong to the next block, not this one.
func findBlock(lines []string, name string) (int, int, bool) {
	start, end := -1, len(lines)
	for i, l := range lines {
		h, ok := header(l)
		if !ok {
			continue
		}
		if start >= 0 {
			end = i
			break
		}
		if h == "cmd."+name || h == `cmd."`+name+`"` || h == "cmd.'"+name+"'" {
			start = i
		}
	}
	if start < 0 {
		return 0, 0, false
	}
	for end-1 > start {
		t := strings.TrimSpace(lines[end-1])
		if t != "" && !strings.HasPrefix(t, "#") {
			break
		}
		end--
	}
	return start, end, true
}

func splice(lines []string, s, e int, repl []string) []string {
	out := append([]string{}, lines[:s]...)
	out = append(out, repl...)
	return append(out, lines[e:]...)
}

func withLock(dir string, fn func() error) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	deadline := time.Now().Add(2 * time.Second)
	for syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) != nil {
		if time.Now().After(deadline) {
			return fmt.Errorf("registry is locked by another byway process (%s)", dir)
		}
		time.Sleep(50 * time.Millisecond)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return fn()
}

func writeAtomic(path string, data []byte) error {
	mode := fs.FileMode(0o644)
	if st, err := os.Stat(path); err == nil {
		mode = st.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".byway-*.toml")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
