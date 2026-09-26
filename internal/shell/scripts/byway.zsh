byway() {
  if [[ "$1" == save ]]; then
    BYWAY_INIT=1 BYWAY_LAST_CMD="$(fc -ln -2 -1 2>/dev/null)" command byway "$@"
  else
    BYWAY_INIT=1 command byway "$@"
  fi
}
_byway() {
  local -a names
  names=("${(@f)$(command byway __names 2>/dev/null | sed -e 's/:/\\:/g' -e 's/\t/:/')}")
  _describe 'command' names
}
(( $+functions[compdef] )) && compdef _byway byway
