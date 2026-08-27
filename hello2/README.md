# hello2

The TUI behind [progapanda.org](https://progapanda.org), rewritten in
[Bubble Tea](https://github.com/charmbracelet/bubbletea).

The original (`./hello`, 2020) was built with [tview](https://github.com/rivo/tview)
and its source was lost — this is a fresh implementation, with the content
brought up to date from my current CV. Both binaries live in the same container
so they can be compared side by side. The site pre-fills `./hello2`; replace it
with `./hello` at the prompt to run the original.

## How it fits in

`progapanda.org` serves an Xterm.js frontend. A Go server upgrades to a
WebSocket, starts a network-less Alpine container per visitor, and pipes its
stdin/stdout over the socket. This program is what runs inside that container.

## Layout

| file | what |
|---|---|
| `content.md` | **all the prose — edit this to change the copy** |
| `main.go` | Bubble Tea model: menu, viewport, focus, layout |
| `content.go` | embeds `content.md` and splits it into sections |
| `render.go` | markdown → styled, wrapped text |

`content.md` is embedded into the binary at build time. Every `# ` heading
starts a new section and becomes its menu entry, in file order. Within a
section, blank lines separate blocks and the opening characters pick the style:

```markdown
## heading

- bullet headline, or a [label](https://url) to make it clickable
  continuation text, hangs under the bullet

[Label](https://url)
[Another](https://url)

anything else is a paragraph
```

A block of consecutive links renders as a label/URL list; `mailto:` is stripped
from what's displayed. Source line breaks are ignored — text is re-wrapped to
the terminal's real width, so `content.md` can stay readable (and it renders on
GitHub).

## Usage

```sh
make run     # run it locally
make test    # wrapping never exceeds the pane width
make build   # cross-compile for linux/amd64 into dist/
make deploy  # push into the running pods (see below)
```

For the complete local browser stack, keep this repository next to
`progapanda.org` and run `make dev` from `progapanda.org`. That command builds
this repository as the local visitor image before starting the web server.

`make build` produces a **linux/amd64** binary for the container, which is why
it goes to `dist/` and not the repo root — running it on an arm64 Mac gives
`exec format error`. Use `make run` locally.

`make deploy` copies the binary and `entrypoint.sh` welcome screen into the
Docker-in-Docker sidecar of each `progapanda-org` pod and rebuilds the local
`progapandist/hello` image tag on top of the pristine upstream image, pinned by
digest as `BASE` in the Makefile. No registry credentials needed. It needs a
working `kubectl` context pointed at the k3s cluster.

Building from the digest rather than from the `progapandist/hello` tag matters:
the tag is what this deploy overwrites, so building `FROM` it would stack a new
layer on the previous deploy every time.

Because DinD stores images in an `emptyDir`, a pod restart wipes it. For
something durable, build and push `progapandist/hello` to Docker Hub instead.

## Keys

`↑`/`↓` navigate · `enter` read (scroll the pane) · `esc` back to menu · `q` quit
