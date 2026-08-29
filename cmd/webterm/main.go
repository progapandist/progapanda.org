package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

type windowSize struct {
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     allowedOrigin,
}

// allowedOrigin keeps other people's pages from opening sandboxes here. A
// missing Origin means a non-browser client (curl, a script), which is not the
// cross-site case this guards against, so it is allowed through.
func allowedOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return parsed.Host == r.Host
}

// One container per session, and the dind sidecar has a fixed memory budget to
// share between all of them. Without a cap, opening sockets in a loop is enough
// to take a pod down.
const maxSessions = 8

var sessions = make(chan struct{}, maxSessions)

// takeSession reserves a slot, returning a release function and whether there
// was room.
func takeSession() (func(), bool) {
	select {
	case sessions <- struct{}{}:
		var once sync.Once
		return func() { once.Do(func() { <-sessions }) }, true
	default:
		return func() {}, false
	}
}

// A session holds a container open, so it cannot last forever. The idle limit
// covers a tab left open; the hard limit covers everything else.
const (
	idleTimeout    = 15 * time.Minute
	sessionTimeout = 60 * time.Minute
)

func containerNameBasedOnPort(ra net.Addr) string {
	if addr, ok := ra.(*net.TCPAddr); ok {
		return fmt.Sprintf("client-%d", addr.Port)
	}
	return "client-0"
}

func runContainer(name string, size *pty.Winsize, pointer string) *exec.Cmd {
	// Respect DOCKER_HOST in production and the user's normal Docker context
	// during local development.
	if out, err := exec.Command("docker", "info").CombinedOutput(); err != nil {
		// Keep docker's own stderr: "exit status 1" on its own says nothing
		// about whether the daemon is down, unreachable, or refusing TLS.
		log.WithError(err).Errorf("docker unavailable: %s", bytes.TrimSpace(out))
		return exec.Command(
			"echo",
			"Oops, you're out of luck. Don't fret though! Refresh the page to reconnect to progapanda.org...",
		)
	}

	return dockerRunCommand(name, size, pointer)
}

func dockerRunCommand(name string, size *pty.Winsize, pointer string) *exec.Cmd {
	args := dockerRunArgs(name, size, pointer)
	shellArgs := append(
		[]string{"-c", `exec docker "$@" 2>/dev/null`, "docker"},
		args...,
	)
	return exec.Command("sh", shellArgs...)
}

// coarsePointer reports whether the browser says it has no mouse. Only the
// page can know this — a terminal cannot tell a phone from a small window — so
// it arrives as a query parameter on the WebSocket upgrade.
func coarsePointer(r *http.Request) string {
	if r.URL.Query().Get("pointer") == "coarse" {
		return "coarse"
	}
	return "fine"
}

// visitorImage is what each visitor gets. Overridable so `make dev` can run a
// natively-built image without clobbering the tag that ships to the cluster —
// a dev build is this machine's architecture, and the cluster is amd64.
func visitorImage() string {
	if image := os.Getenv("VISITOR_IMAGE"); image != "" {
		return image
	}
	return "progapandist/hello2"
}

func dockerRunArgs(name string, size *pty.Winsize, pointer string) []string {
	return []string{
		"run",
		"-it",
		// The daemon is remote, so `docker run` only sets the container's TTY
		// size over the API *after* the container has started; anything the
		// container runs immediately still sees 80x24. We know the real size
		// here, so tell it rather than making it guess.
		"-e",
		fmt.Sprintf("COLUMNS=%d", size.Cols),
		"-e",
		fmt.Sprintf("LINES=%d", size.Rows),
		"-e",
		"POINTER=" + pointer,
		"--cpus=.1",
		"--user=1000:1000",
		"--memory=64M",
		"--memory-swap=64M",
		"--network",
		"none",
		// A fork bomb otherwise reaches ~500 host tasks before the memory
		// limit stops it. 64 is far more than the TUI or stripeek need.
		"--pids-limit",
		"64",
		// Writes would otherwise land in the sidecar's emptyDir, and filling
		// it evicts the whole pod. A tmpfs bounds the damage to itself, and
		// the shell only needs /tmp: history and stripeek's working copy.
		"--read-only",
		"--tmpfs",
		"/tmp:rw,noexec,nosuid,nodev,size=16m",
		"--security-opt",
		"no-new-privileges",
		"--cap-drop=ALL",
		"--rm",
		"--name",
		name,
		visitorImage(),
		"sh",
	}
}

