package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMergeCreatesMergeCommit força um merge não-fast-forward: as duas
// branches avançam, então o git precisa criar um commit de merge.
func TestMergeCreatesMergeCommit(t *testing.T) {
	r := setupRepo(t)
	writeAndCommit(t, r, "base.txt", "base\n", "base")

	st, err := r.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	base := st.Branch

	runGit(t, r.path, "checkout", "-b", "feature")
	writeAndCommit(t, r, "feature.txt", "feature\n", "feature")

	runGit(t, r.path, "checkout", base)
	writeAndCommit(t, r, "main.txt", "main\n", "main")

	if err := r.Merge(context.Background(), "feature"); err != nil {
		t.Fatalf("Merge: %v", err)
	}

	if _, err := os.Stat(filepath.Join(r.path, "feature.txt")); err != nil {
		t.Errorf("feature.txt não existe depois do merge: %v", err)
	}
	if _, err := os.Stat(filepath.Join(r.path, "main.txt")); err != nil {
		t.Errorf("main.txt não existe depois do merge: %v", err)
	}

	commits, err := r.Log(context.Background(), 10)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(commits) == 0 {
		t.Fatal("sem commits depois do merge")
	}
	if len(commits[0].Parents) != 2 {
		t.Errorf("topo do log tem %d parents, esperava 2 (commit de merge): %+v",
			len(commits[0].Parents), commits[0])
	}
}

func TestMergeConflict(t *testing.T) {
	r := setupRepo(t)
	writeAndCommit(t, r, "a.txt", "linha\n", "base")

	st, err := r.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	base := st.Branch

	runGit(t, r.path, "checkout", "-b", "feature")
	writeAndCommit(t, r, "a.txt", "feature\n", "feature")

	runGit(t, r.path, "checkout", base)
	writeAndCommit(t, r, "a.txt", "base\n", "main")

	err = r.Merge(context.Background(), "feature")
	if err == nil {
		t.Fatal("merge com conflito deveria retornar erro")
	}
	if !strings.Contains(err.Error(), "CONFLICT") {
		t.Errorf("erro = %v, esperava mencionar CONFLICT", err)
	}

	st, err = r.Status(context.Background())
	if err != nil {
		t.Fatalf("Status depois: %v", err)
	}
	unmerged := 0
	for _, f := range st.Files {
		if f.IsUnmerged {
			unmerged++
		}
	}
	if unmerged == 0 {
		t.Errorf("status não tem arquivo unmerged: %+v", st.Files)
	}
}

func TestMergeEmptyBranch(t *testing.T) {
	r := setupRepo(t)
	if err := r.Merge(context.Background(), "   "); err == nil {
		t.Fatal("merge com branch vazia deveria retornar erro")
	}
}

func TestCherryPick(t *testing.T) {
	r := setupRepo(t)
	writeAndCommit(t, r, "base.txt", "base\n", "base")

	st, err := r.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	base := st.Branch

	runGit(t, r.path, "checkout", "-b", "feature")
	writeAndCommit(t, r, "feature.txt", "feature\n", "feature commit")

	commits, err := r.Log(context.Background(), 1)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	hash := commits[0].Hash

	runGit(t, r.path, "checkout", base)

	if err := r.CherryPick(context.Background(), hash); err != nil {
		t.Fatalf("CherryPick: %v", err)
	}

	if _, err := os.Stat(filepath.Join(r.path, "feature.txt")); err != nil {
		t.Errorf("feature.txt não existe depois do cherry-pick: %v", err)
	}

	commits, err = r.Log(context.Background(), 1)
	if err != nil {
		t.Fatalf("Log depois: %v", err)
	}
	if commits[0].Subject != "feature commit" {
		t.Errorf("Subject = %q, esperava %q", commits[0].Subject, "feature commit")
	}
	if len(commits[0].Parents) != 1 {
		t.Errorf("parents = %v, esperava 1", commits[0].Parents)
	}
}

func TestCherryPickConflict(t *testing.T) {
	r := setupRepo(t)
	writeAndCommit(t, r, "a.txt", "linha\n", "base")

	st, err := r.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	base := st.Branch

	runGit(t, r.path, "checkout", "-b", "feature")
	writeAndCommit(t, r, "a.txt", "feature\n", "feature")

	commits, err := r.Log(context.Background(), 1)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	hash := commits[0].Hash

	runGit(t, r.path, "checkout", base)
	writeAndCommit(t, r, "a.txt", "base\n", "main")

	err = r.CherryPick(context.Background(), hash)
	if err == nil {
		t.Fatal("cherry-pick com conflito deveria retornar erro")
	}

	st, err = r.Status(context.Background())
	if err != nil {
		t.Fatalf("Status depois: %v", err)
	}
	unmerged := 0
	for _, f := range st.Files {
		if f.IsUnmerged {
			unmerged++
		}
	}
	if unmerged == 0 {
		t.Errorf("status não tem arquivo unmerged: %+v", st.Files)
	}
}

func TestCherryPickEmptyHash(t *testing.T) {
	r := setupRepo(t)
	if err := r.CherryPick(context.Background(), "  "); err == nil {
		t.Fatal("cherry-pick com hash vazio deveria retornar erro")
	}
}
