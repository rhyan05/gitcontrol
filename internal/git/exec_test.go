package git

import (
    "context"
    "testing"
)

func TestRunGitVersion(t *testing.T) {
    r := &Repo{path: "."}
    stdout, stderr, err := r.run(context.Background(), "version")
    if err != nil {
        t.Fatalf("erro: %v, stderr: %s", err, stderr)
    }
    if len(stdout) == 0 {
        t.Fatalf("stdout vazio")
    }
    t.Logf("git version: %s", stdout)
}