//go:build local

package events

// EventKind represents the type of domain event produced by the storage layer.
// Add more kinds as indexing needs evolve.

type EventKind string

const (
	EventMemoryCreated      EventKind = "memory_created"
	EventMemoryEntryCreated EventKind = "memory_entry_created"
)

// Event encapsulates the minimum data required by the local indexer.
// Only IDs are carried – indexer can query full record from storage.

type Event struct {
	Kind     EventKind
	UserID   string
	MemoryID string
	EntryID  *string // optional – present for EntryCreated
}

// Bus is a lightweight in-process pub-sub implementation backed by a buffered channel.

type Bus struct {
	ch chan Event
}

// NewBus creates a bus with the given buffer size.
func NewBus(buffer int) *Bus {
	return &Bus{ch: make(chan Event, buffer)}
}

// Publish attempts to enqueue the event without blocking.
// Returns true if published, false if the buffer is full.
func (b *Bus) Publish(evt Event) bool {
	select {
	case b.ch <- evt:
		return true
	default:
		return false
	}
}

// Subscribe returns a read-only channel for consumers.
func (b *Bus) Subscribe() <-chan Event {
	return b.ch
}

var defaultBus *Bus

// InitDefault initialises the package-level singleton used by storage and indexer.
func InitDefault(buffer int) {
	defaultBus = NewBus(buffer)
}

// Default returns the global bus (may be nil if not initialised).
func Default() *Bus {
	return defaultBus
}

// Publish enqueues via the default bus if initialised.
func Publish(evt Event) bool {
	if defaultBus == nil {
		return false
	}
	return defaultBus.Publish(evt)
}

// Subscribe returns the channel from the default bus if initialised, otherwise a closed channel.
func Subscribe() <-chan Event {
	if defaultBus == nil {
		c := make(chan Event)
		close(c)
		return c
	}
	return defaultBus.Subscribe()
}
