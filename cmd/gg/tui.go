package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rhyan05/gerenciador-git/internal/git"
)

// Índices dos painéis. A ordem aqui é a ordem em que eles aparecem na tela,
// e é usada tanto pelo foco (tab) quanto pelo cursor (j/k).
const (
	focusBranches = iota // painel esquerdo
	focusCommits         // painel do meio
	focusStatus          // painel direito
	numPanels            // quantidade total; também define o tamanho de cursor
)

// Estilos estáticos da TUI.
//
// lipgloss usa cores ANSI de 256 cores; "39" é azul e "240" é um cinza
// escuro. Não dependemos de truecolor pra funcionar em qualquer terminal.
var (
	// focusedStyle destaca o painel ativo: borda arredondada azul.
	focusedStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("39")).
			Padding(0, 1)

	// unfocusedStyle é a versão apagada do painel: borda cinza.
	unfocusedStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	// titleStyle marca o cabeçalho "== Branches ==" etc.
	titleStyle = lipgloss.NewStyle().Bold(true)

	// footerStyle deixa as teclas de ajuda discretas no rodapé.
	footerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// model é TODO o estado da aplicação. A regra do Bubbletea é simples:
// Update só transforma o model; View só lê o model; I/O vive dentro de tea.Cmd.
type model struct {
	repo *git.Repo

	// Dados carregados do repositório. Ficam no model para View ser pura.
	branches []git.Branch
	commits  []git.Commit
	status   *git.StatusResult

	// Erros separados por fonte: se "git status" falhar, ainda mostramos
	// branches e commits em vez de derrubar a tela inteira.
	branchesErr error
	commitsErr  error
	statusErr   error

	// Navegação: um foco global e um cursor independente por painel.
	focus  int
	cursor [numPanels]int

	// Tamanho atual do terminal. Começa em 80x24 e é atualizado por
	// tea.WindowSizeMsg quando o terminal é redimensionado.
	width, height int
	loaded        bool
}

// loadedMsg é a resposta de loadAll: um "pacote" com tudo que a tela precisa.
// Cada fonte tem seu próprio erro para não perder dados parciais.
type loadedMsg struct {
	branches    []git.Branch
	branchesErr error
	commits     []git.Commit
	commitsErr  error
	status      *git.StatusResult
	statusErr   error
}

// run abre o repositório do diretório atual e inicia o loop do Bubbletea.
//
// WithAltScreen usa a tela alternativa: ao sair, o conteúdo anterior do
// terminal volta intacto (nada de poluir o scrollback com frames da TUI).
func run() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("obter cwd: %w", err)
	}

	p := tea.NewProgram(
		// Tamanho inicial de fallback; o WindowSizeMsg real chega logo depois.
		model{repo: git.New(cwd), width: 80, height: 24},
		tea.WithAltScreen(),
	)
	_, err = p.Run()
	return err
}

// loadAll roda em background (é um tea.Cmd), nunca dentro de Update.
// Faz as três leituras em sequência e devolve tudo num único loadedMsg.
func (m model) loadAll() tea.Msg {
	branches, bErr := m.repo.Branches(context.Background())
	commits, cErr := m.repo.Log(context.Background(), 50)
	status, sErr := m.repo.Status(context.Background())
	return loadedMsg{
		branches:    branches,
		branchesErr: bErr,
		commits:     commits,
		commitsErr:  cErr,
		status:      status,
		statusErr:   sErr,
	}
}

// Init é chamado uma vez pelo Bubbletea. Dispara a primeira carga.
func (m model) Init() tea.Cmd {
	return m.loadAll
}

