# rememberit — Implementation Plan

Track progress here. Check off items as they are completed.

---

## Phase 1 — Project Scaffold

- [x] `go mod init github.com/kunalsingh/rememberit`
- [x] Create full directory structure (`cmd`, `internal/domain`, `internal/ports`, `internal/app`, `internal/adapters`)
- [x] Add core dependencies to `go.mod`:
  - [x] `github.com/spf13/cobra` — CLI
  - [x] `modernc.org/sqlite` — pure Go SQLite (no CGO)
  - [x] `github.com/olebedev/when` — natural language time parsing (used directly via HTTP, no Ollama SDK needed)
  - [x] `github.com/oklog/ulid/v2` — sortable unique IDs
  - [x] `golang.org/x/sync` — errgroup for parallel AI calls
- [x] `cmd/rememberit/main.go` — entry point, root cobra command, wire dependencies

---

## Phase 2 — Domain Layer

- [x] `internal/domain/memory.go`
  - [x] `Memory` struct with all fields
  - [x] `MemoryType` enum (`command`, `note`, `reminder`, `url`, `fact`)
  - [x] `ListFilter` struct
- [x] `internal/domain/errors.go`
  - [x] `ErrNotFound`, `ErrOllamaUnavailable`, `ErrInvalidRemindExpr`

---

## Phase 3 — Ports (Interfaces)

- [x] `internal/ports/storage.go` — `StoragePort` interface
- [x] `internal/ports/ai.go` — `AIPort` interface
- [x] `internal/ports/timeparser.go` — `TimeParserPort` interface
- [x] `internal/ports/notifier.go` — `NotifierPort` interface
- [x] `internal/ports/config.go` — `ConfigPort` interface

---

## Phase 4 — SQLite Adapter

- [x] `internal/adapters/sqlite/db.go` — shared `DB` struct, `Open`, `Close`, WAL mode
- [x] `internal/adapters/sqlite/store.go`
  - [x] Schema migration on first run (create tables + indexes)
  - [x] `Save()` — upsert memory, gob-encode embedding blob
  - [x] `GetByID()` — fetch single memory
  - [x] `List()` — filter by type, tag, limit
  - [x] `Delete()` — hard delete with `ErrNotFound` on miss
  - [x] `FindSimilar()` — load all embeddings, compute cosine similarity in-process, return top-k
  - [x] `PendingReminders()` — `remind_at <= now AND reminded_at IS NULL`
  - [x] `MarkReminded()` — set `reminded_at = now`
- [x] `internal/adapters/sqlite/config.go`
  - [x] `ConfigPort` backed by `config` table in same DB

---

## Phase 5 — Ollama Adapter

- [x] `internal/adapters/ollama/client.go`
  - [x] `Embed()` — POST `/api/embeddings`, return `[]float32`
  - [x] `DetectType()` — prompt LLM, parse response into `MemoryType`
  - [x] `ExtractTags()` — prompt LLM, parse JSON array response
  - [x] `Answer()` — build context from memories, call LLM
  - [x] Configurable base URL and model names via `configHelper`
  - [x] Graceful error wrapping as `domain.ErrOllamaUnavailable`

---

## Phase 6 — Time Parser Adapter

- [x] `internal/adapters/timeparser/when.go`
  - [x] `Parse()` — wrap `github.com/olebedev/when`, return `*time.Time`
  - [x] Handle relative: `"in 30 minutes"`, `"in 2 hours"`
  - [x] Handle absolute: `"tomorrow 9am"`, `"Friday 3pm"`, `"8 PM today"`

---

## Phase 7 — Notifier Adapters

- [x] `internal/adapters/notifier/stdout.go`
  - [x] `Notify()` — print formatted reminder to stdout
- [x] `internal/adapters/notifier/notifysend.go`
  - [x] `Notify()` — exec `notify-send` with title + body
  - [x] `IsAvailable()` — detect at runtime, fallback to stdout

---

## Phase 8 — Application Services

