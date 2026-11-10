# Segurança

## Versões suportadas

Apenas a última minor publicada recebe correções de segurança.

## Reportando

Não abra issue pública para vulnerabilidades. Envie um e-mail para o mantenedor
ou use o recurso **Security > Report a vulnerability** do GitHub.

Inclua:

- versão do `gitcontrol`;
- sistema operacional;
- passos para reproduzir;
- impacto e, se possível, uma correção sugerida.

## Escopo

O `gitcontrol` executa o binário `git` do sistema e não gerencia credenciais
diretamente. Tokens do PyPI usados no CI ficam em secrets do GitHub e nunca no
repositório.
