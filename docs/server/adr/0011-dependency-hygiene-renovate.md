---
title: Dependency Hygiene & Renovate Adoption
status: Accepted
date: 2025-07-22
---

# Context

Manual `check-deps.sh` scripts have kept direct dependencies current, but scalability demands automated monitoring, grouped PRs, and vulnerability gating once the repo moves to GitHub.

# Decision

1. Enable **Renovate** with grouping rules: one PR per Go module update batch, weekly cadence.  
2. CI workflow blocks merge unless `go test -race`, `golangci-lint`, and `govulncheck` all pass on the updated branch.  
3. Superseded internal packages must be deleted in the same PR before merge.

# Consequences

• Dependency drift is caught within a week.  
• Security advisories are surfaced automatically.  
• Codebase remains lean; legacy packages do not linger. 