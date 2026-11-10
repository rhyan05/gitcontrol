package git

import (
	"context"
	"fmt"
	"strings"
)

// Branches retorna todas as branches locais e remotas do repositório.
//
// Usa `git for-each-ref` em vez de `git branch -a` porque:
//   - Evita a linha symref "remotes/origin/HEAD -> origin/main" (que não é
//     uma branch real e complicaria o parser).
//   - Dá controle exato sobre o formato de saída.
//
// Formato de cada linha (4 campos separados por tab):
//
//	<refname>\t<short>\t<upstream>\t<HEAD>
//
// Exemplos:
//
//	refs/heads/main\tmain\torigin/main\t*
//	refs/remotes/origin/main\torigin/main\t\t
//
// Repo sem commits: refs/heads e refs/remotes estão vazios, retorna []Branch{}.
func (r *Repo) Branches(ctx context.Context) ([]Branch, error) {
	stdout, stderr, err := r.run(ctx, "for-each-ref",
		"--format=%(refname)%09%(refname:short)%09%(upstream:short)%09%(HEAD)",
		"refs/heads", "refs/remotes",
	)
	if err != nil {
		return nil, fmt.Errorf("git for-each-ref: %w (stderr: %s)", err, stderr)
	}

	if len(stdout) == 0 {
		return []Branch{}, nil
	}

	branches := []Branch{}
	lines := strings.Split(string(stdout), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}
		b, err := parseBranchLine(line)
		if err != nil {
			return nil, err
		}
		branches = append(branches, b)
	}

	return branches, nil
}

// parseBranchLine parseia uma linha do git for-each-ref no formato:
//
//	<refname>\t<short>\t<upstream>\t<HEAD>
//
// Campos:
//   - refname: caminho completo da ref (refs/heads/main, refs/remotes/origin/main)
//   - short:   nome curto (main, origin/main)
//   - upstream: upstream configurado (origin/main) ou vazio
//   - HEAD:    "*" se é a branch atual, vazio caso contrário
func parseBranchLine(line string) (Branch, error) {
	parts := strings.Split(line, "\t")
	if len(parts) != 4 {
		return Branch{}, fmt.Errorf("invalid branch line (%d parts): %q", len(parts), line)
	}

	refname := parts[0]
	short := parts[1]
	upstream := parts[2]
	headMarker := parts[3]

	if refname == "" {
		return Branch{}, fmt.Errorf("empty refname: %q", line)
	}
	if short == "" {
		return Branch{}, fmt.Errorf("empty short name: %q", line)
	}

	isRemote := strings.HasPrefix(refname, "refs/remotes/")
	isHead := headMarker == "*"

	return Branch{
		Name:     short,
		Upstream: upstream,
		IsHead:   isHead,
		IsRemote: isRemote,
	}, nil
}
