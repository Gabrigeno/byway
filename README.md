# byway

Save any shell command by name, run it from anywhere. One registry for fish, zsh and bash. No more scattered git aliases, `~/Makefile` targets and shell functions.

```sh
docker start postgres
byway save pg               # saves the command you just ran
byway pg                    # runs it

git checkout -b feat-x
byway save new --arg 1=feat-x   # -> git checkout -b {{1}}
byway new feat-y
```

## Install

```sh
brew install gabrielegenovesi/byway/byway   # fish: ready. zsh/bash: answer Y on first run
# or
curl -fsSL https://raw.githubusercontent.com/gabrielegenovesi/byway/main/install.sh | sh
```

## Registry

`~/.config/byway/byway.toml` (or `$BYWAY_FILE`). Edit it by hand anytime (`byway edit [name]`); byway keeps your comments.

```toml
[vars]
db = "postgres"

[cmd."docker.pg"]                 # byway docker pg
run  = "docker start {{db}}"
desc = "start postgres"

[cmd."docker.up"]
steps = ["docker.pg", "docker.redis"]
```

Placeholders: `{{1}}` `{{@}}` `{{name}}` `{{1:default}}` `{{env.HOME}}`; `{{!x}}` inserts raw (unquoted). Values are shell-quoted. `{{.Go}}`-style templates are left alone.

## Commands

`byway <name> [args]` · `save` · `add <name> -- cmd` · `ls [ns]` · `show` · `rm` · `edit` · `setup [--uninstall]` · `init <shell>` · `-n` dry-run