// Update trata eventos. Deve ser puro: nada de I/O, nada de Print.
// Eventos tratados hoje:
//   - WindowSizeMsg: o terminal mudou de tamanho (recalcula larguras/alturas);
//   - KeyMsg: q sai, tab/shift+tab trocam o foco, j/k movem o cursor;
//   - loadedMsg: chegada dos dados do loadAll.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// View usa m.width/m.height para recalcular tudo no próximo frame.
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.focus = (m.focus + 1) % numPanels
		case "shift+tab":
			// + numPanels antes do % evita módulo negativo em Go
			// (0 - 1 == -1, e -1 % 3 == -1).
			m.focus = (m.focus + numPanels - 1) % numPanels
		case "j":
			m.cursor[m.focus] = moveCursor(m.cursor[m.focus], 1, m.count(m.focus))
		case "k":
			m.cursor[m.focus] = moveCursor(m.cursor[m.focus], -1, m.count(m.focus))
		}
	case loadedMsg:
		// Copia os dados para o model. A partir daqui View já pode renderizar.
		m.branches = msg.branches
		m.commits = msg.commits
		m.status = msg.status
		m.branchesErr = msg.branchesErr
		m.commitsErr = msg.commitsErr
		m.statusErr = msg.statusErr
		m.loaded = true
	}
	return m, nil
}

// View monta a tela: os três painéis lado a lado em cima e o rodapé embaixo.
// É chamada a cada frame, então não faz I/O nem aloca coisas caras.
func (m model) View() string {
	if !m.loaded {
		return "carregando...\n"
	}

	// JoinHorizontal cola os painéis na mesma linha de base (Top).
	// Cada renderPanel devolve uma caixa já com largura/altura fixas.
	h := m.panelHeight()
	panels := lipgloss.JoinHorizontal(lipgloss.Top,
		m.renderPanel(focusBranches, "== Branches ==", m.branchLines(), h),
		m.renderPanel(focusCommits, "== Commits ==", m.commitLines(), h),
		m.renderPanel(focusStatus, "== Status ==", m.statusLines(), h),
	)

	// O rodapé também é truncado para nunca quebrar linha em terminal estreito.
	footer := footerStyle.Render(truncate("q sair │ tab painel │ j/k mover", m.width))

	return lipgloss.JoinVertical(lipgloss.Left, panels, footer)
}

// panelWidth calcula a largura interna de cada painel.
//
// O terminal é dividido em três. Descontamos 2 células por painel (as duas
// bordas verticais) para que a soma das três caixas caiba em m.width.
// O mínimo de 8 evita pânico/divisão estranha em terminal muito estreito.
func (m model) panelWidth() int {
	w := (m.width - 2*numPanels) / numPanels
	if w < 8 {
		w = 8
	}
	return w
}

// panelHeight calcula a altura interna dos painéis.
//
// m.height inclui as bordas de cima/baixo (1 célula cada) e o rodapé (1 linha).
// Logo: altura interna = altura total - 3. Mínimo de 3 para caber título + linhas.
func (m model) panelHeight() int {
	h := m.height - 3
	if h < 3 {
		h = 3
	}
	return h
}

// renderPanel devolve uma caixa pronta (borda + título + linhas).
//
// Regras de layout:
//   - painel focado usa borda azul; os outros, cinza;
//   - Width/Height são fixos para as três caixas alinharem;
//   - truncamos cada linha na largura útil (largura interna menos o padding
//     horizontal de 2) para texto longo não quebrar e empurrar o layout.
func (m model) renderPanel(panel int, title string, lines []string, height int) string {
	style := unfocusedStyle
	if m.focus == panel {
		style = focusedStyle
	}

	// O título ocupa a primeira linha; por isso sobram height-1 para os itens.
	body := titleStyle.Render(title)
	maxLines := height - 1
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	for _, line := range lines {
		body += "\n" + truncate(line, m.panelWidth()-2)
	}

	return style.
		Width(m.panelWidth()).
		Height(height).
		Render(body)
}

