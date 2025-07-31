# Synapse Client Architecture Design

This document replaces the former `client_design_and_milestones.md`.  It focuses solely on architecture; roadmap and timeline content now lives in `docs/product/roadmap.md`.

## Further Reading
* Concurrency details – `client_concurrency_model.md`
* Read-consistency ADR – `../adr/0012-read-consistency-model.md`

---

Here's a structured, LLM-friendly design document capturing all the client-side components, clearly and holistically.

---

# Synapse MCP Client-Side Architecture

**Goal:** Provide a clear, comprehensive design for implementing client-side logic of Synapse MCP, including context management, policy handling, MCP communication, and client-server interactions.

---

## 1. 📌 **Client-Side Overview**

The Synapse MCP client provides a reliable, performant, and flexible runtime for LLM agents to manage memory contexts and interactions with the Synapse backend. It consists of:

* **MCP Server (local sidecar)**: Exposes JSON-based APIs via stdio or local TCP.
* **Go SDK Runtime**: Handles context snapshots, queues, retries, idempotency.
* **Client-managed Policies**: Define rules for loading, updating, trimming, and managing context fragments.

---

## 2. 🗃 **Context Model**

### Definition

A **Context** consists of multiple small, modular fragments, each independently manageable:

* **`activeContext`**: Current task details.
* **`progress`**: Recent actions.
* **`technicalContext`**: Technical setup.
* **`architecture`**: Structural designs.
* **`projectBrief`**: Goals and vision.

Fragments are plain text or markdown.

### Storage and Management

* Stored as plain files; each write operation uploads a *zip snapshot* of the current directory alongside the entry event.
* MCP server sends the snapshot with every write; backend replaces its copy before applying the entry, guaranteeing consistency.
* No local database; metadata lives in the filesystem (optional `_meta.json`).

---

## 3. 📚 **Memory Policies**

### Types of Policies

| Policy Type          | Definition                                     | Ownership               |
| -------------------- | ---------------------------------------------- | ----------------------- |
| **Default Policies** | Built-in MCP default rules per memory type.    | Embedded in MCP binary. |
| **Custom Policies**  | Client-specific overrides of default policies. | Managed on disk.        |

### Policy Storage (Filesystem)

Default prompt templates ship with the binary.  User overrides live inside the context directory under `_prompts/` and are versioned via Git or simple file copy—no DB rows needed.

---

## 4. 📥 **MCP Server APIs (Local)**

Agents communicate with MCP server via JSON frames:

| Operation          | Description                         |
| ------------------ | ----------------------------------- |
| `open_memory`      | Begin session, get context manifest |
| `context_manifest` | Lists available fragments           |
| `context_get`      | Request fragment content            |
| `context_file`     | Fragment content reply              |
| `context_put`      | Update local fragment               |
| `context_put_ack`  | Confirm fragment stored locally     |
| `add_memories`     | Persist entries + snapshot          |
| `add_ack`          | Async confirmation                  |
| `search_memories`  | Retrieve stored entries             |
| `search_result`    | Return top-k results                |
| `close_session`    | Flush pending changes               |
| `close_ack`        | Final session confirmation          |
| `error`            | Signal issues                       |

---

## 5. 📤 **Client-to-Backend APIs (Cloud)**

MCP server sends data to Synapse backend:

* **Memories**: `POST /v1/memories`, `GET /v1/memories`, `DELETE /v1/memories/{memoryId}`.
* **Context Fragments**: `GET /v1/memories/{memoryId}/fragments`, `PUT /v1/memories/{memoryId}/fragments/{fragmentType}`.
* **Entries**: `POST /v1/memories/{memoryId}/entries`, `GET /v1/memories/{memoryId}/entries`.

---

## 6. 💽 **Local Storage Layout**

Context directory example:

```
~/.synapse/contexts/<memoryId>/
├── prompt.md
├── fragments/
│   ├── intro.md
│   └── faq.md
└── _meta.json
```

Each `add_entry` request includes a zip of this directory (`contextZip`).

---

## 7. 🔄 **Client Interaction Workflow**

**Session Initialization** → **In-Session Operations** → **Session Completion** (flush & close).

---

## 8. 🚩 **Error Handling & Idempotency**

MCP provides idempotency via `x-idempotency-key`; explicit error frames signal issues.

---

## 9. 📏 **Operational Constraints**

(... same table ...)

---

## 10. 📦 **Repository Structure**

(... same tree ...)

---

## 👍 **Benefits**

Reliable | Performant | Flexible | Transparent. 