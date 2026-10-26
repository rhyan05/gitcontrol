package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rhyan05/gerenciador-git/internal/git"
)

func TestMoveCursor(t *testing.T) {
	tests := []struct {
		name          string
		cursor, delta int
		count, want   int
	}{
		{"desce", 0, 1, 3, 1},
		{"não passa do fim", 2, 1, 3, 2},
		{"não passa do início", 0, -1, 3, 0},
		{"lista vazia", 0, 1, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := moveCursor(tt.cursor, tt.delta, tt.count); got != tt.want {
				t.Fatalf("moveCursor(%d, %d, %d) = %d, esperava %d", tt.cursor, tt.delta, tt.count, got, tt.want)
			}
		})
	}
}

func TestTabCyclesFocus(t *testing.T) {
	m := model{focus: focusStatus}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if got := next.(model).focus; got != focusBranches {
		t.Errorf("tab a partir de Status: focus = %d, esperava %d", got, focusBranches)
	}

	prev, _ := m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if got := prev.(model).focus; got != focusCommits {
		t.Errorf("shift+tab a partir de Status: focus = %d, esperava %d", got, focusCommits)
	}
}

func TestViewShowsPanels(t *testing.T) {
	m := model{loaded: true, width: 120, height: 40}
	view := m.View()

	for _, want := range []string{"== Branches ==", "== Commits ==", "== Status =="} {
		if !strings.Contains(view, want) {
			t.Errorf("View não contém %q", want)
		}
	}
}

// O highlight precisa preencher a largura útil; se não preencher, a barra
// de seleção fica "quebrada" no meio do painel.
func TestSelectedLineFillsWidth(t *testing.T) {
	m := model{width: 80}

	got := lipgloss.Width(m.selectedLine("main"))
	want := m.panelWidth() - 2

	if got != want {
		t.Errorf("largura da linha selecionada = %d, esperava %d", got, want)
	}
}

func TestEnterOnCommitOpensDiff(t *testing.T) {
	m := model{
		repo:    git.New("."),
		focus:   focusCommits,
		commits: []git.Commit{{Hash: "abcdef1234567890"}},
		loaded:  true,
	}

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(model)

	if !got.diffOpen {
		t.Error("enter no commit deveria abrir o painel de diff")
	}
	if got.diffHash != "abcdef1234567890" {
		t.Errorf("diffHash = %q, esperava o hash selecionado", got.diffHash)
	}
	if cmd == nil {
		t.Error("enter no commit deveria disparar um tea.Cmd")
	}
}

func TestEnterOutsideCommitsDoesNothing(t *testing.T) {
	m := model{
		focus:   focusBranches,
		commits: []git.Commit{{Hash: "abcdef"}},
		loaded:  true,
	}

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got := next.(model); got.diffOpen {
		t.Error("enter fora de Commits não deveria abrir o diff")
	}
	if cmd != nil {
		t.Error("enter fora de Commits não deveria disparar comando")
	}
}

func TestEscClosesDiff(t *testing.T) {
	m := model{diffOpen: true, diff: "patch", diffHash: "abcdef", loaded: true}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := next.(model)

	if got.diffOpen || got.diff != "" || got.diffHash != "" {
		t.Errorf("esc deveria fechar e limpar o diff: %+v", got)
	}
}

func TestDiffMsgFillsPanel(t *testing.T) {
	m := model{
		loaded:   true,
		width:    120,
		height:   40,
		diffOpen: true,
		diffHash: "abcdef1234567890",
	}

	next, _ := m.Update(diffMsg{hash: "abcdef1234567890", text: "conteudo do patch"})
	got := next.(model)

	if got.diff != "conteudo do patch" {
		t.Errorf("diff = %q, esperava o texto recebido", got.diff)
	}
	view := got.View()
	for _, want := range []string{"== Diff abcdef1 ==", "conteudo do patch"} {
		if !strings.Contains(view, want) {
			t.Errorf("View não contém %q", want)
		}
	}
}

func TestLateDiffMsgIsIgnored(t *testing.T) {
	m := model{loaded: true}

	next, _ := m.Update(diffMsg{hash: "abcdef", text: "velho"})
	if got := next.(model); got.diff != "" || got.diffOpen {
		t.Errorf("resposta sem painel aberto deveria ser ignorada: %+v", got)
	}
}

// A soma das alturas (painéis + diff + rodapé) precisa fechar exatamente a
// altura do terminal, senão o layout "anda" a cada frame.
func TestViewFitsTerminal(t *testing.T) {
	for _, diffOpen := range []bool{false, true} {
		m := model{
			loaded:   true,
			width:    120,
			height:   40,
			diffOpen: diffOpen,
			diff:     "linha 1\nlinha 2",
			diffHash: "abcdef1234567890",
		}

		view := m.View()
		if got := lipgloss.Height(view); got != m.height {
			t.Errorf("diffOpen=%v: altura = %d, esperava %d", diffOpen, got, m.height)
		}
		if got := lipgloss.Width(view); got != m.width {
			t.Errorf("diffOpen=%v: largura = %d, esperava %d", diffOpen, got, m.width)
		}
	}
}
