// Package tray adapts the usecase layer to a system tray (menu bar) UI,
// built on getlantern/systray.
package tray

import (
	"context"
	"fmt"
	"go-sync-status-client/internal/domain"
	"go-sync-status-client/internal/usecase"
	"log/slog"
	"time"

	"github.com/getlantern/systray"
)

// App renders sync status in the system tray.
type App struct {
	service         *usecase.StatusService
	receiverService *usecase.ReceiverService
	logger          *slog.Logger
	refreshInterval time.Duration

	sourceItems   map[string]*systray.MenuItem
	targetItems   map[string]*systray.MenuItem
	receiverItems map[string]*systray.MenuItem
	refreshItem   *systray.MenuItem
	quitItem      *systray.MenuItem

	// sourceStates and fetchFailed track prior observations so
	// notifySourceTransition/notifyFetchFailure/notifyFetchRecovered can
	// detect state changes worth a desktop notification.
	sourceStates map[string]domain.SyncState
	fetchFailed  bool
}

// NewApp builds the tray app. refreshInterval is how often sync status is
// automatically re-checked; a non-positive value disables auto refresh, so
// status only updates when the user clicks Refresh.
func NewApp(service *usecase.StatusService, receiverService *usecase.ReceiverService, logger *slog.Logger, refreshInterval time.Duration) *App {
	return &App{
		service:         service,
		receiverService: receiverService,
		logger:          logger,
		refreshInterval: refreshInterval,
		sourceItems:     make(map[string]*systray.MenuItem),
		targetItems:     make(map[string]*systray.MenuItem),
		receiverItems:   make(map[string]*systray.MenuItem),
		sourceStates:    make(map[string]domain.SyncState),
	}
}

// Run starts the tray event loop. It blocks until Quit is selected.
func (a *App) Run() {
	systray.Run(a.onReady, a.onExit)
}

func (a *App) onReady() {
	a.logger.Debug("tray ready")
	systray.SetIcon(stateIcon(domain.SyncStateUnknown))
	systray.SetTooltip("Sync Status")

	ctx := context.Background()
	sources, err := a.service.Sources(ctx)
	a.logger.Info("initial sources fetch", "count", sources, "error", err)
	if err != nil {
		a.logger.Error("initial sources fetch failed", "error", err)
		systray.SetIcon(stateIcon(domain.SyncStateError))
		systray.SetTooltip(fmt.Sprintf("Sync Status: %v", err))
		a.notifyFetchFailure(err)
		return
	}

	multi := multiServer(sources)
	a.logger.Info("initial sources fetch", "count", len(sources), "multi_server", multi)
	for _, src := range sources {
		item := systray.AddMenuItem(menuLabel(src, multi), src.Detail)
		// item.Disable() // informational row, not an action
		a.sourceItems[src.ID] = item
		a.sourceStates[src.ID] = src.State

		for _, tgt := range src.Targets {
			sub := item.AddSubMenuItem(targetLabel(tgt), "")
			sub.Disable() // informational row, not an action
			a.targetItems[targetKey(src.ID, tgt.ID)] = sub
		}
	}

	a.addReceiverSection(ctx)

	systray.AddSeparator()
	a.refreshItem = systray.AddMenuItem("Refresh", "Re-check sync status")
	a.quitItem = systray.AddMenuItem("Quit", "Quit the sync status client")

	a.setOverallIcon(sources)

	go a.handleClicks(ctx)
}

func (a *App) handleClicks(ctx context.Context) {
	var tick <-chan time.Time
	if a.refreshInterval > 0 {
		ticker := time.NewTicker(a.refreshInterval)
		defer ticker.Stop()
		tick = ticker.C
	}

	for {
		select {
		case <-a.refreshItem.ClickedCh:
			a.logger.Debug("refresh clicked")
			a.refresh(ctx)
		case <-tick:
			a.logger.Debug("auto refresh tick")
			a.refresh(ctx)
		case <-a.quitItem.ClickedCh:
			a.logger.Info("quit clicked")
			systray.Quit()
			return
		}
	}
}

func (a *App) refresh(ctx context.Context) {
	sources, err := a.service.Sources(ctx)
	if err != nil {
		a.logger.Error("refresh sources failed", "error", err)
		systray.SetIcon(stateIcon(domain.SyncStateError))
		systray.SetTooltip(fmt.Sprintf("Sync Status: %v", err))
		a.notifyFetchFailure(err)
		return
	}
	a.notifyFetchRecovered()

	multi := multiServer(sources)
	for _, src := range sources {
		a.notifySourceTransition(src, multi)

		item, ok := a.sourceItems[src.ID]
		if !ok {
			continue
		}
		item.SetTitle(menuLabel(src, multi))
		item.SetTooltip(src.Detail)

		for _, tgt := range src.Targets {
			sub, ok := a.targetItems[targetKey(src.ID, tgt.ID)]
			if !ok {
				continue
			}
			sub.SetTitle(targetLabel(tgt))
		}
	}

	a.refreshReceiverSection(ctx)

	a.setOverallIcon(sources)
}

func (a *App) setOverallIcon(sources []domain.SyncSource) {
	state := usecase.OverallStateOf(sources)
	systray.SetIcon(stateIcon(state))
	systray.SetTooltip(fmt.Sprintf("Sync Status: %s (updated %s)", state, time.Now().Format("15:04:05")))
}

func (a *App) onExit() {
	a.logger.Info("tray exited")
}

// multiServer reports whether sources span more than one distinct
// ServerName, in which case labels should be prefixed with their origin
// server to disambiguate them.
func multiServer(sources []domain.SyncSource) bool {
	names := make(map[string]struct{}, len(sources))
	for _, src := range sources {
		names[src.ServerName] = struct{}{}
		if len(names) > 1 {
			return true
		}
	}
	return false
}

// displayName returns src's name, prefixed with its server when multi is
// true (i.e. more than one server is configured); a single-server setup
// renders exactly as it did before server support was added.
func displayName(src domain.SyncSource, multi bool) string {
	if multi && src.ServerName != "" {
		return src.ServerName + ": " + src.Name
	}
	return src.Name
}

func menuLabel(src domain.SyncSource, multi bool) string {
	return fmt.Sprintf("%s %s — %s (last run %s)", src.State.Symbol(), displayName(src, multi), src.State, formatLastRun(src.UpdatedAt))
}

// formatLastRun renders a source's UpdatedAt for display in a menu label.
// A zero time means the source has never run.
func formatLastRun(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	return t.Format("15:04:05")
}

func targetLabel(tgt domain.SyncTarget) string {
	return fmt.Sprintf("%s %s — %s", tgt.State.Symbol(), tgt.Label, tgt.State)
}

func targetKey(sourceID, targetID string) string {
	return sourceID + "|" + targetID
}
