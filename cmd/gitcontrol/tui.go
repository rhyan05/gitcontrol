package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"

	"github.com/rhyan05/gitcontrol/internal/git"
	"github.com/rhyan05/gitcontrol/internal/workspace"
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
// confirmação de uma ação e digitação da mensagem de commit.
type uiMode int

const (
	modeNormal uiMode = iota
	modeConfirm
	modeCommitInput
)

// inputAction diz o que fazer com o texto digitado no input.
type inputAction int

const (
	inputCommit inputAction = iota
	inputCreateBranch
	inputRenameBranch
	inputStash
	inputTag
	inputAmend
	inputRebase
)

// confirmAction lembra qual ação o usuário confirmou com y.
type confirmAction int

const (
	actionCheckout confirmAction = iota
	actionMerge
	actionCherryPick
	actionPull
	actionPush
	actionAbort
	actionContinue
	actionDeleteBranch
	actionDiscard
	actionRebase
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

	// Multi-repo: todos os caminhos abertos e o índice do ativo. m.repo é
	// sempre o Repo do índice atual.
	repoPaths []string
	repoIndex int

	// Dados carregados do repositório. Ficam no model para View ser pura.
	branches []git.Branch
	commits  []git.Commit
	graph    map[string]string
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

	// Estado dos fluxos interativos (5c-5e + merge/cherry-pick).
	// mode evita que "q", "j" etc. sejam tratados como teclas globais
	// enquanto o usuário confirma uma ação ou digita uma mensagem.
	mode          uiMode
	pendingAction confirmAction
	pendingTarget string // branch ou hash da ação confirmada
	pendingRemote bool
	inputAction   inputAction
	inputTarget   string
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
	graph       []git.GraphRow
	graphErr    error
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

// run abre o repositório informado (ou o diretório atual) e inicia a TUI.
//
// WithAltScreen usa a tela alternativa: ao sair, o conteúdo anterior do
// terminal volta intacto (nada de poluir o scrollback com frames da TUI).
func run(repos []string, active int) error {
	if len(repos) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("obter cwd: %w", err)
		}
		repos = []string{cwd}
	}
	if active < 0 || active >= len(repos) {
		active = 0
	}

	repo := git.New(repos[active])

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
		model{repo: repo, repoPaths: repos, repoIndex: active, watcher: watcher, width: 80, height: 24},
		tea.WithAltScreen(),
	)
	_, err := p.Run()
	return err
}

