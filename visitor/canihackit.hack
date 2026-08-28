
You can try, and the details are worth knowing.

This container runs as an unprivileged user with no capabilities, a read-only
root filesystem, seccomp on, and no-new-privileges set. It has no network at
all. Memory is capped at 64M with swap disabled, CPU at a tenth of a core, and
processes at 64 — so a fork bomb stops at 64, not at whatever the memory limit
happened to allow. The only writable place is /tmp, a 16M tmpfs that vanishes
with the container. Sessions are time-limited and there are only so many at
once.

None of that is the interesting part. The container is a container: it shares a
kernel with its host, and the sidecar it runs inside is privileged. A kernel
bug is the way out, and I have not fixed the kernel.

If you find one — please be kind and email andrey@hey.com to explain how. I
would genuinely rather hear it from you than read about it later.
