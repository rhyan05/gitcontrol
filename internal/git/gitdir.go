package git

import (
	"context"
	"fmt"
	"strings"
)

// GitDir devolve o caminho absoluto do diretório .git do repositório.
//
// É usado pelo watcher de arquivos (Bloco 6). --absolute-git-dir resolve
// também repositórios abertos via caminho relativo.
func (r *Repo) GitDir(ctx context.Context) (string, error) {
	stdout, stderr, err := r.run(ctx, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return "", fmt.Errorf("git rev-parse --absolute-git-dir: %w (stderr: %s)", err, stderr)
	}

	dir := strings.TrimSpace(string(stdout))
	if dir == "" {
		return "", fmt.Errorf("git rev-parse: diretório git vazio")
	}
	return dir, nil
}
