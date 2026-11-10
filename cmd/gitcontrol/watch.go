package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"

	"github.com/rhyan05/gitcontrol/internal/git"
)

// Bloco 6 — refresh automático.
//
// Dois mecanismos trabalham juntos:
//   - fsnotify observa os arquivos de estado do .git (rápido);
//   - um timer a cada refreshInterval garante atualização mesmo se o
//     watcher falhar ou perder algum evento.
//
// Ambos só produzem mensagens; quem realmente recarrega é o loadAll no Update.
const (
	refreshInterval = 3 * time.Second
	watchDebounce   = 150 * time.Millisecond
)

// tickMsg dispara o refresh periódico de fallback.
type tickMsg struct{}

// refreshMsg indica que o watcher viu uma mudança relevante no .git.
type refreshMsg struct{}

// newGitWatcher cria um watcher para os diretórios que guardam estado do git.
//
// Não basta observar .git/HEAD e .git/index diretamente: o git os substitui
// por rename em várias operações, e o watch no arquivo antigo morreria. Por
// isso observamos os diretórios (.git, .git/refs e subpastas de branches) e
// filtramos os eventos depois.
func newGitWatcher(ctx context.Context, repo *git.Repo) (*fsnotify.Watcher, error) {
	dir, err := repo.GitDir(ctx)
	if err != nil {
		return nil, err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("fsnotify: %w", err)
	}

	// refs/remotes pode não existir em repo sem remoto; os.Stat filtra.
	candidates := []string{
		dir,
		filepath.Join(dir, "refs"),
		filepath.Join(dir, "refs", "heads"),
		filepath.Join(dir, "refs", "remotes"),
	}

	added := 0
	for _, path := range candidates {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		if err := watcher.Add(path); err != nil {
			continue
		}
		added++
	}

	if added == 0 {
		watcher.Close()
		return nil, fmt.Errorf("fsnotify: nenhum diretório observável em %s", dir)
	}
	return watcher, nil
}

// isGitStateEvent filtra o ruído do .git (objects, logs, hooks...).
// HEAD, index e packed-refs mudam em checkout/commit; refs/** em commits e
// atualizações de branch remota.
func isGitStateEvent(path string) bool {
	slash := filepath.ToSlash(path)

	// .git/logs/HEAD e .git/logs/refs/** são reflogs: escrevem junto com as
	// operações, mas não são estado que a TUI precise observar.
	if strings.Contains(slash, "/logs/") {
		return false
	}

	switch filepath.Base(path) {
	case "HEAD", "index", "packed-refs":
		return true
	}

	return strings.Contains(slash, "/refs/heads/") ||
		strings.Contains(slash, "/refs/remotes/")
}

// waitForGitChange devolve um tea.Cmd que bloqueia até um evento relevante.
// Depois do debounce, entrega um refreshMsg e é rearmado pelo Update.
func (m model) waitForGitChange() tea.Cmd {
	watcher := m.watcher
	if watcher == nil {
		return nil
	}

	return func() tea.Msg {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return nil
				}
				if !isGitStateEvent(event.Name) {
					continue
				}
				return debounceGitEvents(watcher)

			case _, ok := <-watcher.Errors:
				if !ok {
					return nil
				}
			}
		}
	}
}

// debounceGitEvents espera um silêncio de watchDebounce antes de avisar.
// Um único `git commit` gera vários eventos (HEAD, index, ref, logs); sem o
// debounce cada um viraria um loadAll separado.
func debounceGitEvents(watcher *fsnotify.Watcher) tea.Msg {
	timer := time.NewTimer(watchDebounce)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			return refreshMsg{}

		case event, ok := <-watcher.Events:
			if !ok {
				return refreshMsg{}
			}
			if !isGitStateEvent(event.Name) {
				continue
			}
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(watchDebounce)

		case _, ok := <-watcher.Errors:
			if !ok {
				return refreshMsg{}
			}
		}
	}
}

// tickCmd agenda o próximo refresh de fallback.
func tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}
