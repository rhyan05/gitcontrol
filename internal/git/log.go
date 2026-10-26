package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Separador dos campos do --pretty. Usamos o caractere ASCII Unit Separator
// (\x1f) porque ele não aparece em nomes, e-mails ou subjects. Isso evita o
// problema clássico de usar "|" e quebrar quando a mensagem contém "|".
const logSeparator = "\x1f"

// Log retorna os últimos commits do repositório, limitado a `limit`.
//
// Formato pedido ao git (separado por \x1f):
//
//	%H  hash completo
//	%P  hashes dos pais (separados por espaço; vazio no commit raiz)
//	%an nome do autor
//	%aI data em ISO 8601 estrito (RFC3339, com fuso)
//	%s  subject (primeira linha da mensagem)
func (r *Repo) Log(ctx context.Context, limit int) ([]Commit, error) {
	if limit <= 0 {
		limit = 50
	}

	stdout, stderr, err := r.run(ctx, "log",
		"--pretty=format:%H%x1f%P%x1f%an%x1f%aI%x1f%s",
		"-n", strconv.Itoa(limit),
	)
	if err != nil {
		// Repo sem nenhum commit: dependendo da versão do git, `git log` sai
		// com código != 0 e stdout vazio. Isso não é erro para a TUI: a lista
		// simplesmente começa vazia.
		if len(stdout) == 0 && (strings.Contains(string(stderr), "does not have any commits") ||
			strings.Contains(string(stderr), "bad default revision")) {
			return []Commit{}, nil
		}
		return nil, fmt.Errorf("git log: %w (stderr: %s)", err, stderr)
	}

	if len(stdout) == 0 {
		return []Commit{}, nil
	}

	// Cada commit ocupa uma linha; TrimRight remove só o \n final que o git
	// pode acrescentar, sem comer linhas vazias no meio.
	lines := strings.Split(strings.TrimRight(string(stdout), "\n"), "\n")
	commits := make([]Commit, 0, len(lines))
	for _, line := range lines {
		c, err := parseCommitLine(line)
		if err != nil {
			return nil, err
		}
		commits = append(commits, c)
	}

	return commits, nil
}

// parseCommitLine converte uma linha do formato acima em Commit.
func parseCommitLine(line string) (Commit, error) {
	// Subject pode conter qualquer caractere, então SplitN com limite 5
	// garante que só os quatro primeiros separadores sejam consumidos.
	parts := strings.SplitN(line, logSeparator, 5)
	if len(parts) != 5 {
		return Commit{}, fmt.Errorf("invalid commit line (%d parts): %q", len(parts), line)
	}

	t, err := time.Parse(time.RFC3339, parts[3])
	if err != nil {
		return Commit{}, fmt.Errorf("invalid commit date %q: %w", parts[3], err)
	}

	// Commit raiz não tem pai; strings.Fields("") devolve nil, que é o
	// esperado em Commit.Parents.
	var parents []string
	if parts[1] != "" {
		parents = strings.Fields(parts[1])
	}

	return Commit{
		Hash:    parts[0],
		Parents: parents,
		Author:  parts[2],
		Time:    t,
		Subject: parts[4],
	}, nil
}
