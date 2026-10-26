package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestStatusModifiedUnstaged(t *testing.T) {
	r := setupRepo(t)

	path := filepath.Join(r.path, "arquivo.go")
	if err := os.WriteFile(path, []byte("v1"), 0o644); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	runGit(t, r.path, "add", "arquivo.go")
	runGit(t, r.path, "commit", "-m", "primeiro")

	if err := os.WriteFile(path, []byte("v2"), 0o644); err != nil {
		t.Fatalf("modificar: %v", err)
	}

	got, err := r.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(got.Files) != 1 {
		t.Fatalf("esperava 1 arquivo, veio %d: %+v", len(got.Files), got.Files)
	}
	fs := got.Files[0]
	if fs.Path != "arquivo.go" {
		t.Errorf("path = %q, esperava %q", fs.Path, "arquivo.go")
	}
	if fs.IndexXY != "." {
		t.Errorf("IndexXY = %q, esperava %q", fs.IndexXY, ".")
	}
	if fs.WorktreeXY != "M" {
		t.Errorf("WorktreeXY = %q, esperava %q", fs.WorktreeXY, "M")
	}
}

func TestStatusBranchInfo(t *testing.T) {
	r := setupRepo(t)

	path := filepath.Join(r.path, "arquivo.go")
	if err := os.WriteFile(path, []byte("v1"), 0o644); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	runGit(t, r.path, "add", "arquivo.go")
	runGit(t, r.path, "commit", "-m", "primeiro")

	got, err := r.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if got.Branch == "" {
		t.Errorf("Branch vazio, esperava nome de branch")
	}
	if got.HeadOID == "" {
		t.Errorf("HeadOID vazio, esperava hash do commit")
	}
	if len(got.HeadOID) != 40 {
		t.Errorf("HeadOID len = %d, esperava 40 (SHA-1)", len(got.HeadOID))
	}
}

func TestStatusModifiedStaged(t *testing.T) {
	r := setupRepo(t)

	path := filepath.Join(r.path, "arquivo.go")
	if err := os.WriteFile(path, []byte("v1"), 0o644); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	runGit(t, r.path, "add", "arquivo.go")
	runGit(t, r.path, "commit", "-m", "primeiro")

	// Modifica E faz git add.
	if err := os.WriteFile(path, []byte("v2"), 0o644); err != nil {
		t.Fatalf("modificar: %v", err)
	}
	runGit(t, r.path, "add", "arquivo.go")

	got, err := r.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(got.Files) != 1 {
		t.Fatalf("esperava 1 arquivo, veio %d: %+v", len(got.Files), got.Files)
	}
	fs := got.Files[0]
	if fs.IndexXY != "M" {
		t.Errorf("IndexXY = %q, esperava %q", fs.IndexXY, "M")
	}
	if fs.WorktreeXY != "." {
		t.Errorf("WorktreeXY = %q, esperava %q", fs.WorktreeXY, ".")
	}
}

func TestStatusUnmerged(t *testing.T) {
	r := setupRepo(t)

	// 1. Cria arquivo na main e commita.
	path := filepath.Join(r.path, "arquivo.go")
	if err := os.WriteFile(path, []byte("v1\n"), 0o644); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	runGit(t, r.path, "add", "arquivo.go")
	runGit(t, r.path, "commit", "-m", "base")

	// 2. Cria branch 'feature' com versão diferente.
	runGit(t, r.path, "checkout", "-b", "feature")
	if err := os.WriteFile(path, []byte("v2 feature\n"), 0o644); err != nil {
		t.Fatalf("escrever feature: %v", err)
	}
	runGit(t, r.path, "add", "arquivo.go")
	runGit(t, r.path, "commit", "-m", "feature")

	// 3. Volta pra main e muda o mesmo arquivo.
	runGit(t, r.path, "checkout", "master")
	if err := os.WriteFile(path, []byte("v2 master\n"), 0o644); err != nil {
		t.Fatalf("escrever master: %v", err)
	}
	runGit(t, r.path, "add", "arquivo.go")
	runGit(t, r.path, "commit", "-m", "master")

	// 4. Faz merge da feature — VAI dar conflito (não checa erro).
	cmd := exec.Command("git", "merge", "feature")
	cmd.Dir = r.path
	_ = cmd.Run() // exit code != 0, é esperado

	// 5. Status deve reportar o conflito.
	got, err := r.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(got.Files) != 1 {
		t.Fatalf("esperava 1 arquivo em conflito, veio %d: %+v", len(got.Files), got.Files)
	}
	if !got.Files[0].IsUnmerged {
		t.Errorf("IsUnmerged = false, esperava true")
	}
}

func TestStatusMixed(t *testing.T) {
	r := setupRepo(t)

	// b.txt e c.txt entram no commit base.
	for _, nome := range []string{"b.txt", "c.txt"} {
		if err := os.WriteFile(filepath.Join(r.path, nome), []byte("v1"), 0o644); err != nil {
			t.Fatalf("escrever %s: %v", nome, err)
		}
	}
	runGit(t, r.path, "add", ".")
	runGit(t, r.path, "commit", "-m", "base")

	// a.txt: novo (untracked).
	if err := os.WriteFile(filepath.Join(r.path, "a.txt"), []byte("novo"), 0o644); err != nil {
		t.Fatalf("escrever a.txt: %v", err)
	}
	// b.txt: modificado, sem add.
	if err := os.WriteFile(filepath.Join(r.path, "b.txt"), []byte("v2"), 0o644); err != nil {
		t.Fatalf("modificar b.txt: %v", err)
	}
	// c.txt: modificado e adicionado ao index.
	if err := os.WriteFile(filepath.Join(r.path, "c.txt"), []byte("v2"), 0o644); err != nil {
		t.Fatalf("modificar c.txt: %v", err)
	}
	runGit(t, r.path, "add", "c.txt")

	got, err := r.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(got.Files) != 3 {
		t.Fatalf("esperava 3 arquivos, veio %d: %+v", len(got.Files), got.Files)
	}

	// Indexa por path pra não depender da ordem.
	byPath := map[string]FileStatus{}
	for _, f := range got.Files {
		byPath[f.Path] = f
	}

	a, ok := byPath["a.txt"]
	if !ok {
		t.Fatalf("a.txt não apareceu em Files")
	}
	if !a.IsUntracked {
		t.Errorf("a.txt: IsUntracked = false, esperava true")
	}

	b, ok := byPath["b.txt"]
	if !ok {
		t.Fatalf("b.txt não apareceu em Files")
	}
	if b.IndexXY != "." || b.WorktreeXY != "M" {
		t.Errorf("b.txt: XY = %q/%q, esperava ./M", b.IndexXY, b.WorktreeXY)
	}

	c, ok := byPath["c.txt"]
	if !ok {
		t.Fatalf("c.txt não apareceu em Files")
	}
	if c.IndexXY != "M" || c.WorktreeXY != "." {
		t.Errorf("c.txt: XY = %q/%q, esperava M/.", c.IndexXY, c.WorktreeXY)
	}
}
