package git

import (
	"bytes"
	"context"
	"os"
	"os/exec"
)

// Repo representa um repositório git aberto.
// Por enquanto só guarda o caminho.
type Repo struct {
	path string
}

// New abre um repositório no caminho dado.
// Ainda não valida que é um repo — isso vem depois.
func New(path string) *Repo {
	return &Repo{path: path}
}

// run é a ÚNICA porta de entrada pro binário git.
// Toda operação (status, log, branches, merge...) passa por aqui.
//
// Retorna:
//
//	stdout — saída padrão do git (o que interessa)
//	stderr — mensagens de erro do git
//	err    — erro do exec (nil se exit 0)
func (r *Repo) run(ctx context.Context, args ...string) (stdout, stderr []byte, err error) {
	// 1. Monta os args que SEMPRE vão pro git, antes do comando específico.
	//    Repare: NÃO inclui "git" — o binário vai como 1º arg do CommandContext.
	//    -C <path>                 : roda no repo sem precisar de cd
	//    -c core.quotepath=false   : caminhos UTF-8 literais (sem \303\251)
	//    --no-pager                : nunca abre less/more (travaria a TUI)
	fullArgs := []string{
		"-C", r.path,
		"-c", "core.quotepath=false",
		"--no-pager",
	}

	// 2. Concatena com os args do comando específico.
	//    append(base, args...) — args é variádico, então `args...` expande.
	fullArgs = append(fullArgs, args...)

	// 3. Cria o Cmd. CommandContext cancela se o ctx for cancelado/timeout.
	//    Assinatura: CommandContext(ctx, binário, args...)
	//    Repare: "git" é o binário, fullArgs são os argumentos.
	cmd := exec.CommandContext(ctx, "git", fullArgs...)

	// 4. Captura stdout/stderr em buffers separados.
	//    &outBuf (ponteiro) porque cmd.Stdout espera io.Writer,
	//    e bytes.Buffer implementa io.Writer via método Write.
	//    Se você passar outBuf (valor), compila mas o buffer fica vazio.
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	// 5. Ambiente. append(os.Environ(), ...) é OBRIGATÓRIO:
	//    se você fizer cmd.Env = []string{...}, você substitui TUDO
	//    (perde PATH, HOME, etc.) e `git` deixa de ser encontrado.
	//
	//    GIT_TERMINAL_PROMPT=0 : nunca pede credencial (TUI não tem stdin).
	//    LC_ALL=C              : mensagens de erro em inglês — parseáveis.
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"LC_ALL=C",
	)

	// 6. Stdin nil de propósito: git nunca deve abrir prompt/editor.
	//    Se tentar, falha rápido em vez de roubar o terminal do Bubbletea.
	//    (Já é o padrão, mas explícito documenta a intenção.)
	cmd.Stdin = nil

	// 7. Roda. runErr != nil quando exit code != 0.
	//    Importante: mesmo com runErr != nil, errBuf tem a mensagem do git.
	//    Retornamos os três — o chamador decide o que fazer.
	runErr := cmd.Run()
	return outBuf.Bytes(), errBuf.Bytes(), runErr
}