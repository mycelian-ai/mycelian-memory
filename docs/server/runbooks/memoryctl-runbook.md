# mycelian-service-tools – CLI for Memory Backend

`mycelian-service-tools` is a single-binary command-line client that wraps all public REST endpoints exposed by the Memory service. It replaces ad-hoc `curl` invocations with readable sub-commands, integrated help and generated shell-completion.

## Installation (local dev)

```
make build-mycelian-service-tools
./bin/mycelian-service-tools --help
```

Or build once and keep in `$PATH`:

```
go build -o ~/bin/mycelian-service-tools ./tools/mycelian-service-tools
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
mycelian-service-tools users create --email alice@example.com --name "Alice"
```

• Get a user
```
mycelian-service-tools users get 7f2f0c42-3de6-455f-a476-7d6e1ad7ec2a
```

### 2. Vaults

• Create a vault
```
mycelian-service-tools vaults create -u USER_ID \
  --title "Personal Vault" \
  --desc "My default vault"
```

• List vaults
```
mycelian-service-tools vaults list USER_ID
```

• Get a vault
```
mycelian-service-tools vaults get USER_ID VAULT_ID
```

### 3. Memories

• Create a memory  *(title ≤256 chars, description ≤2048 chars)*
```
mycelian-service-tools memories create USER_ID -v VAULT_ID \
  --type CONVERSATION \
  --title "Project Kick-off" \
  --desc "Chat with PM"
```

• List memories for user
```
mycelian-service-tools memories list USER_ID -v VAULT_ID
```

• Get a memory
```
mycelian-service-tools memories get USER_ID -v VAULT_ID MEMORY_ID
```

### 4. Entries

• Add an entry
```
mycelian-service-tools entries add -u USER_ID -v VAULT_ID -m MEMORY_ID \
  --raw "Discussed design trade-offs" \
  --summary "Design overview"
```

• List entries
```
mycelian-service-tools entries list -u USER_ID -v VAULT_ID -m MEMORY_ID
```

### 5. Search

```
mycelian-service-tools search \
  --user USER_ID \
  --memory MEMORY_ID \
  --query "trade-offs" \
  --topk 5
```

The command prints the raw JSON response.  Pipe through `jq` for nicer formatting:

```
mycelian-service-tools search ... | jq .
```

---

## Shell completion

Generate and source completion for zsh:

```
mycelian-service-tools completion zsh > _memoryctl
fpath+=( "$PWD" )
compinit
```

Bash and fish completions are also supported.