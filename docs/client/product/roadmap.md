# Synapse Product Roadmap

> Last updated: 2025-06-19  
> Status icons: ✅ delivered • ⏳ in-progress • 🔜 planned • 🧪 experimental

## Roadmap File Hierarchy

Detailed breakdowns now live under the `roadmap/` folder (see `.cursor/rules/milestones_roadmap.mdc` for naming conventions).

* Top-level file `roadmap/README.md` summarises active milestones.
* Each milestone has its own folder `milestone-<num>-<slug>/` containing stories and tasks.
* Status in this document remains high-level; drill down into a milestone folder for granular progress.

## Milestone Matrix (At-a-Glance)

| Milestone                        | Theme         | Key Deliverables                                             | Status      |
| -------------------------------- | ------------- | ------------------------------------------------------------ | ----------- |
| **1 – Foundation**               | Core infra    | MCP server skeleton, basic CRUD, Docker, CI pipeline         | ✅ Delivered |
| **2 – Client SDK & Concurrency** | Developer UX  | Go client SDK, shard worker pool, fast local ACK, CLI parity | ✅ Delivered |
| **3 – Context**                  | Rich memory   | `context_get/put`, fragment                                  | ✅ Delivered |


## Current Focus (Milestone 2)

See `memory-bank/activeContext.md` for day-to-day status. Headlines:

* Go Client SDK feature-by-feature migration.
* Concurrency model (ADR-0008) implementation.
* Deprecation of `get_top_k` / `top-entries` (ADR-0013).
* Inflight entry APIs gated as Labs (ADR-0014).

## Future Epics

1. **Context APIs & Policy Engine** (Milestone 3)  
   • Fine-grained fragment storage & retrieval.  
   • Policy DSL (load_on_startup, fetch_if_stale, trim_on_commit).  
   • Snapshot sync on `add_entries`.

2. **Team & Enterprise Features** (Milestone 4)  
   • Multi-user workspaces, RBAC roles.  
   • Usage-based billing & quotas.  
   • Audit trail & compliance exports.

3. **Edge Package** (Milestone 5)  
   • Mobile & IoT SDKs with transparent sync.  
   • Conflict-free Replicated Data Types (CRDT) exploration.  
   • Selective replication / bandwidth throttling.

## De-scoped / Under Evaluation

| Feature | Decision | Rationale |
|---------|----------|-----------|
| Full text search in mirror | Post-v1 | p99 latency acceptable via cloud index |
| AI-assisted tagging | TBD | Needs more customer validation |

---

### How to Propose Changes
1. Open a GitHub issue tagged `roadmap-proposal`.  
2. Attach user feedback / metrics.  
3. Core team reviews weekly and updates this file if accepted.

---

*This roadmap is forward-looking; dates and scope may change based on user feedback and resource constraints.* 