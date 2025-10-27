<div align="center">

  # sameerch's AI Memory Experiments

  [![License](https://img.shields.io/github/license/mycelian-ai/mycelian-memory)](https://github.com/mycelian-ai/mycelian-memory/blob/main/LICENSE)
  [![Discord](https://img.shields.io/discord/1396722077792469104?color=5865F2&logo=discord&logoColor=white)](https://discord.gg/mEqsYcDcAj)
  [![YouTube](https://img.shields.io/badge/Mycelian_Memory-FF0000?logo=youtube&logoColor=white)](https://www.youtube.com/playlist?list=PL45CL__T1vQjpWUHB8yqWq3F_PEYh7LBK)
</div>

---

<div align="center">

*Click the image above to watch a demo of Mycelian Memory in action.*

GitHub Hackday 2025: [Project Presentation and Demo](https://youtu.be/3UptmBLvU-c)

</div>

---

## Intent & Scope

I started this project because I was curious about AI Memory space and wanted to dive deep by building. The journey so far has been very rewarding. I started with an initial design but was proven wrong very quickly, dove deeper into the nitty-grittes, worked with real developers and understood thier pain, and lastly talked to a few real businesess and was able to guide them through high-risk one-way door decisions that will have lasting impact on thier customers and businesses.

At a more fundamental level, I’ve learned that building an AI memory product that solves real business problems is a deep, multidisciplinary challenge. It demands reliable distributed storage, careful data modeling, and serious research into memory management. Real progress requires going deep on a few core problems rather than spreading effort across shallow features.

Throughout the build my focus had been to maximize learnings. Hence, I leaned heavily on AI coding tools for rapid experimentation. That was great for exploration, but it also means this code is not production‑grade. 

### TLDR Setup
1. Checkout the project and install the dependencies.
2. Run `make start-dev-mycelian-server && make start-mcp-streamable-server` to reliably start the service and MCP server.
3. Connect to the MCP server and start tinkering.

### What **NOT** to use it for

Below list is not exclusive, please use your judgement: 
* Production or critical workloads.
* Storing customer data, secrets, or regulated information.
* Long‑lived deployments where durability, correctness, and security are requirements.

### What this project is good for

* Non‑critical personal use & tinkering: Fork it, change things, break things, and observe how behavior changes.
* Architecture exploration with AI agents: Use agents to read the code, draw diagrams, and critique the design, understand the tradeoffs.
* Prototyping eval harnesses: Try retrieval/eval ideas on small, disposable datasets.
* Teaching & demos: Share concepts with teammates or a class, using mock/synthetic data.

_Tip: Make frequent snapshots/backups of your storage database so you can roll back when you break things._

## How?
The framework organizes information in immutable timelines that preserve memory and context fidelity, enabling high precision recall without expensive inference costs during retrieval. Users maintain full control over their memory data, including deletions and corrections.

The architecture is inspired by distributed systems principles, treating memory as an append‑only log that accumulates knowledge over time rather than constantly mutating core state. To learn more about the architecture, see [the architecture document](docs/designs/001_mycelian_memory_architecture.md).

```mermaid
flowchart TD
    Agent[AI Agent] <--> MCP["`**MCP Server**
    _[Mycelian Client]_`"]
    MCP <--> Service[Memory Service]
    Service <--> Postgres[(Postgres)]
    Vector[(Vector DB)] --> Service
    Postgres <--> Worker[Outbox<br/>Worker]
    Worker --> Vector

    %% Add label to Postgres
    Postgres -.- Tables["`**Key Tables:**
    vaults
    memories
    entries
    context
    tx_outbox`"]

    classDef primary fill:#dbeafe,stroke:#1e40af,stroke-width:3px,color:#000
    classDef storage fill:#fee2e2,stroke:#dc2626,stroke-width:3px,color:#000
    classDef async fill:#e9d5ff,stroke:#7c3aed,stroke-width:3px,color:#000
    classDef note fill:#fef3c7,stroke:#d97706,stroke-width:2px,color:#000

    class Agent,MCP,Service primary
    class Postgres,Vector storage
    class Worker async
    class Tables note
```


#### What It Does Today

- **Stores agent memory** via append‑only high fidelity entry logs paired with context snapshots (context shards)
- **Organizes knowledge** through vault‑based scoping
- **Retrieves context** using hybrid search across memory entries and context shards
- **Maintains fidelity** by avoiding lossy summarization chains and graph-based memory complexity
- **Runs locally but designed to run anywhere** with self‑hostable Go backend and pluggable storage/vector database support.
- **Supports ingestion of past recorded conversations**, which will be useful during onboarding an existing agent to Mycelian.
- **Tunned using [LongMemEval](https://github.com/xiaowu0162/LongMemEval/tree/main) benchmark** However, I must warn my fellow developers to not make the decision of memory product purely based on performance on an industry benchmark. What matters is the performance on your usecase.


## Quickstart

### Server Setup

Prerequisites (please refer to [CONTRIBUTING.md](CONTRIBUTING.md)):
1. Docker Desktop
2. Ollama
3. Make & jq

```bash
# 1) Start Ollama (separate terminal)
brew install ollama   # macOS
ollama serve &
ollama pull nomic-embed-text

# 2) Start the backend stack (Postgres, Weaviate, Memory Service)
make start-dev-mycelian-server

# 3) Wait for healthy and verify
curl -s http://localhost:11545/v0/health | jq
```

The stack exposes the API on `http://localhost:11545`.

---

### Ports

| Service | Port | Notes |
| --- | --- | --- |
| MCP server | 11546 | Streamable HTTP endpoint at `/mcp` |
| Memory service (HTTP API) | 11545 | Base URL `http://localhost:11545` |
| Database (Postgres, dev) | 11544 | Host port mapped to container `5432` |
| Vector DB (Weaviate, dev) | 11543 | Host port mapped to container `8080` |

These are authoritative host ports for local/dev. Other databases or vector stores can be used, but should respect these host port assignments for consistency.

---

### MCP Server Configuration

#### For tools that support streamable MCP Servers (e.g. Cursor)

```bash
# Start the MCP server
make start-mcp-streamable-server
```

**Add to Cursor MCP config** (`~/.cursor/mcp.json`):

```json
{
  "mcpServers": {
    "mycelian-memory-streamable": {
      "url": "http://localhost:11546/mcp",
      "alwaysAllow": [
        "add_entry",
        "list_entries",
        "create_vault",
        "list_vaults",
        "list_memories",
        "get_memory",
        "create_memory_in_vault",
        "put_context",
        "get_context",
        "search_memories",
        "await_consistency"
      ]
    }
  }
}
```

#### For tools that require stdio mode (e.g. Claude Desktop)

```bash
# Build the MCP server binary
make build-mcp-server
```

**Add to Claude Desktop config** (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "mycelian-memory": {
      "command": "/path/to/mycelian-memory/bin/mycelian-mcp-server",
      "env": {
        "MEMORY_SERVICE_URL": "http://localhost:11545"
      }
    }
  }
}
```


---

## API overview

Base URL: `http://localhost:11545/v0`

```bash
# Set dev mode API key for local development
export API_KEY="LOCAL_DEV_MODE_NOT_FOR_PRODUCTION"  # pragma: allowlist secret
export MCP_PORT="11546"

# Health (no auth required)
curl -s http://localhost:11545/v0/health

# Create a vault
curl -s -X POST http://localhost:11545/v0/vaults \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"title":"notes"}'

# Create a memory inside a vault
curl -s -X POST http://localhost:11545/v0/vaults/<vaultId>/memories \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"title":"demo","memoryType":"NOTES"}'

# Put and get context (plain text)
curl -s -X PUT http://localhost:11545/v0/vaults/<vaultId>/memories/<memoryId>/contexts \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: text/plain; charset=utf-8" \
  --data-binary @context.txt

curl -s http://localhost:11545/v0/vaults/<vaultId>/memories/<memoryId>/contexts \
  -H "Authorization: Bearer $API_KEY" -H "Accept: text/plain"

# Search (requires index + embeddings to be healthy)
curl -s -X POST http://localhost:11545/v0/search \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"query":"hello", "limit":10}'
```

Auth: development mode accepts a single dev API key. Use the Go SDK helper `client.NewWithDevMode(...)` during local development instead of pasting keys.

---

## Configuration (environment)

All server configuration uses the `MEMORY_SERVER_` prefix. Useful vars:

- `MEMORY_SERVER_HTTP_PORT` (default `11545`)
- `MEMORY_SERVER_BUILD_TARGET` (`cloud-dev` by default)
- `MEMORY_SERVER_DEV_MODE` (`true|false`)
- `MEMORY_SERVER_POSTGRES_DSN` (Postgres connection string)
- `MEMORY_SERVER_SEARCH_INDEX_URL` (Weaviate host, e.g. `weaviate:8080`)
- `MEMORY_SERVER_EMBED_PROVIDER` (default `ollama`)
- `MEMORY_SERVER_EMBED_MODEL` (default `nomic-embed-text`)
- `MEMORY_SERVER_HEALTH_INTERVAL_SECONDS` (default `30`)
- `MEMORY_SERVER_HEALTH_PROBE_TIMEOUT_SECONDS` (default `2`)
- `MEMORY_SERVER_MAX_CONTEXT_CHARS` (default `65536`)
- `OLLAMA_URL` (default `http://localhost:11434`)

See `server/internal/config/config.go` for defaults and descriptions. Docker compose examples live in `deployments/docker/`.

---

## Repository layout

```text
cmd/
  memory-service/         # HTTP API server
  mycelian-mcp-server/    # MCP server (stdio/HTTP)
client/                   # Go SDK (typed, minimal surface)
server/                   # Service code, internal packages, Makefile
deployments/docker/       # Compose files for local/dev
tools/                    # CLI and service tools
docs/                     # ADRs, designs, API reference
```

For detailed information about the monorepo structure, versioning, and development workflow, see [docs/monorepo.md](docs/monorepo.md).

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for complete development setup, workflow, and contribution guidelines. Day-to-day coding expectations and command references live in [AGENTS.md](AGENTS.md).

---

## License
Apache 2.0 — see the [LICENSE](LICENSE) file for details.

---

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=mycelian-ai/mycelian-memory&type=Date)](https://star-history.com/#mycelian-ai/mycelian-memory&Date)
