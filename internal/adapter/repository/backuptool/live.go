package backuptool

import (
	"context"
	"encoding/json"
	"fmt"
	"go-sync-status-client/internal/domain"
	"go-sync-status-client/internal/usecase"
	"net/http"
	"net/url"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const (
	// liveReadTimeout bounds how long Watch waits for the next message
	// before treating the connection as dead. The server pushes the full
	// status at least every 30s, so this allows for one missed push.
	liveReadTimeout = 75 * time.Second

	// liveReadLimit caps a single message's size. The library default
	// (32 KiB) is too small for a server with many jobs and receivers.
	liveReadLimit = 4 << 20

	// liveMinBackoff and liveMaxBackoff bound the reconnect delay, which
	// doubles after every attempt that never received a message.
	liveMinBackoff = time.Second
	liveMaxBackoff = time.Minute
)

// liveStatus mirrors the openapi LiveStatus schema: one message on the
// GET /api/live WebSocket, carrying the same data as /api/status and
// /api/receivers together.
type liveStatus struct {
	Type      string             `json:"type"`
	Jobs      []jobSnapshot      `json:"jobs"`
	Receivers []receiverSnapshot `json:"receivers"`
}

// Watch implements usecase.ChangeWatcher by following the GET /api/live
// WebSocket. While connected, every pushed message replaces the snapshot
// ListSources and ListReceivers serve from, and onChange is called. When
// the connection drops, the snapshot is discarded (so they fall back to
// plain HTTP requests) and the disconnect time recorded (see Connections),
// onChange is called once more, and Watch reconnects with exponential
// backoff. It returns once ctx is done.
func (r *Repository) Watch(ctx context.Context, onChange func()) {
	backoff := liveMinBackoff
	for {
		received, err := r.followLive(ctx, onChange)
		if ctx.Err() != nil {
			return
		}
		if r.markDisconnected(received) {
			onChange()
		}
		if received {
			backoff = liveMinBackoff
		}
		r.logger.Warn("backuptool: live status disconnected", "base_url", r.baseURL, "error", err, "retry_in", backoff)

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, liveMaxBackoff)
	}
}

// followLive opens one GET /api/live connection and stores every message
// it receives until the connection fails or ctx is done. received reports
// whether at least one message was stored.
func (r *Repository) followLive(ctx context.Context, onChange func()) (received bool, err error) {
	ticket, err := r.mintLiveTicket(ctx)
	if err != nil {
		return false, err
	}

	liveURL := r.baseURL + "/api/live?ticket=" + url.QueryEscape(ticket)
	conn, _, err := websocket.Dial(ctx, liveURL, &websocket.DialOptions{HTTPClient: r.httpClient}) //nolint:bodyclose // websocket.Dial documents that callers never close resp.Body
	if err != nil {
		return false, fmt.Errorf("backuptool: dial live status: %w", err)
	}
	defer func() { _ = conn.CloseNow() }()
	conn.SetReadLimit(liveReadLimit)

	r.logger.Info("backuptool: live status connected", "base_url", r.baseURL)
	for {
		var msg liveStatus
		readCtx, cancel := context.WithTimeout(ctx, liveReadTimeout)
		err := wsjson.Read(readCtx, conn, &msg)
		cancel()
		if err != nil {
			return received, fmt.Errorf("backuptool: read live status: %w", err)
		}
		if msg.Type != "status" {
			r.logger.Debug("backuptool: ignoring live message", "type", msg.Type)
			continue
		}

		r.logger.Debug("backuptool: live status received", "jobs", len(msg.Jobs), "receivers", len(msg.Receivers))
		r.setLive(&msg)
		received = true
		onChange()
	}
}

// mintLiveTicket fetches a one-time ticket for opening GET /api/live via
// POST /api/live/ticket, since the WebSocket handshake itself is not
// authorized by the bearer token.
func (r *Repository) mintLiveTicket(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/api/live/ticket", nil)
	if err != nil {
		return "", fmt.Errorf("backuptool: build live ticket request: %w", err)
	}
	if r.token != "" {
		req.Header.Set("Authorization", "Bearer "+r.token)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("backuptool: request live ticket: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("backuptool: unexpected live ticket status %s", resp.Status)
	}

	var body struct {
		Ticket string `json:"ticket"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("backuptool: decode live ticket: %w", err)
	}
	return body.Ticket, nil
}

// markDisconnected discards the live snapshot after a connection attempt
// ended. It records the disconnect time if live updates were just lost
// (received) or this is the first failed attempt, and reports whether it
// did, i.e. whether Connections changed.
func (r *Repository) markDisconnected(received bool) bool {
	r.liveMu.Lock()
	defer r.liveMu.Unlock()
	r.live = nil
	if !received && !r.disconnectedAt.IsZero() {
		return false
	}
	r.disconnectedAt = time.Now()
	return true
}

// recordPoll notes a successful HTTP poll (see Connections).
func (r *Repository) recordPoll() {
	r.liveMu.Lock()
	defer r.liveMu.Unlock()
	r.lastPoll = time.Now()
}

// Connections implements usecase.ConnectionRepository, reporting this
// server's single connection.
func (r *Repository) Connections() []domain.Connection {
	r.liveMu.RLock()
	defer r.liveMu.RUnlock()
	return []domain.Connection{{
		Live:           r.live != nil,
		DisconnectedAt: r.disconnectedAt,
		LastPoll:       r.lastPoll,
	}}
}

// setLive replaces the snapshot served by ListSources/ListReceivers; nil
// discards it, making them fall back to HTTP requests.
func (r *Repository) setLive(msg *liveStatus) {
	r.liveMu.Lock()
	defer r.liveMu.Unlock()
	r.live = msg
}

// liveSnapshot returns the most recent live status message, or nil while
// not connected.
func (r *Repository) liveSnapshot() *liveStatus {
	r.liveMu.RLock()
	defer r.liveMu.RUnlock()
	return r.live
}

var (
	_ usecase.ChangeWatcher        = (*Repository)(nil)
	_ usecase.ConnectionRepository = (*Repository)(nil)
)
