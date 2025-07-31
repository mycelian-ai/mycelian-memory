Recommended one-liners:

1. Fresh build + guaranteed single instance (keeps any DB data)  
   ```bash
   docker compose up -d --build --force-recreate
   ```
   • `--build` rebuilds the image from the current source code.  
   • `--force-recreate` removes any existing containers and starts exactly one new copy.

2. Same as above **but with a brand-new Spanner emulator (empty database)**  
   ```bash
   docker compose down -v --remove-orphans \
     && docker compose up -d --build
   ```
   • `down -v` stops and **deletes containers, networks, and volumes** so the emulator starts with a clean slate.  
   • The follow-up `up -d --build` rebuilds images and launches the stack.

Pick #1 when you just changed Go code or the Dockerfile; pick #2 when you also want to wipe all emulator data.


Waviate query 

Ways to poke around the running Weaviate container

1. Built-in GraphQL “Playground” (quickest)  
   • Open your browser at  
     http://localhost:8082  
   • The home page ships with a GraphQL UI where you can:  
     – Inspect the schema.  
     – Run ad-hoc queries / mutations and see results formatted.  
     – Save snippets while you experiment.

2. Plain cURL / httpie from a terminal  
   • Search:  
     ```bash
     curl -s -X POST http://localhost:8082/v1/graphql \
       -H 'Content-Type: application/json' \
       -d '{"query":"{ Get { MemoryEntry(limit:10) { entryId summary } } }"}' | jq .
     ```  
   • Inspect objects:  
     ```bash
     curl -s 'http://localhost:8082/v1/objects?limit=20&class=MemoryEntry' | jq .
     ```

3. Exec a shell inside the container (handy when host tools are missing)  
   ```bash
   docker compose exec weaviate sh
   # then use curl inside
   ```

4. Language clients (heavier but nice for notebooks / scripts)  
   • Python: `pip install weaviate-client`, then  
     ```python
     import weaviate
     c = weaviate.Client("http://localhost:8082")
     print(c.schema.get())
     res = c.query.get("MemoryEntry", ["entryId", "summary"]).with_limit(5).do()
     ```  
   • Go: `github.com/weaviate/weaviate-go-client/v5`, identical to what the indexer uses.

5. Third-party UIs  
   • Postman / Insomnia – import the GraphQL endpoint and use their explorers.  
   • Weaviate Console (open-source React UI) – for larger projects you can run it separately and point it at the same base URL, but the built-in playground is usually enough for dev.

Tip: Add an alias for quick cURL

```bash
alias wvql='curl -s -X POST http://localhost:8082/v1/graphql \
  -H "Content-Type: application/json" -d'
# usage:
wvql '{"query":"{ Aggregate { MemoryEntry { meta { count } } } }"}' | jq .
```

No additional containers or tools are required—the browser playground and cURL cover 95 % of ad-hoc needs.
.

go test ./e2e -v -run TestRDESingleEntryIngestion 