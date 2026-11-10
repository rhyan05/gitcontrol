package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBranchCreateDeleteRename(t *testing.T) {
	ctx := context.Background()
	r := setupRepo(t)
	writeAndCommit(t, r, "a.txt", "v1", "primeiro")

	if err := r.CreateBranch(ctx, "feature", ""); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}
	if err := r.RenameBranch(ctx, "feature", "feature-2"); err != nil {
		t.Fatalf("RenameBranch: %v", err)
	}
	if err := r.DeleteBranch(ctx, "feature-2", false); err != nil {
		t.Fatalf("DeleteBranch: %v", err)
	}

	branches, err := r.Branches(ctx)
	if err != nil {
		t.Fatalf("Branches: %v", err)
	}
	for _, branch := range branches {
		if branch.Name == "feature" || branch.Name == "feature-2" {
			t.Fatalf("branch não foi removida: %+v", branches)
		}
	}
}

func TestStashPushListPop(t *testing.T) {
	ctx := context.Background()
	r := setupRepo(t)
	writeAndCommit(t, r, "a.txt", "v1", "primeiro")

	if err := os.WriteFile(filepath.Join(r.path, "a.txt"), []byte("v2"), 0o644); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	if err := r.StashPush(ctx, "wip", true); err != nil {
		t.Fatalf("StashPush: %v", err)
	}

	stashes, err := r.StashList(ctx)
	if err != nil {
		t.Fatalf("StashList: %v", err)
	}
	if len(stashes) != 1 || stashes[0].Index != 0 || !strings.Contains(stashes[0].Message, "wip") {
		t.Fatalf("stashes = %+v", stashes)
	}

	if err := r.StashPop(ctx, 0); err != nil {
		t.Fatalf("StashPop: %v", err)
	}
	stashes, err = r.StashList(ctx)
	if err != nil {
		t.Fatalf("StashList depois: %v", err)
	}
	if len(stashes) != 0 {
		t.Fatalf("stash não foi removido: %+v", stashes)
	}
}

func TestTags(t *testing.T) {
	ctx := context.Background()
	r := setupRepo(t)
	writeAndCommit(t, r, "a.txt", "v1", "primeiro")

	if err := r.CreateTag(ctx, "v1.0.0", "release"); err != nil {
		t.Fatalf("CreateTag: %v", err)
	}
	tags, err := r.Tags(ctx)
	if err != nil {
		t.Fatalf("Tags: %v", err)
	}
	if len(tags) != 1 || tags[0].Name != "v1.0.0" || !tags[0].Annotated {
		t.Fatalf("tags = %+v", tags)
	}

	if err := r.DeleteTag(ctx, "v1.0.0"); err != nil {
		t.Fatalf("DeleteTag: %v", err)
	}
	tags, err = r.Tags(ctx)
	if err != nil {
		t.Fatalf("Tags depois: %v", err)
	}
	if len(tags) != 0 {
		t.Fatalf("tag não foi removida: %+v", tags)
	}
}

func TestResetAndDiscard(t *testing.T) {
	ctx := context.Background()
	r := setupRepo(t)
	writeAndCommit(t, r, "a.txt", "v1", "primeiro")
	writeAndCommit(t, r, "a.txt", "v2", "segundo")

	if err := r.Reset(ctx, "HEAD~1", ResetSoft); err != nil {
		t.Fatalf("Reset soft: %v", err)
	}
	status, err := r.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(status.Files) == 0 {
		t.Fatal("reset soft deveria manter a mudança staged")
	}

	if err := r.Discard(ctx, []string{"a.txt"}); err != nil {
		t.Fatalf("Discard tracked: %v", err)
	}
	if err := os.WriteFile(filepath.Join(r.path, "novo.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("escrever untracked: %v", err)
	}
	if err := r.Discard(ctx, []string{"novo.txt"}); err != nil {
		t.Fatalf("Discard untracked: %v", err)
	}
	if _, err := os.Stat(filepath.Join(r.path, "novo.txt")); !os.IsNotExist(err) {
		t.Fatalf("untracked não foi removido: %v", err)
	}
}

func TestAmend(t *testing.T) {
	ctx := context.Background()
	r := setupRepo(t)
	writeAndCommit(t, r, "a.txt", "v1", "primeiro")

	if err := r.Amend(ctx, "mensagem nova"); err != nil {
		t.Fatalf("Amend: %v", err)
	}
	commits, err := r.Log(ctx, 1)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if commits[0].Subject != "mensagem nova" {
		t.Fatalf("subject = %q", commits[0].Subject)
	}
}

func TestRemotesAddRemove(t *testing.T) {
	ctx := context.Background()
	r := setupRepo(t)

	if err := r.AddRemote(ctx, "origin", "https://example.com/repo.git"); err != nil {
		t.Fatalf("AddRemote: %v", err)
	}
	remotes, err := r.Remotes(ctx)
	if err != nil {
		t.Fatalf("Remotes: %v", err)
	}
	if len(remotes) != 1 || remotes[0].Name != "origin" {
		t.Fatalf("remotes = %+v", remotes)
	}

	if err := r.RemoveRemote(ctx, "origin"); err != nil {
		t.Fatalf("RemoveRemote: %v", err)
	}
	remotes, err = r.Remotes(ctx)
	if err != nil {
		t.Fatalf("Remotes depois: %v", err)
	}
	if len(remotes) != 0 {
		t.Fatalf("remoto não foi removido: %+v", remotes)
	}
}

func TestRebaseConflictAbort(t *testing.T) {
	ctx := context.Background()
	r, base := conflictRepo(t)

	runGit(t, r.path, "checkout", "feature")
	if err := r.Rebase(ctx, base); err == nil {
		t.Fatal("rebase deveria conflitar")
	}

	op, err := r.CurrentOperation(ctx)
	if err != nil {
		t.Fatalf("CurrentOperation: %v", err)
	}
	if op != OperationRebase {
		t.Fatalf("operação = %q, esperava rebase", op)
	}

	if err := r.Abort(ctx); err != nil {
		t.Fatalf("Abort rebase: %v", err)
	}
	op, err = r.CurrentOperation(ctx)
	if err != nil {
		t.Fatalf("CurrentOperation depois: %v", err)
	}
	if op != OperationNone {
		t.Fatalf("operação depois = %q", op)
	}
}
