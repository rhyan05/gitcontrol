package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestShow verifica que Show devolve o patch do commit pedido, com o
// cabeçalho do arquivo e a linha adicionada.
func TestShow(t *testing.T) {
	r := setupRepo(t)
	path := filepath.Join(r.path, "arquivo.go")

	if err := os.WriteFile(path, []byte("v1\n"), 0o644); err != nil {
		t.Fatalf("escrever v1: %v", err)
	}
	runGit(t, r.path, "add", "arquivo.go")
	runGit(t, r.path, "commit", "-m", "primeiro")

	if err := os.WriteFile(path, []byte("v1\nv2\n"), 0o644); err != nil {
		t.Fatalf("escrever v2: %v", err)
	}
	runGit(t, r.path, "add", "arquivo.go")
	runGit(t, r.path, "commit", "-m", "segundo")

	commits, err := r.Log(context.Background(), 10)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("esperava 2 commits, veio %d", len(commits))
	}

	got, err := r.Show(context.Background(), commits[0].Hash)
	if err != nil {
		t.Fatalf("Show: %v", err)
	}
	for _, want := range []string{"segundo", "arquivo.go", "+v2"} {
		if !strings.Contains(got, want) {
			t.Errorf("Show não contém %q:\n%s", want, got)
		}
	}
}

// TestShowInvalid verifica que hash inexistente devolve erro claro.
func TestShowInvalid(t *testing.T) {
	r := setupRepo(t)

	if _, err := r.Show(context.Background(), "0000000000000000000000000000000000000000"); err == nil {
		t.Fatal("Show com hash inválido deveria retornar erro")
	}
}
