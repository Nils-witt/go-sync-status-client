package backuptool

import (
	"context"
	"go-sync-status-client/internal/domain"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const testTicket = "t-123"

// newLiveServer fakes a go-backup-tool dashboard: it mints testTicket, then
// pushes each of msgs over GET /api/live in order, waiting for a value on
// next before each one. One more value on next then closes the socket from
// the server side; otherwise it stays open until the client leaves.
// GET /api/status and /api/receivers fail the test, since a connected
// client must serve from the live snapshot instead.
func newLiveServer(t *testing.T, next <-chan struct{}, msgs ...liveStatus) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/live/ticket", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Errorf("ticket Authorization = %q, want %q", got, "Bearer secret")
		}
		_, _ = w.Write([]byte(`{"ticket":"` + testTicket + `"}`))
	})
	mux.HandleFunc("GET /api/live", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("ticket"); got != testTicket {
			http.Error(w, "bad ticket", http.StatusForbidden)
			return
		}
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer func() { _ = conn.CloseNow() }()
		ctx := conn.CloseRead(r.Context())

		for _, msg := range msgs {
			select {
			case <-ctx.Done():
				return
			case <-next:
			}
			if err := wsjson.Write(ctx, conn, msg); err != nil {
				return
			}
		}
		select {
		case <-ctx.Done():
		case <-next:
			_ = conn.Close(websocket.StatusGoingAway, "test disconnect")
		}
	})
	fail := func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected polling request %s while live", r.URL.Path)
		http.Error(w, "unexpected", http.StatusInternalServerError)
	}
	mux.HandleFunc("GET /api/status", fail)
	mux.HandleFunc("GET /api/receivers", fail)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestWatch_ServesPushedSnapshots(t *testing.T) {
	next := make(chan struct{})
	srv := newLiveServer(t, next,
		liveStatus{Type: "status", Jobs: []jobSnapshot{{Name: "docs", State: runStateRunning}}},
		liveStatus{
			Type:      "status",
			Jobs:      []jobSnapshot{{Name: "docs", State: runStateOK}},
			Receivers: []receiverSnapshot{{ID: "edp-daily", State: "ok"}},
		},
	)

	repo := NewRepository(srv.URL, WithBearerToken("secret"))
	ctx, cancel := context.WithCancel(context.Background())
	changes := make(chan struct{}, 4)
	done := make(chan struct{})
	go func() {
		defer close(done)
		repo.Watch(ctx, func() { changes <- struct{}{} })
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})

	waitChange := func() {
		t.Helper()
		next <- struct{}{}
		select {
		case <-changes:
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for onChange")
		}
	}

	waitChange()
	sources, err := repo.ListSources(ctx)
	if err != nil {
		t.Fatalf("ListSources: %v", err)
	}
	if len(sources) != 1 || sources[0].State != domain.SyncStateSyncing {
		t.Fatalf("sources = %+v, want one Syncing source", sources)
	}

	waitChange()
	sources, err = repo.ListSources(ctx)
	if err != nil {
		t.Fatalf("ListSources: %v", err)
	}
	if len(sources) != 1 || sources[0].State != domain.SyncStateSynced {
		t.Fatalf("sources = %+v, want one Synced source", sources)
	}
	receivers, err := repo.ListReceivers(ctx)
	if err != nil {
		t.Fatalf("ListReceivers: %v", err)
	}
	if len(receivers) != 1 || receivers[0].ID != "edp-daily" || receivers[0].State != domain.SyncStateSynced {
		t.Fatalf("receivers = %+v, want one Synced edp-daily receiver", receivers)
	}
}

func TestWatch_DisconnectFallsBackToPolling(t *testing.T) {
	next := make(chan struct{})
	srv := newLiveServer(t, next, liveStatus{Type: "status", Jobs: []jobSnapshot{{Name: "docs", State: runStateOK}}})

	repo := NewRepository(srv.URL, WithBearerToken("secret"))
	ctx, cancel := context.WithCancel(context.Background())
	changes := make(chan struct{}, 4)
	done := make(chan struct{})
	go func() {
		defer close(done)
		repo.Watch(ctx, func() { changes <- struct{}{} })
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})

	next <- struct{}{}
	select {
	case <-changes:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for onChange")
	}
	if repo.liveSnapshot() == nil {
		t.Fatal("liveSnapshot() = nil after a pushed message, want non-nil")
	}
	if conn := repo.Connections()[0]; !conn.Live || !conn.DisconnectedAt.IsZero() {
		t.Fatalf("Connections()[0] = %+v while live, want Live and no DisconnectedAt", conn)
	}

	next <- struct{}{}
	select {
	case <-changes:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for onChange after disconnect")
	}
	if repo.liveSnapshot() != nil {
		t.Fatal("liveSnapshot() != nil after disconnect, want nil so List* fall back to HTTP")
	}
	if conn := repo.Connections()[0]; conn.Live || conn.DisconnectedAt.IsZero() {
		t.Fatalf("Connections()[0] = %+v after disconnect, want not Live with DisconnectedAt set", conn)
	}
}
