package domain

import "time"

// Connection describes how the client currently receives status from one
// server: pushed live updates, or polling as a fallback.
type Connection struct {
	// ServerName identifies which configured server this connection is
	// to. Populated by a multi-server-aware repository; empty when there's
	// only one implicit source.
	ServerName string
	// Live reports whether live updates are currently being received.
	Live bool
	// DisconnectedAt is when live updates were last lost, or first failed
	// to connect. Zero if that never happened.
	DisconnectedAt time.Time
	// LastPoll is when status was last fetched successfully by polling.
	// Zero if it never was.
	LastPoll time.Time
}
