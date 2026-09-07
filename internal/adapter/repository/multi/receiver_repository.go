package multi

import (
	"context"
	"go-sync-status-client/internal/domain"
	"go-sync-status-client/internal/usecase"
	"log/slog"
)

// ReceiverEntry is one named child repository to fan out to.
type ReceiverEntry struct {
	// Name identifies the server this repository talks to, e.g. for
	// tagging error rows.
	Name string
	Repo usecase.ReceiverRepository
}

// ReceiverRepository is a ReceiverRepository that merges results from
// several named child repositories.
type ReceiverRepository struct {
	entries []ReceiverEntry
	logger  *slog.Logger
}

// NewReceiverRepository builds a ReceiverRepository that fans out to every
// entry.
func NewReceiverRepository(logger *slog.Logger, entries ...ReceiverEntry) *ReceiverRepository {
	return &ReceiverRepository{entries: entries, logger: logger}
}

// ListReceivers queries every child repository concurrently. A child that
// fails does not fail the whole call: its error is logged and it
// contributes a single synthetic error receiver instead, so the rest of the
// servers' receivers are still returned.
func (r *ReceiverRepository) ListReceivers(ctx context.Context) ([]domain.Receiver, error) {
	receivers := fanOut(r.entries, func(entry ReceiverEntry) []domain.Receiver {
		return r.fetch(ctx, entry)
	})
	return receivers, nil
}

// fetch queries one entry, tagging every returned receiver with its server
// name and a server-qualified ID (so same-named receivers across servers
// don't collide in the tray's per-receiver maps). On failure, it returns a
// single synthetic error receiver representing the whole server.
func (r *ReceiverRepository) fetch(ctx context.Context, entry ReceiverEntry) []domain.Receiver {
	receivers, err := entry.Repo.ListReceivers(ctx)
	if err != nil {
		r.logger.Error("multi: list receivers failed", "server", entry.Name, "error", err)
		return []domain.Receiver{{
			ID:         entry.Name + "|__server__",
			ServerName: entry.Name,
			State:      domain.SyncStateError,
			Path:       err.Error(),
		}}
	}

	tagged := make([]domain.Receiver, len(receivers))
	for i, rcv := range receivers {
		rcv.ServerName = entry.Name
		rcv.ID = entry.Name + "|" + rcv.ID
		tagged[i] = rcv
	}
	return tagged
}

var _ usecase.ReceiverRepository = (*ReceiverRepository)(nil)
