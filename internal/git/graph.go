package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// GraphRow é uma linha do grafo. Graph contém os conectores ASCII e Commit é
// nil nas linhas que só desenham arestas (ex.: "|\").
type GraphRow struct {
	Graph  string  `json:"graph"`
	Commit *Commit `json:"commit,omitempty"`
}

// Graph devolve o histórico com o grafo ASCII gerado pelo próprio git.
func (r *Repo) Graph(ctx context.Context, limit int) ([]GraphRow, error) {
	if limit <= 0 {
		limit = 50
	}

	stdout, stderr, err := r.run(ctx, "log",
		"--graph", "--no-color",
		"--pretty=format:%H%x1f%P%x1f%an%x1f%aI%x1f%s",
		"-n", strconv.Itoa(limit),
	)
	if err != nil {
		if len(stdout) == 0 {
			return []GraphRow{}, nil
		}
		return nil, fmt.Errorf("git log --graph: %w (stderr: %s)", err, stderr)
	}
	if len(stdout) == 0 {
		return []GraphRow{}, nil
	}

	rows := []GraphRow{}
	for _, line := range strings.Split(strings.TrimRight(string(stdout), "\n"), "\n") {
		sep := strings.Index(line, logSeparator)
		if sep == -1 {
			if graph := strings.TrimRight(line, " "); graph != "" {
				rows = append(rows, GraphRow{Graph: graph})
			}
			continue
		}

		start := sep - 40
		if start < 0 || !isHex40(line[start:sep]) {
			return nil, fmt.Errorf("linha de grafo inválida: %q", line)
		}

		commit, err := parseCommitLine(line[start:])
		if err != nil {
			return nil, err
		}
		rows = append(rows, GraphRow{
			Graph:  strings.TrimRight(line[:start], " "),
			Commit: &commit,
		})
	}

	return rows, nil
}

func isHex40(s string) bool {
	if len(s) != 40 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
