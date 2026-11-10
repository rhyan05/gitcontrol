package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestBranchesEmpty(t *testing.T) {
	r := setupRepo(t)

	got, err := r.Branches(context.Background())
	if err != nil {
		t.Fatalf("Branches em repo vazio não deveria dar erro: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("esperava 0 branches, veio %d: %+v", len(got), got)
	}
}

func TestBranchesOne(t *testing.T) {
	r := setupRepo(t)

	path := filepath.Join(r.path, "arquivo.go")
	if err := os.WriteFile(path, []byte("v1"), 0o644); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	runGit(t, r.path, "add", "arquivo.go")
	runGit(t, r.path, "commit", "-m", "primeiro")

	got, err := r.Branches(context.Background())
	if err != nil {
		t.Fatalf("Branches: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("esperava 1 branch, veio %d: %+v", len(got), got)
	}
	b := got[0]
	if !b.IsHead {
		t.Errorf("IsHead = false, esperava true (é a única branch)")
	}
	if b.Name == "" {
		t.Errorf("Name vazio")
	}
	if b.Upstream != "" {
		t.Errorf("Upstream = %q, esperava vazio (sem remoto)", b.Upstream)
	}
	if b.IsRemote {
		t.Errorf("IsRemote = true, esperava false (branch local)")
	}
}

func TestBranchesMultiple(t *testing.T) {
	r := setupRepo(t)

	path := filepath.Join(r.path, "arquivo.go")
	if err := os.WriteFile(path, []byte("v1"), 0o644); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	runGit(t, r.path, "add", "arquivo.go")
	runGit(t, r.path, "commit", "-m", "primeiro")
	runGit(t, r.path, "checkout", "-b", "feature")

	got, err := r.Branches(context.Background())
	if err != nil {
		t.Fatalf("Branches: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("esperava 2 branches, veio %d: %+v", len(got), got)
	}

	headCount := 0
	names := map[string]bool{}
	for _, b := range got {
		if b.IsHead {
			headCount++
		}
		names[b.Name] = true
	}
	if headCount != 1 {
		t.Errorf("esperava exatamente 1 IsHead, veio %d", headCount)
	}
	if !names["feature"] {
		t.Errorf("branch %q não apareceu nas branches: %+v", "feature", names)
	}
}
