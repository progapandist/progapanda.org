#!/bin/sh

# 29 columns wide, so it fits the narrowest phone and never needs a width
# check. Static output cannot reflow — nothing survives the `exec` below to
# repaint it — so the only banner that always looks right is one that fits
# everywhere.
printf '\033[1;38;5;212m'
cat << 'EOF'
████  ████   ███   ████  ███
█   █ █   █ █   █ █     █   █
████  ████  █   █ █  ██ █████
█     █  █  █   █ █   █ █   █
█     █   █  ███   ███  █   █

████   ███  █   █ ████   ███
█   █ █   █ ██  █ █   █ █   █
████  █████ █ █ █ █   █ █████
█     █   █ █  ██ █   █ █   █
█     █   █ █   █ ████  █   █
EOF
printf '\033[0m\033[1;38;5;81mPORTFOLIO / TUI\033[0m  \033[38;5;245m● ONLINE\033[0m\n'
echo ""
printf '\033[38;5;255mWelcome, stranger.\033[0m\n'
printf 'Private, network-less container.\n'
printf '\033[38;5;81mPress Enter\033[0m to continue.\n'
echo ""

exec "$@"
