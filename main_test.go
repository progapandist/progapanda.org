package main

import (
	"reflect"
	"testing"
)

func TestDockerRunCommandKeepsVisitorContainerIsolated(t *testing.T) {
	cmd := dockerRunCommand("client-1234")

	want := []string{
		"docker", "run", "-it",
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
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Fatalf("unexpected Docker command\n got: %#v\nwant: %#v", cmd.Args, want)
	}
	if cmd.Stderr == nil {
		t.Fatal("Docker diagnostics must not be attached to the visitor PTY")
	}
}
