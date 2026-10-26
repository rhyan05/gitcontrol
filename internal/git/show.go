package git

import (
	"context"
	"fmt"
)

// Show devolve o diff completo de um commit (metadados + patch).
//
// Flags usadas:
//   - --no-ext-diff : ignora diffs externos configurados pelo usuário
//     (ex.: diff-so-fancy). Queremos o patch cru para renderizar na TUI.
//   - --color=never : sem sequências ANSI; a TUI aplica as próprias cores.
//
// O --no-pager global já está em run(); sem ele o git abriria um pager e
// travaria o processo, pois o stdin da TUI está fechado.
func (r *Repo) Show(ctx context.Context, hash string) (string, error) {
	if hash == "" {
		return "", fmt.Errorf("git show: hash vazio")
	}

	stdout, stderr, err := r.run(ctx, "show", "--no-ext-diff", "--color=never", hash)
	if err != nil {
		return "", fmt.Errorf("git show %s: %w (stderr: %s)", hash, err, stderr)
	}

	return string(stdout), nil
}
