// Package demo provides an in-memory StatusRepository seeded with fixed
// demo data, standing in for a real sync backend.
package demo

import (
	"context"
	"go-sync-status-client/internal/domain"
	"go-sync-status-client/internal/usecase"
	"time"
)

// Repository is a StatusRepository backed by a static, in-memory demo
// dataset.
type Repository struct{}

// NewRepository creates a Repository backed by static demo data.
func NewRepository() *Repository {
	return &Repository{}
}

// ListSources returns the fixed demo dataset.
func (r *Repository) ListSources(_ context.Context) ([]domain.SyncSource, error) {
	now := time.Now()
	return []domain.SyncSource{
		{
			ID:        "photos",
			Name:      "Photos",
			State:     domain.SyncStateSynced,
			Detail:    "12,482 items",
			UpdatedAt: now.Add(-2 * time.Minute),
		},
		{
			ID:        "documents",
			Name:      "Documents",
			State:     domain.SyncStateSyncing,
			Detail:    "134 of 900 files",
			UpdatedAt: now,
		},
		{
			ID:        "backup",
			Name:      "Backup",
			State:     domain.SyncStateError,
			Detail:    "Connection timed out",
			UpdatedAt: now.Add(-10 * time.Minute),
		},
		{
			ID:        "music",
			Name:      "Music",
			State:     domain.SyncStatePaused,
			Detail:    "Paused by user",
			UpdatedAt: now.Add(-1 * time.Hour),
		},
	}, nil
}

var _ usecase.StatusRepository = (*Repository)(nil)
