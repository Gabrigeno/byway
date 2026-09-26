# byway — design

Date: 2026-09-26
Status: approved in brainstorming, pending written-spec review

## 1. Purpose

`byway` is an open-source CLI that acts as a single, global, personal registry of
custom shell commands. It replaces the scattered mix of git aliases, `~/Makefile`
targets, and shell functions people accumulate, with one tool that works the same
from fish, zsh, and bash.

Positioning: lower level than Taskfile/just (no per-project files, no hand-written
YAML required), more "call by name" than snippet managers like pet/navi (which
focus on searching).

### Priorities (in order)

1. **Instant capture** — run a command, then `byway save <name>` stores it from
   shell history; `byway <name>` runs it. Zero friction.
2. **Parametric and composable commands** — positional args, named vars,
   defaults, shared variables, commands built from other commands.
3. **Portability** — one human-readable file, versionable in dotfiles, identical
   on every machine, with namespaces.

### Success criteria

- After `brew install`, fish users have full integration with no manual steps;
  zsh/bash users answer one `Y` prompt.
- `cmd; byway save name; byway name` round-trips correctly in fish, zsh, bash.
- Hand comments in the registry file survive every `save`/`add`/`rm`.
- A substituted argument can never inject shell code (unless explicitly raw).

### Out of scope for v1

Fuzzy TUI picker, remote sync, per-project `.byway.toml`, importers from
gitconfig/Makefile/shell functions. The file format must not preclude any of them.

## 2. Registry file

Location: `$BYWAY_FILE` if set, else `$XDG_CONFIG_HOME/byway/byway.toml`, else
`~/.config/byway/byway.toml`. Created on first write.

```toml
[vars]
db_name = "postgres"
db_img  = "postgres:17"

[cmd.sb]
run = "git status -sb"

[cmd.new]
run  = "git checkout main && git pull && git checkout -b {{1}}"
desc = "new branch from up-to-date main"

[cmd."docker.postgres"]
run = "docker start {{db_name}}"

[cmd."docker.rebuild"]
steps = ["docker.postgres-run", "docker.memcached-run"]
```

Command fields:

| Field   | Type     | Meaning |
|---------|----------|---------|
| `run`   | string   | Template executed via `sh -c` after resolution. |
| `steps` | []string | Names of other commands run in sequence; stop at first non-zero exit. |
| `desc`  | string   | Optional description shown by `ls` and completions. |
| `shell` | string   | Optional interpreter override (`fish`, `bash`, `zsh`); default `sh`. |

Rules:
- Exactly one of `run` / `steps` must be set; otherwise load error.
- Names: `[a-z0-9_-]` segments joined by `.`. A dot defines a namespace:
  `byway docker postgres` ≡ `byway docker.postgres`.
- `steps` entries are invoked with no args; they see the same `[vars]` and any
  `--name=val` flags given to the parent. Cycles are detected at load time.
- Reading uses `github.com/BurntSushi/toml`. Writing is **textual** (append a
  block / remove a block / replace a block) so user comments and formatting are
  preserved. Writes are atomic (temp file + rename) under a file lock.

## 3. Templates and arguments

Resolved by byway before handing the string to the shell, so `$1` / `$VAR` in
`run` stay literal shell syntax.

| Placeholder          | Meaning |
|----------------------|---------|
| `{{1}}`, `{{2}}` …   | Positional arguments. |
| `{{@}}`              | All remaining positional arguments, each quoted. |
| `{{name}}`           | Value from `--name=val` flag, else `[vars]`. |
| `{{1:main}}`, `{{name:dflt}}` | With default. |
| `{{env.HOME}}`       | Environment variable. |
| `{{!1}}`, `{{!name}}` | Raw substitution (no escaping) — explicit opt-in. |

Rules:
- Every non-raw substituted value is shell-escaped for the target shell.
- Missing required value → `byway new: missing argument 1`, nothing runs.
- Extra positional args with no `{{@}}` in the template are appended, quoted,
  to the end of the command (so wrappers like `docker compose ps` accept flags).
- `--name=val` flags are parsed only before a literal `--`; everything after
  `--` is positional.
- Precedence for `{{name}}`: CLI flag → `[vars]` → default → error.

## 4. CLI

