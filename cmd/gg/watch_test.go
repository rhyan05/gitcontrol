package main

import (
	"context"
	"os/exec"
	"testing"

	"github.com/rhyan05/gerenciador-git/internal/git"
)

func TestIsGitStateEvent(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/repo/.git/HEAD", true},
		{"/repo/.git/index", true},
		{"/repo/.git/packed-refs", true},
		{"/repo/.git/refs/heads/main", true},
		{"/repo/.git/refs/remotes/origin/main", true},
		{"/repo/.git/objects/ab/cdef", false},
		{"/repo/.git/logs/HEAD", false},
		{"/repo/.git/config", false},
	}
	for _, tt := range tests {
		if got := isGitStateEvent(tt.path); got != tt.want {
			t.Errorf("isGitStateEvent(%q) = %v, esperava %v", tt.path, got, tt.want)
		}
	}
}

// Garante que o watcher consegue observar um repo recém-criado. O teste de
// entrega de eventos fica manual para não depender do timing do SO.
func TestNewGitWatcher(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("git", "init", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	watcher, err := newGitWatcher(context.Background(), git.New(dir))
	if err != nil {
		t.Fatalf("newGitWatcher: %v", err)
	}
	defer watcher.Close()
}
