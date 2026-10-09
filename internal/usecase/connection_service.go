package usecase

import "go-sync-status-client/internal/domain"

// ConnectionRepository is the port an adapter must satisfy to report how
// status is currently being received from each server. Defined here, on
// the consumer side, so this package stays in control of the contract.
type ConnectionRepository interface {
	Connections() []domain.Connection
}

// ConnectionService exposes connection state to presentation adapters
// (e.g. the tray UI) without exposing how it is tracked.
type ConnectionService struct {
	repo ConnectionRepository
}

// NewConnectionService creates a ConnectionService backed by repo.
func NewConnectionService(repo ConnectionRepository) *ConnectionService {
	return &ConnectionService{repo: repo}
}

// Connections returns the connection state of every configured server.
func (s *ConnectionService) Connections() []domain.Connection {
	return s.repo.Connections()
}
