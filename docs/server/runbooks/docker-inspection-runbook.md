# Docker Inspection Runbook

> Quick reference for viewing status, logs, and diagnostics of the **memory-backend** docker stack (or any Docker Compose project).

---

## 1. Prerequisites

* Docker ≥ 24.x installed and running.
* Project stack started via `docker compose up -d` from repo root.
* Terminal positioned in project root unless noted.

---

## 2. List Running Containers

| Purpose | Command |
|---------|---------|
| Show all running containers | `docker ps` |
| Filter to stack containers | `docker ps --filter label=com.docker.compose.project=memory-backend` |
| Abbreviated status (name & uptime) | `docker ps --format "{{.Names}} {{.Status}}"` |

> **Tip** – Add more `--filter name=<string>` clauses to narrow to a service (e.g. `memory-service`).

---

## 3. Inspect Container State & Restarts

```
# Generic template
$ docker inspect -f '{{ .Name }}  Status={{ .State.Status }}  Restarts={{ .RestartCount }}' <container-name>

# Example
$ docker inspect -f '{{ .Name }}  Status={{ .State.Status }}  Restarts={{ .RestartCount }}' memory-backend-memory-service-1
```

* **Status** – `running`, `exited`, `paused`, etc.
* **RestartCount** – non-zero indicates crashes or manual restarts.

---

## 4. View Logs

| Scenario | Command |
|----------|---------|
| Last N lines (default 100) | `docker logs <container>` |
| Follow live | `docker logs -f <container>` |
| Since 30s/5m/1h | `docker logs --since 30s <container>` |
| Timestamps | `docker logs --timestamps <container>` |

Examples:
```
# Tail & follow memory-service logs
$ docker logs -f memory-backend-memory-service-1

# Last 5 minutes of indexer logs
$ docker logs --since 5m memory-backend-indexer-prototype-1
```

---

## 5. Shell Into a Container

```
$ docker exec -it <container> /bin/sh   # Alpine images
$ docker exec -it <container> /bin/bash # Debian/Ubuntu images
```

Useful for inspecting the filesystem, running health checks, or debugging.

---

## 6. Check Resource Usage

```
# Live CPU / memory per container
$ docker stats --no-stream
```

Add `--format` to customise output, e.g.:
```
$ docker stats --no-stream --format "{{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}"
```

---

## 7. Examine Environment & Config

```
# Environment variables
$ docker inspect -f '{{ range $index, $value := .Config.Env }}\n{{ $value }}{{ end }}' <container>

# Port mappings
$ docker port <container>
```

---

## 8. Health Checks (if configured)

```
# Show health status field
$ docker inspect -f '{{ .Name }} health={{ .State.Health.Status }}' <container>
```

A status of `unhealthy` usually triggers automatic restarts.

---

## 9. Debugging Restart Loops

1. Identify containers in a `restarting` state:
   ```
   $ docker ps --filter "status=restarting"
   ```
2. Inspect restart counts & last exit code:
   ```
   $ docker inspect -f '{{ .Name }} RestartCount={{ .RestartCount }} LastExit={{ .State.ExitCode }}' <container>
   ```
3. View error message:
   ```
   $ docker inspect -f '{{ .State.Error }}' <container>
   ```
4. Check logs around each crash:
   ```
   $ docker logs --since 2m <container>
   ```

---

## 10. Cleanup Helpers

| Action | Command |
|--------|---------|
| Stop all stack containers | `docker compose down` |
| Remove dangling images | `docker image prune` |
| Remove unused volumes | `docker volume prune` |

---

## 11. Quick Cheatsheet

```
# List stack containers with status
alias dps='docker ps --filter label=com.docker.compose.project=memory-backend --format "{{.Names}} {{.Status}}"'

# Tail 100 lines and follow logs for memory-service
alias dmem='docker logs -n 100 -f memory-backend-memory-service-1'

# Inspect restart counts for both critical services
alias drestarts='docker inspect -f "{{ .Name }} {{ .RestartCount }}" memory-backend-memory-service-1 memory-backend-indexer-prototype-1'
```

Add these to your `~/.zshrc` or `~/.bashrc` for convenience.

---

_Last updated: 2025-07-27_ 