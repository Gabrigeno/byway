function byway --description 'run saved commands'
    if test "$argv[1]" = save
        BYWAY_INIT=1 BYWAY_LAST_CMD=(string join \n -- $history[2] $history[1] | string collect) command byway $argv
    else
        BYWAY_INIT=1 command byway $argv
    end
end
complete -c byway -f -n __fish_is_first_arg -a '(command byway __names 2>/dev/null)'
