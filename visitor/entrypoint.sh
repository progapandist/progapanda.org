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
printf '\033[0m\033[1;38;5;81mPORTFOLIO / TUI\033[0m  \033[1;38;5;84m● ONLINE\033[0m\n'
echo ""
# Every line stays inside 32 columns; see the banner note above.
printf '\033[38;5;255mWelcome, stranger.\033[0m\n'
echo ""
printf 'This is a real Linux shell in a\n'
printf 'container of your own. No\n'
printf 'network, 64 MB, nobody else.\n'
echo ""
printf '\033[38;5;212mTry to break it\033[0m — fork bombs,\n'
printf 'rm -rf, whatever you fancy. A\n'
printf 'reload hands you a pristine one.\n'
echo ""
printf '\033[38;5;81mPress Enter\033[0m, or poke around.\n'
echo ""

exec "$@"
