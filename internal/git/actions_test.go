package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// writeAndCommit cria/atualiza um arquivo, faz stage e commit. Atalho para
// montar o estado inicial dos testes sem repetir cinco linhas em cada um.
func writeAndCommit(t *testing.T, r *Repo, name, content, message string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(r.path, name), []byte(content), 0o644); err != nil {
		t.Fatalf("escrever %s: %v", name, err)
	}
	runGit(t, r.path, "add", name)
	runGit(t, r.path, "commit", "-m", message)
}

func TestCheckoutLocal(t *testing.T) {
	r := setupRepo(t)
	writeAndCommit(t, r, "a.txt", "v1", "primeiro")

	st, err := r.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	base := st.Branch
	if base == "" {
		t.Fatal("branch inicial vazia")
	}

	runGit(t, r.path, "checkout", "-b", "feature")

	if err := r.Checkout(context.Background(), base); err != nil {
		t.Fatalf("Checkout local: %v", err)
	}

	st, err = r.Status(context.Background())
	if err != nil {
		t.Fatalf("Status depois do checkout: %v", err)
	}
	if st.Branch != base {
		t.Errorf("branch atual = %q, esperava %q", st.Branch, base)
	}
}

func TestCheckoutRemoteCreatesTracking(t *testing.T) {
	r := setupRepo(t)
	writeAndCommit(t, r, "a.txt", "v1", "primeiro")

	commits, err := r.Log(context.Background(), 1)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("esperava 1 commit, veio %d", len(commits))
	}

	// Simula uma branch remota já buscada, sem precisar de rede.
	runGit(t, r.path, "remote", "add", "origin", r.path)
	runGit(t, r.path, "update-ref", "refs/remotes/origin/feature", commits[0].Hash)

	if err := r.Checkout(context.Background(), "origin/feature"); err != nil {
		t.Fatalf("Checkout remoto: %v", err)
	}

	st, err := r.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if st.Branch != "feature" {
		t.Errorf("branch atual = %q, esperava %q (tracking local)", st.Branch, "feature")
	}

	branches, err := r.Branches(context.Background())
	if err != nil {
		t.Fatalf("Branches: %v", err)
	}
	found := false
	for _, b := range branches {
		if b.Name == "feature" && !b.IsRemote {
			found = true
			if !b.IsHead {
				t.Error("feature local deveria ser HEAD")
			}
			if b.Upstream != "origin/feature" {
				t.Errorf("Upstream = %q, esperava %q", b.Upstream, "origin/feature")
			}
		}
	}
	if !found {
		t.Error("branch local feature não foi criada")
	}
}

func TestCheckoutInvalid(t *testing.T) {
	r := setupRepo(t)
	writeAndCommit(t, r, "a.txt", "v1", "primeiro")

	if err := r.Checkout(context.Background(), "nao-existe"); err == nil {
		t.Fatal("checkout de branch inexistente deveria retornar erro")
	}
}

func TestStage(t *testing.T) {
	r := setupRepo(t)

	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(r.path, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("escrever %s: %v", name, err)
		}
	}

	if err := r.Stage(context.Background(), []string{"a.txt", "b.txt"}); err != nil {
		t.Fatalf("Stage: %v", err)
	}

	st, err := r.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(st.Files) != 2 {
		t.Fatalf("esperava 2 arquivos, veio %d: %+v", len(st.Files), st.Files)
	}
	for _, f := range st.Files {
		if f.IsUntracked {
			t.Errorf("%s continua untracked depois do Stage", f.Path)
		}
		if f.IndexXY != "A" {
			t.Errorf("%s: IndexXY = %q, esperava %q", f.Path, f.IndexXY, "A")
		}
	}
}

func TestStageEmpty(t *testing.T) {
	r := setupRepo(t)
	if err := r.Stage(context.Background(), nil); err == nil {
		t.Fatal("Stage sem paths deveria retornar erro")
	}
}

func TestUnstage(t *testing.T) {
	r := setupRepo(t)
	writeAndCommit(t, r, "a.txt", "v1", "primeiro")

	if err := os.WriteFile(filepath.Join(r.path, "a.txt"), []byte("v2"), 0o644); err != nil {
		t.Fatalf("modificar a.txt: %v", err)
	}
	runGit(t, r.path, "add", "a.txt")

	st, err := r.Status(context.Background())
	if err != nil {
		t.Fatalf("Status antes: %v", err)
	}
	if len(st.Files) != 1 || st.Files[0].IndexXY != "M" {
		t.Fatalf("pré-condição falhou, esperava M staged: %+v", st.Files)
	}

	if err := r.Unstage(context.Background(), []string{"a.txt"}); err != nil {
		t.Fatalf("Unstage: %v", err)
	}

	st, err = r.Status(context.Background())
	if err != nil {
		t.Fatalf("Status depois: %v", err)
	}
	if len(st.Files) != 1 {
		t.Fatalf("esperava 1 arquivo, veio %d: %+v", len(st.Files), st.Files)
	}
	f := st.Files[0]
	if f.IndexXY != "." || f.WorktreeXY != "M" {
		t.Errorf("XY = %q/%q, esperava . / M", f.IndexXY, f.WorktreeXY)
	}
}

func TestUnstageEmpty(t *testing.T) {
	r := setupRepo(t)
	if err := r.Unstage(context.Background(), nil); err == nil {
		t.Fatal("Unstage sem paths deveria retornar erro")
	}
}

func TestCommit(t *testing.T) {
	r := setupRepo(t)

	if err := os.WriteFile(filepath.Join(r.path, "a.txt"), []byte("v1"), 0o644); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	if err := r.Stage(context.Background(), []string{"a.txt"}); err != nil {
		t.Fatalf("Stage: %v", err)
	}
	if err := r.Commit(context.Background(), "primeiro commit"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	commits, err := r.Log(context.Background(), 10)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("esperava 1 commit, veio %d", len(commits))
	}
	if commits[0].Subject != "primeiro commit" {
		t.Errorf("Subject = %q, esperava %q", commits[0].Subject, "primeiro commit")
	}
}

func TestCommitNothingStaged(t *testing.T) {
	r := setupRepo(t)
	writeAndCommit(t, r, "a.txt", "v1", "primeiro")

	if err := r.Commit(context.Background(), "nada staged"); err == nil {
		t.Fatal("Commit sem mudança staged deveria retornar erro")
	}
}

func TestCommitEmptyMessage(t *testing.T) {
	r := setupRepo(t)
	writeAndCommit(t, r, "a.txt", "v1", "primeiro")

	if err := os.WriteFile(filepath.Join(r.path, "a.txt"), []byte("v2"), 0o644); err != nil {
		t.Fatalf("modificar: %v", err)
	}
	runGit(t, r.path, "add", "a.txt")

	if err := r.Commit(context.Background(), "   "); err == nil {
		t.Fatal("Commit com mensagem vazia deveria retornar erro")
	}
}
