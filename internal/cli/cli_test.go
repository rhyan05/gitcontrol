package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rhyan05/gitcontrol/internal/git"
)

// response espelha o envelope para os testes. Data fica cru para cada teste
// decodificar no tipo que espera.
type response struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// newRepo cria um repositório temporário e devolve o Repo + o diretório
// (o path interno do core é privado, então os testes precisam do dir).
func newRepo(t *testing.T) (*git.Repo, string) {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
	return git.New(dir), dir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func writeAndCommit(t *testing.T, dir, name, content, message string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("escrever %s: %v", name, err)
	}
	runGit(t, dir, "add", name)
	runGit(t, dir, "commit", "-m", message)
}

// runCLI roda o contrato e garante que a saída é exatamente um objeto JSON.
func runCLI(t *testing.T, repo *git.Repo, args ...string) (response, int) {
	t.Helper()
	var out bytes.Buffer
	code := Run(repo, args, &out)

	var resp response
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("saída não é JSON válido: %v\n%s", err, out.String())
	}
	return resp, code
}

func TestStatus(t *testing.T) {
	r, dir := newRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("escrever: %v", err)
	}

	resp, code := runCLI(t, r, "status")
	if code != 0 || !resp.OK {
		t.Fatalf("status falhou: code=%d resp=%+v", code, resp)
	}

	var st git.StatusResult
	if err := json.Unmarshal(resp.Data, &st); err != nil {
		t.Fatalf("decodificar status: %v", err)
	}
	if len(st.Files) != 1 || st.Files[0].Path != "a.txt" {
		t.Errorf("files = %+v, esperava a.txt", st.Files)
	}
}

func TestLog(t *testing.T) {
	r, dir := newRepo(t)
	writeAndCommit(t, dir, "a.txt", "v1", "primeiro commit")

	resp, code := runCLI(t, r, "log")
	if code != 0 || !resp.OK {
		t.Fatalf("log falhou: code=%d resp=%+v", code, resp)
	}

	var commits []git.Commit
	if err := json.Unmarshal(resp.Data, &commits); err != nil {
		t.Fatalf("decodificar log: %v", err)
	}
	if len(commits) != 1 || commits[0].Subject != "primeiro commit" {
		t.Errorf("commits = %+v", commits)
	}
}

func TestBranches(t *testing.T) {
	r, dir := newRepo(t)
	writeAndCommit(t, dir, "a.txt", "v1", "primeiro")

	resp, code := runCLI(t, r, "branches")
	if code != 0 || !resp.OK {
		t.Fatalf("branches falhou: code=%d resp=%+v", code, resp)
	}

	var branches []git.Branch
	if err := json.Unmarshal(resp.Data, &branches); err != nil {
		t.Fatalf("decodificar branches: %v", err)
	}
	heads := 0
	for _, b := range branches {
		if b.IsHead {
			heads++
		}
	}
	if len(branches) == 0 || heads != 1 {
		t.Errorf("branches = %+v, esperava pelo menos 1 com IsHead", branches)
	}
}

