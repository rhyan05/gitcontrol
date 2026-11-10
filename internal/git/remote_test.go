package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRemotes(t *testing.T) {
	r := setupRepo(t)
	bare := filepath.Join(t.TempDir(), "remote.git")
	runGit(t, filepath.Dir(bare), "init", "--bare", bare)
	runGit(t, r.path, "remote", "add", "origin", bare)

	remotes, err := r.Remotes(context.Background())
	if err != nil {
		t.Fatalf("Remotes: %v", err)
	}
	if len(remotes) != 1 || remotes[0].Name != "origin" || remotes[0].URL != bare {
		t.Fatalf("remotes = %+v", remotes)
	}
}

func TestPushPullFetchLocal(t *testing.T) {
	ctx := context.Background()
	r := setupRepo(t)
	bare := filepath.Join(t.TempDir(), "remote.git")
	runGit(t, filepath.Dir(bare), "init", "--bare", bare)
	runGit(t, r.path, "remote", "add", "origin", bare)

	writeAndCommit(t, r, "a.txt", "v1", "primeiro")

	if err := r.Push(ctx, "origin", ""); err != nil {
		t.Fatalf("Push inicial: %v", err)
	}
	st, err := r.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if st.Upstream == "" {
		t.Fatal("Push deveria configurar o upstream")
	}

	other := filepath.Join(t.TempDir(), "other")
	runGit(t, filepath.Dir(other), "clone", bare, other)
	runGit(t, other, "config", "user.email", "test@test.com")
	runGit(t, other, "config", "user.name", "Test")
	writeAndCommit(t, &Repo{path: other}, "b.txt", "v2", "segundo")
	runGit(t, other, "push", "origin", "HEAD")

	if err := r.Fetch(ctx, "origin"); err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	st, err = r.Status(ctx)
	if err != nil {
		t.Fatalf("Status depois do fetch: %v", err)
	}
	if st.Behind == 0 {
		t.Fatalf("esperava behind > 0 depois do fetch: %+v", st)
	}

	if err := r.Pull(ctx, "origin", ""); err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if _, err := os.Stat(filepath.Join(r.path, "b.txt")); err != nil {
		t.Fatalf("b.txt não veio no pull: %v", err)
	}
}
