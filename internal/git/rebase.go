package git

import (
	"context"
	"fmt"
	"strings"
)

// Rebase reaplica a branch atual sobre upstream.
func (r *Repo) Rebase(ctx context.Context, upstream string) error {
	upstream = strings.TrimSpace(upstream)
	if upstream == "" {
		return fmt.Errorf("git rebase: upstream vazio")
	}

	stdout, stderr, err := r.run(ctx, "rebase", upstream)
	if err != nil {
		return gitError("git rebase "+upstream, stdout, stderr, err)
	}
	return nil
}