func TestShow(t *testing.T) {
	r, dir := newRepo(t)
	writeAndCommit(t, dir, "a.txt", "v1\n", "primeiro")

	st, err := r.Status(t.Context())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}

	resp, code := runCLI(t, r, "show", st.HeadOID)
	if code != 0 || !resp.OK {
		t.Fatalf("show falhou: code=%d resp=%+v", code, resp)
	}

	var data struct {
		Hash string `json:"hash"`
		Diff string `json:"diff"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("decodificar show: %v", err)
	}
	if data.Hash != st.HeadOID {
		t.Errorf("hash = %q, esperava %q", data.Hash, st.HeadOID)
	}
	if !strings.Contains(data.Diff, "primeiro") || !strings.Contains(data.Diff, "+v1") {
		t.Errorf("diff inesperado:\n%s", data.Diff)
	}
}

func TestUsageErrors(t *testing.T) {
	r, _ := newRepo(t)

	tests := [][]string{
		nil,
		{"desconhecido"},
		{"show"},
		{"checkout"},
		{"merge"},
		{"cherry-pick"},
		{"branch-create"},
		{"branch-delete", "a", "b"},
		{"branch-rename", "a"},
		{"remote-add", "a"},
		{"remote-remove"},
		{"rebase"},
		{"reset"},
		{"stash-pop", "x"},
		{"tag-create"},
		{"tag-delete"},
		{"discard"},
		{"diff"},
		{"fetch", "a", "b"},
		{"pull", "a", "b", "c"},
		{"push", "a", "b", "c"},
		{"resolve"},
		{"abort", "extra"},
		{"continue", "extra"},
		{"graph", "extra"},
		{"remotes", "extra"},
		{"stage"},
		{"unstage"},
		{"commit"},
		{"status", "extra"},
	}
	for _, args := range tests {
		resp, code := runCLI(t, r, args...)
		if code == 0 || resp.OK {
			t.Errorf("args %v deveriam falhar: code=%d resp=%+v", args, code, resp)
		}
		if resp.Error == nil || resp.Error.Code != "usage" {
			t.Errorf("args %v: error = %+v, esperava code=usage", args, resp.Error)
		}
	}
}

func TestBrokenRepo(t *testing.T) {
	dir := t.TempDir()
	// O temp pode estar dentro de um repo (ex.: home do usuário). O ceiling
	// impede o git de subir diretórios e achar esse repo pai.
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir))
	repo := git.New(dir)

	resp, code := runCLI(t, repo, "status")
	if code == 0 || resp.OK {
		t.Fatalf("repo inválido deveria falhar: code=%d resp=%+v", code, resp)
	}
	if resp.Error == nil || resp.Error.Code != "not_a_repo" {
		t.Errorf("error = %+v, esperava code=not_a_repo", resp.Error)
	}
}

func TestCheckoutAction(t *testing.T) {
	r, dir := newRepo(t)
	writeAndCommit(t, dir, "a.txt", "v1", "primeiro")

	st, err := r.Status(t.Context())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	base := st.Branch
	runGit(t, dir, "checkout", "-b", "feature")

	resp, code := runCLI(t, r, "checkout", base)
	if code != 0 || !resp.OK {
		t.Fatalf("checkout falhou: code=%d resp=%+v", code, resp)
	}

	st, err = r.Status(t.Context())
	if err != nil {
		t.Fatalf("Status depois: %v", err)
	}
	if st.Branch != base {
		t.Errorf("branch = %q, esperava %q", st.Branch, base)
	}
}

func TestMergeAction(t *testing.T) {
	r, dir := newRepo(t)
	writeAndCommit(t, dir, "base.txt", "base", "base")

	st, err := r.Status(t.Context())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	base := st.Branch

	runGit(t, dir, "checkout", "-b", "feature")
	writeAndCommit(t, dir, "feature.txt", "feature", "feature")
	runGit(t, dir, "checkout", base)

	resp, code := runCLI(t, r, "merge", "feature")
	if code != 0 || !resp.OK {
		t.Fatalf("merge falhou: code=%d resp=%+v", code, resp)
	}
	if _, err := os.Stat(filepath.Join(dir, "feature.txt")); err != nil {
		t.Errorf("feature.txt não existe depois do merge: %v", err)
	}
}

func TestCherryPickAction(t *testing.T) {
	r, dir := newRepo(t)
	writeAndCommit(t, dir, "base.txt", "base", "base")

	st, err := r.Status(t.Context())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	base := st.Branch

	runGit(t, dir, "checkout", "-b", "feature")
	writeAndCommit(t, dir, "feature.txt", "feature", "feature commit")

	commits, err := r.Log(t.Context(), 1)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	hash := commits[0].Hash

	runGit(t, dir, "checkout", base)

	resp, code := runCLI(t, r, "cherry-pick", hash)
	if code != 0 || !resp.OK {
		t.Fatalf("cherry-pick falhou: code=%d resp=%+v", code, resp)
	}

	commits, err = r.Log(t.Context(), 1)
	if err != nil {
		t.Fatalf("Log depois: %v", err)
	}
	if commits[0].Subject != "feature commit" {
		t.Errorf("Subject = %q, esperava feature commit", commits[0].Subject)
	}
}

func TestStageUnstageCommitActions(t *testing.T) {
	r, dir := newRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("v1"), 0o644); err != nil {
		t.Fatalf("escrever: %v", err)
	}

	// stage
	resp, code := runCLI(t, r, "stage", "a.txt")
	if code != 0 || !resp.OK {
		t.Fatalf("stage falhou: code=%d resp=%+v", code, resp)
	}
	if string(resp.Data) != "null" {
		t.Errorf("data de ação = %s, esperava null", resp.Data)
	}

	st, err := r.Status(t.Context())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(st.Files) != 1 || st.Files[0].IndexXY != "A" {
		t.Fatalf("depois do stage: %+v", st.Files)
	}

	// unstage
	resp, code = runCLI(t, r, "unstage", "a.txt")
	if code != 0 || !resp.OK {
		t.Fatalf("unstage falhou: code=%d resp=%+v", code, resp)
	}

	st, err = r.Status(t.Context())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(st.Files) != 1 || !st.Files[0].IsUntracked {
		t.Fatalf("depois do unstage: %+v", st.Files)
	}

	// stage de novo + commit (mensagem em vários args)
	if resp, code = runCLI(t, r, "stage", "a.txt"); code != 0 || !resp.OK {
		t.Fatalf("segundo stage falhou: %+v", resp)
	}
	if resp, code = runCLI(t, r, "commit", "primeiro", "commit"); code != 0 || !resp.OK {
		t.Fatalf("commit falhou: %+v", resp)
	}

	commits, err := r.Log(t.Context(), 10)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(commits) != 1 || commits[0].Subject != "primeiro commit" {
		t.Errorf("commits = %+v", commits)
	}
}

func TestDiffFile(t *testing.T) {
	r, dir := newRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "novo.txt"), []byte("linha\n"), 0o644); err != nil {
		t.Fatalf("escrever: %v", err)
	}

	resp, code := runCLI(t, r, "diff", "novo.txt")
	if code != 0 || !resp.OK {
		t.Fatalf("diff falhou: code=%d resp=%+v", code, resp)
	}

	var data struct {
		Path string `json:"path"`
		Diff string `json:"diff"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("decodificar diff: %v", err)
	}
	if data.Path != "novo.txt" || !strings.Contains(data.Diff, "+linha") {
		t.Errorf("diff inesperado: %+v", data)
	}
}

