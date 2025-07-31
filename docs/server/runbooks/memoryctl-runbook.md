# memoryctl – CLI for Memory Backend

`memoryctl` is a single-binary command-line client that wraps all public REST endpoints exposed by the Memory service.  It aims to replace ad-hoc `curl` invocations with readable sub-commands, integrated help and generated shell-completion.

## Installation (local dev)

```
go run ./cmd/memoryctl --help
```

Or build once and keep in `$PATH`:

```
go build -o ~/bin/memoryctl ./cmd/memoryctl
```

## Global options

| Flag | Default | Description |
|------|---------|-------------|
| `--api` | `http://localhost:8080` | Base URL of the running memory-service |
| `--vault` | *(none)* | Vault ID. Required for most memory operations. |

All sub-commands inherit `--api`.

---

## Commands & examples

### 1. Users

• Create a user
```
memoryctl users create --email alice@example.com --name "Alice"
```

• Get a user
```
memoryctl users get 7f2f0c42-3de6-455f-a476-7d6e1ad7ec2a
```

### 2. Vaults

• Create a vault
```
memoryctl vaults create -u USER_ID \
  --title "Personal Vault" \
  --desc "My default vault"
```

• List vaults
```
memoryctl vaults list USER_ID
```

• Get a vault
```
memoryctl vaults get USER_ID VAULT_ID
```

### 3. Memories

• Create a memory  *(title ≤256 chars, description ≤2048 chars)*
```
memoryctl memories create USER_ID -v VAULT_ID \
  --type CONVERSATION \
  --title "Project Kick-off" \
  --desc "Chat with PM"
```

• List memories for user
```
memoryctl memories list USER_ID -v VAULT_ID
```

• Get a memory
```
memoryctl memories get USER_ID -v VAULT_ID MEMORY_ID
```

### 4. Entries

• Add an entry
```
memoryctl entries add -u USER_ID -v VAULT_ID -m MEMORY_ID \
  --raw "Discussed design trade-offs" \
  --summary "Design overview"
```

• List entries
```
memoryctl entries list -u USER_ID -v VAULT_ID -m MEMORY_ID
```

### 5. Search

```
memoryctl search \
  --user USER_ID \
  --memory MEMORY_ID \
  --query "trade-offs" \
  --topk 5
```

The command prints the raw JSON response.  Pipe through `jq` for nicer formatting:

```
memoryctl search ... | jq .
```

---

## Shell completion

Generate and source completion for zsh:

```
memoryctl completion zsh > _memoryctl
fpath+=( "$PWD" )
compinit
```

Bash and fish completions are also supported.