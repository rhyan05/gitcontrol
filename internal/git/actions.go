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
		detail := strings.TrimSpace(string(stderr))
		if detail == "" {
			detail = strings.TrimSpace(string(stdout))
		}
		if detail == "" {
			return fmt.Errorf("git commit: %w", err)
		}
		return fmt.Errorf("git commit: %w (%s)", err, detail)
	}
	return nil
}
