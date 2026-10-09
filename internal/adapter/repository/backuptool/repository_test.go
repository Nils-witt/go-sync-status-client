package backuptool

import (
	"go-sync-status-client/internal/domain"
	"testing"
)

func TestToTargetState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   runState
		want domain.SyncState
	}{
		{runStateOK, domain.SyncStateSynced},
		{runStateRunning, domain.SyncStateSyncing},
		{runStateIdle, domain.SyncStatePaused},
		{runStateIncomplete, domain.SyncStateIncomplete},
		{runStateFailed, domain.SyncStateError},
		{"bogus", domain.SyncStateUnknown},
	}
	for _, tt := range tests {
		if got := toTargetState(tt.in); got != tt.want {
			t.Errorf("toTargetState(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestToSyncSourceKeepsIncompleteJobAsError(t *testing.T) {
	t.Parallel()

	src := toSyncSource(jobSnapshot{
		Name:    "photos",
		State:   runStateIncomplete,
		Targets: []targetSnapshot{{Server: "s3", Bucket: "b", State: runStateIncomplete}},
	})
	if src.State != domain.SyncStateError {
		t.Errorf("source state = %v, want %v", src.State, domain.SyncStateError)
	}
	if src.Targets[0].State != domain.SyncStateIncomplete {
		t.Errorf("target state = %v, want %v", src.Targets[0].State, domain.SyncStateIncomplete)
	}
}