func TestGraphCLI(t *testing.T) {
	r, dir := newRepo(t)
	writeAndCommit(t, dir, "a.txt", "v1", "primeiro")

	resp, code := runCLI(t, r, "graph")
	if code != 0 || !resp.OK {
		t.Fatalf("graph falhou: code=%d resp=%+v", code, resp)
	}

	var rows []git.GraphRow
	if err := json.Unmarshal(resp.Data, &rows); err != nil {
		t.Fatalf("decodificar graph: %v", err)
	}
	if len(rows) == 0 || rows[0].Commit == nil || rows[0].Commit.Subject != "primeiro" {
		t.Errorf("graph inesperado: %+v", rows)
	}
}

func TestRemotesCLI(t *testing.T) {
	r, dir := newRepo(t)
	bare := filepath.Join(t.TempDir(), "remote.git")
	runGit(t, filepath.Dir(bare), "init", "--bare", bare)
	runGit(t, dir, "remote", "add", "origin", bare)

	resp, code := runCLI(t, r, "remotes")
	if code != 0 || !resp.OK {
		t.Fatalf("remotes falhou: code=%d resp=%+v", code, resp)
	}

	var remotes []git.Remote
	if err := json.Unmarshal(resp.Data, &remotes); err != nil {
		t.Fatalf("decodificar remotes: %v", err)
	}
	if len(remotes) != 1 || remotes[0].Name != "origin" {
		t.Errorf("remotes = %+v", remotes)
	}
}

