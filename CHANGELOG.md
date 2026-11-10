# Changelog

## 0.1.2

- TUI persiste a lista de repositórios e o ativo em
  `~/.config/gitcontrol/workspace.json` (ou `%AppData%` no Windows).
- Release workflow publica as notas do CHANGELOG e anexa o wheel do PyPI.
- CI valida o wheel também no macOS.
- Dependabot, branch protection, templates de issue/PR, CONTRIBUTING e SECURITY.

## 0.1.1

- CI do Rust compila o sidecar antes do `cargo check`.
- CI ganhou smoke test do wheel PyPI (build, instalação em venv, `--version` e
  `--json status`).
- Testes adicionais para grafo, troca de repositório e ações novas da TUI.
- Screenshot e metadados do repositório atualizados.
- Release workflow validado ponta a ponta e publicando no PyPI via secret.

## 0.1.0

- Core Go com status, log, grafo, branches, show, diff, checkout, merge,
  cherry-pick, stage/unstage e commit.
- Remotos: fetch, pull e push com upstream automático.
- Conflitos: resolver, abortar e continuar merge/cherry-pick.
- Multi-repo na TUI (`[`/`]`) e lista de repositórios no desktop.
- CLI JSON (`gitcontrol --json`) com `--repo` e `--version`.
- TUI em Bubble Tea com diff, grafo e refresh automático.
- Desktop Tauri + React com diff de working tree, ações de remoto e conflitos.
- Instaladores Windows (MSI/NSIS).
- Pacote universal no PyPI com binários para Windows/Linux/macOS (x64/arm64).
- Licença MIT.
