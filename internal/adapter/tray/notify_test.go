package tray

import (
	"go-sync-status-client/internal/domain"
	"testing"
)

func TestClassifyTransition(t *testing.T) {
	tests := []struct {
		name  string
		known bool
		prev  domain.SyncState
		cur   domain.SyncState
		want  transitionKind
	}{
		{
			name:  "first observation is never a notification, even if already erroring",
			known: false,
			prev:  domain.SyncStateUnknown,
			cur:   domain.SyncStateError,
			want:  transitionNone,
		},
		{
			name:  "unchanged state is not a notification",
			known: true,
			prev:  domain.SyncStateSynced,
			cur:   domain.SyncStateSynced,
			want:  transitionNone,
		},
		{
			name:  "synced to error is an error occurring",
			known: true,
			prev:  domain.SyncStateSynced,
			cur:   domain.SyncStateError,
			want:  transitionErrorOccurred,
		},
		{
			name:  "syncing to error is an error occurring",
			known: true,
			prev:  domain.SyncStateSyncing,
			cur:   domain.SyncStateError,
			want:  transitionErrorOccurred,
		},
		{
			name:  "error to synced is an error clearing",
			known: true,
			prev:  domain.SyncStateError,
			cur:   domain.SyncStateSynced,
			want:  transitionErrorCleared,
		},
		{
			name:  "error to syncing is an error clearing",
			known: true,
			prev:  domain.SyncStateError,
			cur:   domain.SyncStateSyncing,
			want:  transitionErrorCleared,
		},
		{
			name:  "synced to syncing is not error-related",
			known: true,
			prev:  domain.SyncStateSynced,
			cur:   domain.SyncStateSyncing,
			want:  transitionNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyTransition(tt.known, tt.prev, tt.cur); got != tt.want {
				t.Errorf("classifyTransition(%v, %v, %v) = %v, want %v", tt.known, tt.prev, tt.cur, got, tt.want)
			}
		})
	}
}
