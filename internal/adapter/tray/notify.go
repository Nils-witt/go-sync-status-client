package tray

import (
	"go-sync-status-client/internal/domain"

	"github.com/gen2brain/beeep"
)

func init() {
	beeep.AppName = "Sync Status"
}

// transitionKind classifies how a source's state changed between two
// observations, for desktop-notification purposes.
type transitionKind int

const (
	transitionNone transitionKind = iota
	transitionErrorOccurred
	transitionErrorCleared
)

// classifyTransition decides whether moving from prev to cur is worth a
// desktop notification. known is false the first time a source is observed
// (e.g. right after startup), so a pre-existing error doesn't fire a
// spurious "error occurred" notification the moment the app starts tracking
// it.
func classifyTransition(known bool, prev, cur domain.SyncState) transitionKind {
	if !known || prev == cur {
		return transitionNone
	}
	switch {
	case cur == domain.SyncStateError:
		return transitionErrorOccurred
	case prev == domain.SyncStateError:
		return transitionErrorCleared
	default:
		return transitionNone
	}
}

// notifySourceTransition updates the tracked state for src and, if it just
// entered or left the Error state, sends a desktop notification.
func (a *App) notifySourceTransition(src domain.SyncSource) {
	prev, known := a.sourceStates[src.ID]
	a.sourceStates[src.ID] = src.State

	switch classifyTransition(known, prev, src.State) {
	case transitionErrorOccurred:
		a.sendNotification(beeep.Alert, src.Name+": sync error", src.Detail, domain.SyncStateError)
	case transitionErrorCleared:
		a.sendNotification(beeep.Notify, src.Name+": sync restored", src.Detail, domain.SyncStateSynced)
	}
}

// notifyFetchFailure notifies once when fetching sync status starts
// failing; repeated failures on later refreshes stay silent until it
// recovers.
func (a *App) notifyFetchFailure(err error) {
	if a.fetchFailed {
		return
	}
	a.fetchFailed = true
	a.sendNotification(beeep.Alert, "Sync status unavailable", err.Error(), domain.SyncStateError)
}

// notifyFetchRecovered notifies once when fetching sync status succeeds
// again after a prior failure.
func (a *App) notifyFetchRecovered() {
	if !a.fetchFailed {
		return
	}
	a.fetchFailed = false
	a.sendNotification(beeep.Notify, "Sync status restored", "Reconnected to sync status source", domain.SyncStateSynced)
}

// sendNotification sends a desktop notification via send (beeep.Alert or
// beeep.Notify), logging rather than failing if the OS notification
// backend is unavailable (e.g. a headless session).
func (a *App) sendNotification(send func(title, message string, icon any) error, title, message string, iconState domain.SyncState) {
	if err := send(title, message, stateIcon(iconState)); err != nil {
		a.logger.Warn("desktop notification failed", "title", title, "error", err)
	}
}
