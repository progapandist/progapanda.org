package main

import (
	"reflect"
	"strings"
	"testing"
)

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
