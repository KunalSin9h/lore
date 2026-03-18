---
name: yaad
description: yaad is a simple local memory engine for humans and agents. Use it to remember anything worth keeping across sessions, and to recall it later in natural language. Invoke when starting work to load prior context, when something is worth saving, or when the user asks about something from the past.
argument-hint: "[query] or [add <content>] or [add <content> --remind 'when']"
allowed-tools: Bash
---

yaad is a local-first memory engine. Everything is stored on the user's machine. No cloud, no accounts.

## Check yaad is installed

```bash
which yaad
```

If not found, tell the user to install it first:

```bash
curl -fsSL https://yaad.knl.co.in/install.sh | bash
```

Then stop — do not proceed until yaad is available.

---

## Commands

### Save a memory

```bash
yaad add "<content>"
yaad add "<content>" --remind "in 30 minutes"
yaad add "<content>" --remind "tomorrow 9am"
yaad add "<content>" --tag <tag>           # repeatable
yaad add "<content>" --type command|note|url|fact|reminder
```

### Recall with natural language

```bash
yaad ask "<question>"
```

### Browse saved memories

```bash
yaad list                    # 20 most recent
yaad list --type command     # filter by type
yaad list --tag postgres     # filter by tag
yaad list --remind           # pending reminders only
yaad list --limit 50
```

### Get full detail of one memory

```bash
yaad get <id>                # 10-char ULID prefix is enough
```

### Delete

```bash
yaad delete <id>
yaad delete <id> -y          # skip confirmation
```

---

## When to save

Save proactively when you encounter:

- A command that solved a non-obvious problem
- A port, hostname, or infrastructure detail the user will look up again
- A decision made about architecture, approach, or tooling
- A URL for docs or an API being actively used
- A time-based reminder the user sets
- Any fact the user explicitly says they want to remember

## Writing good memories

Put all context directly in the content — the AI embeds the full string, so searchable context belongs there:

```bash
# good — self-contained, findable later
yaad add "staging db is postgres on port 5433" --tag postgres
yaad add "prod uses nginx, config at /etc/nginx/sites-enabled/app"
yaad add "API rate limit is 100 req/min per token" --tag api
yaad add "https://docs.stripe.com/api/charges" --tag stripe --tag reference
yaad add "deploy checklist: run migrations, restart workers, clear cache"
yaad add "standup at 10am" --remind "tomorrow 9:45am"

# avoid — no context, won't recall well
yaad add "port 5433"
yaad add "check this later"
```

## When to recall

- At the start of a session: run `yaad list` to surface relevant prior context
- When the user asks "what was that X?" or "do I have anything about Y?" — run `yaad ask`
- When a task looks familiar — check if the user has prior notes on it

## Reminder setup

To surface reminders in the terminal, the user should add to their shell config:

```bash
# ~/.bashrc or ~/.zshrc
export PROMPT_COMMAND="yaad check; $PROMPT_COMMAND"

# zsh only
precmd() { yaad check }
```
