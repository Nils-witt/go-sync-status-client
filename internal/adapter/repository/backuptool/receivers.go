package backuptool

import (
	"context"
	"encoding/json"
	"fmt"
	"go-sync-status-client/internal/domain"
	"go-sync-status-client/internal/usecase"
	"net/http"
	"time"
)

// receiverSnapshot mirrors the go-backup-tool dashboard API's
// GET /api/receivers response: one storage destination that receives
// synced data.
type receiverSnapshot struct {
	ID         string    `json:"id"`
	Path       string    `json:"path"`
	Retention  string    `json:"retention"`
	State      string    `json:"state"`
	LastKey    string    `json:"last_key"`
	LastSeen   time.Time `json:"last_seen"`
	StaleAfter string    `json:"stale_after"`
}

// ListReceivers implements usecase.ReceiverRepository. While Watch is
// connected, it maps the latest live status message; otherwise it fetches
// and maps GET /api/receivers.
func (r *Repository) ListReceivers(ctx context.Context) ([]domain.Receiver, error) {
	if live := r.liveSnapshot(); live != nil {
		return r.toReceivers(live.Receivers), nil
	}

	url := r.baseURL + "/api/receivers"
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		r.logger.Error("backuptool: build request failed", "url", url, "error", err)
		return nil, fmt.Errorf("backuptool: build request: %w", err)
	}
	if r.token != "" {
		req.Header.Set("Authorization", "Bearer "+r.token)
	}

	r.logger.Debug("backuptool: requesting receivers", "url", url)
	resp, err := r.httpClient.Do(req)
	if err != nil {
		r.logger.Error("backuptool: request failed", "url", url, "error", err)
		return nil, fmt.Errorf("backuptool: request receivers: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		r.logger.Error("backuptool: unexpected status", "url", url, "status", resp.Status)
		return nil, fmt.Errorf("backuptool: unexpected status %s", resp.Status)
	}

	var snapshots []receiverSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&snapshots); err != nil {
		r.logger.Error("backuptool: decode response failed", "url", url, "error", err)
		return nil, fmt.Errorf("backuptool: decode response: %w", err)
	}

	r.logger.Debug("backuptool: receivers fetched", "receivers", len(snapshots), "elapsed", time.Since(start))
	return r.toReceivers(snapshots), nil
}

func (r *Repository) toReceivers(snapshots []receiverSnapshot) []domain.Receiver {
	receivers := make([]domain.Receiver, 0, len(snapshots))
	for _, s := range snapshots {
		receivers = append(receivers, r.toReceiver(s))
	}
	return receivers
}

func (r *Repository) toReceiver(s receiverSnapshot) domain.Receiver {
	retention, err := time.ParseDuration(s.Retention)
	if err != nil {
		r.logger.Debug("backuptool: parse retention failed", "id", s.ID, "retention", s.Retention, "error", err)
	}
	staleAfter, err := time.ParseDuration(s.StaleAfter)
	if err != nil {
		r.logger.Debug("backuptool: parse stale_after failed", "id", s.ID, "stale_after", s.StaleAfter, "error", err)
	}

	lastSeen := s.LastSeen
	if !lastSeen.IsZero() {
		lastSeen = lastSeen.Local()
	}

	return domain.Receiver{
		ID:         s.ID,
		Path:       s.Path,
		State:      toReceiverState(s.State),
		LastKey:    s.LastKey,
		LastSeen:   lastSeen,
		Retention:  retention,
		StaleAfter: staleAfter,
	}
}

// toReceiverState maps the backend's receiver state string onto
// domain.SyncState. Only "ok" has been observed in practice; any other
// value (including ones this client doesn't yet recognize) is treated as
// Error, since flagging a possible problem is the safer default for a
// status-monitoring UI.
func toReceiverState(s string) domain.SyncState {
	if s == "ok" {
		return domain.SyncStateSynced
	}
	return domain.SyncStateError
}

var _ usecase.ReceiverRepository = (*Repository)(nil)
