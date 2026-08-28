
You can try — and you're welcome to.

This container runs as an unprivileged user with no Linux capabilities, a
read-only root filesystem, seccomp on, and no-new-privileges set. It has no
network. Memory is capped at 64M with swap off, CPU at a tenth of a core, and
processes at 64 — so a fork bomb stops at 64. The only writable place is /tmp,
a small tmpfs that disappears with the container. Sessions don't last forever.

If you do find your way out, please be kind and email andrey@hey.com to tell me
how. I'd genuinely like to know.
