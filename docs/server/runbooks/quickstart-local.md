# Local Quick-Start (macOS)

Run the backend with SQLite, **Waviate**, and **Ollama** – external services you install & run yourself.

## 1 — Install & Run Required Services

### Waviate (vector store)
```bash
# Docker (preferred)
docker run -p 8080:8080 semitechnologies/weaviate:1.24.12 --host 0.0.0.0 --port 8080 --persistencyDataPath ./data

# OR Homebrew (macOS)
brew install weaviate
weaviate --port 8080 &
```

### Ollama (embedding service)
```bash
# Homebrew (preferred)
brew install ollama

# Start the server
ollama serve &                 # http://localhost:11434

# Pull the embedding model used by Synapse (once)
ollama pull mxbai-embed-large

# OR Docker
# docker run -p 11434:11434 ollama/ollama:latest
```

Verify services:
```bash
curl http://localhost:8080/v1/meta | jq '.version'
curl http://localhost:11434/api/tags | jq    # should list pulled models
```

## 2 — Run the backend
```bash
make local     # builds binary, uses ~/.synapse-memory/memory.db
```

If Waviate or Ollama isn’t reachable the backend logs a warning with these installation steps. 