// Package tray adapts the usecase layer to a system tray (menu bar) UI,
// built on getlantern/systray.
package tray

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/getlantern/systray"

	"go-sync-status-client/internal/domain"
	"go-sync-status-client/internal/usecase"
)

// App renders sync status in the system tray.
type App struct {
	service         *usecase.StatusService
	logger          *slog.Logger
	refreshInterval time.Duration

	sourceItems map[string]*systray.MenuItem
	targetItems map[string]*systray.MenuItem
	refreshItem *systray.MenuItem
	quitItem    *systray.MenuItem
}

// NewApp builds the tray app. refreshInterval is how often sync status is
// automatically re-checked; a non-positive value disables auto refresh, so
// status only updates when the user clicks Refresh.
func NewApp(service *usecase.StatusService, logger *slog.Logger, refreshInterval time.Duration) *App {
	return &App{
		service:         service,
		logger:          logger,
		refreshInterval: refreshInterval,
		sourceItems:     make(map[string]*systray.MenuItem),
		targetItems:     make(map[string]*systray.MenuItem),
	}
}

// Run starts the tray event loop. It blocks until Quit is selected.
func (a *App) Run() {
	systray.Run(a.onReady, a.onExit)
}

func (a *App) onReady() {
	a.logger.Debug("tray ready")
	systray.SetTitle("Sync")
	systray.SetTooltip("Sync Status")

	ctx := context.Background()
	sources, err := a.service.Sources(ctx)
	if err != nil {
		a.logger.Error("initial sources fetch failed", "error", err)
		systray.SetTitle("Sync ✕")
		systray.SetTooltip(fmt.Sprintf("Sync Status: %v", err))
		return
	}

	for _, src := range sources {
		item := systray.AddMenuItem(menuLabel(src), src.Detail)
		// item.Disable() // informational row, not an action
		a.sourceItems[src.ID] = item

		for _, tgt := range src.Targets {
			sub := item.AddSubMenuItem(targetLabel(tgt), "")
			sub.Disable() // informational row, not an action
			a.targetItems[targetKey(src.ID, tgt.ID)] = sub
		}
	}

	systray.AddSeparator()
	a.refreshItem = systray.AddMenuItem("Refresh", "Re-check sync status")
	a.quitItem = systray.AddMenuItem("Quit", "Quit the sync status client")

	a.setOverallTitle(ctx)

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
		systray.SetTitle("Sync ✕")
		systray.SetTooltip(fmt.Sprintf("Sync Status: %v", err))
		return
	}

	for _, src := range sources {
		item, ok := a.sourceItems[src.ID]
		if !ok {
			continue
		}
		item.SetTitle(menuLabel(src))
		item.SetTooltip(src.Detail)

		for _, tgt := range src.Targets {
			sub, ok := a.targetItems[targetKey(src.ID, tgt.ID)]
			if !ok {
				continue
			}
			sub.SetTitle(targetLabel(tgt))
		}
	}

	a.setOverallTitle(ctx)
}

func (a *App) setOverallTitle(ctx context.Context) {
	state, err := a.service.OverallState(ctx)
	if err != nil {
		a.logger.Error("overall state fetch failed", "error", err)
		systray.SetTitle("Sync ✕")
		return
	}
	systray.SetTitle(fmt.Sprintf("Sync %s", state.Symbol()))
	systray.SetTooltip(fmt.Sprintf("Sync Status: %s (updated %s)", state, time.Now().Format("15:04:05")))
}

func (a *App) onExit() {
	a.logger.Info("tray exited")
}

func menuLabel(src domain.SyncSource) string {
	return fmt.Sprintf("%s %s — %s", src.State.Symbol(), src.Name, src.State)
}

func targetLabel(tgt domain.SyncTarget) string {
	return fmt.Sprintf("%s %s — %s", tgt.State.Symbol(), tgt.Label, tgt.State)
}

func targetKey(sourceID, targetID string) string {
	return sourceID + "|" + targetID
}
