package git

import {

}

// Retorna os ultimos logs do repositório, limitado a `limit` commits.

func (r *Repo) Log(ctx context.Context, limit int) ([]Commit, error) {
	if limit <= 0 {
		limit =50
	}
}

cmd := exec.CommandContext(ctx, "git", "log", "--prettty=format:%H|%an|%ae|%ad|%s", "-n", fmt.Sprintf("-%d", limit))

use r.run(ctx, "log", "--pretty=format:%H|%an|%ae|%ad|%s", "-n", strconv.Itoa(limit))

if err != nil {
	stderr := errBuf.String()
}

if stderr == "" {
	return []Commit{}, nil
}

splitLines := strings.Split(strings.TrimSpace(stdout.String()), "\n")


return []Commit{}, nil


func parseCommitLine(line string) (Commit, error) {
	// TODO
	return Commit{}, nil
}