// loadAll roda em background (é um tea.Cmd), nunca dentro de Update.
// Faz as três leituras em sequência e devolve tudo num único loadedMsg.
func (m model) loadAll() tea.Msg {
	branches, bErr := m.repo.Branches(context.Background())
	commits, cErr := m.repo.Log(context.Background(), 50)
	graph, gErr := m.repo.Graph(context.Background(), 50)
	status, sErr := m.repo.Status(context.Background())
	return loadedMsg{
		branches:    branches,
		branchesErr: bErr,
		commits:     commits,
		commitsErr:  cErr,
		graph:       graph,
		graphErr:    gErr,
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
		case modeConfirm:
			return m.updateConfirm(msg)
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
		case "M":
			// Merge da branch selecionada na branch atual (com confirmação).
			m.startMerge()
		case "C":
			// Cherry-pick do commit selecionado (com confirmação).
			m.startCherryPick()
		case "n":
			m.startInput(inputCreateBranch)
		case "R":
			m.startRenameInput()
		case "D":
			m.startDeleteBranch()
		case "r":
			m.startRebase()
		case "z":
			m.startInput(inputStash)
		case "Z":
			return m, m.stashPop()
		case "t":
			m.startInput(inputTag)
		case "A":
			m.startInput(inputAmend)
		case "X":
			m.startDiscard()
		case "f":
			// Fetch é read-only: roda direto, sem confirmação.
			return m, m.fetch()
		case "p":
			m.startPull()
		case "P":
			m.startPush()
		case "a":
			m.startAbort()
		case "x":
			m.startContinue()
		case "e":
			if cmd := m.editSelected(); cmd != nil {
				return m, cmd
			}
		case "[":
			return m, m.switchRepo(-1)
		case "]":
			return m, m.switchRepo(1)
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
		m.graph = make(map[string]string, len(msg.graph))
		for _, row := range msg.graph {
			if row.Commit != nil {
				m.graph[row.Commit.Hash] = row.Graph
			}
		}
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
		return m, m.submitInput()
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

// updateConfirm trata y/n/esc enquanto a confirmação está aberta.
func (m model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		return m, m.runConfirmed()
	case "n", "esc":
		m.mode = modeNormal
		m.pendingTarget = ""
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
	m.mode = modeConfirm
	m.pendingAction = actionCheckout
	m.pendingTarget = b.Name
	m.pendingRemote = b.IsRemote
	m.notice = ""
}

// startMerge arma o merge da branch selecionada na branch atual.
func (m *model) startMerge() {
	if m.focus != focusBranches || len(m.branches) == 0 {
		return
	}

	b := m.branches[m.cursor[focusBranches]]
	m.mode = modeConfirm
	m.pendingAction = actionMerge
	m.pendingTarget = b.Name
	m.pendingRemote = b.IsRemote
	m.notice = ""
}

// startCherryPick arma o cherry-pick do commit selecionado.
// pendingTarget guarda o hash completo; o footer mostra só os 7 primeiros.
func (m *model) startCherryPick() {
	if m.focus != focusCommits || len(m.commits) == 0 {
		return
	}

	hash := m.commits[m.cursor[focusCommits]].Hash
	m.mode = modeConfirm
	m.pendingAction = actionCherryPick
	m.pendingTarget = hash
	m.pendingRemote = false
	m.notice = ""
}

// startPull arma o pull do branch atual.
func (m *model) startPull() {
	m.mode = modeConfirm
	m.pendingAction = actionPull
	m.pendingTarget = m.currentBranch()
	m.notice = ""
}

// startPush arma o push do branch atual.
func (m *model) startPush() {
	m.mode = modeConfirm
	m.pendingAction = actionPush
	m.pendingTarget = m.currentBranch()
	m.notice = ""
}

// startAbort arma o abort da operação em andamento (merge/cherry-pick).
func (m *model) startAbort() {
	if m.status == nil || m.status.Operation == git.OperationNone {
		m.notice = "nenhuma operação em andamento"
		return
	}
	m.mode = modeConfirm
	m.pendingAction = actionAbort
	m.pendingTarget = string(m.status.Operation)
	m.notice = ""
}

// startContinue arma a conclusão da operação em andamento.
func (m *model) startContinue() {
	if m.status == nil || m.status.Operation == git.OperationNone {
		m.notice = "nenhuma operação em andamento"
		return
	}
	m.mode = modeConfirm
	m.pendingAction = actionContinue
	m.pendingTarget = string(m.status.Operation)
	m.notice = ""
}

// fetch roda direto: é read-only e não muda o working tree.
func (m *model) fetch() tea.Cmd {
	repo := m.repo
	return func() tea.Msg {
		if err := repo.Fetch(context.Background(), ""); err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{notice: "fetch concluído"}
	}
}

// editSelected abre o arquivo conflitado no editor do usuário.
func (m *model) editSelected() tea.Cmd {
	if m.focus != focusStatus {
		return nil
	}
	files := m.visibleStatus()
	if len(files) == 0 {
		return nil
	}
	file := files[m.cursor[focusStatus]]
	if !file.IsUnmerged {
		m.notice = "arquivo não está em conflito"
		return nil
	}

	editor := strings.TrimSpace(os.Getenv("EDITOR"))
	if editor == "" {
		if runtime.GOOS == "windows" {
			editor = "notepad"
		} else {
			editor = "vi"
		}
	}
	parts := strings.Fields(editor)
	args := append(parts[1:], file.Path)
	cmd := exec.Command(parts[0], args...)

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{notice: "editado: " + file.Path + " (s para marcar resolvido)"}
	})
}

