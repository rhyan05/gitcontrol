package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rhyan05/gitcontrol/internal/git"
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
	if got.diffKey != "abcdef1234567890" {
		t.Errorf("diffKey = %q, esperava o hash selecionado", got.diffKey)
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
	m := model{diffOpen: true, diff: "patch", diffKey: "abcdef", diffLabel: "abcdef", diffScroll: 2, loaded: true}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := next.(model)

	if got.diffOpen || got.diff != "" || got.diffKey != "" || got.diffLabel != "" || got.diffScroll != 0 {
		t.Errorf("esc deveria fechar e limpar o diff: %+v", got)
	}
}

func TestDiffMsgFillsPanel(t *testing.T) {
	m := model{
		loaded:    true,
		width:     120,
		height:    40,
		diffOpen:  true,
		diffKey:   "abcdef1234567890",
		diffLabel: "abcdef1",
	}

	next, _ := m.Update(diffMsg{key: "abcdef1234567890", text: "conteudo do patch"})
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

	next, _ := m.Update(diffMsg{key: "abcdef", text: "velho"})
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
			diffKey:  "abcdef1234567890",
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

func TestCheckoutConfirmFlow(t *testing.T) {
	m := model{
		repo:     git.New("."),
		focus:    focusBranches,
		branches: []git.Branch{{Name: "feature"}},
		loaded:   true,
	}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(model)
	if got.mode != modeConfirm {
		t.Fatalf("modo = %v, esperava modeConfirm", got.mode)
	}
	if got.pendingAction != actionCheckout {
		t.Errorf("pendingAction = %v, esperava actionCheckout", got.pendingAction)
	}
	if got.pendingTarget != "feature" {
		t.Errorf("pendingTarget = %q, esperava feature", got.pendingTarget)
	}

	next, cmd := got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	got = next.(model)
	if got.mode != modeNormal {
		t.Errorf("depois de y: modo = %v, esperava modeNormal", got.mode)
	}
	if cmd == nil {
		t.Error("y deveria disparar o checkout")
	}
}

func TestCheckoutCancel(t *testing.T) {
	m := model{
		repo:          git.New("."),
		focus:         focusBranches,
		branches:      []git.Branch{{Name: "feature"}},
		loaded:        true,
		mode:          modeConfirm,
		pendingAction: actionCheckout,
		pendingTarget: "feature",
	}

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	got := next.(model)
	if got.mode != modeNormal || got.pendingTarget != "" {
		t.Errorf("n deveria cancelar: mode=%v target=%q", got.mode, got.pendingTarget)
	}
	if cmd != nil {
		t.Error("n não deveria disparar comando")
	}
}

func TestRemoteBranchLabels(t *testing.T) {
	m := model{
		focus: focusCommits, // evita o highlight de seleção nas branches
		branches: []git.Branch{
			{Name: "main", IsHead: true, Upstream: "origin/main"},
			{Name: "origin/feature", IsRemote: true},
		},
	}

	lines := strings.Join(m.branchLines(), "\n")
	for _, want := range []string{"→ origin/main", "[remota]"} {
		if !strings.Contains(lines, want) {
			t.Errorf("branchLines não contém %q:\n%s", want, lines)
		}
	}
}

func TestRemoteCheckoutFooterMentionsLocalBranch(t *testing.T) {
	m := model{
		focus:    focusBranches,
		branches: []git.Branch{{Name: "origin/feature", IsRemote: true}},
		loaded:   true,
	}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(model)
	if !got.pendingRemote {
		t.Error("pendingRemote deveria ser true")
	}
	if got.pendingAction != actionCheckout {
		t.Errorf("pendingAction = %v, esperava actionCheckout", got.pendingAction)
	}
	if !strings.Contains(got.footerText(), "cria branch local") {
		t.Errorf("footer = %q, esperava mencionar a branch local", got.footerText())
	}
}

func TestMergeConfirmFlow(t *testing.T) {
	m := model{
		repo:     git.New("."),
		focus:    focusBranches,
		branches: []git.Branch{{Name: "main", IsHead: true}, {Name: "feature"}},
		status:   &git.StatusResult{Branch: "main"},
		loaded:   true,
	}
	m.cursor[focusBranches] = 1 // feature

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("M")})
	got := next.(model)

	if got.mode != modeConfirm {
		t.Fatalf("M deveria abrir confirmação: modo = %v", got.mode)
	}
	if got.pendingAction != actionMerge {
		t.Errorf("pendingAction = %v, esperava actionMerge", got.pendingAction)
	}
	if got.pendingTarget != "feature" {
		t.Errorf("pendingTarget = %q, esperava feature", got.pendingTarget)
	}
	if !strings.Contains(got.footerText(), "merge feature em main") {
		t.Errorf("footer = %q", got.footerText())
	}

	next, cmd := got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if cmd == nil {
		t.Error("y deveria disparar o merge")
	}
	if mode := next.(model).mode; mode != modeNormal {
		t.Errorf("modo = %v, esperava modeNormal", mode)
	}
}

