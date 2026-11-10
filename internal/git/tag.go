package git

import (
	"context"
	"fmt"
	"strings"
)

// Tag representa uma tag do repositório.
type Tag struct {
	Name      string `json:"name"`
	Hash      string `json:"hash"`
	Message   string `json:"message,omitempty"`
	Annotated bool   `json:"annotated,omitempty"`
}

// Tags lista tags locais.
func (r *Repo) Tags(ctx context.Context) ([]Tag, error) {
	stdout, stderr, err := r.run(ctx, "for-each-ref",
		"--format=%(refname:short)%09%(objectname)%09%(objecttype)%09%(contents:subject)",
		"refs/tags",
	)
	if err != nil {
		return nil, fmt.Errorf("git for-each-ref refs/tags: %w (stderr: %s)", err, stderr)
	}
	if len(stdout) == 0 {
		return []Tag{}, nil
	}

	tags := []Tag{}
	for _, line := range strings.Split(strings.TrimRight(string(stdout), "\n"), "\n") {
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) != 4 {
			return nil, fmt.Errorf("linha de tag inválida: %q", line)
		}
		tags = append(tags, Tag{
			Name:      parts[0],
			Hash:      parts[1],
			Annotated: parts[2] == "tag",
			Message:   parts[3],
		})
	}
	return tags, nil
}

// CreateTag cria uma tag no HEAD. Com mensagem, cria tag anotada.
func (r *Repo) CreateTag(ctx context.Context, name, message string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("git tag: nome vazio")
	}

	args := []string{"tag"}
	if message = strings.TrimSpace(message); message != "" {
		args = append(args, "-a", name, "-m", message)
	} else {
		args = append(args, name)
	}

	stdout, stderr, err := r.run(ctx, args...)
	if err != nil {
		return gitError("git tag "+name, stdout, stderr, err)
	}
	return nil
}

// DeleteTag remove uma tag local.
func (r *Repo) DeleteTag(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("git tag -d: nome vazio")
	}

	stdout, stderr, err := r.run(ctx, "tag", "-d", name)
	if err != nil {
		return gitError("git tag -d "+name, stdout, stderr, err)
	}
	return nil
}
