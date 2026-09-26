byway() {
  if [[ "$1" == save ]]; then
    BYWAY_INIT=1 BYWAY_LAST_CMD="$(fc -ln -2 -1 2>/dev/null)" command byway "$@"
  else
    BYWAY_INIT=1 command byway "$@"
  fi
}
_byway() {
  COMPREPLY=($(compgen -W "$(command byway __names 2>/dev/null | cut -f1)" -- "${COMP_WORDS[COMP_CWORD]}"))
}
complete -F _byway byway
