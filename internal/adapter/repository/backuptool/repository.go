// Package backuptool provides a StatusRepository backed by a go-backup-tool
// instance's dashboard API (see its openapi.yaml, operation getStatus at
// GET /api/status).
package backuptool

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"go-sync-status-client/internal/domain"
	"go-sync-status-client/internal/usecase"
)

// Repository is a StatusRepository backed by a go-backup-tool instance's
// dashboard API.
type Repository struct {
	baseURL    string
	token      string
	httpClient *http.Client
	logger     *slog.Logger
}

// Option configures a Repository.
type Option func(*Repository)

// WithHTTPClient overrides the default HTTP client (10s timeout).
func WithHTTPClient(c *http.Client) Option {
	return func(r *Repository) { r.httpClient = c }
}

// WithBearerToken sets the dashboard session token sent as
// "Authorization: Bearer <token>". Only required when the target instance
// has webui.username or OIDC configured; open instances ignore it.
func WithBearerToken(token string) Option {
	return func(r *Repository) { r.token = token }
}

// WithLogger overrides the default no-op logger.
func WithLogger(logger *slog.Logger) Option {
	return func(r *Repository) { r.logger = logger }
}

// NewRepository builds a Repository that talks to the go-backup-tool
// dashboard API at baseURL (e.g. "http://localhost:8081").
func NewRepository(baseURL string, opts ...Option) *Repository {
	r := &Repository{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     slog.New(slog.DiscardHandler),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// runState mirrors the openapi RunState schema.
type runState string

const (
	runStateIdle       runState = "idle"
	runStateRunning    runState = "running"
	runStateOK         runState = "ok"
	runStateIncomplete runState = "incomplete"
	runStateFailed     runState = "failed"
)

// jobSnapshot mirrors the openapi JobSnapshot schema returned by
// GET /api/status.
type jobSnapshot struct {
	Name      string           `json:"name"`
	Interval  string           `json:"interval"`
	State     runState         `json:"state"`
	LastStart time.Time        `json:"last_start"`
	LastEnd   time.Time        `json:"last_end"`
	Size      string           `json:"size"`
	Error     string           `json:"error"`
	Targets   []targetSnapshot `json:"targets"`
}

// targetSnapshot mirrors the openapi Target schema nested under a
// JobSnapshot: one destination (server/bucket pair) the job replicates to.
type targetSnapshot struct {
	Server string   `json:"server"`
	Bucket string   `json:"bucket"`
	Kind   string   `json:"kind"`
	State  runState `json:"state"`
}

// ListSources implements usecase.StatusRepository by fetching and mapping
// GET /api/status.
func (r *Repository) ListSources(ctx context.Context) ([]domain.SyncSource, error) {
	url := r.baseURL + "/api/status"
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		r.logger.Error("backuptool: build request failed", "url", url, "error", err)
		return nil, fmt.Errorf("backuptool: build request: %w", err)
	}
	if r.token != "" {
		req.Header.Set("Authorization", "Bearer "+r.token)
	}

	r.logger.Debug("backuptool: requesting status", "url", url)
	resp, err := r.httpClient.Do(req)
	if err != nil {
		r.logger.Error("backuptool: request failed", "url", url, "error", err)
		return nil, fmt.Errorf("backuptool: request status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		r.logger.Error("backuptool: unexpected status", "url", url, "status", resp.Status)
		return nil, fmt.Errorf("backuptool: unexpected status %s", resp.Status)
	}

	var jobs []jobSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&jobs); err != nil {
		r.logger.Error("backuptool: decode response failed", "url", url, "error", err)
		return nil, fmt.Errorf("backuptool: decode response: %w", err)
	}

	sources := make([]domain.SyncSource, 0, len(jobs))
	for _, j := range jobs {
		sources = append(sources, toSyncSource(j))
	}
	r.logger.Debug("backuptool: status fetched", "jobs", len(sources), "elapsed", time.Since(start))
	return sources, nil
}

func toSyncSource(j jobSnapshot) domain.SyncSource {
	updatedAt := j.LastEnd
	if updatedAt.IsZero() {
		updatedAt = j.LastStart
	}

	targets := make([]domain.SyncTarget, 0, len(j.Targets))
	for _, t := range j.Targets {
		targets = append(targets, toSyncTarget(t))
	}

	return domain.SyncSource{
		ID:        j.Name,
		Name:      j.Name,
		State:     toSyncState(j.State),
		Detail:    detail(j),
		UpdatedAt: updatedAt,
		Targets:   targets,
	}
}

func toSyncTarget(t targetSnapshot) domain.SyncTarget {
	return domain.SyncTarget{
		ID:    t.Server + ":" + t.Bucket,
		Label: fmt.Sprintf("%s (%s)", t.Server, t.Kind),
		State: toSyncState(t.State),
	}
}

// toSyncState maps the backend's five-state RunState onto the tray's
// four-state SyncState. idle ("never run yet") reads as Paused rather than
// Unknown, since Unknown is reserved for states this client doesn't
// recognize; incomplete (partial target failure) folds into Error.
func toSyncState(s runState) domain.SyncState {
	switch s {
	case runStateOK:
		return domain.SyncStateSynced
	case runStateRunning:
		return domain.SyncStateSyncing
	case runStateIdle:
		return domain.SyncStatePaused
	case runStateIncomplete, runStateFailed:
		return domain.SyncStateError
	default:
		return domain.SyncStateUnknown
	}
}

func detail(j jobSnapshot) string {
	if j.Error != "" {
		return j.Error
	}
	if j.Size != "" {
		return j.Size
	}
	return j.Interval
}

var _ usecase.StatusRepository = (*Repository)(nil)
