package shell

import (
	"embed"
	"fmt"
)

//go:embed scripts/byway.fish scripts/byway.zsh scripts/byway.bash
var scripts embed.FS

// InitScript returns the integration snippet for sh.
func InitScript(sh string) (string, error) {
	b, err := scripts.ReadFile("scripts/byway." + sh)
	if err != nil {
		return "", fmt.Errorf("unsupported shell %q (supported: fish, zsh, bash)", sh)
	}
	return string(b), nil
}
