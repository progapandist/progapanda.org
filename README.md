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

The TUI inside the container lives in a separate repo:
[hello2](https://github.com/progapandist/hello2).

Wire format is binary frames, first byte is the type: `0` raw terminal bytes,
`1` a JSON `{"rows":n,"cols":n}` resize. The first message a client sends must
be a resize — the server needs it to size the PTY.

## Stack

- Go, [gorilla/mux](https://github.com/gorilla/mux) + [gorilla/websocket](https://github.com/gorilla/websocket), [creack/pty](https://github.com/creack/pty)
- [Xterm.js](https://xtermjs.org) :computer:, built with [Vite](https://vite.dev)
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the TUI :tv:
- Docker :ship: — Docker-in-Docker sidecar per pod
- k3s with [k3sup](https://github.com/alexellis/k3sup) :tomato: on a Digital Ocean droplet

## Running it

```sh
make dev      # frontend + local visitor image, then serve on :4567
make build    # frontend, linux binary, container image
make deploy   # push, apply, roll the pods, reinstall the TUI
```

`make dev` builds the visitor image from `../hello2`; point `HELLO2_DIR`
elsewhere if it lives somewhere else.

`make deploy` finishes the job: rolling these pods recreates the
Docker-in-Docker sidecars, whose image cache is an `emptyDir`, so the visitor
image is wiped every time. It therefore ends by running `make deploy` in the
hello2 repo to put the TUI back. That is also why `HELLO2_DIR` is checked
before anything is pushed — failing after the roll would leave visitors on the
stale upstream image.

Changed only the TUI or its copy? Deploy from the hello2 repo directly; nothing
here needs to restart.

Deploys need `KUBECONFIG` pointed at the k3s cluster (defaults to
`~/kubeconfig`).

## Why?

This started as research into scalable online coding environments for
programming students at [Le Wagon](https://www.lewagon.com). It stuck around
as a portfolio.

## Other OSS projects

- [lewagon/wait-on-check-action](https://github.com/lewagon/wait-on-check-action), the GitHub Action that can be used to halt any workflow until required checks for a given ref pass successfully.
- [lewagon/foot_traffic](https://github.com/lewagon/foot_traffic), pure Ruby DSL for Chrome scripting based on Ferrum. No Selenium required. Works from any script. Simulate web app usage scenarios in production or locally.
- [lewagon/quay-github-actions-dispatch](https://github.com/lewagon/quay-github-actions-dispatch), a tiny web service for securely forwarding Quay build notifications to Github Action's repository_dispatch webhook. A missing link for creating powerful build flows with Quay and GHA.

## Contact me

andrey@hey.com

## License

MIT
