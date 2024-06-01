rename-session uniddb
send "nvim ." C-m
new-window -t 2 -n terminal
new-window -t 3 -n tailwind-watch
send "make tailwind-watch" C-m
new-window -t 4 -n air-watch
send "make air" C-m
new-window -t 5 -n templ-watch
send "make templ-watch" C-m
select-window -t 1
