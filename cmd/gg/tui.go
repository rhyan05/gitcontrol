package main

import (
	"context"
	"fmt"
	"os"
	"strings"

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

	// Cores por tipo de dado (item 4c):
	//   - hash de commit em cinza, pra não competir com o subject;
	//   - branch atual (HEAD) em verde;
	//   - arquivo untracked em amarelo.
	hashStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	headStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	untrackedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))

	// selectedStyle destaca a linha inteira no painel focado (item 4d).
	// ColorWhitespace(true) é o que faz o fundo cobrir também o padding
	// adicionado por Width, virando uma barra contínua em vez de um bloco
	// só atrás do texto.
	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("237")).
			Foreground(lipgloss.Color("15")).
			Bold(true).
			ColorWhitespace(true)
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

	// Painel de diff do commit selecionado (5a/5b).
	// diffOpen controla a visibilidade; diffHash guarda qual commit foi
	// pedido para descartar respostas atrasadas de outro commit.
	diffOpen bool
	diff     string
	diffErr  error
	diffHash string
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

// diffMsg é a resposta assíncrona de git show para o painel de diff.
type diffMsg struct {
	hash string
	text string
	err  error
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
//   - KeyMsg: q sai, tab/shift+tab trocam o foco, j/k movem o cursor,
//     enter abre o diff do commit e esc fecha;
//   - loadedMsg: chegada dos dados do loadAll;
//   - diffMsg: chegada do diff pedido ao git show.
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
		case "enter":
			// O diff só faz sentido no painel de commits. Nos outros,
			// enter será usado por checkout/stage nos próximos blocos.
			if cmd := m.showSelectedCommit(); cmd != nil {
				return m, cmd
			}
		case "esc":
			// Fechar também limpa diffHash: respostas em voo serão
			// descartadas quando chegarem.
			m.diffOpen = false
			m.diff = ""
			m.diffErr = nil
			m.diffHash = ""
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
	case diffMsg:
		// Descarta resposta atrasada se o painel foi fechado ou se outro
		// commit foi selecionado enquanto este git show rodava.
		if !m.diffOpen || m.diffHash != msg.hash {
			return m, nil
		}
		m.diff = msg.text
		m.diffErr = msg.err
	}
	return m, nil
}

// showSelectedCommit prepara o model e devolve o tea.Cmd que roda git show.
//
// Recebe *model para poder marcar diffOpen/diffHash no model local do Update;
// o I/O em si acontece dentro da closure, nunca aqui.
func (m *model) showSelectedCommit() tea.Cmd {
	if m.focus != focusCommits || len(m.commits) == 0 {
		return nil
	}

	hash := m.commits[m.cursor[focusCommits]].Hash
	m.diffOpen = true
	m.diff = "carregando..."
	m.diffErr = nil
	m.diffHash = hash

	repo := m.repo
	return func() tea.Msg {
		text, err := repo.Show(context.Background(), hash)
		return diffMsg{hash: hash, text: text, err: err}
	}
}

// View monta a tela: os três painéis lado a lado em cima, o diff (quando
// aberto) no meio e o rodapé embaixo. Chamada a cada frame, então não faz I/O.
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

	views := []string{panels}
	help := "q sair │ tab painel │ j/k mover │ enter diff"
	if m.diffOpen {
		views = append(views, m.renderDiffPanel())
		help = "esc fechar │ q sair │ tab painel │ j/k mover"
	}

	// O rodapé também é truncado para nunca quebrar linha em terminal estreito.
	views = append(views, footerStyle.Render(truncate(help, m.width)))

	return lipgloss.JoinVertical(lipgloss.Left, views...)
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
// Sem diff: m.height inclui as bordas de cima/baixo (2) e o rodapé (1),
// logo altura interna = m.height - 3.
//
// Com diff: além das bordas dos painéis (2) e do rodapé (1), entram as duas
// bordas do painel de diff. O espaço restante é dividido com diffHeight().
func (m model) panelHeight() int {
	h := m.height - 3
	if m.diffOpen {
		h = m.height - 5 - m.diffHeight()
	}
	if h < 3 {
		h = 3
	}
	return h
}

// diffHeight calcula a altura interna do painel de diff: ~40% do espaço útil.
// O mínimo de 4 garante título + pelo menos três linhas de patch.
func (m model) diffHeight() int {
	h := (m.height - 5) * 2 / 5
	if h < 4 {
		h = 4
	}
	return h
}

