// Package usecase contains the application's business logic. It depends
// only on domain and on the ports it declares here — never on a concrete
// adapter.
package usecase

import (
	"context"
	"log/slog"

	"go-sync-status-client/internal/domain"
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
	if len(sources) == 0 {
		return domain.SyncStateUnknown, nil
	}

	rank := func(s domain.SyncState) int {
		switch s {
		case domain.SyncStateError:
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

	worst := sources[0].State
	for _, src := range sources[1:] {
		if rank(src.State) > rank(worst) {
			worst = src.State
		}
	}
	return worst, nil
}
