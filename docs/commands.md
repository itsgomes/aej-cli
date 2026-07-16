# Comandos do AEJ

Este arquivo reúne os comandos mais comuns do `aej` com uso rápido e casos de uso mínimos.

## Autenticação

### `aej login`
- Caso de uso: configurar URL do Jira, e-mail e API token.
- Exemplo:
  ```bash
  aej login
  ```

## Consulta e acompanhamento

### `aej me`
- Caso de uso: ver informações do usuário atual.
- Exemplo:
  ```bash
  aej me
  ```

### `aej mine`
- Caso de uso: listar as últimas issues atribuídas a você.
- Exemplo:
  ```bash
  aej mine
  aej mine --status "Em andamento"
  ```

### `aej board`
- Caso de uso: listar boards disponíveis ou abrir as issues de um board.
- Exemplo:
  ```bash
  aej board
  aej board 1712
  aej board 1712 --full
  ```

### `aej search [TERMO]`
- Caso de uso: buscar issues por texto, tag ou versão.
- Exemplo:
  ```bash
  aej search "bug de login"
  aej search --tag backend
  aej search --version 2.1
  ```

### `aej issue <CHAVE>`
- Caso de uso: abrir os detalhes de uma issue específica.
- Exemplo:
  ```bash
  aej issue DEV-123
  ```

### `aej logs`
- Caso de uso: ver o tempo trabalhado em um período.
- Exemplo:
  ```bash
  aej logs
  aej logs --days 15
  aej logs --date 16-07-2026
  ```

## Ações em issues

### `aej assign <CHAVE>`
- Caso de uso: atribuir uma issue a você, a outro usuário ou remover o responsável.
- Exemplo:
  ```bash
  aej assign DEV-123
  aej assign DEV-123 --to usuario@empresa.com
  aej assign DEV-123 --unassign
  ```

### `aej transition <CHAVE>`
- Caso de uso: alterar o status de uma issue por seleção interativa.
- Exemplo:
  ```bash
  aej transition DEV-123
  ```

### `aej comment <CHAVE> <COMENTÁRIO>`
- Caso de uso: adicionar um comentário em uma issue.
- Exemplo:
  ```bash
  aej comment DEV-123 "Correção disponível para validação"
  ```

### `aej open <CHAVE>`
- Caso de uso: abrir a issue diretamente no navegador.
- Exemplo:
  ```bash
  aej open DEV-123
  ```

### `aej log <CHAVE> <TEMPO> [COMENTÁRIO]`
- Caso de uso: registrar tempo gasto em uma issue.
- Exemplo:
  ```bash
  aej log DEV-123 2h
  aej log DEV-123 30m "Revisão de código"
  aej log DEV-123 "1h 30m" "Implementando feature"
  ```

## Flags globais

- `--json`: retorna a saída em JSON.
- `--timing`: mostra o tempo total de execução.

Exemplo:

```bash
aej issue DEV-123 --json
aej search "deploy" --timing
```
