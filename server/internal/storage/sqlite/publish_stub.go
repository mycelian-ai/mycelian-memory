//go:build !local

package sqlite

func publishMemoryCreated(_, _ string) {}

func publishMemoryEntryCreated(_, _, _ string) {}
