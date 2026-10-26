package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestLogEmpty verifica que Log em um repo sem commits retorna lista vazia,
// sem erro. Git moderno (2.51+) retorna exit 0 com stdout vazio nesse caso;
// versões antigas retornavam exit 128. O Log trata ambos como lista vazia.
func TestLogEmpty(t *testing.T) {
	r := setupRepo(t)

	got, err := r.Log(context.Background(), 10)
	if err != nil {
		t.Fatalf("Log em repo vazio não deveria dar erro: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("esperava 0 commits, veio %d: %+v", len(got), got)
	}
}

// TestLogOneCommit verifica que um único commit é parseado com todos os
// campos preenchidos (Hash, Author, Subject, Time).
func TestLogOneCommit(t *testing.T) {
	r := setupRepo(t)

	path := filepath.Join(r.path, "arquivo.go")
	if err := os.WriteFile(path, []byte("v1"), 0o644); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	runGit(t, r.path, "add", "arquivo.go")
	runGit(t, r.path, "commit", "-m", "primeiro commit")

	got, err := r.Log(context.Background(), 10)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("esperava 1 commit, veio %d: %+v", len(got), got)
	}

	c := got[0]
	if len(c.Hash) != 40 {
		t.Errorf("Hash = %q (len %d), esperava 40 chars (SHA-1)", c.Hash, len(c.Hash))
	}
	if c.Author != "Test" {
		t.Errorf("Author = %q, esperava %q", c.Author, "Test")
	}
	if c.Subject != "primeiro commit" {
		t.Errorf("Subject = %q, esperava %q", c.Subject, "primeiro commit")
	}
	if c.Time.IsZero() {
		t.Errorf("Time está zero, esperava valor")
	}
}

// TestLogMultipleCommits verifica que múltiplos commits são retornados em
// ordem cronológica reversa (mais recente primeiro), que é o que git log
// faz por padrão.
func TestLogMultipleCommits(t *testing.T) {
	r := setupRepo(t)

	path := filepath.Join(r.path, "arquivo.go")

	// 3 commits com conteúdos diferentes (git não commita sem mudança).
	for i, subject := range []string{"a", "b", "c"} {
		content := []byte("v" + string(rune('1'+i)))
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatalf("escrever %s: %v", subject, err)
		}
		runGit(t, r.path, "add", "arquivo.go")
		runGit(t, r.path, "commit", "-m", subject)
	}

	got, err := r.Log(context.Background(), 10)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("esperava 3 commits, veio %d", len(got))
	}

	// git log retorna mais recente primeiro: c, b, a.
	want := []string{"c", "b", "a"}
	for i, w := range want {
		if got[i].Subject != w {
			t.Errorf("got[%d].Subject = %q, esperava %q", i, got[i].Subject, w)
		}
	}
}

// TestLogLimit verifica que o parâmetro limit é respeitado.
func TestLogLimit(t *testing.T) {
	r := setupRepo(t)

	path := filepath.Join(r.path, "arquivo.go")

	// 3 commits.
	for i, subject := range []string{"a", "b", "c"} {
		content := []byte("v" + string(rune('1'+i)))
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatalf("escrever %s: %v", subject, err)
		}
		runGit(t, r.path, "add", "arquivo.go")
		runGit(t, r.path, "commit", "-m", subject)
	}

	got, err := r.Log(context.Background(), 2)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("esperava 2 commits, veio %d", len(got))
	}

	// Os 2 mais recentes são "c" e "b".
	if got[0].Subject != "c" {
		t.Errorf("got[0].Subject = %q, esperava %q", got[0].Subject, "c")
	}
	if got[1].Subject != "b" {
		t.Errorf("got[1].Subject = %q, esperava %q", got[1].Subject, "b")
	}
}

// TestLogParents verifica que Parents é preenchido corretamente:
// root commit (primeiro) tem Parents vazio, filho tem 1 parent.
func TestLogParents(t *testing.T) {
	r := setupRepo(t)

	path := filepath.Join(r.path, "arquivo.go")

	// 2 commits em sequência.
	if err := os.WriteFile(path, []byte("v1"), 0o644); err != nil {
		t.Fatalf("escrever v1: %v", err)
	}
	runGit(t, r.path, "add", "arquivo.go")
	runGit(t, r.path, "commit", "-m", "primeiro")

	if err := os.WriteFile(path, []byte("v2"), 0o644); err != nil {
		t.Fatalf("escrever v2: %v", err)
	}
	runGit(t, r.path, "add", "arquivo.go")
	runGit(t, r.path, "commit", "-m", "segundo")

	got, err := r.Log(context.Background(), 10)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("esperava 2 commits, veio %d", len(got))
	}

	// got[0] = mais recente (segundo), got[1] = mais antigo (primeiro, root).
	if len(got[0].Parents) != 1 {
		t.Errorf("filho: Parents = %v, esperava 1 elemento", got[0].Parents)
	}
	if len(got[1].Parents) != 0 {
		t.Errorf("root: Parents = %v, esperava 0 elementos", got[1].Parents)
	}
}
