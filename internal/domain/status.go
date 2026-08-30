// Package domain holds the core entities and value objects for sync status
// tracking. It has no dependency on any framework, transport, or storage
// technology.
package domain

import "time"

// SyncState is the state a SyncSource can be in.
type SyncState int

const (
	SyncStateUnknown SyncState = iota
	SyncStateSynced
	SyncStateSyncing
	SyncStatePaused
	SyncStateError
)

func (s SyncState) String() string {
	switch s {
	case SyncStateSynced:
		return "Synced"
	case SyncStateSyncing:
		return "Syncing"
	case SyncStatePaused:
		return "Paused"
	case SyncStateError:
		return "Error"
	default:
		return "Unknown"
	}
}

// Symbol returns a short glyph used to prefix the source in menu labels.
func (s SyncState) Symbol() string {
	switch s {
	case SyncStateSynced:
		return "✓"
	case SyncStateSyncing:
		return "↻"
	case SyncStatePaused:
		return "⏸"
	case SyncStateError:
		return "✕"
	default:
		return "?"
	}
}

// SyncSource is a single item being kept in sync (a folder, a library, a
// remote endpoint, ...).
type SyncSource struct {
	ID        string
	Name      string
	State     SyncState
	Detail    string
	UpdatedAt time.Time
	Targets   []SyncTarget
}

// SyncTarget is one destination a SyncSource replicates to (e.g. a specific
// server/bucket pair). A source may fan out to several targets, each with
// its own state.
type SyncTarget struct {
	ID    string
	Label string
	State SyncState
}
