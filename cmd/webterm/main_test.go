package main

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/creack/pty"
)

type splitReader struct {
	chunks [][]byte
}

func (r *splitReader) Read(p []byte) (int, error) {
	if len(r.chunks) == 0 {
		return 0, io.EOF
	}
	chunk := r.chunks[0]
	r.chunks = r.chunks[1:]
	return copy(p, chunk), nil
}

type recordingWriter struct {
	data bytes.Buffer
}

func (w *recordingWriter) WriteMessage(_ int, data []byte) error {
	_, err := w.data.Write(data)
	return err
}

func TestDockerRunCommandKeepsVisitorContainerIsolated(t *testing.T) {
	want := []string{
		"run", "-it",
		"-e", "COLUMNS=120",
		"-e", "LINES=42",
		"-e", "POINTER=coarse",
		"--cpus=.1",
		"--user=1000:1000",
		"--memory=64M",
		"--memory-swap=64M",
		"--network", "none",
		"--pids-limit", "64",
		"--read-only",
		"--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=16m",
		"--security-opt", "no-new-privileges",
		"--cap-drop=ALL",
		"--rm",
		"--name", "client-1234",
		"progapandist/hello2",
		"sh",
	}
	size := &pty.Winsize{Rows: 42, Cols: 120}
	if got := dockerRunArgs("client-1234", size, "coarse"); !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected Docker arguments\n got: %#v\nwant: %#v", got, want)
	}

	cmd := dockerRunCommand("client-1234", size, "coarse")
	if cmd.Stderr != nil {
		t.Fatal("command stderr must remain available for PTY attachment")
	}
}

func TestCoarsePointerComesFromTheQueryString(t *testing.T) {
	for query, want := range map[string]string{
		"/term?pointer=coarse": "coarse",
		"/term?pointer=fine":   "fine",
		"/term":                "fine",
		"/term?pointer=bogus":  "fine",
	} {
		r := httptest.NewRequest(http.MethodGet, query, nil)
		if got := coarsePointer(r); got != want {
			t.Errorf("%s: got %q, want %q", query, got, want)
		}
	}
}

func TestVisitorImageIsOverridable(t *testing.T) {
	if got := visitorImage(); got != "progapandist/hello2" {
		t.Errorf("default: got %q", got)
	}
	t.Setenv("VISITOR_IMAGE", "progapandist/hello2:dev")
	if got := visitorImage(); got != "progapandist/hello2:dev" {
		t.Errorf("override: got %q", got)
	}
}

// A shutdown and a session ending together must not both try to stop the same
// container, and nothing may be left behind for the shutdown to miss.
func TestTakeContainersDrainsTheSetOnce(t *testing.T) {
	trackContainer("client-1")
	trackContainer("client-2")
	forgetContainer("client-1")

	got := takeContainers()
	if len(got) != 1 || got[0] != "client-2" {
		t.Fatalf("got %v, want [client-2]", got)
	}
	if again := takeContainers(); len(again) != 0 {
		t.Errorf("second take returned %v, want nothing", again)
	}
}

// Another site must not be able to open sandboxes here, but a non-browser
// client sends no Origin at all and is not the threat.
func TestAllowedOrigin(t *testing.T) {
	for _, tc := range []struct {
		origin, host string
		want         bool
	}{
		{"", "progapanda.org", true},
		{"https://progapanda.org", "progapanda.org", true},
		{"http://localhost:4567", "localhost:4567", true},
		{"https://evil.example", "progapanda.org", false},
		{"https://progapanda.org.evil.example", "progapanda.org", false},
		{"://nonsense", "progapanda.org", false},
	} {
		r := httptest.NewRequest(http.MethodGet, "/term", nil)
		r.Host = tc.host
		if tc.origin != "" {
			r.Header.Set("Origin", tc.origin)
		}
		if got := allowedOrigin(r); got != tc.want {
			t.Errorf("origin %q host %q: got %v, want %v", tc.origin, tc.host, got, tc.want)
		}
	}
}

// The slot has to come back when a session ends, and releasing twice must not
// hand out a slot that was never taken.
func TestSessionSlotsAreBounded(t *testing.T) {
	var releases []func()
	for i := 0; i < maxSessions; i++ {
		release, ok := takeSession()
		if !ok {
			t.Fatalf("slot %d refused, expected %d to be available", i, maxSessions)
		}
		releases = append(releases, release)
	}
	if _, ok := takeSession(); ok {
		t.Fatal("expected the next session to be refused")
	}

	releases[0]()
	releases[0]() // a second release must not free a slot twice
	if _, ok := takeSession(); !ok {
		t.Fatal("expected a slot after releasing one")
	}
	if _, ok := takeSession(); ok {
		t.Fatal("double release handed out a slot that was never taken")
	}
	for _, release := range releases[1:] {
		release()
	}
}

func TestDecodeWindowSize(t *testing.T) {
	size, err := decodeWindowSize(strings.NewReader(`{"rows":42,"cols":120}`))
	if err != nil {
		t.Fatal(err)
	}
	if size.Rows != 42 || size.Cols != 120 {
		t.Fatalf("unexpected terminal size: %#v", size)
	}
}

func TestDecodeWindowSizeRejectsZeroDimensions(t *testing.T) {
	if _, err := decodeWindowSize(strings.NewReader(`{"rows":0,"cols":120}`)); err == nil {
		t.Fatal("expected zero-sized terminal to be rejected")
	}
}

func TestCopyOutputPreservesSplitUTF8(t *testing.T) {
	src := &splitReader{chunks: [][]byte{
		[]byte("left \xe2"),
		[]byte("\x94"),
		[]byte("\x80 right"),
	}}
	dst := &recordingWriter{}

	err := copyOutput(dst, src)
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF, got %v", err)
	}
	if got, want := dst.data.String(), "left ─ right"; got != want {
		t.Fatalf("forwarded %q, want %q", got, want)
	}
}
