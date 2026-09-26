package shell

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const marker = "# byway"

// brew installs this for fish; its presence counts as installed.
var fishVendor = []string{
	"/opt/homebrew/share/fish/vendor_conf.d/byway.fish",
	"/usr/local/share/fish/vendor_conf.d/byway.fish",
	"/home/linuxbrew/.linuxbrew/share/fish/vendor_conf.d/byway.fish",
}

func DetectShell(shellEnv string) string {
	switch b := filepath.Base(shellEnv); b {
	case "fish", "zsh", "bash":
		return b
	}
	return ""
}

// ponytail: bash uses ~/.bashrc only; macOS login bash reads ~/.bash_profile, add when someone asks.
func RCPath(sh, home string) (string, error) {
	switch sh {
	case "zsh":
		return filepath.Join(home, ".zshrc"), nil
	case "bash":
		return filepath.Join(home, ".bashrc"), nil
	case "fish":
		return filepath.Join(home, ".config", "fish", "conf.d", "byway.fish"), nil
	}
	return "", fmt.Errorf("unsupported shell %q (supported: fish, zsh, bash)", sh)
}

func Setup(sh, home string) (string, error) {
	p, err := RCPath(sh, home)
	if err != nil {
		return "", err
	}
	if sh == "fish" {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return "", err
		}
		return p, os.WriteFile(p, []byte("command byway init fish | source\n"), 0o644)
	}
	data, err := os.ReadFile(p)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	if hasMarker(string(data)) {
		return p, nil
	}
	line := fmt.Sprintf("eval \"$(command byway init %s)\"  %s\n", sh, marker)
	if len(data) > 0 && !strings.HasSuffix(string(data), "\n") {
		line = "\n" + line
	}
	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}
	defer f.Close()
	_, err = f.WriteString(line)
	return p, err
}

func Uninstall(sh, home string) (string, error) {
	p, err := RCPath(sh, home)
	if err != nil {
		return "", err
	}
	if sh == "fish" {
		if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
		return p, nil
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return p, nil
	}
	if err != nil {
		return "", err
	}
	var keep []string
	for _, l := range strings.SplitAfter(string(data), "\n") {
		if !isMarkerLine(l) {
			keep = append(keep, l)
		}
	}
	return p, os.WriteFile(p, []byte(strings.Join(keep, "")), 0o644)
}

func Installed(sh, home string) bool {
	p, err := RCPath(sh, home)
	if err != nil {
		return false
	}
	if sh == "fish" {
		for _, c := range append([]string{p}, fishVendor...) {
			if _, err := os.Stat(c); err == nil {
				return true
			}
		}
		return false
	}
	data, _ := os.ReadFile(p)
	return hasMarker(string(data))
}

func isMarkerLine(l string) bool {
	t := strings.TrimSpace(l)
	return strings.HasSuffix(t, marker) && strings.Contains(t, "byway init")
}

func hasMarker(s string) bool {
	for _, l := range strings.Split(s, "\n") {
		if isMarkerLine(l) {
			return true
		}
	}
	return false
}

func declinedPath() string {
	dir := os.Getenv("XDG_STATE_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(dir, "byway", "declined")
}

func Declined() bool {
	_, err := os.Stat(declinedPath())
	return err == nil
}

func Decline() error {
	p := declinedPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, nil, 0o644)
}
