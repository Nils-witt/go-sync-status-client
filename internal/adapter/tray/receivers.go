package tray

import (
	"context"
	"fmt"
	"go-sync-status-client/internal/domain"
	"go-sync-status-client/internal/usecase"

	"github.com/getlantern/systray"
)

// addReceiverSection fetches receivers and renders them as a separate menu
// section: its own separator, a disabled header, then one row per server
// whose submenu holds one row per receiver on that server. Unlike a sources
// fetch failure, a receivers fetch failure doesn't abort the whole menu
// build — it just renders a single disabled "unavailable" row in the
// receivers section.
func (a *App) addReceiverSection(ctx context.Context) {
	receivers, err := a.receiverService.Receivers(ctx)
	if err != nil {
		a.logger.Error("initial receivers fetch failed", "error", err)
		systray.AddSeparator()
		systray.AddMenuItem("Receivers", "").Disable()
		systray.AddMenuItem(fmt.Sprintf("%s Receivers unavailable — %v", domain.SyncStateError.Symbol(), err), "").Disable()
		return
	}

	groups := groupReceiversByServer(receivers)
	a.logger.Info("initial receivers fetch", "count", len(receivers), "servers", len(groups))

	systray.AddSeparator()
	systray.AddMenuItem("Receivers", "").Disable()
	for _, g := range groups {
		serverItem := systray.AddMenuItem(receiverServerLabel(g), "")
		a.receiverServerItems[g.serverName] = serverItem
		for _, rcv := range g.receivers {
			item := serverItem.AddSubMenuItem(receiverLabel(rcv), receiverDetail(rcv))
			a.receiverItems[receiverKey(rcv)] = item
			a.receiverStates[receiverKey(rcv)] = rcv.State
		}
	}
}

// refreshReceiverSection re-fetches receivers and updates existing server
// and receiver rows in place. It never touches the sources-driven overall
// icon/tooltip; a receivers fetch failure is logged and otherwise ignored,
// leaving the section showing its last-known values. Servers or receivers
// that weren't present at startup aren't added.
func (a *App) refreshReceiverSection(ctx context.Context) {
	receivers, err := a.receiverService.Receivers(ctx)
	if err != nil {
		a.logger.Error("refresh receivers failed", "error", err)
		return
	}

	multi := multiServerReceivers(receivers)
	for _, g := range groupReceiversByServer(receivers) {
		if serverItem, ok := a.receiverServerItems[g.serverName]; ok {
			serverItem.SetTitle(receiverServerLabel(g))
		}

		for _, rcv := range g.receivers {
			a.notifyReceiverTransition(rcv, multi)

			item, ok := a.receiverItems[receiverKey(rcv)]
			if !ok {
				continue
			}
			item.SetTitle(receiverLabel(rcv))
			item.SetTooltip(receiverDetail(rcv))
		}
	}
}

// receiverGroup is the set of receivers reported by one server.
type receiverGroup struct {
	serverName string
	receivers  []domain.Receiver
}

// groupReceiversByServer groups receivers by ServerName, keeping servers
// and the receivers within each in the order they were first seen.
func groupReceiversByServer(receivers []domain.Receiver) []receiverGroup {
	var groups []receiverGroup
	index := make(map[string]int)
	for _, rcv := range receivers {
		i, ok := index[rcv.ServerName]
		if !ok {
			i = len(groups)
			index[rcv.ServerName] = i
			groups = append(groups, receiverGroup{serverName: rcv.ServerName})
		}
		groups[i].receivers = append(groups[i].receivers, rcv)
	}
	return groups
}

// receiverServerLabel renders a server row: the worst state among its
// receivers, its name, and how many receivers it has.
func receiverServerLabel(g receiverGroup) string {
	states := make([]domain.SyncState, len(g.receivers))
	for i, rcv := range g.receivers {
		states[i] = rcv.State
	}

	name := g.serverName
	if name == "" {
		name = "Server"
	}
	return fmt.Sprintf("%s %s (%d)", usecase.WorstState(states...).Symbol(), name, len(g.receivers))
}

// receiverKey identifies rcv across servers, since receiver IDs are only
// unique within a single server.
func receiverKey(rcv domain.Receiver) string {
	return rcv.ServerName + "|" + rcv.ID
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

// receiverLabel renders a receiver row. It's nested under its server's
// row, so unlike displayReceiverName it never needs a server prefix.
func receiverLabel(rcv domain.Receiver) string {
	return fmt.Sprintf("%s %s", rcv.State.Symbol(), rcv.ID)
}

func receiverDetail(rcv domain.Receiver) string {
	if rcv.Path == "" {
		return ""
	}
	return fmt.Sprintf("%s (last key: %s, at: %s)", rcv.State, rcv.LastKey, formatLastRun(rcv.LastSeen))
}
