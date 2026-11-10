package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// conflictRepo monta um merge conflitante e devolve o repo e a branch base.
func conflictRepo(t *testing.T) (*Repo, string) {
	t.Helper()
	r := setupRepo(t)
	writeAndCommit(t, r, "a.txt", "base\n", "base")

	st, err := r.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	base := st.Branch

	runGit(t, r.path, "checkout", "-b", "feature")
	writeAndCommit(t, r, "a.txt", "feature\n", "feature")
	runGit(t, r.path, "checkout", base)
	writeAndCommit(t, r, "a.txt", "main\n", "main")

	return r, base
}

func TestMergeConflictContinue(t *testing.T) {
	ctx := context.Background()
	r, _ := conflictRepo(t)

	if err := r.Merge(ctx, "feature"); err == nil {
		t.Fatal("merge deveria conflitar")
	}

	op, err := r.CurrentOperation(ctx)
	if err != nil {
		t.Fatalf("CurrentOperation: %v", err)
	}
	if op != OperationMerge {
		t.Fatalf("operação = %q, esperava merge", op)
	}

	st, err := r.Status(ctx)
	if err != nil {
		t.Fatalf("Status em conflito: %v", err)
	}
	if st.Operation != OperationMerge {
		t.Errorf("Status.Operation = %q, esperava merge", st.Operation)
	}
	if len(st.Files) == 0 || !st.Files[0].IsUnmerged {
		t.Errorf("esperava arquivo unmerged: %+v", st.Files)
	}

	if err := os.WriteFile(filepath.Join(r.path, "a.txt"), []byte("resolved\n"), 0o644); err != nil {
		t.Fatalf("escrever resolução: %v", err)
	}
	if err := r.Resolve(ctx, []string{"a.txt"}); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if err := r.Continue(ctx); err != nil {
		t.Fatalf("Continue: %v", err)
	}

	op, err = r.CurrentOperation(ctx)
	if err != nil {
		t.Fatalf("CurrentOperation depois: %v", err)
	}
	if op != OperationNone {
		t.Errorf("operação depois = %q, esperava none", op)
	}
}

func TestMergeConflictAbort(t *testing.T) {
	ctx := context.Background()
	r, _ := conflictRepo(t)

	if err := r.Merge(ctx, "feature"); err == nil {
		t.Fatal("merge deveria conflitar")
	}
	if err := r.Abort(ctx); err != nil {
		t.Fatalf("Abort: %v", err)
	}

	op, err := r.CurrentOperation(ctx)
	if err != nil {
		t.Fatalf("CurrentOperation: %v", err)
	}
	if op != OperationNone {
		t.Errorf("operação depois = %q, esperava none", op)
	}
	content, err := os.ReadFile(filepath.Join(r.path, "a.txt"))
	if err != nil {
		t.Fatalf("ler a.txt: %v", err)
	}
	if strings.TrimSpace(string(content)) != "main" {
		t.Errorf("conteúdo = %q, esperava main", content)
	}
}
