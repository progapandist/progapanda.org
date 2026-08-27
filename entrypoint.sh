#!/bin/sh

cat << EOF > canihackit.hack

You can try! Fork bombs and excessive memory allocations should not work though. Containers also don't have any network. If you manage to break out of this container into a host system—please be kind and shoot me an email at andrey@lewagon.org to explain how you did it :)

EOF

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
printf '\033[38;5;81mPress Enter\033[0m to launch the pre-filled \033[1;38;5;212m./hello2\033[0m command.\n'
printf '\033[38;5;245mWant the 2020 original instead? Replace it with ./hello.\033[0m\n'
echo ""

exec "$@"
