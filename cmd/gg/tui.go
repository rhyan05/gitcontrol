package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"

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

// uiMode separa a tela normal dos fluxos que capturam teclas:
// confirmação de checkout e digitação da mensagem de commit.
type uiMode int

const (
	modeNormal uiMode = iota
	modeConfirmCheckout
	modeCommitInput
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

	// remoteStyle distingue branches remotas das locais (5c).
	remoteStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))

	// upstreamStyle deixa o "→ origin/main" discreto ao lado da branch local.
	upstreamStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

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

	// Painel de diff (5a/5b/5f).
	// diffOpen controla a visibilidade; diffKey identifica o pedido atual
	// (hash do commit ou path do arquivo) para descartar respostas atrasadas;
	// diffLabel é o texto exibido no título. diffScroll é o offset vertical.
	diffOpen   bool
	diff       string
	diffErr    error
	diffKey    string
	diffLabel  string
	diffScroll int

	// Estado dos fluxos interativos (5c-5e).
	// mode evita que "q", "j" etc. sejam tratados como teclas globais
	// enquanto o usuário confirma um checkout ou digita uma mensagem.
	mode          uiMode
	pendingBranch string
	pendingRemote bool
	commitText    string
	notice        string

	// watcher observa .git para refresh automático (Bloco 6). Pode ser nil
	// quando fsnotify não está disponível; o timer de fallback continua.
	watcher *fsnotify.Watcher
}

// actionMsg é a resposta assíncrona de checkout/stage/unstage/commit.
// notice é a mensagem de sucesso; err, quando presente, vence na exibição.
type actionMsg struct {
	notice string
	err    error
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

// diffMsg é a resposta assíncrona de git show/git diff para o painel.
// key é o hash do commit ou o path do arquivo que originou o pedido.
type diffMsg struct {
	key  string
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

	repo := git.New(cwd)

	// Watcher é opcional: se fsnotify falhar (permissão, .git exótico),
	// seguimos só com o timer de fallback. O defer fecha ao sair da TUI.
	watcher, watchErr := newGitWatcher(context.Background(), repo)
	if watchErr == nil {
		defer watcher.Close()
	} else {
		watcher = nil
	}

	p := tea.NewProgram(
		// Tamanho inicial de fallback; o WindowSizeMsg real chega logo depois.
		model{repo: repo, watcher: watcher, width: 80, height: 24},
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

// Init é chamado uma vez pelo Bubbletea. Dispara a primeira carga, o timer
// de fallback e, se disponível, a escuta do watcher de .git.
func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.loadAll, tickCmd()}
	if cmd := m.waitForGitChange(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

// Update trata eventos. Deve ser puro: nada de I/O, nada de Print.
// Eventos tratados hoje:
//   - WindowSizeMsg: o terminal mudou de tamanho (recalcula larguras/alturas);
//   - KeyMsg: navegação normal, confirmação de checkout e input de commit;
//   - loadedMsg: chegada dos dados do loadAll;
//   - diffMsg: chegada do diff pedido ao git show;
//   - actionMsg: resultado de checkout/stage/unstage/commit.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// View usa m.width/m.height para recalcular tudo no próximo frame.
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		// ctrl+c sai de qualquer modo, inclusive digitando mensagem.
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// Modos que capturam o teclado têm prioridade sobre a navegação.
		switch m.mode {
		case modeCommitInput:
			return m.updateCommitInput(msg)
		case modeConfirmCheckout:
			return m.updateCheckoutConfirm(msg)
		}

		switch msg.String() {
		case "q":
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
			switch m.focus {
			case focusCommits:
				if cmd := m.showSelectedCommit(); cmd != nil {
					return m, cmd
				}
			case focusBranches:
				m.startCheckout()
			}
		case "s":
			if cmd := m.stageSelected(); cmd != nil {
				return m, cmd
			}
		case "u":
			if cmd := m.unstageSelected(); cmd != nil {
				return m, cmd
			}
		case "c":
			m.mode = modeCommitInput
			m.commitText = ""
			m.notice = ""
		case "d":
			// Diff do arquivo selecionado no working tree (5f).
			if cmd := m.showSelectedFile(); cmd != nil {
				return m, cmd
			}
		case "pgdown":
			if m.diffOpen {
				m.scrollDiff(m.diffPage())
			}
		case "pgup":
			if m.diffOpen {
				m.scrollDiff(-m.diffPage())
			}
		case "home":
			if m.diffOpen {
				m.diffScroll = 0
			}
		case "end":
			if m.diffOpen {
				m.diffScroll = m.diffMaxScroll()
			}
		case "esc":
			// Fechar também limpa diffKey: respostas em voo serão
			// descartadas quando chegarem.
			m.diffOpen = false
			m.diff = ""
			m.diffErr = nil
			m.diffKey = ""
			m.diffLabel = ""
			m.diffScroll = 0
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

		// Depois de checkout/stage/commit as listas podem encolher.
		// Sem o clamp, o cursor ficaria apontando para fora do slice.
		for i := 0; i < numPanels; i++ {
			m.cursor[i] = clampCursor(m.cursor[i], m.count(i))
		}
	case diffMsg:
		// Descarta resposta atrasada se o painel foi fechado ou se outro
		// commit/arquivo foi selecionado enquanto o comando rodava.
		if !m.diffOpen || m.diffKey != msg.key {
			return m, nil
		}
		m.diff = msg.text
		m.diffErr = msg.err
		m.diffScroll = 0
	case tickMsg:
		// Fallback periódico: recarrega e rearma o próximo tick.
		return m, tea.Batch(m.loadAll, tickCmd())
	case refreshMsg:
		// O watcher acordou: recarrega e volta a escutar o próximo evento.
		cmd := m.waitForGitChange()
		if cmd == nil {
			return m, m.loadAll
		}
		return m, tea.Batch(m.loadAll, cmd)
	case actionMsg:
		// Qualquer ação pode mudar commits/branches: fecha o diff antigo.
		m.diffOpen = false
		m.diff = ""
		m.diffErr = nil
		m.diffKey = ""
		m.diffLabel = ""
		m.diffScroll = 0

		if msg.err != nil {
			m.notice = "erro: " + msg.err.Error()
		} else {
			m.notice = msg.notice
		}

		// Recarrega tudo para refletir a ação (ex.: novo commit no topo).
		return m, m.loadAll
	}
	return m, nil
}

// updateCommitInput trata teclas enquanto a mensagem de commit é digitada.
// Sem cursor móvel: o texto sempre cresce/encolhe no fim.
func (m model) updateCommitInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		return m, m.submitCommit()
	case tea.KeyEsc:
		m.mode = modeNormal
		m.commitText = ""
	case tea.KeyBackspace:
		m.commitText = removeLastRune(m.commitText)
	case tea.KeyRunes:
		// Runes cobre letras, acentos, espaços e texto colado de uma vez.
		m.commitText += string(msg.Runes)
	}
	return m, nil
}

