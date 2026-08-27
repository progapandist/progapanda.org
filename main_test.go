package main

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
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
		"--cpus=.1",
		"--user=1000:1000",
		"--memory=64M",
		"--memory-swap=64M",
		"--network", "none",
		"--rm",
		"--name", "client-1234",
		"progapandist/hello",
		"sh",
	}
	if got := dockerRunArgs("client-1234"); !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected Docker arguments\n got: %#v\nwant: %#v", got, want)
	}

	cmd := dockerRunCommand("client-1234")
	if cmd.Stderr != nil {
		t.Fatal("command stderr must remain available for PTY attachment")
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