// live tracks the containers this process started. --rm only removes a
// container when it exits, and these never do on their own: cleanup is the
// deferred stop below, which a killed process never reaches. Without this,
// Ctrl-C on `make dev` orphans every open session's container indefinitely.
var live = struct {
	sync.Mutex
	names map[string]struct{}
}{names: map[string]struct{}{}}

func trackContainer(name string) {
	live.Lock()
	defer live.Unlock()
	live.names[name] = struct{}{}
}

func forgetContainer(name string) {
	live.Lock()
	defer live.Unlock()
	delete(live.names, name)
}

// takeContainers returns every tracked name and empties the set, so a shutdown
// and a session ending at the same moment cannot both stop the same container.
func takeContainers() []string {
	live.Lock()
	defer live.Unlock()
	names := make([]string, 0, len(live.names))
	for name := range live.names {
		names = append(names, name)
	}
	live.names = map[string]struct{}{}
	return names
}

// stopVisitorContainers stops everything still running, in parallel: each stop
// waits a second for the shell to go away, and doing that in series would make
// shutdown take as long as there are sessions.
func stopVisitorContainers() {
	var wg sync.WaitGroup
	for _, name := range takeContainers() {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			stopContainer(name)
		}(name)
	}
	wg.Wait()
}

func stopContainer(name string) {
	_ = exec.Command(
		"docker",
		"stop",
		"--time",
		"1",
		name,
	).Run()
}

func readInitialWindowSize(conn *websocket.Conn) (*pty.Winsize, error) {
	messageType, reader, err := conn.NextReader()
	if err != nil {
		return nil, err
	}
	if messageType != websocket.BinaryMessage {
		return nil, fmt.Errorf("expected binary terminal-size message")
	}

	var dataType [1]byte
	if _, err := io.ReadFull(reader, dataType[:]); err != nil {
		return nil, err
	}
	if dataType[0] != 1 {
		return nil, fmt.Errorf("expected terminal-size message, got type %d", dataType[0])
	}

	return decodeWindowSize(reader)
}

func decodeWindowSize(reader io.Reader) (*pty.Winsize, error) {
	size := windowSize{}
	if err := json.NewDecoder(reader).Decode(&size); err != nil {
		return nil, err
	}
	if size.Rows == 0 || size.Cols == 0 {
		return nil, fmt.Errorf("terminal dimensions must be positive")
	}
	return &pty.Winsize{Rows: size.Rows, Cols: size.Cols}, nil
}

