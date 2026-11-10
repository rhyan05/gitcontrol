package git

import (
	"context"
	"testing"
)

func TestGraphEmpty(t *testing.T) {
	r := setupRepo(t)

	rows, err := r.Graph(context.Background(), 10)
	if err != nil {
		t.Fatalf("Graph vazio: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("esperava 0 linhas, veio %d", len(rows))
	}
}

func TestGraphWithMerge(t *testing.T) {
	r := setupRepo(t)
	writeAndCommit(t, r, "a.txt", "base\n", "base")

	st, err := r.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	base := st.Branch

	runGit(t, r.path, "checkout", "-b", "feature")
	writeAndCommit(t, r, "feature.txt", "feature\n", "feature")
	runGit(t, r.path, "checkout", base)
	writeAndCommit(t, r, "main.txt", "main\n", "main")
	runGit(t, r.path, "merge", "--no-edit", "feature")

	rows, err := r.Graph(context.Background(), 10)
	if err != nil {
		t.Fatalf("Graph: %v", err)
	}

	commits := 0
	hasConnector := false
	for _, row := range rows {
		if row.Commit != nil {
			commits++
		}
		if row.Graph == "" {
			continue
		}
		for _, ch := range row.Graph {
			if ch == '|' || ch == '\\' || ch == '/' {
				hasConnector = true
			}
		}
	}
	if commits < 4 {
		t.Errorf("esperava pelo menos 4 commits no grafo, veio %d", commits)
	}
	if !hasConnector {
		t.Errorf("grafo sem conectores: %+v", rows)
	}
}
