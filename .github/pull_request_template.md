## Resumo

<!-- O que muda e por quê? -->

## Tipo

- [ ] correção
- [ ] funcionalidade
- [ ] documentação
- [ ] CI/infra
- [ ] refatoração

## Checklist

- [ ] `go test ./...`
- [ ] `go vet ./...`
- [ ] `gofmt -l .` sem saída
- [ ] `cd desktop && npm run lint && npm run test && npm run build`
- [ ] `cargo check --manifest-path desktop/src-tauri/Cargo.toml`
- [ ] versão consistente (`python scripts/check_version.py`), se aplicável
