package tray

import (
	"go-sync-status-client/internal/domain"
	"testing"
	"time"
)

func TestConnectionLabel(t *testing.T) {
	disconnected := time.Date(2026, 10, 9, 14, 3, 0, 0, time.Local)
	polled := time.Date(2026, 10, 9, 14, 5, 30, 0, time.Local)

	tests := []struct {
		name  string
		conn  domain.Connection
		multi bool
		want  string
	}{
		{
			name: "live",
			conn: domain.Connection{ServerName: testServerOne, Live: true, DisconnectedAt: disconnected, LastPoll: polled},
			want: "✓ Live updates connected",
		},
		{
			name:  "live, multi server",
			conn:  domain.Connection{ServerName: testServerOne, Live: true},
			multi: true,
			want:  "✓ one: Live updates connected",
		},
		{
			name: "connecting, polled",
			conn: domain.Connection{LastPoll: polled},
			want: "? Live updates connecting — last poll 14:05:30",
		},
		{
			name: "disconnected, never polled",
			conn: domain.Connection{DisconnectedAt: disconnected},
			want: "⏸ Live updates disconnected since 14:03:00 — polling, last poll never",
		},
		{
			name:  "disconnected, polled, multi server",
			conn:  domain.Connection{ServerName: testServerTwo, DisconnectedAt: disconnected, LastPoll: polled},
			multi: true,
			want:  "⏸ two: Live updates disconnected since 14:03:00 — polling, last poll 14:05:30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := connectionLabel(tt.conn, tt.multi); got != tt.want {
				t.Errorf("connectionLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}