func handleWebsocket(w http.ResponseWriter, r *http.Request) {
	l := log.WithField("remoteaddr", r.RemoteAddr)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		l.WithError(err).Error("Unable to upgrade connection")
		return
	}
	log.Printf("New connection: %v", conn.RemoteAddr())

	release, ok := takeSession()
	if !ok {
		rejected.Add(1)
		l.Warn("Refused a session: all slots busy")
		conn.WriteMessage(websocket.BinaryMessage, []byte(
			"\r\nEvery sandbox on this server is busy. Refresh in a moment.\r\n"))
		conn.Close()
		return
	}
	defer release()
	served.Add(1)

	// Closing the connection unblocks the read loop below, which tears the
	// container down through the usual path.
	expiry := time.AfterFunc(sessionTimeout, func() {
		l.Info("Session reached the time limit")
		conn.Close()
	})
	defer expiry.Stop()
	conn.SetReadDeadline(time.Now().Add(idleTimeout))

	initialSize, err := readInitialWindowSize(conn)
	if err != nil {
		l.WithError(err).Error("Unable to read initial terminal size")
		conn.Close()
		return
	}

	containerName := containerNameBasedOnPort(conn.RemoteAddr())
	trackContainer(containerName)
	cmd := runContainer(containerName, initialSize, coarsePointer(r))

	tty, err := pty.StartWithSize(cmd, initialSize)
	if err != nil {
		l.WithError(err).Error("Unable to start pty/cmd")
		conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		return
	}

	var stopOnce sync.Once
	stop := func() {
		stopOnce.Do(func() {
			forgetContainer(containerName)
			stopContainer(containerName)
		})
	}
	clientDone := make(chan struct{})
	var clientDoneOnce sync.Once
	markClientDone := func() {
		clientDoneOnce.Do(func() {
			close(clientDone)
		})
	}

	defer func() {
		markClientDone()
		stop()
		cmd.Process.Kill()
		cmd.Process.Wait()
		tty.Close()
		conn.Close()
	}()

	// Constantly read websocket and copy to tty
	go func() {
		for {
			messageType, reader, err := conn.NextReader()
			if err == nil {
				conn.SetReadDeadline(time.Now().Add(idleTimeout))
			}
			if err != nil {
				markClientDone()
				if !websocket.IsCloseError(
					err,
					websocket.CloseNormalClosure,
					websocket.CloseGoingAway,
					websocket.CloseNoStatusReceived,
					websocket.CloseAbnormalClosure,
				) {
					l.WithError(err).Warn("Terminal input closed unexpectedly")
				}
				stop()
				return
			}

			if messageType == websocket.TextMessage {
				l.Warn("Unexpected text message")
				continue
			}

			var dataType [1]byte
			if _, err := io.ReadFull(reader, dataType[:]); err != nil {
				l.WithError(err).Warn("Unable to read terminal message type")
				markClientDone()
				stop()
				return
			}

			switch dataType[0] {
			// It's a binary data message
			case 0:
				copied, err := io.Copy(tty, reader)
				if err != nil {
					l.WithError(err).Warnf("Terminal input closed after %d bytes", copied)
					markClientDone()
					stop()
					return
				}
			case 1:
				resizeMessage, err := decodeWindowSize(reader)
				if err != nil {
					l.WithError(err).Warn("Unable to decode terminal resize message")
					continue
				}
				log.WithField("resizeMessage", resizeMessage).Info("Resizing terminal")
				if err := pty.Setsize(tty, resizeMessage); err != nil {
					l.WithError(err).Error("Unable to resize terminal")
				}
			default:
				l.WithField("dataType", dataType[0]).Error("Unknown data type")
			}
		}
	}()

	// Preserve the PTY byte stream exactly. Xterm owns the stateful UTF-8
	// decoder because a multibyte character may span WebSocket messages.
	if err := copyOutput(conn, tty); err != nil && err != io.EOF {
		select {
		case <-clientDone:
		default:
			l.WithError(err).Warn("Terminal process ended unexpectedly")
		}
	}
	stop()
}

type messageWriter interface {
	WriteMessage(messageType int, data []byte) error
}

func copyOutput(dst messageWriter, src io.Reader) error {
	buf := make([]byte, 32*1024)
	for {
		n, readErr := src.Read(buf)
		if n > 0 {
			if err := dst.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				return err
			}
		}
		if readErr != nil {
			return readErr
		}
	}
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/term", handleWebsocket)
	// Only mounted when a password is set, so a missing secret cannot leave it
	// open by accident.
	if os.Getenv("ADMIN_PASSWORD") != "" {
		r.HandleFunc("/admin", adminHandler)
	}
	// /tja is the same page; the frontend reads the path and launches that
	// program instead of the portfolio TUI.
	r.HandleFunc("/tja", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "dist/index.html")
	})
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("dist")))
	server := &http.Server{Addr: ":4567", Handler: r}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-signals
		log.WithField("signal", sig).Info("Stopping visitor containers before exit")
		stopVisitorContainers()
		server.Close()
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.WithError(err).Fatal("Something went wrong with the webserver")
	}
}
