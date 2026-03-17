package domain

import "time"

// MemoryType classifies what kind of information is stored.
type MemoryType string

const (
	MemoryTypeCommand  MemoryType = "command"
	MemoryTypeNote     MemoryType = "note"
	MemoryTypeReminder MemoryType = "reminder"
	MemoryTypeURL      MemoryType = "url"
	MemoryTypeFact     MemoryType = "fact"
)

// Memory is the core domain entity — one piece of remembered information.
type Memory struct {
	ID         string
	Content    string
	ForLabel   string     // human context: "why did I save this?"
	Type       MemoryType
	Tags       []string
	WorkingDir string     // cwd captured automatically at save time
	Hostname   string     // machine identity
	CreatedAt  time.Time
	RemindAt   *time.Time // nil if not a reminder
	RemindedAt *time.Time // nil until notification fires
	Embedding  []float32  // vector for semantic search
}

// ListFilter controls which memories StoragePort.List returns.
type ListFilter struct {
	Type          MemoryType
	Tag           string
	Limit         int
	OnlyReminders bool
}
