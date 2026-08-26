package main

import (
	"bytes"
	"errors"
	"io"
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

func TestCopyOutputPreservesSplitUTF8(t *testing.T) {
	// Split the three-byte box-drawing character across separate PTY reads.
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
