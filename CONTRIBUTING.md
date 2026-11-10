# Contribuindo

## Requisitos

- Go 1.27+
- Node.js 22+ e npm
- Rust 1.90+ (MSVC no Windows)

## Antes de abrir um PR

```powershell
go test ./...
go vet ./...
gofmt -l .
cd desktop
npm run lint
npm run test
npm run build
cargo check --manifest-path src-tauri/Cargo.toml
```

Se mudar a versão, rode também `python scripts/check_version.py` na raiz.

## Estilo

- Código Go segue `gofmt`; comentários curtos e em português.
- Frontend segue ESLint/Prettier existentes.
- Mudanças de contrato JSON precisam de teste em `internal/cli` e atualização do README.
- Novas teclas da TUI devem ser cobertas em `cmd/gitcontrol/tui_test.go`.

## Commit

Use mensagens curtas no padrão do histórico, por exemplo:

```text
core: adiciona diff de arquivo
tui: atalho para fetch
desktop: painel de conflitos
ci: valida wheel no macOS
```