// count devolve quantos itens existem no painel. É o limite superior do cursor.
// Para o status contamos só os visíveis (ignorados ficam de fora).
func (m model) count(panel int) int {
	switch panel {
	case focusBranches:
		return len(m.branches)
	case focusCommits:
		return len(m.commits)
	case focusStatus:
		return len(m.visibleStatus())
	}
	return 0
}

// branchLines monta as linhas do painel de branches.
// O marcador ">" só aparece no painel focado; "*" indica a branch atual.
func (m model) branchLines() []string {
	if m.branchesErr != nil {
		return []string{"erro: " + m.branchesErr.Error()}
	}
	if len(m.branches) == 0 {
		return []string{"(nenhuma branch)"}
	}

	lines := make([]string, 0, len(m.branches))
	for i, b := range m.branches {
		name := "  " + b.Name
		if b.IsHead {
			name = "* " + b.Name
		}
		lines = append(lines, cursorMark(i == m.cursor[focusBranches], m.focus == focusBranches)+name)
	}
	return lines
}

// commitLines monta as linhas do painel de commits: hash curto + subject.
func (m model) commitLines() []string {
	if m.commitsErr != nil {
		return []string{"erro: " + m.commitsErr.Error()}
	}
	if len(m.commits) == 0 {
		return []string{"(nenhum commit)"}
	}

	lines := make([]string, 0, len(m.commits))
	for i, c := range m.commits {
		lines = append(lines, cursorMark(i == m.cursor[focusCommits], m.focus == focusCommits)+shortHash(c.Hash)+" "+c.Subject)
	}
	return lines
}

// statusLines monta as linhas do painel de status: código XY + caminho.
func (m model) statusLines() []string {
	if m.statusErr != nil {
		return []string{"erro: " + m.statusErr.Error()}
	}

	files := m.visibleStatus()
	if len(files) == 0 {
		return []string{"(limpo)"}
	}

	lines := make([]string, 0, len(files))
	for i, f := range files {
		lines = append(lines, cursorMark(i == m.cursor[focusStatus], m.focus == focusStatus)+statusCode(f)+" "+f.Path)
	}
	return lines
}

// visibleStatus esconde ignorados: são ruído no painel.
func (m model) visibleStatus() []git.FileStatus {
	if m.status == nil {
		return nil
	}
	files := make([]git.FileStatus, 0, len(m.status.Files))
	for _, f := range m.status.Files {
		if !f.IsIgnored {
			files = append(files, f)
		}
	}
	return files
}

// cursorMark devolve o prefixo da linha. O ">" existe só quando o item está
// selecionado E o painel está focado; assim só há um cursor visível na tela.
func cursorMark(selected, focused bool) string {
	if selected && focused {
		return "> "
	}
	return "  "
}

// statusCode resume o estado do arquivo em duas letras, no estilo do git:
// "??" para untracked, "UU" para conflito e XY para o resto.
func statusCode(f git.FileStatus) string {
	if f.IsUntracked {
		return "??"
	}
	if f.IsUnmerged {
		return "UU"
	}
	return f.IndexXY + f.WorktreeXY
}

// shortHash mostra os 7 primeiros caracteres, como o git faz por padrão.
func shortHash(hash string) string {
	if len(hash) > 7 {
		return hash[:7]
	}
	return hash
}

// moveCursor move o cursor e o prende em [0, count-1].
// Lista vazia (count == 0) sempre devolve 0 para não gerar índice -1.
func moveCursor(cursor, delta, count int) int {
	if count <= 0 {
		return 0
	}
	cursor += delta
	if cursor < 0 {
		return 0
	}
	if cursor >= count {
		return count - 1
	}
	return cursor
}

// truncate corta o texto na largura visual informada.
//
// Usa lipgloss.MaxWidth porque ele mede células (não bytes): acentos e
// caracteres largos não quebram o alinhamento. Também preserva sequências
// ANSI, o que será importante quando as linhas ganharem cor (4c/4d).
func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	return lipgloss.NewStyle().MaxWidth(width).Render(s)
}
