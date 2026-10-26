package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDiffFileModified cobre o caso mais comum: arquivo versionado com
// mudança não staged no working tree.
func TestDiffFileModified(t *testing.T) {
	r := setupRepo(t)
	writeAndCommit(t, r, "a.txt", "v1\n", "primeiro")

	if err := os.WriteFile(filepath.Join(r.path, "a.txt"), []byte("v1\nv2\n"), 0o644); err != nil {
		t.Fatalf("modificar: %v", err)
	}

	got, err := r.DiffFile(context.Background(), "a.txt")
	if err != nil {
		t.Fatalf("DiffFile: %v", err)
	}
	for _, want := range []string{"a.txt", " v1", "+v2"} {
		if !strings.Contains(got, want) {
			t.Errorf("diff não contém %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "new file mode") {
		t.Errorf("arquivo versionado não deveria aparecer como novo:\n%s", got)
	}
}

// TestDiffFileStaged garante que o diff compara com o HEAD, então mudanças
// já adicionadas ao index também aparecem.
func TestDiffFileStaged(t *testing.T) {
	r := setupRepo(t)
	writeAndCommit(t, r, "a.txt", "v1\n", "primeiro")

	if err := os.WriteFile(filepath.Join(r.path, "a.txt"), []byte("v1\nv2\n"), 0o644); err != nil {
		t.Fatalf("modificar: %v", err)
	}
	runGit(t, r.path, "add", "a.txt")

	got, err := r.DiffFile(context.Background(), "a.txt")
	if err != nil {
		t.Fatalf("DiffFile: %v", err)
	}
	if !strings.Contains(got, "+v2") {
		t.Errorf("diff staged não contém +v2:\n%s", got)
	}
}

// TestDiffFileUntracked usa o fallback --no-index contra /dev/null.
func TestDiffFileUntracked(t *testing.T) {
	r := setupRepo(t)

	if err := os.WriteFile(filepath.Join(r.path, "novo.txt"), []byte("conteudo\n"), 0o644); err != nil {
		t.Fatalf("escrever: %v", err)
	}

	got, err := r.DiffFile(context.Background(), "novo.txt")
	if err != nil {
		t.Fatalf("DiffFile untracked: %v", err)
	}
	for _, want := range []string{"novo.txt", "+conteudo"} {
		if !strings.Contains(got, want) {
			t.Errorf("diff untracked não contém %q:\n%s", want, got)
		}
	}
}

// TestDiffFileUnmodified protege contra o falso positivo de tratar um arquivo
// versionado sem mudanças como untracked e imprimir o arquivo inteiro.
func TestDiffFileUnmodified(t *testing.T) {
	r := setupRepo(t)
	writeAndCommit(t, r, "a.txt", "v1\n", "primeiro")

	got, err := r.DiffFile(context.Background(), "a.txt")
	if err != nil {
		t.Fatalf("DiffFile: %v", err)
	}
	if strings.TrimSpace(got) != "" {
		t.Errorf("arquivo sem mudanças deveria ter diff vazio:\n%s", got)
	}
}

func TestDiffFileEmptyPath(t *testing.T) {
	r := setupRepo(t)
	if _, err := r.DiffFile(context.Background(), "   "); err == nil {
		t.Fatal("DiffFile com path vazio deveria retornar erro")
	}
}

func TestGitDir(t *testing.T) {
	r := setupRepo(t)

	dir, err := r.GitDir(context.Background())
	if err != nil {
		t.Fatalf("GitDir: %v", err)
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("GitDir = %q, esperava caminho absoluto", dir)
	}
	if _, err := os.Stat(filepath.Join(dir, "HEAD")); err != nil {
		t.Errorf("GitDir %q não contém HEAD: %v", dir, err)
	}
}
