# Ports verification (local/dev)

Authoritative host ports:

- MCP: 11546
- Memory service: 11545
- Postgres: 11544 -> 5432
- Weaviate: 11543 -> 8080

## Container port mappings

```bash
$ docker ps --format "table {{.Names}}\t{{.Ports}}" | grep -E "memory|weaviate|postgres|mcp"
memory-service               0.0.0.0:11545->11545/tcp, [::]:11545->11545/tcp
memory-postgres              0.0.0.0:11544->5432/tcp, [::]:11544->5432/tcp
weaviate                     0.0.0.0:11543->8080/tcp, [::]:11543->8080/tcp
mycelian-memory-mcp-server   0.0.0.0:11546->11546/tcp, [::]:11546->11546/tcp
```

## Endpoint checks

Memory service health (11545):

```bash
$ curl -sS http://localhost:11545/v0/health
{"status":"healthy","timestamp":"2025-09-09T04:13:58Z"}
```

Weaviate meta (11543):

```bash
$ curl -sS http://localhost:11543/v1/meta | jq '{status:"ok",version:.version}'
{
  "status": "ok",
  "version": "1.31.2"
}
```

Postgres readiness (container listens on 5432; host maps 11544 -> 5432):

```bash
$ docker exec memory-postgres pg_isready -h localhost -p 5432 -U memory -d memory
localhost:5432 - accepting connections
```
