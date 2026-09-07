package domain

import "time"

// Receiver is a backup storage destination that receives synced data (a
// go-backup-tool "receiver" location), shown as its own tray menu section.
type Receiver struct {
	ID   string
	Path string
	// ServerName identifies which configured server this receiver came
	// from. Populated by a multi-server-aware repository; empty when
	// there's only one implicit source.
	ServerName string
	State      SyncState
	LastKey    string
	LastSeen   time.Time
	Retention  time.Duration
	StaleAfter time.Duration
}
