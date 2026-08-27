#!/bin/sh

printf '\033[1;38;5;212m'
cat << 'EOF'
████  ████   ███   ████  ███  ████   ███  █   █ ████   ███         ███  ████   ████
█   █ █   █ █   █ █     █   █ █   █ █   █ ██  █ █   █ █   █       █   █ █   █ █
████  ████  █   █ █  ██ █████ ████  █████ █ █ █ █   █ █████       █   █ ████  █  ██
█     █  █  █   █ █   █ █   █ █     █   █ █  ██ █   █ █   █       █   █ █  █  █   █
█     █   █  ███   ███  █   █ █     █   █ █   █ ████  █   █   ██   ███  █   █  ███
EOF
printf '\033[0m\033[1;38;5;81mPORTFOLIO / TUI\033[0m  \033[38;5;245m● ONLINE\033[0m\n'
echo ""
printf '\033[38;5;255mWelcome, stranger.\033[0m Your private, network-less container is ready.\n'
printf '\033[38;5;81mPress Enter\033[0m to continue.\n'
echo ""

exec "$@"
