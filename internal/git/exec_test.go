package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupRepo cria um repositório git temporário já configurado para commitar.
func setupRepo(t *testing.T) *Repo {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
	return New(dir)
}

func TestRunStatus(t *testing.T) {
	repo := setupRepo(t)
	if err := os.WriteFile(filepath.Join(repo.path, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, _, err := repo.run(context.Background(), "status", "--porcelain")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "a.txt") {
		t.Fatalf("status não listou a.txt: %q", out)
	}
}

// runGit roda um comando git no diretório e falha o teste em caso de erro.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestStatusUntracked(t *testing.T) {
	r := setupRepo(t)

	// Cria um arquivo novo, sem git add.
	path := filepath.Join(r.path, "novo.txt")
	if err := os.WriteFile(path, []byte("conteudo"), 0o644); err != nil {
		t.Fatalf("escrever arquivo: %v", err)
	}

	got, err := r.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(got.Files) != 1 {
		t.Fatalf("esperava 1 arquivo, veio %d: %+v", len(got.Files), got.Files)
	}
	fs := got.Files[0]
	if fs.Path != "novo.txt" {
		t.Errorf("path = %q, esperava %q", fs.Path, "novo.txt")
	}
	if !fs.IsUntracked {
		t.Errorf("IsUntracked = false, esperava true")
	}
}
