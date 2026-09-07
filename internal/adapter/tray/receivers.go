package tray

import (
	"context"
	"fmt"
	"go-sync-status-client/internal/domain"

	"github.com/getlantern/systray"
)

// addReceiverSection fetches receivers and renders them as a separate menu
// section (its own separator, a disabled header, then one row per
// receiver). Unlike a sources fetch failure, a receivers fetch failure
// doesn't abort the whole menu build — it just renders a single disabled
// "unavailable" row in the receivers section.
func (a *App) addReceiverSection(ctx context.Context) {
	receivers, err := a.receiverService.Receivers(ctx)
	if err != nil {
		a.logger.Error("initial receivers fetch failed", "error", err)
		systray.AddSeparator()
		systray.AddMenuItem("Receivers", "").Disable()
		systray.AddMenuItem(fmt.Sprintf("%s Receivers unavailable — %v", domain.SyncStateError.Symbol(), err), "").Disable()
		return
	}

	multi := multiServerReceivers(receivers)
	a.logger.Info("initial receivers fetch", "count", len(receivers), "multi_server", multi)

	systray.AddSeparator()
	systray.AddMenuItem("Receivers", "").Disable()
	for _, rcv := range receivers {
		item := systray.AddMenuItem(receiverLabel(rcv, multi), receiverDetail(rcv))
		a.receiverItems[rcv.ID] = item
	}
}

// refreshReceiverSection re-fetches receivers and updates existing rows in
// place. It never touches the sources-driven overall icon/tooltip; a
// receivers fetch failure is logged and otherwise ignored, leaving the
// section showing its last-known values.
func (a *App) refreshReceiverSection(ctx context.Context) {
	receivers, err := a.receiverService.Receivers(ctx)
	if err != nil {
		a.logger.Error("refresh receivers failed", "error", err)
		return
	}

	multi := multiServerReceivers(receivers)
	for _, rcv := range receivers {
		item, ok := a.receiverItems[rcv.ID]
		if !ok {
			continue
		}
		item.SetTitle(receiverLabel(rcv, multi))
		item.SetTooltip(receiverDetail(rcv))
	}
}

// multiServerReceivers reports whether receivers span more than one
// distinct ServerName, in which case labels should be prefixed with their
// origin server to disambiguate them.
func multiServerReceivers(receivers []domain.Receiver) bool {
	names := make(map[string]struct{}, len(receivers))
	for _, rcv := range receivers {
		names[rcv.ServerName] = struct{}{}
		if len(names) > 1 {
			return true
		}
	}
	return false
}

// displayReceiverName returns rcv's ID, prefixed with its server when multi
// is true (i.e. more than one server is configured).
func displayReceiverName(rcv domain.Receiver, multi bool) string {
	if multi && rcv.ServerName != "" {
		return rcv.ServerName + ": " + rcv.ID
	}
	return rcv.ID
}

func receiverLabel(rcv domain.Receiver, multi bool) string {
	return fmt.Sprintf("%s %s — %s (last seen %s)", rcv.State.Symbol(), displayReceiverName(rcv, multi), rcv.State, formatLastRun(rcv.LastSeen))
}

func receiverDetail(rcv domain.Receiver) string {
	if rcv.Path == "" {
		return ""
	}
	return fmt.Sprintf("%s (last key: %s, retention: %s)", rcv.Path, rcv.LastKey, rcv.Retention)
}