// updateCheckoutConfirm trata y/n/esc enquanto a confirmação está aberta.
func (m model) updateCheckoutConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		return m, m.doCheckout()
	case "n", "esc":
		m.mode = modeNormal
		m.pendingBranch = ""
		m.pendingRemote = false
		m.notice = ""
	}
	return m, nil
}

// startCheckout arma a confirmação de checkout para a branch selecionada.
// Só marca o estado; o git só roda depois do "y".
func (m *model) startCheckout() {
	if m.focus != focusBranches || len(m.branches) == 0 {
		return
	}

	b := m.branches[m.cursor[focusBranches]]
	m.pendingBranch = b.Name
	m.pendingRemote = b.IsRemote
	m.mode = modeConfirmCheckout
	m.notice = ""
}

// doCheckout devolve o tea.Cmd que executa o checkout confirmado.
// Para branch remota, o core cria a local de tracking automaticamente.
func (m *model) doCheckout() tea.Cmd {
	branch := m.pendingBranch
	m.mode = modeNormal
	m.pendingBranch = ""
	m.pendingRemote = false

	repo := m.repo
	return func() tea.Msg {
		if err := repo.Checkout(context.Background(), branch); err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{notice: "checkout: " + branch}
	}
}

// stageSelected e unstageSelected agem no arquivo selecionado do status.
func (m *model) stageSelected() tea.Cmd {
	return m.fileAction(true)
}

func (m *model) unstageSelected() tea.Cmd {
	return m.fileAction(false)
}