// switchRepo troca o repositório ativo e recria o watcher.
func (m *model) switchRepo(delta int) tea.Cmd {
	if len(m.repoPaths) <= 1 {
		return nil
	}
	if m.watcher != nil {
		m.watcher.Close()
		m.watcher = nil
	}

	m.repoIndex = (m.repoIndex + delta + len(m.repoPaths)) % len(m.repoPaths)
	m.repo = git.New(m.repoPaths[m.repoIndex])
	_ = workspace.Save(workspace.Config{Repos: m.repoPaths, Active: m.repoIndex})
	m.loaded = false
	m.branches = nil
	m.commits = nil
	m.graph = nil
	m.status = nil
	m.cursor = [numPanels]int{}

	if watcher, err := newGitWatcher(context.Background(), m.repo); err == nil {
		m.watcher = watcher
	}

	cmds := []tea.Cmd{m.loadAll}
	if cmd := m.waitForGitChange(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

// startInput abre o input de texto para a ação informada.
func (m *model) startInput(action inputAction) {
	m.mode = modeCommitInput
	m.inputAction = action
	m.inputTarget = ""
	m.commitText = ""
	m.notice = ""
}

// startRenameInput renomeia a branch selecionada (ou a atual).
func (m *model) startRenameInput() {
	old := m.currentBranch()
	if m.focus == focusBranches && len(m.branches) > 0 {
		old = m.branches[m.cursor[focusBranches]].Name
	}
	m.startInput(inputRenameBranch)
	m.inputTarget = old
}

// startDeleteBranch arma a remoção da branch selecionada.
func (m *model) startDeleteBranch() {
	if m.focus != focusBranches || len(m.branches) == 0 {
		return
	}
	branch := m.branches[m.cursor[focusBranches]]
	if branch.IsHead {
		m.notice = "não dá para deletar a branch atual"
		return
	}
	m.mode = modeConfirm
	m.pendingAction = actionDeleteBranch
	m.pendingTarget = branch.Name
	m.notice = ""
}

// startRebase arma o rebase da branch atual sobre a selecionada.
func (m *model) startRebase() {
	if m.focus != focusBranches || len(m.branches) == 0 {
		return
	}
	branch := m.branches[m.cursor[focusBranches]]
	if branch.Name == m.currentBranch() {
		m.notice = "escolha outra branch para rebase"
		return
	}
	m.mode = modeConfirm
	m.pendingAction = actionRebase
	m.pendingTarget = branch.Name
	m.notice = ""
}

// startDiscard arma o descarte do arquivo selecionado.
func (m *model) startDiscard() {
	if m.focus != focusStatus {
		return
	}
	files := m.visibleStatus()
	if len(files) == 0 {
		return
	}
	m.mode = modeConfirm
	m.pendingAction = actionDiscard
	m.pendingTarget = files[m.cursor[focusStatus]].Path
	m.notice = ""
}

// stashPop aplica o stash mais recente sem confirmação.
func (m *model) stashPop() tea.Cmd {
	repo := m.repo
	return func() tea.Msg {
		if err := repo.StashPop(context.Background(), 0); err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{notice: "stash aplicado"}
	}
}

// runConfirmed executa a ação que estava aguardando y.
// O I/O acontece na closure; Update apenas lê o resultado via actionMsg.
func (m *model) runConfirmed() tea.Cmd {
	action := m.pendingAction
	target := m.pendingTarget

	m.mode = modeNormal
	m.pendingTarget = ""
	m.pendingRemote = false

	repo := m.repo
	return func() tea.Msg {
		var err error
		notice := ""

		switch action {
		case actionCheckout:
			err = repo.Checkout(context.Background(), target)
			notice = "checkout: " + target
		case actionMerge:
			err = repo.Merge(context.Background(), target)
			notice = "merge: " + target
		case actionCherryPick:
			err = repo.CherryPick(context.Background(), target)
			notice = "cherry-pick: " + shortHash(target)
		case actionPull:
			err = repo.Pull(context.Background(), "", "")
			notice = "pull concluído"
		case actionPush:
			err = repo.Push(context.Background(), "", "")
			notice = "push concluído"
		case actionAbort:
			err = repo.Abort(context.Background())
			notice = "operação abortada"
		case actionContinue:
			err = repo.Continue(context.Background())
			notice = "operação concluída"
		case actionDeleteBranch:
			err = repo.DeleteBranch(context.Background(), target, false)
			notice = "branch removida: " + target
		case actionDiscard:
			err = repo.Discard(context.Background(), []string{target})
			notice = "descartado: " + target
		case actionRebase:
			err = repo.Rebase(context.Background(), target)
			notice = "rebase em " + target
		}

		if err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{notice: notice}
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

// submitInput valida e despacha o texto do input conforme a ação.
func (m *model) submitInput() tea.Cmd {
	text := strings.TrimSpace(m.commitText)
	action := m.inputAction
	target := m.inputTarget

	if text == "" && action != inputStash && action != inputAmend {
		if action == inputCommit {
			m.notice = "mensagem de commit vazia"
		} else {
			m.notice = "texto vazio"
		}
		return nil
	}

	m.mode = modeNormal
	m.commitText = ""
	m.inputTarget = ""
	repo := m.repo

	return func() tea.Msg {
		ctx := context.Background()
		var err error
		notice := ""

		switch action {
		case inputCommit:
			err = repo.Commit(ctx, text)
			notice = "commit: " + text
		case inputCreateBranch:
			fields := strings.Fields(text)
			start := ""
			if len(fields) > 1 {
				start = fields[1]
			}
			err = repo.CreateBranch(ctx, fields[0], start)
			notice = "branch criada: " + fields[0]
		case inputRenameBranch:
			err = repo.RenameBranch(ctx, target, text)
			notice = "branch renomeada: " + text
		case inputStash:
			err = repo.StashPush(ctx, text, true)
			notice = "stash criado"
		case inputTag:
			fields := strings.Fields(text)
			err = repo.CreateTag(ctx, fields[0], strings.Join(fields[1:], " "))
			notice = "tag criada: " + fields[0]
		case inputAmend:
			err = repo.Amend(ctx, text)
			notice = "commit amendado"
		case inputRebase:
			err = repo.Rebase(ctx, text)
			notice = "rebase em " + text
		}

		if err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{notice: notice}
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
		text = fmt.Sprintf("%s%s │ enter confirmar │ esc cancelar", m.inputPrompt(), m.commitText)
	case modeConfirm:
		switch m.pendingAction {
		case actionCheckout:
			text = "checkout " + m.pendingTarget + "?"
			if m.pendingRemote {
				text += " cria branch local"
			}
		case actionMerge:
			text = "merge " + m.pendingTarget + " em " + m.currentBranch() + "?"
		case actionCherryPick:
			text = "cherry-pick " + shortHash(m.pendingTarget) + " em " + m.currentBranch() + "?"
		case actionPull:
			text = "pull em " + m.currentBranch() + "?"
		case actionPush:
			text = "push de " + m.currentBranch() + "?"
		case actionAbort:
			text = "abortar " + m.pendingTarget + "?"
		case actionContinue:
			text = "continuar " + m.pendingTarget + "?"
		case actionDeleteBranch:
			text = "deletar branch " + m.pendingTarget + "?"
		case actionDiscard:
			text = "descartar " + m.pendingTarget + "?"
		case actionRebase:
			text = "rebase em " + m.pendingTarget + "?"
		}
		text += " (y/n)"
	default:
		// Ordem proposital: ações antes das teclas de navegação, para não
		// serem cortadas em terminal de 80 colunas.
		text = "q sair │ tab │ j/k │ n branch │ R renomear │ D deletar │ r rebase │ z/Z stash │ t tag │ A amend │ X discard │ M merge │ C cherry-pick │ f fetch │ p pull │ P push │ a/x conflito │ enter │ d diff │ s/u stage │ c commit"
		if m.diffOpen {
			text = "esc fechar │ pgup/pgdn scroll │ home/end topo/fim │ q sair"
		}
		if len(m.repoPaths) > 1 {
			text = fmt.Sprintf("[%d/%d] %s │ ", m.repoIndex+1, len(m.repoPaths), m.repoPaths[m.repoIndex]) + text
		}
	}

	if m.notice != "" {
		text = m.notice + " │ " + text
	}
	return text
}

// inputPrompt devolve o rótulo do input atual.
func (m model) inputPrompt() string {
	switch m.inputAction {
	case inputCreateBranch:
		return "nova branch (nome [start])> "
	case inputRenameBranch:
		return "renomear " + m.inputTarget + " para> "
	case inputStash:
		return "stash (mensagem opcional)> "
	case inputTag:
		return "tag (nome [mensagem])> "
	case inputAmend:
		return "amend (vazio mantém a mensagem)> "
	case inputRebase:
		return "rebase em> "
	default:
		return "commit> "
	}
}

// currentBranch é o alvo do merge/cherry-pick exibido na confirmação.
// Fallback "HEAD" cobre repo sem commits ou detached HEAD.
func (m model) currentBranch() string {
	if m.status == nil || m.status.Branch == "" {
		return "HEAD"
	}
	return m.status.Branch
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
		prefix := m.graph[c.Hash]
		if prefix == "" {
			prefix = "*"
		}

		if m.isSelected(focusCommits, i) {
			// Sem hashStyle aqui: cor interna + reset apagaria o fundo
			// do highlight no meio da linha.
			lines = append(lines, m.selectedLine(prefix+" "+hash+" "+c.Subject))
			continue
		}

		lines = append(lines, "  "+prefix+" "+hashStyle.Render(hash)+" "+c.Subject)
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
