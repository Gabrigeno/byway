# byway

Save a shell command by name and run it from fish, zsh or bash. All commands live in one TOML file.

![demo](docs/demo.gif)

```sh
docker start postgres
byway save pg
byway pg

git checkout -b feat-x
byway save new --arg 1=feat-x
byway new feat-y
```

`byway save` takes the last command from your history. `--arg 1=feat-x` replaces `feat-x` with `{{1}}`, the first argument.

## Install

```sh
brew install Gabrigeno/byway/byway
```

```sh
curl -fsSL https://raw.githubusercontent.com/Gabrigeno/byway/main/install.sh | sh
```

```sh
go install github.com/Gabrigeno/byway/cmd/byway@latest
byway setup
```

Binaries for macOS and Linux are also on the [releases page](https://github.com/Gabrigeno/byway/releases).

With brew, fish works right away. For zsh and bash, run any byway command and answer Y, or run `byway setup`. `save` needs this to read your history.

## Registry

`~/.config/byway/byway.toml`, or `$BYWAY_FILE`. You can edit it by hand (`byway edit [name]`); comments are kept.

```toml
[vars]
db = "postgres"

[cmd."docker.pg"]
run  = "docker start {{db}}"
desc = "start postgres"

[cmd."docker.up"]
steps = ["docker.pg", "docker.redis"]
```

`docker.pg` runs as `byway docker pg`. `steps` runs other commands in order.

| Placeholder | Value |
|---|---|
| `{{1}}`, `{{2}}` | positional arguments |
| `{{@}}` | all remaining arguments |
| `{{name}}` | `--name=value`, else `[vars]` |
| `{{1:main}}` | default if missing |
| `{{env.HOME}}` | environment variable |
| `{{!1}}` | inserted as is, not quoted |

Values are shell-quoted. `{{.Name}}` (Go/docker templates) is left alone.

## Commands

```
byway <name> [args]          run
byway -n <name> [args]       print instead of running
byway save <name>            save the last command (--arg, -d, -f)
byway add <name> -- <cmd>    save a command, or pipe it on stdin
byway ls [namespace]
byway show <name>
byway edit [name]
byway rm <name> -y
byway setup [--uninstall]
byway init <fish|zsh|bash>
```

## Why not pet, navi or just?

pet and navi are snippet managers: you search and pick from a list. just runs recipes from a justfile in the project. byway is closer to aliases: you call commands by name, they take arguments, work in every shell, and you create them from what you just ran.
