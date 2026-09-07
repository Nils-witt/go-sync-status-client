package multi

import (
	"context"
	"errors"
	"go-sync-status-client/internal/domain"
	"io"
	"log/slog"
	"sort"
	"testing"
)

type stubRepo struct {
	sources []domain.SyncSource
	err     error
}

func (s stubRepo) ListSources(_ context.Context) ([]domain.SyncSource, error) {
	return s.sources, s.err
}

var testLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func TestListSources_TagsAndQualifiesIDs(t *testing.T) {
	repo := NewRepository(testLogger,
		Entry{Name: "one", Repo: stubRepo{sources: []domain.SyncSource{{ID: "docs", Name: "Docs"}}}},
		Entry{Name: "two", Repo: stubRepo{sources: []domain.SyncSource{{ID: "docs", Name: "Docs"}}}},
	)

	sources, err := repo.ListSources(context.Background())
	if err != nil {
		t.Fatalf("ListSources: %v", err)
	}
	if len(sources) != 2 {
		t.Fatalf("len(sources) = %d, want 2", len(sources))
	}

	sort.Slice(sources, func(i, j int) bool { return sources[i].ID < sources[j].ID })
	if sources[0].ID != "one|docs" || sources[0].ServerName != "one" {
		t.Errorf("sources[0] = %+v, want ID %q ServerName %q", sources[0], "one|docs", "one")
	}
	if sources[1].ID != "two|docs" || sources[1].ServerName != "two" {
		t.Errorf("sources[1] = %+v, want ID %q ServerName %q", sources[1], "two|docs", "two")
	}
}

func TestListSources_FailedServerYieldsErrorRowNotWholeFailure(t *testing.T) {
	repo := NewRepository(testLogger,
		Entry{Name: "up", Repo: stubRepo{sources: []domain.SyncSource{{ID: "docs", Name: "Docs", State: domain.SyncStateSynced}}}},
		Entry{Name: "down", Repo: stubRepo{err: errors.New("connection refused")}},
	)

	sources, err := repo.ListSources(context.Background())
	if err != nil {
		t.Fatalf("ListSources: %v", err)
	}
	if len(sources) != 2 {
		t.Fatalf("len(sources) = %d, want 2, got %+v", len(sources), sources)
	}

	byServer := make(map[string]domain.SyncSource, len(sources))
	for _, src := range sources {
		byServer[src.ServerName] = src
	}
	if byServer["up"].State != domain.SyncStateSynced {
		t.Errorf("up server source state = %v, want Synced", byServer["up"].State)
	}
	if byServer["down"].State != domain.SyncStateError {
		t.Errorf("down server source state = %v, want Error", byServer["down"].State)
	}
}
