package tray

import (
	"fmt"
	"go-sync-status-client/internal/domain"

	"github.com/getlantern/systray"
)

// addConnectionSection renders how status is being received from each
// server (live updates, or polling as a fallback) as a separate menu
// section: its own separator, a disabled header, then one row per server.
func (a *App) addConnectionSection() {
	connections := a.connectionService.Connections()
	multi := len(connections) > 1

	systray.AddSeparator()
	systray.AddMenuItem("Connection", "").Disable()
	for _, c := range connections {
		a.connectionItems[c.ServerName] = systray.AddMenuItem(connectionLabel(c, multi), "")
	}
}

// refreshConnectionSection updates existing connection rows in place.
func (a *App) refreshConnectionSection() {
	connections := a.connectionService.Connections()
	multi := len(connections) > 1
	for _, c := range connections {
		item, ok := a.connectionItems[c.ServerName]
		if !ok {
			continue
		}
		item.SetTitle(connectionLabel(c, multi))
	}
}

// connectionLabel describes c, prefixed with its server when multi is true
// (i.e. more than one server is configured). While live updates are down,
// it says since when, and when status was last polled instead.
func connectionLabel(c domain.Connection, multi bool) string {
	prefix := ""
	if multi && c.ServerName != "" {
		prefix = " " + c.ServerName
	}

	switch {
	case c.Live:
		return fmt.Sprintf("%s%s", domain.SyncStateSynced.Symbol(), prefix)
	case c.DisconnectedAt.IsZero():
		return fmt.Sprintf("%s%s last poll %s", domain.SyncStateUnknown.Symbol(), prefix, formatLastRun(c.LastPoll))
	default:
		return fmt.Sprintf("%s%s disconnected since %s, last poll %s",
			domain.SyncStatePaused.Symbol(), prefix, formatLastRun(c.DisconnectedAt), formatLastRun(c.LastPoll))
	}
}
