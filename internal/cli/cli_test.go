package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rhyan05/gerenciador-git/internal/git"
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
