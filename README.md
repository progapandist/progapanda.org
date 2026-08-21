# hello2

The TUI behind [progapanda.org](https://progapanda.org), rewritten in
[Bubble Tea](https://github.com/charmbracelet/bubbletea).

The original (`./hello`, 2020) was built with [tview](https://github.com/rivo/tview)
and its source was lost — this is a fresh implementation, with the content
brought up to date from my current CV. Both binaries live in the same container
so they can be compared side by side: type `./hello` or `./hello2`.

## How it fits in

`progapanda.org` serves a Svelte + Xterm.js frontend. A Go server upgrades to a
WebSocket, starts a network-less Alpine container per visitor, and pipes its
stdin/stdout over the socket. This program is what runs inside that container.

## Layout

| file | what |
|---|---|
| `main.go` | Bubble Tea model: menu, viewport, focus, layout |
| `content.go` | all the prose, in a tiny line-prefix markup |
| `render.go` | markup → styled, wrapped text |

Markup, one block per blank-line-separated chunk:

```
# heading
- bullet headline
  continuation text, hangs under the bullet
@ Label|https://url
anything else is a paragraph
```

Source line breaks are ignored; text is re-wrapped to the terminal's real
width, so the content files can stay readable.

## Usage

```sh
make run     # run it locally
make test    # wrapping never exceeds the pane width
make build   # cross-compile for linux/amd64
make deploy  # push into the running pods (see below)
```

`make deploy` copies the binary into the Docker-in-Docker sidecar of each
`progapanda-org` pod and rebuilds the local `progapandist/hello` image tag from
the cached base. No registry, no credentials. It needs a working `kubectl`
context pointed at the k3s cluster.

Because DinD stores images in an `emptyDir`, a pod restart wipes it. For
something durable, build and push `progapandist/hello` to Docker Hub instead.

## Keys

`↑`/`↓` navigate · `enter` read (scroll the pane) · `esc` back to menu · `q` quit
