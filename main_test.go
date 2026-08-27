package main

import (
	"reflect"
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
