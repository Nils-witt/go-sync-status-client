package usecase

import (
	"context"
	"go-sync-status-client/internal/domain"
	"log/slog"
)

// ReceiverRepository is the port an adapter must satisfy to supply
// receivers. Defined here, on the consumer side, so this package stays in
// control of the contract.
type ReceiverRepository interface {
	ListReceivers(ctx context.Context) ([]domain.Receiver, error)
}

// ReceiverService exposes receivers to presentation adapters (e.g. the tray
// UI) without exposing how that data is sourced.
type ReceiverService struct {
	repo   ReceiverRepository
	logger *slog.Logger
}

// NewReceiverService creates a ReceiverService backed by repo.
func NewReceiverService(repo ReceiverRepository, logger *slog.Logger) *ReceiverService {
	return &ReceiverService{repo: repo, logger: logger}
}

// Receivers returns every tracked receiver.
func (s *ReceiverService) Receivers(ctx context.Context) ([]domain.Receiver, error) {
	receivers, err := s.repo.ListReceivers(ctx)
	if err != nil {
		s.logger.Error("list receivers failed", "error", err)
		return nil, err
	}
	s.logger.Debug("listed receivers", "count", len(receivers))
	return receivers, nil
}