// fileAction evita duplicar o cmd de stage/unstage; só o verbo muda.
func (m *model) fileAction(stage bool) tea.Cmd {
	if m.focus != focusStatus {
		return nil
	}

	files := m.visibleStatus()
	if len(files) == 0 {
		return nil
	}
	path := files[m.cursor[focusStatus]].Path
	repo := m.repo

	return func() tea.Msg {
		verb := "unstage"
		var err error
		if stage {
			verb = "stage"
			err = repo.Stage(context.Background(), []string{path})
		} else {
			err = repo.Unstage(context.Background(), []string{path})
		}
		if err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{notice: verb + ": " + path}
	}
}

// submitCommit valida a mensagem e devolve o cmd que cria o commit.
// Mensagem vazia mantém o input aberto e avisa no rodapé.
func (m *model) submitCommit() tea.Cmd {
	message := strings.TrimSpace(m.commitText)
	if message == "" {
		m.notice = "mensagem de commit vazia"
		return nil
	}

	m.mode = modeNormal
	m.commitText = ""
	repo := m.repo

	return func() tea.Msg {
		if err := repo.Commit(context.Background(), message); err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{notice: "commit: " + message}
	}
}

// showSelectedCommit prepara o model e devolve o tea.Cmd que roda git show.
//
// Recebe *model para poder marcar diffOpen/diffKey no model local do Update;
// o I/O em si acontece dentro da closure, nunca aqui.
func (m *model) showSelectedCommit() tea.Cmd {
	if m.focus != focusCommits || len(m.commits) == 0 {
		return nil
	}

	hash := m.commits[m.cursor[focusCommits]].Hash
	m.diffOpen = true
	m.diff = "carregando..."
	m.diffErr = nil
	m.diffKey = hash
	m.diffLabel = shortHash(hash)
	m.diffScroll = 0

	repo := m.repo
	return func() tea.Msg {
		text, err := repo.Show(context.Background(), hash)
		return diffMsg{key: hash, text: text, err: err}
	}
}

