# Configuration

Configuration is read from `~/.lorerc` (key = value format). Create it with commented defaults:

```bash
lore config init
```

---

## Config file (`~/.lorerc`)

```ini
# lore configuration

# Ollama server
ollama.url           = http://localhost:11434

# Embedding model — nomic-embed-text, mxbai-embed-large, all-minilm
ollama.embed_model   = nomic-embed-text

# Chat model — llama3.2:3b, mistral, gemma2:2b, phi3
ollama.chat_model    = llama3.2:3b

# Reminder daemon poll interval
reminder.poll_interval = 30s

# Notifier adapters — comma-separated, all fire together
# cli         — styled box printed to terminal (default, no dependencies)
# notify-send — desktop notification via notify-send (Linux only)
notifier = cli
```

---

## All keys

| Key | Default | Description |
|---|---|---|
| `ollama.url` | `http://localhost:11434` | Ollama server URL |
| `ollama.embed_model` | `nomic-embed-text` | Model for generating embeddings |
| `ollama.chat_model` | `llama3.2:3b` | Model for query answering and type detection |
| `reminder.poll_interval` | `30s` | How often the daemon polls for due reminders |
| `notifier` | `cli` | Notifier adapter(s), comma-separated |

---

## Priority

```
built-in defaults  <  ~/.lorerc  <  CLI flags
```

CLI flags override the rc file for a single invocation:

```bash
lore --chat-model mistral ask "what was that command?"
lore --ollama-url http://192.168.1.5:11434 add "remote ollama note"
lore --notifier cli,notify-send check
```

---

## Config commands

```bash
lore config init                            # create ~/.lorerc with defaults
lore config list                            # show all set values
lore config set ollama.chat_model mistral   # update a value
lore config get ollama.chat_model           # read a value
lore config path                            # print rc file location
```

---

## Data storage

Memories are stored at `$XDG_DATA_HOME/lore/memories.db`, defaulting to `~/.local/share/lore/memories.db`.

---

## Notifiers

Notifiers are composable — set multiple to fire all at once:

```bash
lore config set notifier cli,notify-send
```

| Adapter | Platform | Requirement |
|---|---|---|
| `cli` | All | None — prints a styled box to the terminal |
| `notify-send` | Linux | `notify-send` must be installed (`libnotify-bin`) |