func TestBranchActionsCLI(t *testing.T) {
	r, dir := newRepo(t)
	writeAndCommit(t, dir, "a.txt", "v1", "primeiro")

	resp, code := runCLI(t, r, "branch-create", "feature")
	if code != 0 || !resp.OK {
		t.Fatalf("branch-create: %+v", resp)
	}
	if resp, code = runCLI(t, r, "branch-rename", "feature", "feature-2"); code != 0 || !resp.OK {
		t.Fatalf("branch-rename: %+v", resp)
	}
	if resp, code = runCLI(t, r, "branch-delete", "feature-2"); code != 0 || !resp.OK {
		t.Fatalf("branch-delete: %+v", resp)
	}
}

func TestStashCLI(t *testing.T) {
	r, dir := newRepo(t)
	writeAndCommit(t, dir, "a.txt", "v1", "primeiro")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("v2"), 0o644); err != nil {
		t.Fatalf("escrever: %v", err)
	}

	if resp, code := runCLI(t, r, "stash", "wip"); code != 0 || !resp.OK {
		t.Fatalf("stash: %+v", resp)
	}

	resp, code := runCLI(t, r, "stash-list")
	if code != 0 || !resp.OK {
		t.Fatalf("stash-list: %+v", resp)
	}
	var stashes []git.Stash
	if err := json.Unmarshal(resp.Data, &stashes); err != nil {
		t.Fatalf("decodificar stash: %v", err)
	}
	if len(stashes) != 1 {
		t.Fatalf("stashes = %+v", stashes)
	}

	if resp, code = runCLI(t, r, "stash-pop", "0"); code != 0 || !resp.OK {
		t.Fatalf("stash-pop: %+v", resp)
	}
}

func TestTagsCLI(t *testing.T) {
	r, dir := newRepo(t)
	writeAndCommit(t, dir, "a.txt", "v1", "primeiro")

	if resp, code := runCLI(t, r, "tag-create", "v1.0.0", "release"); code != 0 || !resp.OK {
		t.Fatalf("tag-create: %+v", resp)
	}
	resp, code := runCLI(t, r, "tags")
	if code != 0 || !resp.OK {
		t.Fatalf("tags: %+v", resp)
	}
	var tags []git.Tag
	if err := json.Unmarshal(resp.Data, &tags); err != nil {
		t.Fatalf("decodificar tags: %v", err)
	}
	if len(tags) != 1 || tags[0].Name != "v1.0.0" {
		t.Fatalf("tags = %+v", tags)
	}
	if resp, code = runCLI(t, r, "tag-delete", "v1.0.0"); code != 0 || !resp.OK {
		t.Fatalf("tag-delete: %+v", resp)
	}
}

func TestDiscardAmendResetCLI(t *testing.T) {
	r, dir := newRepo(t)
	writeAndCommit(t, dir, "a.txt", "v1", "primeiro")
	writeAndCommit(t, dir, "a.txt", "v2", "segundo")

	if resp, code := runCLI(t, r, "amend", "segundo amend"); code != 0 || !resp.OK {
		t.Fatalf("amend: %+v", resp)
	}
	if resp, code := runCLI(t, r, "reset", "HEAD~1", "soft"); code != 0 || !resp.OK {
		t.Fatalf("reset: %+v", resp)
	}
	if resp, code := runCLI(t, r, "discard", "a.txt"); code != 0 || !resp.OK {
		t.Fatalf("discard: %+v", resp)
	}
}
