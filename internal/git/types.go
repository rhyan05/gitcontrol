package git

import "time"

// FileStatus descreve o estado de um arquivo no working tree.
type FileStatus struct {
	Path        string `json:"path"`
	OldPath     string `json:"old_path,omitempty"`
	IndexXY     string `json:"index_xy"`
	WorktreeXY  string `json:"worktree_xy"`
	IsUntracked bool   `json:"untracked,omitempty"`
	IsIgnored   bool   `json:"ignored,omitempty"`
	IsUnmerged  bool   `json:"unmerged,omitempty"`
}

// StatusResult é o retorno de Status: metadados do branch + arquivos com mudança.
type StatusResult struct {
	Branch   string       `json:"branch"`
	HeadOID  string       `json:"head_oid"`
	Upstream string       `json:"upstream,omitempty"`
	Ahead    int          `json:"ahead,omitempty"`
	Behind   int          `json:"behind,omitempty"`
	Files    []FileStatus `json:"files"`
}

// Commit representa um commit do git.
type Commit struct {
	Hash    string    `json:"hash"`
	Parents []string  `json:"parents,omitempty"`
	Author  string    `json:"author"`
	Time    time.Time `json:"time"`
	Subject string    `json:"subject"`
}

type Branch struct {
	Name     string `json:"name"`
	Upstream string `json:"upstream,omitempty"`
	IsHead   bool   `json:"head,omitempty"`
	IsRemote bool   `json:"remote,omitempty"` // true se começa com "origin/" etc
}
