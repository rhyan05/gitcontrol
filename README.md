# gg

Gerenciador de Git com três interfaces sobre o mesmo core:

- **TUI** em Go/Bubble Tea para operar tudo pelo terminal.
- **CLI JSON** (`gg --json`) com contrato estável para automação.
- **Desktop** em Tauri + React para uma experiência visual.

![Interface do gg](desktop/docs/gg.png)

## O que funciona

- Branches: listar, checkout (local e remota com tracking), merge.
- Commits: log, diff completo, cherry-pick.
- Working tree: status, stage/unstage por arquivo, commit.
- Refresh automático na TUI (fsnotify + timer) e na GUI (polling curto).
- Instaladores Windows: `.msi` e `.exe` (NSIS).

Conflitos são reportados; a resolução continua no terminal (`git merge --abort`, editar e `git add`).
Pull/push/fetch, grafo de commits e multi-repo ainda não fazem parte do escopo.

## Arquitetura

```text
cmd/gg/            entrada única: TUI ou --json
internal/git/      core Go (status, log, branches, show, diff, ações)
internal/cli/      contrato JSON (envelope ok/data ou ok/error)
desktop/           app Tauri + React + Tailwind v4
  src/             UI (componentes, features, hooks, sidecar client)
  src-tauri/       runtime Rust, sidecar e ícones
```

O desktop não reimplementa Git: ele chama o sidecar `gg --json`, que é o mesmo
binário do core testado em Go.

## Requisitos

- Go 1.27+
- Node.js 22+ e npm
- Rust 1.90+ com toolchain MSVC
- Visual Studio Build Tools com **Desktop development with C++**
- WebView2 Runtime (Windows)

## TUI / CLI

```powershell
go test ./...
go run ./cmd/gg              # abre a TUI
go run ./cmd/gg --json status
go run ./cmd/gg --json log
go run ./cmd/gg --json branches
go run ./cmd/gg --json show <hash>
go run ./cmd/gg --json checkout <branch>
go run ./cmd/gg --json merge <branch>
go run ./cmd/gg --json cherry-pick <hash>
go run ./cmd/gg --json stage <path...>
go run ./cmd/gg --json unstage <path...>
go run ./cmd/gg --json commit "mensagem"
```

O contrato é sempre um objeto JSON:

```json
{"ok":true,"data":{...}}
{"ok":false,"error":{"code":"git_error","message":"..."}}
```

Exit code `0` em sucesso e `1` em erro.

## Desktop

```powershell
cd desktop
npm install
npm run tauri -- dev
```

O `beforeDevCommand` compila o sidecar automaticamente (`npm run sidecar`) e
inicia o Vite. Para gerar os instaladores:

```powershell
cd desktop
npm run tauri -- build
```

Artefatos:

- `desktop/src-tauri/target/release/bundle/msi/gg_0.1.0_x64_en-US.msi`
- `desktop/src-tauri/target/release/bundle/nsis/gg_0.1.0_x64-setup.exe`

Atalhos da GUI: `/` busca, `r` atualiza, `Ctrl/Cmd+K` abre a paleta de comandos,
`Esc` fecha diálogos. O painel de diff é redimensionável pela alça acima dele.

## Qualidade

```powershell
go test ./...
cd desktop
npm run lint
npm run test
npm run build
```

## Licença

Uso pessoal. Ajuste a licença antes de publicar.
