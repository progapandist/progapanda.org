package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
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
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func containerNameBasedOnPort(ra net.Addr) string {
	if addr, ok := ra.(*net.TCPAddr); ok {
		return fmt.Sprintf("client-%d", addr.Port)
	}
	return "client-0"
}

func runContainer(name string) *exec.Cmd {
	var cmd *exec.Cmd
	// Blocking
	_, err := net.DialTimeout("tcp", "127.0.0.1:2376", 1*time.Second)

	if err != nil {
		fmt.Println(err)
		cmd = exec.Command(
			"echo",
			"Oops, you're out of luck. Don't fret though! Refresh the page to reconnect to progapanda.org...",
		)
		return cmd
	}

	cmd = exec.Command(
		"docker",
		"run",
		"-it",
		"--cpus=.1",
		"--user=1000:1000",
		"--memory=64M",
		"--kernel-memory=32M",
		"--memory-swap=64M",
		"--network",
		"none",
		"--rm",
		"--name",
		name,
		"progapandist/hello",
		"sh",
	)

	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	return cmd
}

func stopContainer(name string) {
	out, _ := exec.Command(
		"docker",
		"stop",
		name,
	).Output()
	log.Printf("Stopped container %s", out)
}

func handleWebsocket(w http.ResponseWriter, r *http.Request) {
	l := log.WithField("remoteaddr", r.RemoteAddr)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		l.WithError(err).Error("Unable to upgrade connection")
		return
	}
	log.Printf("New connection: %v", conn.RemoteAddr())

	containerName := containerNameBasedOnPort(conn.RemoteAddr())
	cmd := runContainer(containerName)

	tty, err := pty.Start(cmd)
	if err != nil {
		l.WithError(err).Error("Unable to start pty/cmd")
		conn.WriteMessage(websocket.BinaryMessage, []byte("\r\nUnable to start session: "+err.Error()))
		conn.Close()
		return
	}

	defer func() {
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
				l.WithError(err).Error("Unable to grab next reader")
				stopContainer(containerName)
				return
			}

			if messageType == websocket.TextMessage {
				l.Warn("Unexpected text message")
				conn.WriteMessage(websocket.TextMessage, []byte("Unexpected text message"))
				continue
			}

			dataTypeBuf := make([]byte, 1)
			read, err := reader.Read(dataTypeBuf)
			if err != nil {
				l.WithError(err).Error("Unable to read message type from reader")
				conn.WriteMessage(websocket.TextMessage, []byte("Unable to read message type from reader"))
				return
			}

			if read != 1 {
				l.WithField("bytes", read).Error("Unexpected number of bytes read")
				return
			}

			switch dataTypeBuf[0] {
			// It's a binary data message
			case 0:
				copied, err := io.Copy(tty, reader)
				if err != nil {
					l.WithError(err).Errorf("Error after copying %d bytes", copied)
				}
			case 1:
				decoder := json.NewDecoder(reader)
				resizeMessage := windowSize{}
				err := decoder.Decode(&resizeMessage)
				if err != nil {
					conn.WriteMessage(websocket.TextMessage, []byte("Error decoding resize message: "+err.Error()))
					continue
				}
				log.WithField("resizeMessage", resizeMessage).Info("Resizing terminal")
				err = pty.Setsize(tty, &pty.Winsize{
					Rows: resizeMessage.Rows,
					Cols: resizeMessage.Cols,
				})
				if err != nil {
					l.WithError(err).Error("Unable to resize terminal")
				}
			default:
				l.WithField("dataType", dataTypeBuf[0]).Error("Unknown data type")
			}
		}
	}()

	// Constantly read from process and copy to websocket.
	if err := copyOutput(conn, tty); err != nil {
		l.WithError(err).Error("Terminal output stream ended")
		stopContainer(containerName)
	}
}

type messageWriter interface {
	WriteMessage(messageType int, data []byte) error
}

// copyOutput preserves the PTY byte stream exactly. In particular, it must not
// decode each read independently because UTF-8 characters can span reads.
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
