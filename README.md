# progapanda.org

Source code for :link: https://progapanda.org

A terminal in your browser. Every visitor gets a private, network-less
container running a Go TUI portfolio.

## How it works

1. The browser loads Xterm.js and opens a WebSocket to `/term`.
2. The Go server starts a throwaway Docker container per visitor
   (`--network none`, 0.1 CPU, 64M, `--rm`) behind a PTY.
3. PTY bytes and keystrokes are piped over the socket, raw. Xterm owns UTF-8
   decoding, because a multibyte character can straddle two messages.

The TUI inside the container is `cmd/hello2` in this repo. It began life as a
separate repository and its history came along with it.

Wire format is binary frames, first byte is the type: `0` raw terminal bytes,
`1` a JSON `{"rows":n,"cols":n}` resize. The first message a client sends must
be a resize — the server needs it to size the PTY.

## Layout

| path | what |
|---|---|
| `cmd/webterm` | the web server: WebSocket, PTY, one container per visitor |
| `cmd/hello2` | the TUI that runs inside that container |
| `cmd/hello2/content.md` | **all the prose — edit this to change the copy** |
| `visitor` | everything else that ships in the visitor container |
| `src` | Xterm.js frontend, built with Vite |
| `k8s` | Deployment, Service, Ingress, cert |
| `tools/icons.py` | regenerates the favicon and link-preview image |
| `tools/stripeek_fixture.py` | regenerates the scrubbed stripeek capture |

`cmd/hello2/content.md` is embedded into the TUI binary at build time. Every
`# ` heading starts a new section and becomes its menu entry, in file order.
Within a section, blank lines separate blocks and the opening characters pick
the style:

```markdown
## heading

- bullet headline, or a [label](https://url) to make it clickable
  continuation text, hangs under the bullet

[Label](https://url)
[Another](https://url)

anything else is a paragraph
```

Source line breaks are ignored — text is re-wrapped to the terminal's real
width, so `content.md` stays readable (and renders on GitHub).

The original TUI (`./hello`, 2020) was built with
[tview](https://github.com/rivo/tview) and its source was lost. Both binaries
live in the visitor image so they can be compared: the site pre-fills
`./hello2`, replace it with `./hello` at the prompt.

## stripeek in the sandbox

The container also carries [stripeek](https://github.com/progapandist/stripeek),
a Stripe API/webhook inspector, pinned by version in `Dockerfile.visitor`. There
is no network and no Stripe key, so it runs against a captured session loaded
from `STRIPEEK_HISTORY_PATH` — 65 calls and 35 webhooks, including subscription
schedules and the whole invoice lifecycle.

Stripe object ids embed a fragment derived from the account that made them, so a
raw capture leaks the account id in every id it contains.
`tools/stripeek_fixture.py` rewrites that fragment consistently — ids are opaque
strings, so every cross-reference survives — and refuses to write a fixture that
still contains it.

`visitor/stripeek` is a wrapper, not the binary. On a coarse pointer or under
80 columns it prints a note and exits, because the inspector is wide and
keyboard-driven. `stripeek.bin` is still there for anyone who insists.

## Stack

- Go, [gorilla/mux](https://github.com/gorilla/mux) + [gorilla/websocket](https://github.com/gorilla/websocket), [creack/pty](https://github.com/creack/pty)
- [Xterm.js](https://xtermjs.org) :computer:, built with [Vite](https://vite.dev)
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the TUI :tv:
- Docker :ship: — Docker-in-Docker sidecar per pod
- k3s with [k3sup](https://github.com/alexellis/k3sup) :tomato: on a Digital Ocean droplet

## Running it

```sh
make dev         # the whole site on :4567
make run-tui     # just the TUI, in this terminal
make test
make deploy      # everything: push, apply, roll the pods, reinstall the TUI
make deploy-tui  # only the TUI — no restart, no downtime
```

`make deploy` rolls the pods, which recreates the Docker-in-Docker sidecars.
Their image cache is an `emptyDir`, so the visitor image is wiped every time —
which is why `deploy` ends by running `deploy-tui` to put it back.

Changed only the TUI or its copy? `make deploy-tui`. It copies a fresh binary
into each running pod and rebuilds the visitor image tag in place: no registry,
no restart, live for the next visitor.

`make deploy-tui` cross-compiles to `build/`, not `dist/`. `dist/` is the
frontend bundle; the TUI binary is linux/amd64 and will not run on your Mac —
use `make run-tui`.

Deploys need `KUBECONFIG` pointed at the k3s cluster (defaults to
`~/kubeconfig`).

## Keys

`↑`/`↓` navigate · `enter` read (scroll the pane) · `esc` back to menu · `q` quit

## Why?

This started as research into scalable online coding environments for
programming students at [Le Wagon](https://www.lewagon.com). It stuck around
as a portfolio.

## Other OSS projects

- [lewagon/wait-on-check-action](https://github.com/lewagon/wait-on-check-action), the GitHub Action that can be used to halt any workflow until required checks for a given ref pass successfully.
- [lewagon/foot_traffic](https://github.com/lewagon/foot_traffic), pure Ruby DSL for Chrome scripting based on Ferrum. No Selenium required. Works from any script. Simulate web app usage scenarios in production or locally.
- [lewagon/quay-github-actions-dispatch](https://github.com/lewagon/quay-github-actions-dispatch), a tiny web service for securely forwarding Quay build notifications to Github Action's repository_dispatch webhook. A missing link for creating powerful build flows with Quay and GHA.

## Contact me

Andy Baranov — andrey@hey.com

## License

MIT
