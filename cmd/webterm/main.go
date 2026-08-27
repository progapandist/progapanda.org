package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"

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
	CheckOrigin:     func(r *http.Request) bool { return true },
}

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
		"--rm",
		"--name",
		name,
		visitorImage(),
		"sh",
	}
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

	initialSize, err := readInitialWindowSize(conn)
	if err != nil {
		l.WithError(err).Error("Unable to read initial terminal size")
		conn.Close()
		return
	}

	containerName := containerNameBasedOnPort(conn.RemoteAddr())
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
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("dist")))

	if err := http.ListenAndServe(":4567", r); err != nil {
		log.WithError(err).Fatal("Something went wrong with the webserver")
	}
}
