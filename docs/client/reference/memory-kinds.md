# Memory Kinds (SDK Enum Reference)

**Doc status:** draft – MVP scope

---

## Purpose
`MemoryKind` is a closed enum in the Go Client SDK that selects which trio of prompt templates (entry-capture, summary, context) should be used when a memory is created.  Keeping the set small ensures prompt maintenance remains manageable.

For MVP we ship **two** public memory types that map onto **five** fundamental kinds – only the first two are active.

| Enum `MemoryKind` | Typical public memory types | Prompt directory | Description |
|-------------------|-----------------------------|------------------|-------------|
| `CHAT`   | `conversation` | `prompts/default/chat/`   | Time-ordered dialogue, episodic recall |
| `CODE`   | `coding`       | `prompts/default/code/`   | Code snippets, review comments |
| `DOC`    | `technical_doc`, `knowledge_base` (future) | `prompts/default/doc/` | Long-form docs, facts |
| `PROFILE`| `user_profile` (future) | `prompts/default/profile/` | Stable user or agent facts |
| `TASK`   | `task_list` (future) | `prompts/default/task/` | To-dos, goals, prospective memory |

Only `CHAT` and `CODE` will have prompt files in the MVP container image.  The remaining kinds are placeholders for future milestones; their prompt directories can be empty without affecting runtime.

---

## Mapping helper (SDK)
```go
// Public API value provided by callers
 type MemoryType string

const (
    TypeConversation MemoryType = "conversation"
    TypeCoding       MemoryType = "coding"
    // future: TypeTechnicalDoc, TypeUserProfile, TypeTaskList...
)

// Internal enum used by prompt loader
 type MemoryKind string
 const (
    KindChat    MemoryKind = "chat"
    KindCode    MemoryKind = "code"
    KindDoc     MemoryKind = "doc"
    KindProfile MemoryKind = "profile"
    KindTask    MemoryKind = "task"
 )

var typeToKind = map[MemoryType]MemoryKind{
    TypeConversation: KindChat,
    TypeCoding:       KindCode,
}
```

The SDK's `CreateMemory` helper will look up `typeToKind` and embed both values in the request so that downstream components can rely on the stable enum.

---

## Prompt file layout recap
```
prompts/
└── default/
    ├── chat/
    │   ├── entry_capture_prompt.txt
    │   ├── summary_prompt.txt
    │   └── context_prompt.txt
    └── code/
        ├── entry_capture_prompt.txt
        ├── summary_prompt.txt
        └── context_prompt.txt
```

Each file ≤ 200 words (~150-180 tokens).  Override directory follows the same structure.

---

## Cognitive classes (future extension)
While the MVP excludes per-entry cognitive tagging, the following classes are planned for later phases. They explain *how* an entry may be used during retrieval and ranking.

| Class | What it captures | Typical examples | Primary default in `MemoryKind` |
|-------|------------------|------------------|---------------------------------|
| **Episodic** | Time-stamped events & dialogue turns | Daily stand-up notes, chat history | `CHAT` |
| **Semantic** | Stable facts & knowledge | "User's name is Alex", technology facts | `PROFILE`, `DOC` |
| **Procedural** | How-to instructions or code | Deployment steps, function snippet | `CODE` |
| **Prospective / Working** | To-dos and short-term goals | "Reminder to run benchmark" | `TASK` |
| **Affective** (optional) | Sentiment & preferences | "User dislikes jargon" | (none yet) |

No API changes are required to introduce these later: a `cognitive_class` enum can be added to the `Entry` record without affecting existing data.

*Last updated: 2025-06-22* 