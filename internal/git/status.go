package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

func (r *Repo) Status(ctx context.Context) (*StatusResult, error) {
	stdout, stderr, err := r.run(ctx, "status", "--porcelain=v2", "--branch", "--ignored", "-z")
	if err != nil {
		return nil, fmt.Errorf("git status: %w (stderr: %s)", err, stderr)
	}

	result := &StatusResult{Files: []FileStatus{}}
	entries := strings.Split(string(stdout), "\x00")

	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		if entry == "" {
			continue
		}

		switch entry[0] {
		case '#':
			if err := parseHeader(entry, result); err != nil {
				return nil, err
			}

		case '1':
			fs, err := parseChanged(entry)
			if err != nil {
				return nil, err
			}
			result.Files = append(result.Files, fs)

		case '2':
			if i+1 >= len(entries) {
				return nil, fmt.Errorf("renamed entry without original path")
			}
			fs, err := parseRenamed(entry, entries[i+1])
			if err != nil {
				return nil, err
			}
			result.Files = append(result.Files, fs)
			i++ // consome o caminho original, que vem na entrada seguinte

		case '?':
			fs, err := parseUntracked(entry)
			if err != nil {
				return nil, err
			}
			result.Files = append(result.Files, fs)

		case '!':
			fs, err := parseIgnored(entry)
			if err != nil {
				return nil, err
			}
			result.Files = append(result.Files, fs)

		case 'u':
			fs, err := parseUnmerged(entry)
			if err != nil {
				return nil, err
			}
			result.Files = append(result.Files, fs)

		default:
			return nil, fmt.Errorf("unexpected porcelain line: %q", entry)
		}
	}

	return result, nil
}

// parseHeader preenche result com os metadados do branch a partir de uma
// linha começando com "# ". Headers desconhecidos são ignorados (git pode
// adicionar novos em versões futuras).
func parseHeader(line string, result *StatusResult) error {
	if !strings.HasPrefix(line, "# ") {
		return fmt.Errorf("invalid status header: %q", line)
	}

	resto := strings.TrimPrefix(line, "# ")
	parts := strings.SplitN(resto, " ", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid status header (no value): %q", line)
	}

	key, value := parts[0], parts[1]

	switch key {
	case "branch.oid":
		result.HeadOID = value
	case "branch.head":
		result.Branch = value
	case "branch.upstream":
		result.Upstream = value
	case "branch.ab":
		// Formato: "+<ahead> -<behind>", ex: "+2 -1"
		ab := strings.Fields(value)
		if len(ab) != 2 {
			return fmt.Errorf("invalid branch.ab value: %q", value)
		}
		ahead, err := strconv.Atoi(strings.TrimPrefix(ab[0], "+"))
		if err != nil {
			return fmt.Errorf("invalid ahead value %q: %w", ab[0], err)
		}
		behind, err := strconv.Atoi(strings.TrimPrefix(ab[1], "-"))
		if err != nil {
			return fmt.Errorf("invalid behind value %q: %w", ab[1], err)
		}
		result.Ahead = ahead
		result.Behind = behind
	default:
		// Header desconhecido: ignora, não é erro.
	}

	return nil
}

func parseChanged(line string) (FileStatus, error) {
	fields := strings.SplitN(line, " ", 9)

	if len(fields) != 9 || fields[0] != "1" {
		return FileStatus{}, fmt.Errorf("invalid changed entry: %q", line)
	}

	xy := fields[1]
	if len(xy) != 2 {
		return FileStatus{}, fmt.Errorf("invalid XY status: %q", xy)
	}

	path := fields[8]
	if path == "" {
		return FileStatus{}, fmt.Errorf("empty path in changed entry: %q", line)
	}

	return FileStatus{
		Path:       path,
		IndexXY:    string(xy[0]),
		WorktreeXY: string(xy[1]),
	}, nil
}

func parseRenamed(entry, oldPath string) (FileStatus, error) {
	// Formato: 2 <XY> <sub> <mH> <mI> <mW> <hH> <hI> <X><score> <path>
	// Com -z, <origPath> vem como a próxima entrada separada por NUL.
	fields := strings.SplitN(entry, " ", 10)

	if len(fields) != 10 || fields[0] != "2" {
		return FileStatus{}, fmt.Errorf("invalid renamed entry: %q", entry)
	}

	xy := fields[1]
	if len(xy) != 2 {
		return FileStatus{}, fmt.Errorf("invalid XY status: %q", xy)
	}

	path := fields[9]
	if path == "" || oldPath == "" {
		return FileStatus{}, fmt.Errorf("empty path in renamed entry: %q", entry)
	}

	return FileStatus{
		Path:       path,
		OldPath:    oldPath,
		IndexXY:    string(xy[0]),
		WorktreeXY: string(xy[1]),
	}, nil
}

func parseUntracked(entry string) (FileStatus, error) {
	if !strings.HasPrefix(entry, "? ") {
		return FileStatus{}, fmt.Errorf("invalid untracked entry: %q", entry)
	}

	path := strings.TrimPrefix(entry, "? ")
	if path == "" {
		return FileStatus{}, fmt.Errorf("empty path in untracked entry")
	}

	return FileStatus{
		Path:        path,
		IsUntracked: true,
	}, nil
}

func parseIgnored(entry string) (FileStatus, error) {
	if !strings.HasPrefix(entry, "! ") {
		return FileStatus{}, fmt.Errorf("invalid ignored entry: %q", entry)
	}

	path := strings.TrimPrefix(entry, "! ")
	if path == "" {
		return FileStatus{}, fmt.Errorf("empty path in ignored entry")
	}

	return FileStatus{
		Path:      path,
		IsIgnored: true,
	}, nil
}

func parseUnmerged(entry string) (FileStatus, error) {
	// Formato: u <XY> <sub> <m1> <m2> <m3> <mW> <h1> <h2> <h3> <path>
	fields := strings.SplitN(entry, " ", 11)

	if len(fields) != 11 || fields[0] != "u" {
		return FileStatus{}, fmt.Errorf("invalid unmerged entry: %q", entry)
	}

	xy := fields[1]
	if len(xy) != 2 {
		return FileStatus{}, fmt.Errorf("invalid XY status: %q", xy)
	}

	path := fields[10]
	if path == "" {
		return FileStatus{}, fmt.Errorf("empty path in unmerged entry")
	}

	return FileStatus{
		Path:       path,
		IndexXY:    string(xy[0]),
		WorktreeXY: string(xy[1]),
		IsUnmerged: true,
	}, nil
}
