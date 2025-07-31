//go:build local

package sqlite

import "memory-backend/internal/events"

func publishMemoryCreated(userID, memoryID string) {
	events.Publish(events.Event{
		Kind:     events.EventMemoryCreated,
		UserID:   userID,
		MemoryID: memoryID,
	})
}

func publishMemoryEntryCreated(userID, memoryID, entryID string) {
	events.Publish(events.Event{
		Kind:     events.EventMemoryEntryCreated,
		UserID:   userID,
		MemoryID: memoryID,
		EntryID:  &entryID,
	})
}
