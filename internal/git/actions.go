package git

import (
	"context"
	"fmt"
	"strings"
)

// Checkout troca a branch atual.
//
// Aceita branch local ("main") ou remota ("origin/feature"). Para remota,
// cria uma branch local de tracking (ou usa a local, se já existir).
//
// A detecção é feita pelas refs reais (refs/heads e refs/remotes), e não pelo
// formato do nome: branches locais também podem conter "/" (ex.: feature/x).
func (r *Repo) Checkout(ctx context.Context, branch string) error {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return fmt.Errorf("git checkout: branch vazia")
	}

	switch {
	case r.hasRef(ctx, "refs/heads/"+branch):
		return r.checkout(ctx, branch)

	case r.hasRef(ctx, "refs/remotes/"+branch):
		local := localBranchName(branch)
		// Se a local já existe, só troca para ela; não recria com --track.
		if r.hasRef(ctx, "refs/heads/"+local) {
			return r.checkout(ctx, local)
		}
		return r.checkoutTracking(ctx, local, branch)

	default:
		// Deixa o git produzir a mensagem real (branch não encontrada etc.).
		return r.checkout(ctx, branch)
	}
}

// checkout executa o checkout simples de uma branch já existente.
func (r *Repo) checkout(ctx context.Context, branch string) error {
	_, stderr, err := r.run(ctx, "checkout", branch)
	if err != nil {
		return fmt.Errorf("git checkout %s: %w (stderr: %s)", branch, err, stderr)
	}
	return nil
}

// checkoutTracking cria a branch local já configurando o upstream remoto.
func (r *Repo) checkoutTracking(ctx context.Context, local, remote string) error {
	_, stderr, err := r.run(ctx, "checkout", "-b", local, "--track", remote)
	if err != nil {
		return fmt.Errorf("git checkout -b %s --track %s: %w (stderr: %s)", local, remote, err, stderr)
	}
	return nil
}

// hasRef informa se uma ref exata existe.
// `git show-ref --verify --quiet` não imprime nada e sai != 0 quando não existe.
func (r *Repo) hasRef(ctx context.Context, ref string) bool {
	_, _, err := r.run(ctx, "show-ref", "--verify", "--quiet", ref)
	return err == nil
}

// localBranchName deriva o nome local de uma branch remota:
//
//	origin/feature     -> feature
//	origin/feature/x   -> feature/x
//
// Remove só o primeiro segmento (o nome do remoto), preservando "feature/x".
func localBranchName(remoteBranch string) string {
	if i := strings.Index(remoteBranch, "/"); i >= 0 {
		return remoteBranch[i+1:]
	}
	return remoteBranch
}

// Stage adiciona arquivos ao index (git add).
//
// O "--" separa flags de paths: sem ele, um arquivo chamado "-x" seria
// interpretado como opção do git.
func (r *Repo) Stage(ctx context.Context, paths []string) error {
	if len(paths) == 0 {
		return fmt.Errorf("git add: nenhum path informado")
	}

	args := append([]string{"add", "--"}, paths...)
	_, stderr, err := r.run(ctx, args...)
	if err != nil {
		return fmt.Errorf("git add: %w (stderr: %s)", err, stderr)
	}
	return nil
}

// Unstage remove arquivos do index, mantendo as mudanças no working tree
// (equivalente a `git reset HEAD -- <paths>`).
func (r *Repo) Unstage(ctx context.Context, paths []string) error {
	if len(paths) == 0 {
		return fmt.Errorf("git reset: nenhum path informado")
	}

	args := append([]string{"reset", "-q", "HEAD", "--"}, paths...)
	_, stderr, err := r.run(ctx, args...)
	if err != nil {
		return fmt.Errorf("git reset: %w (stderr: %s)", err, stderr)
	}
	return nil
}

// Commit cria um commit com a mensagem informada (git commit -m).
//
// Valida a mensagem antes de chamar o git: mensagem só com espaços viraria
// um commit vazio ou abriria o editor, e a TUI não tem stdin interativo.
// Em caso de falha, o git costuma escrever o motivo em stdout (ex.:
// "nothing to commit"), então usamos stderr e caímos para stdout se preciso.
func (r *Repo) Commit(ctx context.Context, message string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		return fmt.Errorf("git commit: mensagem vazia")
	}

	stdout, stderr, err := r.run(ctx, "commit", "-m", message)
	if err != nil {
		return gitError("git commit", stdout, stderr, err)
	}
	return nil
}

// Amend reescreve o commit atual. Mensagem vazia mantém a original.
func (r *Repo) Amend(ctx context.Context, message string) error {
	args := []string{"commit", "--amend", "--no-edit"}
	if message = strings.TrimSpace(message); message != "" {
		args = []string{"commit", "--amend", "-m", message}
	}

	stdout, stderr, err := r.run(ctx, args...)
	if err != nil {
		return gitError("git commit --amend", stdout, stderr, err)
	}
	return nil
}

// Merge incorpora a branch informada na branch atual (git merge).
//
// --no-edit evita abrir editor para a mensagem do merge commit: a TUI roda
// sem stdin interativo. A estratégia é a padrão do git (fast-forward quando
// possível, merge commit quando as branches divergiram).
//
// Em conflito, o git deixa o repositório em estado de merge (MERGE_HEAD) e
// a TUI apenas reporta o erro; a resolução fica no terminal.
func (r *Repo) Merge(ctx context.Context, branch string) error {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return fmt.Errorf("git merge: branch vazia")
	}

	stdout, stderr, err := r.run(ctx, "merge", "--no-edit", branch)
	if err != nil {
		// Conflito é reportado no stdout ("CONFLICT ..."), não no stderr.
		return gitError("git merge "+branch, stdout, stderr, err)
	}
	return nil
}

// CherryPick aplica o commit informado na branch atual (git cherry-pick).
//
// --no-edit mantém a mensagem original sem abrir editor. Em conflito, o repo
// fica em estado de cherry-pick (CHERRY_PICK_HEAD) e o chamador decide como
// resolver/abortar.
func (r *Repo) CherryPick(ctx context.Context, hash string) error {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return fmt.Errorf("git cherry-pick: hash vazio")
	}

	stdout, stderr, err := r.run(ctx, "cherry-pick", "--no-edit", hash)
	if err != nil {
		return gitError("git cherry-pick "+hash, stdout, stderr, err)
	}
	return nil
}

// gitError monta um erro com a mensagem mais útil do git.
// Dependendo do comando, a explicação fica no stderr ou no stdout (merge e
// cherry-pick, por exemplo, escrevem "CONFLICT" no stdout e detalhes de
// rerere no stderr). Por isso juntamos os dois, sem duplicar vazios.
func gitError(op string, stdout, stderr []byte, err error) error {
	parts := make([]string, 0, 2)
	for _, raw := range [][]byte{stdout, stderr} {
		part := strings.TrimSpace(string(raw))
		if part != "" {
			parts = append(parts, part)
		}
	}

	if len(parts) == 0 {
		return fmt.Errorf("%s: %w", op, err)
	}
	return fmt.Errorf("%s: %w (%s)", op, err, strings.Join(parts, "\n"))
}
