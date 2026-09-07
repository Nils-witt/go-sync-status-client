// Package multi provides a StatusRepository that fans out to several named
// child repositories (typically one per configured server) and merges their
// results, so a single down server degrades to one error row instead of
// blanking out the whole status list.
package multi

import (
	"context"
	"go-sync-status-client/internal/domain"
	"go-sync-status-client/internal/usecase"
	"log/slog"
)

// Entry is one named child repository to fan out to.
type Entry struct {
	// Name identifies the server this repository talks to, e.g. for
	// prefixing source labels and tagging error rows.
	Name string
	Repo usecase.StatusRepository
}

// Repository is a StatusRepository that merges results from several named
// child repositories.
type Repository struct {
	entries []Entry
	logger  *slog.Logger
}

// NewRepository builds a Repository that fans out to every entry.
func NewRepository(logger *slog.Logger, entries ...Entry) *Repository {
	return &Repository{entries: entries, logger: logger}
}

// ListSources queries every child repository concurrently. A child that
// fails does not fail the whole call: its error is logged and it
// contributes a single synthetic error source instead, so the rest of the
// servers' sources are still returned.
func (r *Repository) ListSources(ctx context.Context) ([]domain.SyncSource, error) {
	sources := fanOut(r.entries, func(entry Entry) []domain.SyncSource {
		return r.fetch(ctx, entry)
	})
	return sources, nil
}

// fetch queries one entry, tagging every returned source with its server
// name and a server-qualified ID (so same-named jobs across servers don't
// collide in the tray's per-source maps). On failure, it returns a single
// synthetic error source representing the whole server.
func (r *Repository) fetch(ctx context.Context, entry Entry) []domain.SyncSource {
	sources, err := entry.Repo.ListSources(ctx)
	if err != nil {
		r.logger.Error("multi: list sources failed", "server", entry.Name, "error", err)
		return []domain.SyncSource{{
			ID:         entry.Name + "|__server__",
			Name:       entry.Name,
			ServerName: entry.Name,
			State:      domain.SyncStateError,
			Detail:     err.Error(),
		}}
	}

	tagged := make([]domain.SyncSource, len(sources))
	for i, src := range sources {
		src.ServerName = entry.Name
		src.ID = entry.Name + "|" + src.ID
		tagged[i] = src
	}
	return tagged
}

var _ usecase.StatusRepository = (*Repository)(nil)