// showSelectedFile abre o diff do arquivo selecionado no painel de status
// (5f). Compartilha o mesmo painel e o mesmo guard de resposta do commit.
func (m *model) showSelectedFile() tea.Cmd {
	if m.focus != focusStatus {
		return nil
	}

	files := m.visibleStatus()
	if len(files) == 0 {
		return nil
	}
	path := files[m.cursor[focusStatus]].Path

	m.diffOpen = true
	m.diff = "carregando..."
	m.diffErr = nil
	m.diffKey = path
	m.diffLabel = path
	m.diffScroll = 0

	repo := m.repo
	return func() tea.Msg {
		text, err := repo.DiffFile(context.Background(), path)
		return diffMsg{key: path, text: text, err: err}
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
	if m.diffOpen {
		views = append(views, m.renderDiffPanel())
	}

	// O rodapé também é truncado para nunca quebrar linha em terminal estreito.
	views = append(views, footerStyle.Render(truncate(m.footerText(), m.width)))

	return lipgloss.JoinVertical(lipgloss.Left, views...)
}

// footerText escolhe a linha de ajuda de acordo com o modo atual.
// Em input/confirmacão o texto muda; notice (sucesso/erro) é prefixado.
func (m model) footerText() string {
	var text string

	switch m.mode {
	case modeCommitInput:
		text = fmt.Sprintf("commit> %s │ enter commitar │ esc cancelar", m.commitText)
	case modeConfirmCheckout:
		text = "checkout " + m.pendingBranch + "?"
		if m.pendingRemote {
			text += " cria branch local"
		}
		text += " (y/n)"
	default:
		text = "q sair │ tab painel │ j/k mover │ enter abrir │ d diff │ s/u stage │ c commit"
		if m.diffOpen {
			text = "esc fechar │ pgup/pgdn scroll │ home/end topo/fim │ q sair"
		}
	}

	if m.notice != "" {
		text = m.notice + " │ " + text
	}
	return text
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

// renderDiffPanel desenha o diff do commit ou arquivo selecionado usando a
// largura toda e respeitando o offset de scroll.
func (m model) renderDiffPanel() string {
	width := m.width - 2 // desconta as duas bordas verticais
	if width < 8 {
		width = 8
	}
	height := m.diffHeight()
	visible := m.diffVisibleLines()

	lines := m.diffLines()
	start := clampScroll(m.diffScroll, len(lines), visible)
	end := start + visible
	if end > len(lines) {
		end = len(lines)
	}

	title := "== Diff =="
	if m.diffLabel != "" {
		title = "== Diff " + m.diffLabel
		if len(lines) > visible {
			title += fmt.Sprintf(" (%d-%d/%d)", start+1, end, len(lines))
		}
		title += " =="
	}
	title = truncate(title, width-2)

	body := titleStyle.Render(title)
	for _, line := range lines[start:end] {
		// width-2 é a área útil depois do padding horizontal de 1 de cada lado.
		body += "\n" + truncate(line, width-2)
	}

	return focusedStyle.
		Width(width).
		Height(height).
		Render(body)
}

// diffLines transforma o texto cru em linhas exibíveis. Centralizado para
// render e cálculo de scroll usarem exatamente a mesma contagem.
func (m model) diffLines() []string {
	switch {
	case m.diffErr != nil:
		return []string{"erro: " + m.diffErr.Error()}
	case m.diff == "":
		return []string{"(diff vazio)"}
	default:
		return strings.Split(m.diff, "\n")
	}
}

// diffVisibleLines é quantas linhas de diff cabem (o título ocupa uma).
func (m model) diffVisibleLines() int {
	n := m.diffHeight() - 1
	if n < 1 {
		n = 1
	}
	return n
}

// diffPage é o passo de pgup/pgdn: meia tela, no mínimo 1.
func (m model) diffPage() int {
	n := m.diffVisibleLines() / 2
	if n < 1 {
		n = 1
	}
	return n
}

// diffMaxScroll é o maior offset válido para o conteúdo atual.
func (m model) diffMaxScroll() int {
	max := len(m.diffLines()) - m.diffVisibleLines()
	if max < 0 {
		return 0
	}
	return max
}

// scrollDiff move o diff e prende o offset nos limites.
func (m *model) scrollDiff(delta int) {
	m.diffScroll = clampScroll(m.diffScroll+delta, len(m.diffLines()), m.diffVisibleLines())
}

// clampScroll prende offset em [0, total-visible]. (total <= visible => 0.)
func clampScroll(offset, total, visible int) int {
	max := total - visible
	if max < 0 {
		max = 0
	}
	if offset < 0 {
		return 0
	}
	if offset > max {
		return max
	}
	return offset
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
// Marcadores:
//   - "*" e verde: branch atual (HEAD);
//   - "→ origin/x" em cinza: upstream de uma branch local;
//   - "[remota]" em ciano: branch que vive só no remoto. Enter nela cria a
//     branch local de tracking (o core cuida disso no Checkout).
//
// A linha selecionada é montada sem cores internas e recebe o highlight por
// fora (ver selectedLine).
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

		// Remota explícita ou upstream da local (nunca os dois).
		suffix := ""
		if b.IsRemote {
			suffix = " [remota]"
		} else if b.Upstream != "" {
			suffix = " → " + b.Upstream
		}

		// Texto puro, sem ANSI: é o que vai para o highlight.
		plain := name + suffix
		if m.isSelected(focusBranches, i) {
			lines = append(lines, m.selectedLine(plain))
			continue
		}

		// Fora do highlight, cada trecho ganha sua cor.
		text := name
		switch {
		case b.IsHead:
			text = headStyle.Render(name)
		case b.IsRemote:
			text = remoteStyle.Render(name)
		}
		if suffix != "" {
			if b.IsRemote {
				text += remoteStyle.Render(suffix)
			} else {
				text += upstreamStyle.Render(suffix)
			}
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

// clampCursor garante que um cursor existente continue válido depois de um
// reload (a lista pode ter encolhido). Sem isso, o próximo Enter usaria um
// índice fora do slice e a TUI quebraria.
func clampCursor(cursor, count int) int {
	if count <= 0 {
		return 0
	}
	if cursor < 0 {
		return 0
	}
	if cursor >= count {
		return count - 1
	}
	return cursor
}

// removeLastRune remove a última runa (não o último byte), para backspace
// funcionar corretamente com acentos e emojis.
func removeLastRune(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return ""
	}
	return string(runes[:len(runes)-1])
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
