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
	rcv := domain.Receiver{ID: testReceiverID, ServerName: testServerOne, State: domain.SyncStateSynced, LastSeen: time.Date(2026, 9, 7, 19, 15, 7, 0, time.UTC)}

	want := "✓ " + testReceiverID
	if got := receiverLabel(rcv); got != want {
		t.Errorf("receiverLabel(%+v) = %q, want %q", rcv, got, want)
	}
}

func TestGroupReceiversByServer(t *testing.T) {
	receivers := []domain.Receiver{
		{ID: testReceiverID, ServerName: testServerTwo},
		{ID: testReceiverID, ServerName: testServerOne},
		{ID: testReceiverID2, ServerName: testServerTwo},
	}

	got := groupReceiversByServer(receivers)
	if len(got) != 2 {
		t.Fatalf("groupReceiversByServer() returned %d groups, want 2: %+v", len(got), got)
	}
	if got[0].serverName != testServerTwo || len(got[0].receivers) != 2 ||
		got[0].receivers[0].ID != testReceiverID || got[0].receivers[1].ID != testReceiverID2 {
		t.Errorf("group 0 = %+v, want server %q with [%s %s]", got[0], testServerTwo, testReceiverID, testReceiverID2)
	}
	if got[1].serverName != testServerOne || len(got[1].receivers) != 1 {
		t.Errorf("group 1 = %+v, want server %q with 1 receiver", got[1], testServerOne)
	}
}

func TestReceiverServerLabel(t *testing.T) {
	tests := []struct {
		name  string
		group receiverGroup
		want  string
	}{
		{
			name: "worst state wins",
			group: receiverGroup{serverName: testServerOne, receivers: []domain.Receiver{
				{ID: testReceiverID, State: domain.SyncStateSynced},
				{ID: testReceiverID2, State: domain.SyncStateError},
			}},
			want: domain.SyncStateError.Symbol() + " " + testServerOne + " (2)",
		},
		{
			name:  "empty server name falls back",
			group: receiverGroup{receivers: []domain.Receiver{{ID: testReceiverID, State: domain.SyncStateSynced}}},
			want:  domain.SyncStateSynced.Symbol() + " Server (1)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := receiverServerLabel(tt.group); got != tt.want {
				t.Errorf("receiverServerLabel(%+v) = %q, want %q", tt.group, got, tt.want)
			}
		})
	}
}

func TestReceiverKey(t *testing.T) {
	a := receiverKey(domain.Receiver{ID: testReceiverID, ServerName: testServerOne})
	b := receiverKey(domain.Receiver{ID: testReceiverID, ServerName: testServerTwo})
	if a == b {
		t.Errorf("receiverKey collides across servers: %q", a)
	}
}
