package tray

import (
	"go-sync-status-client/internal/domain"
	"testing"
	"time"
)

const (
	testReceiverID  = "edp-daily"
	testServerOne   = "one"
	testServerTwo   = "two"
	testReceiverID2 = "edp-hourly"
)

func TestMultiServerReceivers(t *testing.T) {
	tests := []struct {
		name      string
		receivers []domain.Receiver
		want      bool
	}{
		{
			name:      "no receivers",
			receivers: nil,
			want:      false,
		},
		{
			name:      "single server",
			receivers: []domain.Receiver{{ID: testReceiverID, ServerName: testServerOne}, {ID: testReceiverID2, ServerName: testServerOne}},
			want:      false,
		},
		{
			name:      "multiple servers",
			receivers: []domain.Receiver{{ID: testReceiverID, ServerName: testServerOne}, {ID: testReceiverID, ServerName: testServerTwo}},
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := multiServerReceivers(tt.receivers); got != tt.want {
				t.Errorf("multiServerReceivers(%+v) = %v, want %v", tt.receivers, got, tt.want)
			}
		})
	}
}

func TestDisplayReceiverName(t *testing.T) {
	tests := []struct {
		name  string
		rcv   domain.Receiver
		multi bool
		want  string
	}{
		{
			name:  "single server, no prefix",
			rcv:   domain.Receiver{ID: testReceiverID, ServerName: testServerOne},
			multi: false,
			want:  testReceiverID,
		},
		{
			name:  "multi server, prefixed",
			rcv:   domain.Receiver{ID: testReceiverID, ServerName: testServerOne},
			multi: true,
			want:  testServerOne + ": " + testReceiverID,
		},
		{
			name:  "multi server, empty server name falls back to ID",
			rcv:   domain.Receiver{ID: testReceiverID},
			multi: true,
			want:  testReceiverID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := displayReceiverName(tt.rcv, tt.multi); got != tt.want {
				t.Errorf("displayReceiverName(%+v, %v) = %q, want %q", tt.rcv, tt.multi, got, tt.want)
			}
		})
	}
}

func TestReceiverLabel(t *testing.T) {
	lastSeen := time.Date(2026, 9, 7, 19, 15, 7, 0, time.UTC)
	rcv := domain.Receiver{ID: testReceiverID, State: domain.SyncStateSynced, LastSeen: lastSeen}

	want := "✓ " + testReceiverID + " — Synced (last seen " + lastSeen.Format("15:04:05") + ")"
	if got := receiverLabel(rcv, false); got != want {
		t.Errorf("receiverLabel(%+v, false) = %q, want %q", rcv, got, want)
	}
}
