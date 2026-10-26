package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
