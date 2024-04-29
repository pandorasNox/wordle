#!/usr/bin/env ash

CMD1=${1:?"first parameter, which is expected to be a command, is not set"}
CMD2=${2:?"second parameter, which is expected to be a command, is not set"}

# tmux new-session -d -s my-session-name 'watch -n 5 ps';
tmux new-session -d -s my-session-name "${CMD1}";

# tmux split-window -h 'watch -n 5 ls';
tmux split-window -h "${CMD2}";

# Use this to connect whenever you want 
tmux -2 attach-session -t my-session-name;