func TestCherryPickConfirmFlow(t *testing.T) {
	hash := "abcdef1234567890"
	m := model{
		repo:    git.New("."),
		focus:   focusCommits,
		commits: []git.Commit{{Hash: hash}},
		status:  &git.StatusResult{Branch: "main"},
		loaded:  true,
	}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
	got := next.(model)

	if got.mode != modeConfirm {
		t.Fatalf("C deveria abrir confirmação: modo = %v", got.mode)
	}
	if got.pendingAction != actionCherryPick {
		t.Errorf("pendingAction = %v, esperava actionCherryPick", got.pendingAction)
	}
	if got.pendingTarget != hash {
		t.Errorf("pendingTarget = %q, esperava %q", got.pendingTarget, hash)
	}
	if !strings.Contains(got.footerText(), "cherry-pick abcdef1 em main") {
		t.Errorf("footer = %q", got.footerText())
	}

	next, cmd := got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if cmd == nil {
		t.Error("y deveria disparar o cherry-pick")
	}
	if mode := next.(model).mode; mode != modeNormal {
		t.Errorf("modo = %v, esperava modeNormal", mode)
	}
}

func TestMergeAndCherryPickIgnoredOutsidePanel(t *testing.T) {
	base := model{
		repo:     git.New("."),
		branches: []git.Branch{{Name: "feature"}},
		commits:  []git.Commit{{Hash: "abcdef"}},
		loaded:   true,
	}

	// M fora de branches
	base.focus = focusCommits
	next, _ := base.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("M")})
	if mode := next.(model).mode; mode != modeNormal {
		t.Errorf("M em commits não deveria abrir confirmação")
	}

	// C fora de commits
	base.focus = focusBranches
	next, _ = base.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
	if mode := next.(model).mode; mode != modeNormal {
		t.Errorf("C em branches não deveria abrir confirmação")
	}
}

func TestCommitInputFlow(t *testing.T) {
	m := model{repo: git.New("."), loaded: true}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	got := next.(model)
	if got.mode != modeCommitInput {
		t.Fatalf("c deveria abrir o input: modo = %v", got.mode)
	}

	next, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("oi")})
	got = next.(model)
	if got.commitText != "oi" {
		t.Fatalf("commitText = %q, esperava oi", got.commitText)
	}

	next, _ = got.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	got = next.(model)
	if got.commitText != "o" {
		t.Errorf("backspace: commitText = %q, esperava o", got.commitText)
	}

	next, cmd := got.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got = next.(model)
	if got.mode != modeNormal || cmd == nil || got.commitText != "" {
		t.Errorf("enter deveria enviar: mode=%v cmd=%v text=%q", got.mode, cmd, got.commitText)
	}
}

func TestCommitEmptyMessageKeepsInputOpen(t *testing.T) {
	m := model{repo: git.New("."), loaded: true, mode: modeCommitInput}

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(model)

	if cmd != nil {
		t.Error("mensagem vazia não deveria disparar comando")
	}
	if got.mode != modeCommitInput {
		t.Error("input deveria continuar aberto")
	}
	if got.notice != "mensagem de commit vazia" {
		t.Errorf("notice = %q", got.notice)
	}
}

