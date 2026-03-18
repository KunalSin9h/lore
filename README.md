<img width="1287" height="443" alt="image" src="https://github.com/user-attachments/assets/782e0ea7-143d-43e4-8e22-10c6b19a2740" />

## yaad

> The simplest local memory engine — for you and your agents.

> [yaad.knl.co.in](https://yaad.knl.co.in/)

No servers. No SDKs. No complexity. Save anything, recall it with natural language. Works for humans in the terminal and for AI agents as a skill. Everything runs locally via Ollama — no cloud, no accounts.

```bash
# Save anything — context in the content makes it findable
yaad add "staging db is postgres on port 5433" --tag postgres
yaad add "prod nginx config at /etc/nginx/sites-enabled/app"
yaad add "deploy checklist: run migrations, restart workers, clear cache"

# Set a reminder
yaad add "book conference ticket" --remind "in 30 minutes"

# Ask anything
yaad ask "what's the staging db port?"
yaad ask "do I have anything due tonight?"
```

---

## Table of Contents

- [Features](#features)
- [Requirements](#requirements)
- [Installation](#installation)
- [Agent Integration](#agent-integration)
- [Configuration](#configuration)
- [Usage](#usage)
- [Reminders](#reminders)
- [Architecture](#architecture)
- [Project structure](#project-structure)
- [Contributing](#contributing)
- [License](#license)

---

## Features

- **Simple by design** — one binary, one command to save, one to recall
- **Query-first** — natural language search powered by local embeddings + LLM
- **Agent Skill** — install via `npx skills add kunalsin9h/yaad`, works with Claude Code, Cursor, Codex, and 39+ agents
- **Smart reminders** — parse `"in 30 minutes"`, `"tomorrow 9am"`, `"Friday 3pm"` into real deadlines
- **Rich metadata** — every memory captures working directory, hostname, and timestamp automatically
- **Fully local** — all AI runs via [Ollama](https://ollama.com), no data leaves your machine
- **Offline-safe** — saves gracefully even when Ollama is not running
- **Ports & Adapters architecture** — every component is swappable (storage, AI, notifier)

---

## Requirements

- [Ollama](https://ollama.com) running locally

Pull the required models once:

```bash
ollama pull mxbai-embed-large  # embeddings
ollama pull llama3.2:3b        # reasoning (or any chat model you prefer)
```

---

## Installation

**Linux / macOS** — one-liner:

```bash
curl -fsSL https://yaad.knl.co.in/install.sh | bash
```

**Go install:**

```bash
go install github.com/kunalsin9h/yaad/cmd/yaad@latest
```

**Pre-built binaries** — download from [GitHub Releases](https://github.com/KunalSin9h/yaad/releases).

**Build from source** (requires [Go](https://go.dev) 1.21+):

```bash
git clone https://github.com/kunalsin9h/yaad
cd yaad
make install
```

---

## Agent Integration

yaad ships as an [Agent Skill](https://github.com/vercel-labs/skills) — compatible with Claude Code, Cursor, Codex CLI, Gemini CLI, and any agent that supports the open skills standard.

```bash
# install globally — available in every project and agent session
npx skills add kunalsin9h/yaad -g

# or project-scoped
npx skills add kunalsin9h/yaad
```

Once installed, your agent can save and recall memory across sessions:

```bash
/yaad "what's the prod db setup?"                          # recall
/yaad add "prod uses nginx, config at /etc/nginx/..."      # save
/yaad add "review PR #42" --remind "tomorrow 9am"          # remind
```

---

## Configuration

Configuration is read from `~/.yaadrc`. Generate it with commented defaults:

```bash
yaad config init
```

Common commands:

```bash
yaad config set ollama.chat_model mistral
yaad config list
```

See [CONFIG.md](./CONFIG.md) for all keys, notifier options, CLI flag overrides, and data storage location.

---

## Usage

### Save a memory

```bash
yaad add "<content>" [flags]

Flags:
      --remind  string   When to remind you ("in 30 minutes", "tomorrow 9am")
      --type    string   Override type detection: command|note|reminder|url|fact
      --tag     string   Add a tag (repeatable)
```

Put context directly in the content — the AI embeds the full string, so searchable context belongs there:

```bash
# facts and commands
yaad add "staging db is postgres on port 5433" --tag postgres
yaad add "prod login: ssh -i ~/.ssh/id_rsa user@bastion.internal"
yaad add "API rate limit is 100 req/min per token" --tag api

# URLs
yaad add "stripe charges API: https://docs.stripe.com/api/charges" --tag stripe

# reminders
yaad add "book conference ticket" --remind "in 30 minutes"
yaad add "submit PR for review" --remind "tomorrow 9am"
```

### Query your memories

```bash
yaad ask "what's the staging db port?"
yaad ask "how do I log into prod?"
yaad ask "do I have anything due tonight?"
```

### Browse memories

```bash
yaad list                   # 20 most recent
yaad list --type command    # only commands
yaad list --tag postgres    # by tag
yaad list --remind          # pending reminders only
yaad list --limit 50
```

### Get full details

```bash
yaad get 01KKXKKJ3Q         # by ID (prefix is fine)
```

### Delete

```bash
yaad delete 01KKXKKJ3Q      # prompts for confirmation
yaad delete 01KKXKKJ3Q -y   # skip confirmation
```

---

## Reminders

### Inline — shell `PROMPT_COMMAND` (recommended)

Reminders surface directly in your terminal on every prompt — no background process needed.

Add to your `~/.bashrc` or `~/.zshrc`:

```bash
export PROMPT_COMMAND="yaad check; $PROMPT_COMMAND"
```

For `zsh`, add to `~/.zshrc`:

```zsh
precmd() { yaad check }
```

### Background daemon — systemd user service

```bash
yaad daemon install          # writes ~/.config/systemd/user/yaad.service
systemctl --user enable --now yaad
```

---

## Architecture

`yaad` follows the **Ports and Adapters** (Hexagonal) pattern. The domain and application logic are fully isolated from infrastructure — every adapter is replaceable without touching business logic.

```
CLI (Cobra)
    │
Application Layer
    ├── MemoryService   — add, ask, list, delete
    └── ReminderService — check, daemon
         │
    Ports (interfaces)
    ├── StoragePort   ← SQLiteAdapter (modernc.org/sqlite, pure Go)
    ├── AIPort        ← OllamaAdapter (direct HTTP, no SDK dep)
    ├── TimeParserPort← WhenAdapter   (github.com/olebedev/when)
    ├── NotifierPort  ← NotifySend / Stdout (auto-detected)
    └── ConfigPort    ← SQLiteAdapter (same DB, config table)
```

Swapping any layer requires implementing one interface. For example, to use ChromaDB for vector search, write a `ChromaAdapter` that satisfies `StoragePort` — the rest of the app is unchanged.

---

## Project structure

```
yaad/
├── cmd/yaad/main.go          # entry point + dependency wiring
├── internal/
│   ├── domain/                     # Memory, MemoryType, errors — no deps
│   ├── ports/                      # interfaces only — no deps
│   ├── app/                        # business logic — depends only on ports
│   └── adapters/
│       ├── sqlite/                 # StoragePort + ConfigPort
│       ├── ollama/                 # AIPort (direct HTTP)
│       ├── timeparser/             # TimeParserPort
│       └── notifier/               # NotifierPort (notify-send + stdout)
├── skills/yaad/                    # Agent Skill (npx skills add kunalsin9h/yaad)
├── SPEC.md                         # product specification
└── Makefile
```

---

## Contributing

Contributions are welcome. The architecture is designed to make new adapters easy to add:

- **New AI backend** (OpenAI, Gemini) → implement `ports.AIPort`
- **New storage backend** (ChromaDB, Postgres) → implement `ports.StoragePort`
- **New notifier** (macOS, Slack, email) → implement `ports.NotifierPort`

Please open an issue before starting large changes.

---

## License

MIT © [Kunal Singh](https://github.com/kunalsin9h)
