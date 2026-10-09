// Package usecase contains the application's business logic. It depends
// only on domain and on the ports it declares here — never on a concrete
// adapter.
package usecase

import (
	"context"
	"go-sync-status-client/internal/domain"
	"log/slog"
)

// StatusRepository is the port an adapter must satisfy to supply sync
// sources. Defined here, on the consumer side, so this package stays in
// control of the contract.
type StatusRepository interface {
	ListSources(ctx context.Context) ([]domain.SyncSource, error)
}

// StatusService exposes sync status to presentation adapters (e.g. the tray
// UI) without exposing how that status is sourced.
type StatusService struct {
	repo   StatusRepository
	logger *slog.Logger
}

// NewStatusService creates a StatusService backed by repo.
func NewStatusService(repo StatusRepository, logger *slog.Logger) *StatusService {
	return &StatusService{repo: repo, logger: logger}
}

// Sources returns every tracked sync source.
func (s *StatusService) Sources(ctx context.Context) ([]domain.SyncSource, error) {
	sources, err := s.repo.ListSources(ctx)
	if err != nil {
		s.logger.Error("list sources failed", "error", err)
		return nil, err
	}
	s.logger.Debug("listed sources", "count", len(sources))
	return sources, nil
}

// OverallState reduces every source to a single worst-case state, suitable
// for a tray icon summary: Error beats Syncing beats Paused beats Synced.
func (s *StatusService) OverallState(ctx context.Context) (domain.SyncState, error) {
	sources, err := s.repo.ListSources(ctx)
	if err != nil {
		s.logger.Error("list sources failed", "error", err)
		return domain.SyncStateUnknown, err
	}
	return OverallStateOf(sources), nil
}

// OverallStateOf reduces sources to a single worst-case state, suitable for
// a tray icon summary: Error beats Syncing beats Paused beats Synced.
// Exposed as a pure function so callers that already fetched sources (e.g.
// the tray, once per refresh) don't need a second ListSources round trip
// just to derive the overall state.
func OverallStateOf(sources []domain.SyncSource) domain.SyncState {
	states := make([]domain.SyncState, len(sources))
	for i, src := range sources {
		states[i] = src.State
	}
	return WorstState(states...)
}

// WorstState reduces states to the single most severe one: Error beats
// Incomplete beats Syncing beats Paused beats Synced. It returns
// SyncStateUnknown when states is empty.
func WorstState(states ...domain.SyncState) domain.SyncState {
	if len(states) == 0 {
		return domain.SyncStateUnknown
	}

	rank := func(s domain.SyncState) int {
		switch s {
		case domain.SyncStateError:
			return 5
		case domain.SyncStateIncomplete:
			return 4
		case domain.SyncStateSyncing:
			return 3
		case domain.SyncStatePaused:
			return 2
		case domain.SyncStateSynced:
			return 1
		default:
			return 0
		}
	}

	worst := states[0]
	for _, s := range states[1:] {
		if rank(s) > rank(worst) {
			worst = s
		}
	}
	return worst
}