// renderDiffPanel desenha o diff do commit selecionado usando a largura toda.
//
// Mostramos apenas as primeiras linhas que cabem na altura calculada. O diff
// é somente leitura nesta etapa; scroll vem junto com o Bloco 5 completo.
func (m model) renderDiffPanel() string {
	width := m.width - 2 // desconta as duas bordas verticais
	if width < 8 {
		width = 8
	}
	height := m.diffHeight()

	title := "== Diff =="
	if m.diffHash != "" {
		title = "== Diff " + shortHash(m.diffHash) + " =="
	}

	var lines []string
	switch {
	case m.diffErr != nil:
		lines = []string{"erro: " + m.diffErr.Error()}
	case m.diff == "":
		lines = []string{"(diff vazio)"}
	default:
		lines = strings.Split(m.diff, "\n")
	}

	body := titleStyle.Render(title)
	maxLines := height - 1 // o título ocupa a primeira linha
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	for _, line := range lines {
		// width-2 é a área útil depois do padding horizontal de 1 de cada lado.
		body += "\n" + truncate(line, width-2)
	}

	return focusedStyle.
		Width(width).
		Height(height).
		Render(body)
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
//
// A branch atual ganha "*" e cor verde. A linha selecionada é montada sem
// cores internas e recebe o highlight por fora (ver selectedLine).
func (m model) branchLines() []string {
	if m.branchesErr != nil {
		return []string{"erro: " + m.branchesErr.Error()}
	}
	if len(m.branches) == 0 {
		return []string{"(nenhuma branch)"}
	}

	lines := make([]string, 0, len(m.branches))
	for i, b := range m.branches {
		// Texto puro, sem ANSI: é o que vai para o highlight.
		plain := "  " + b.Name
		if b.IsHead {
			plain = "* " + b.Name
		}

		if m.isSelected(focusBranches, i) {
			lines = append(lines, m.selectedLine(plain))
			continue
		}

		// Fora do highlight, o "* nome" da HEAD fica verde.
		text := plain
		if b.IsHead {
			text = headStyle.Render(plain)
		}
		lines = append(lines, "  "+text)
	}
	return lines
}

// commitLines monta as linhas do painel de commits: hash curto + subject.
// O hash fica cinza; o subject mantém a cor padrão do terminal.
func (m model) commitLines() []string {
	if m.commitsErr != nil {
		return []string{"erro: " + m.commitsErr.Error()}
	}
	if len(m.commits) == 0 {
		return []string{"(nenhum commit)"}
	}

	lines := make([]string, 0, len(m.commits))
	for i, c := range m.commits {
		hash := shortHash(c.Hash)

		if m.isSelected(focusCommits, i) {
			// Sem hashStyle aqui: cor interna + reset apagaria o fundo
			// do highlight no meio da linha.
			lines = append(lines, m.selectedLine(hash+" "+c.Subject))
			continue
		}

		lines = append(lines, "  "+hashStyle.Render(hash)+" "+c.Subject)
	}
	return lines
}

// statusLines monta as linhas do painel de status: código XY + caminho.
// Untracked fica amarelo para saltar aos olhos, como no lazygit.
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
		plain := statusCode(f) + " " + f.Path

		if m.isSelected(focusStatus, i) {
			lines = append(lines, m.selectedLine(plain))
			continue
		}

		text := plain
		if f.IsUntracked {
			text = untrackedStyle.Render(plain)
		}
		lines = append(lines, "  "+text)
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

// isSelected diz se o item `index` está sob o cursor e se o painel está ativo.
// Assim só existe um cursor visível por vez, sempre no painel focado.
func (m model) isSelected(panel, index int) bool {
	return m.focus == panel && m.cursor[panel] == index
}

// selectedLine aplica o highlight (item 4d) numa linha de texto puro.
//
// Importante: `text` NÃO deve conter ANSI. Cores internas terminam com reset
// (\x1b[0m), que apagaria o fundo do highlight no meio da linha. Por isso as
// linhas selecionadas são montadas sem estilo e pintadas só aqui.
func (m model) selectedLine(text string) string {
	width := m.panelWidth() - 2
	if width < 3 {
		return "> " + text
	}

	// "> " ocupa 2 células. Truncar ANTES de pintar evita que Width quebre a
	// linha em duas e garante que o fundo tenha exatamente a largura útil.
	line := truncate("> "+text, width)
	return selectedStyle.Width(width).Render(line)
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
// ANSI, o que permite colorir as linhas sem desalinhar os painéis.
func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	return lipgloss.NewStyle().MaxWidth(width).Render(s)
}