func TestStageAndUnstageKeys(t *testing.T) {
	for _, key := range []string{"s", "u"} {
		m := model{
			repo:   git.New("."),
			focus:  focusStatus,
			loaded: true,
			status: &git.StatusResult{Files: []git.FileStatus{{Path: "a.txt"}}},
		}

		next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		if cmd == nil {
			t.Errorf("%s no status deveria disparar comando", key)
		}
		if mode := next.(model).mode; mode != modeNormal {
			t.Errorf("%s não deveria mudar de modo", key)
		}
	}
}

func TestEscInInputDoesNotCloseDiff(t *testing.T) {
	m := model{
		loaded:     true,
		mode:       modeCommitInput,
		commitText: "msg",
		diffOpen:   true,
	}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := next.(model)

	if got.mode != modeNormal || got.commitText != "" {
		t.Errorf("esc deveria cancelar o input: mode=%v text=%q", got.mode, got.commitText)
	}
	if !got.diffOpen {
		t.Error("esc no input não deveria fechar o diff")
	}
}

func TestCursorClampedAfterReload(t *testing.T) {
	m := model{loaded: true}
	m.cursor[focusCommits] = 5

	next, _ := m.Update(loadedMsg{commits: []git.Commit{{Hash: "a"}}})
	if got := next.(model).cursor[focusCommits]; got != 0 {
		t.Errorf("cursor = %d, esperava 0 depois do reload", got)
	}
}

func TestDKeyOpensFileDiff(t *testing.T) {
	m := model{
		repo:       git.New("."),
		focus:      focusStatus,
		loaded:     true,
		status:     &git.StatusResult{Files: []git.FileStatus{{Path: "a.txt"}}},
		diffScroll: 3,
	}

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	got := next.(model)

	if cmd == nil {
		t.Fatal("d no status deveria disparar o diff do arquivo")
	}
	if !got.diffOpen {
		t.Error("d deveria abrir o painel de diff")
	}
	if got.diffKey != "a.txt" || got.diffLabel != "a.txt" {
		t.Errorf("diffKey=%q diffLabel=%q, esperava a.txt", got.diffKey, got.diffLabel)
	}
	if got.diffScroll != 0 {
		t.Errorf("scroll = %d, esperava resetar em 0", got.diffScroll)
	}
}

func TestDiffMsgResetsScroll(t *testing.T) {
	m := model{
		loaded:     true,
		diffOpen:   true,
		diffKey:    "abc",
		diffLabel:  "abc",
		diffScroll: 4,
	}

	next, _ := m.Update(diffMsg{key: "abc", text: "linha"})
	if got := next.(model).diffScroll; got != 0 {
		t.Errorf("scroll = %d, esperava 0 ao receber novo diff", got)
	}
}

func TestClampScroll(t *testing.T) {
	tests := []struct {
		name                   string
		offset, total, visible int
		want                   int
	}{
		{"inicio", 0, 10, 5, 0},
		{"meio", 5, 10, 5, 5},
		{"passa do fim", 9, 10, 5, 5},
		{"negativo", -1, 10, 5, 0},
		{"conteudo menor", 3, 3, 5, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clampScroll(tt.offset, tt.total, tt.visible); got != tt.want {
				t.Errorf("clampScroll(%d, %d, %d) = %d, esperava %d", tt.offset, tt.total, tt.visible, got, tt.want)
			}
		})
	}
}

func TestDiffScrollKeys(t *testing.T) {
	m := model{
		loaded:    true,
		width:     120,
		height:    20,
		diffOpen:  true,
		diffKey:   "abc",
		diffLabel: "abc",
		diff:      "a\nb\nc\nd\ne\nf\ng\nh",
	}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	scrolled := next.(model)
	if scrolled.diffScroll == 0 {
		t.Fatal("pgdown deveria rolar o diff")
	}

	next, _ = scrolled.Update(tea.KeyMsg{Type: tea.KeyEnd})
	atEnd := next.(model)
	if atEnd.diffScroll != atEnd.diffMaxScroll() {
		t.Errorf("end: scroll = %d, esperava %d", atEnd.diffScroll, atEnd.diffMaxScroll())
	}

	next, _ = atEnd.Update(tea.KeyMsg{Type: tea.KeyHome})
	if got := next.(model).diffScroll; got != 0 {
		t.Errorf("home: scroll = %d, esperava 0", got)
	}
}