```
byway <name> [args]             run (also "byway docker postgres")
byway save <name> [-d desc] [--arg N=value]... [-f]
                                store last command from shell history
byway add <name> [-d desc] [-f] -- <cmd...>
byway add <name> [-d desc] [-f] < file     (reads command from stdin)
byway ls [namespace]            list name + desc (or run if no desc)
byway show <name> [args]        print resolved command, do not run
byway rm <name> [-y]            delete (confirms unless -y)
byway edit [name]               open $EDITOR at the command's line
byway setup [--shell S] [--uninstall]
byway init <fish|zsh|bash>      print integration script
byway run <name> [args]         explicit run (for names colliding with subcommands)
```

Global flags: `-n/--dry-run` (print resolved command, exit 0), `--file PATH`.
They must appear **before** `<name>` (`byway -n new x`); every `--k=v` after
`<name>` is a template variable, never a byway flag.

- Subcommand names are reserved: `save ls` is rejected with a hint.
- `save`/`add` refuse to overwrite an existing name without `-f`.
- `save --arg 1=feat-x` replaces occurrences of `feat-x` in the captured command
  with `{{1}}` (whole-token matches only; error if not found).
- Execution: inherits cwd, env, stdin/stdout/stderr and TTY (so `docker exec -it`
  works); byway exits with the command's exit code.
- Unknown name → `byway: "postgress" not found. Did you mean: postgres?`
  (Levenshtein ≤ 2).

## 5. Shell integration

`byway init <shell>` prints a script that:
1. Defines a `byway` shell function wrapping the binary, exporting
   `BYWAY_INIT=1`.
2. For `save`, passes the previous command via `BYWAY_LAST_CMD`:
   fish `$history[1]`, bash/zsh `fc -ln -1` (trimmed).
3. Registers completions (names, namespaces; `desc` as hint on fish/zsh).

Automatic installation:
- **fish**: the Homebrew formula installs `share/fish/vendor_conf.d/byway.fish`
  and `share/fish/vendor_completions.d/byway.fish`. Nothing to do.
- **zsh/bash**: when the binary runs without `BYWAY_INIT` in an interactive TTY,
  it detects the shell from `$SHELL` and asks once:
  `Add byway integration to ~/.zshrc? [Y/n]`. Yes → appends
  `eval "$(byway init zsh)"  # byway` idempotently (marker comment). No →
  records the refusal in `$XDG_STATE_HOME/byway/declined` and never asks again.
  Never prompts when stdin/stdout is not a TTY.
- `install.sh` (non-brew) installs the binary and runs `byway setup`.
- `byway setup --uninstall` removes the marker line.

`save` without `BYWAY_LAST_CMD` → error explaining `byway setup` or
`byway add name -- cmd`.

## 6. Code layout (Go)

```
cmd/byway/main.go       wiring only
internal/store/         load (BurntSushi/toml), validate, cycle check, textual edits, atomic write, lock
internal/tmpl/          pure: Resolve(template, args, flags, vars, env, shell) (string, error)
internal/runner/        exec sh -c / steps, TTY inheritance, exit code propagation
internal/capture/       --arg value→placeholder substitution on captured command
internal/shell/         embedded init scripts; rc setup/uninstall; shell escaping per shell
internal/cli/           cobra commands + name dispatch, "did you mean"
```

Dependencies: `spf13/cobra`, `BurntSushi/toml`. Nothing else without reason.

## 7. Errors

- Invalid TOML → message with line:col and hint `byway edit`; file never rewritten.
- Validation errors (both/neither `run`/`steps`, bad name, cycle, unknown step)
  reported at load with the command name.
- Atomic writes; lock timeout → clear error.

## 8. Testing

- Unit, table-driven: `tmpl` (escaping, defaults, raw, `{{@}}`, missing args),
  `capture`, `store` (comment preservation, add/replace/remove block, cycles).
- Integration: real `fish -c`, `zsh -c`, `bash -c` sessions sourcing
  `byway init`, running a command, `byway save`, asserting file contents and
  `byway <name>` output. CI matrix: macOS + Ubuntu.

## 9. Release

- goreleaser: darwin/linux × amd64/arm64 binaries, GitHub Releases, tap
  `homebrew-byway` with formula installing fish vendor conf + completions.
- `install.sh` for non-brew installs.
- License MIT.
