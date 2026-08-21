#!/bin/sh

cat << EOF > canihackit.hack

You can try! Fork bombs and excessive memory allocations should not work though. Containers also don't have any network. If you manage to break out of this container into a host system—please be kind and shoot me an email at andrey@lewagon.org to explain how you did it :)

EOF

cat << EOF > welcome
 _____  _____   ____   _____          _____        _   _ _____            ____  _____   _____ 
|  __ \|  __ \ / __ \ / ____|   /\   |  __ \ /\   | \ | |  __ \   /\     / __ \|  __ \ / ____|
| |__) | |__) | |  | | |  __   /  \  | |__) /  \  |  \| | |  | | /  \   | |  | | |__) | |  __ 
|  ___/|  _  /| |  | | | |_ | / /\ \ |  ___/ /\ \ | .   | |  | |/ /\ \  | |  | |  _  /| | |_ |
| |    | | \ \| |__| | |__| |/ ____ \| |  / ____ \| |\  | |__| / ____ \ | |__| | | \ \| |__| |
|_|    |_|  \_\\____/ \_____/_/    \_\_| /_/    \_\_| \_|_____/_/    \_(_)____/|_|  \_\\_____|

EOF

cat welcome

echo "Welcome, stranger! Open this page in any desktop browser."
echo ""
echo "Type ./hello  for the original 2020 TUI (built with tview)."
echo "Type ./hello2 for the 2025 rewrite    (built with Bubble Tea)."

exec "$@"