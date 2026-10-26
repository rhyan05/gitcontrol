package git

import (
	"bytes"
	"context"
	"fmt"
	"strings"
)

// DiffFile devolve o diff de um arquivo no working tree.
//
// Para arquivos versionados usamos `git diff HEAD`: assim mudanças staged e
// unstaged aparecem juntas, que é o que o usuário espera ao inspecionar o
// arquivo. Para untracked não existe HEAD, então comparamos com /dev/null.
func (r *Repo) DiffFile(ctx context.Context, path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("git diff: path vazio")
	}

	if !r.isTracked(ctx, path) {
		return r.diffUntracked(ctx, path)
	}

	stdout, stderr, err := r.run(ctx, "diff", "HEAD", "--no-ext-diff", "--color=never", "--", path)
	if err != nil {
		// Repo sem nenhum commit: HEAD ainda não existe. Cai para o diff
		// normal (working tree). ponytail: index não é incluído aqui; juntar
		// `git diff --cached` se repo sem commit virar caso real.
		stdout, stderr, err = r.run(ctx, "diff", "--no-ext-diff", "--color=never", "--", path)
		if err != nil {
			return "", fmt.Errorf("git diff %s: %w (stderr: %s)", path, err, stderr)
		}
	}

	return string(stdout), nil
}

// diffUntracked compara o arquivo com /dev/null usando --no-index.
//
// Esse modo do git usa exit code 1 quando encontra diferenças (é o
// comportamento normal, não um erro). Só tratamos como erro quando não veio
// nenhum diff e o comando falhou de verdade (ex.: arquivo inexistente).
func (r *Repo) diffUntracked(ctx context.Context, path string) (string, error) {
	// Git for Windows reconhece /dev/null; os.DevNull ("NUL") não serve aqui.
	stdout, stderr, err := r.run(ctx, "diff", "--no-index", "--no-ext-diff", "--color=never", "--", "/dev/null", path)
	if err != nil && len(bytes.TrimSpace(stdout)) == 0 {
		return "", fmt.Errorf("git diff --no-index %s: %w (stderr: %s)", path, err, stderr)
	}
	return string(stdout), nil
}

// isTracked diz se o path já está no index. Usado para escolher entre o diff
// contra o HEAD e o diff contra /dev/null.
func (r *Repo) isTracked(ctx context.Context, path string) bool {
	_, _, err := r.run(ctx, "ls-files", "--error-unmatch", "--", path)
	return err == nil
}