- [x] `internal/app/memory_service.go`
  - [x] `Add()`:
    - [x] Parse `--remind` via `TimeParserPort`
    - [x] Capture `WorkingDir` + `Hostname` automatically
    - [x] Parallelize: `Embed`, `DetectType`, `ExtractTags` (goroutines + errgroup)
    - [x] `StoragePort.Save()`
    - [x] Return created memory
  - [x] `Ask()`:
    - [x] `AIPort.Embed(question)`
    - [x] `StoragePort.FindSimilar(topK=5)`
    - [x] `AIPort.Answer(question, memories)`
  - [x] `List()`
  - [x] `Delete()`
- [x] `internal/app/reminder_service.go`
  - [x] `CheckAndFire()` — fetch pending, notify each, mark reminded
  - [x] `RunDaemon()` — poll loop with configurable interval

---

## Phase 9 — CLI Commands (Cobra)

- [x] `rememberit add` command
  - [x] Flags: `--for`, `--remind`, `--type`, `--tag`
  - [x] Call `MemoryService.Add()`
  - [x] Print confirmation with ID, type, tags, remind_at
- [x] `rememberit ask` command
  - [x] Accept question as positional arg
  - [x] Print LLM answer
- [x] `rememberit list` command
  - [x] Flags: `--type`, `--tag`, `--limit`, `--remind`
  - [x] Render table: ID (short), type, content (truncated), for_label, created_at (relative)
- [x] `rememberit get` command
  - [x] By full ID
  - [x] Print full memory details
- [x] `rememberit delete` command
  - [x] Confirm prompt before deleting (`--force` to skip)
- [x] `rememberit check` command
  - [x] Silent unless reminders are due
  - [x] Designed for `PROMPT_COMMAND` integration
- [x] `rememberit daemon` command
  - [x] `start` — run poll loop in foreground (systemd manages background)
  - [x] `install` — write systemd user service file
- [x] `rememberit config` command
  - [x] `set <key> <value>`
  - [x] `get <key>`
  - [x] `list`

---

## Phase 10 — Wiring (Dependency Injection)

- [x] `cmd/rememberit/main.go`
  - [x] Determine DB path (XDG_DATA_HOME aware)
  - [x] Instantiate SQLiteAdapter (shared DB for Store + Config)
  - [x] Instantiate OllamaAdapter (direct HTTP, no SDK dependency)
  - [x] Instantiate WhenAdapter
  - [x] Instantiate NotifierAdapter (auto-detect notify-send)
  - [x] Instantiate MemoryService + ReminderService
  - [x] Pass services into cobra commands via closure

---

## Phase 11 — Quality & Polish

- [ ] Unit tests for `MemoryService.Add()` with mock adapters
- [ ] Unit tests for `ReminderService.CheckAndFire()` with mock adapters
- [ ] Integration test: SQLite adapter round-trip (save → list → delete)
- [ ] Integration test: Ollama embed + FindSimilar
- [ ] `rememberit check` shell integration docs in README
- [x] Graceful handling: Ollama not running → skip embed, save with empty embedding, warn user
- [x] `--help` text for all commands reviewed and accurate

---

## Phase 12 — Distribution

- [x] `Makefile` with `build`, `install`, `test` targets
- [ ] `go install github.com/kunalsingh/rememberit/cmd/rememberit@latest` works
- [ ] Systemd service template at `contrib/rememberit.service`

---

## Implementation Order (Recommended)

> Build vertically — one thin slice at a time, each slice is runnable.

1. ~~**Slice 1**: Scaffold + domain + ports + SQLite adapter + `add` (no AI, stores content only) + `list`~~ ✅
2. ~~**Slice 2**: Ollama embed on `add` + `FindSimilar` + `ask` (full semantic query)~~ ✅
3. ~~**Slice 3**: Time parser + `--remind` flag + `check` command~~ ✅
4. ~~**Slice 4**: Notifier adapters + daemon command + systemd install~~ ✅
5. ~~**Slice 5**: AI type detection + tag extraction on `add`~~ ✅
6. **Slice 6**: Tests, polish, distribution ← next
