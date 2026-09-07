package multi

import (
	"context"
	"errors"
	"go-sync-status-client/internal/domain"
	"sort"
	"testing"
)

type stubReceiverRepo struct {
	receivers []domain.Receiver
	err       error
}

func (s stubReceiverRepo) ListReceivers(_ context.Context) ([]domain.Receiver, error) {
	return s.receivers, s.err
}

func TestListReceivers_TagsAndQualifiesIDs(t *testing.T) {
	repo := NewReceiverRepository(testLogger,
		ReceiverEntry{Name: testServerOne, Repo: stubReceiverRepo{receivers: []domain.Receiver{{ID: "edp-daily"}}}},
		ReceiverEntry{Name: testServerTwo, Repo: stubReceiverRepo{receivers: []domain.Receiver{{ID: "edp-daily"}}}},
	)

	receivers, err := repo.ListReceivers(context.Background())
	if err != nil {
		t.Fatalf("ListReceivers: %v", err)
	}
	if len(receivers) != 2 {
		t.Fatalf("len(receivers) = %d, want 2", len(receivers))
	}

	sort.Slice(receivers, func(i, j int) bool { return receivers[i].ID < receivers[j].ID })
	if receivers[0].ID != "one|edp-daily" || receivers[0].ServerName != testServerOne {
		t.Errorf("receivers[0] = %+v, want ID %q ServerName %q", receivers[0], "one|edp-daily", testServerOne)
	}
	if receivers[1].ID != "two|edp-daily" || receivers[1].ServerName != testServerTwo {
		t.Errorf("receivers[1] = %+v, want ID %q ServerName %q", receivers[1], "two|edp-daily", testServerTwo)
	}
}

func TestListReceivers_FailedServerYieldsErrorRowNotWholeFailure(t *testing.T) {
	repo := NewReceiverRepository(testLogger,
		ReceiverEntry{Name: "up", Repo: stubReceiverRepo{receivers: []domain.Receiver{{ID: "edp-daily", State: domain.SyncStateSynced}}}},
		ReceiverEntry{Name: "down", Repo: stubReceiverRepo{err: errors.New("connection refused")}},
	)

	receivers, err := repo.ListReceivers(context.Background())
	if err != nil {
		t.Fatalf("ListReceivers: %v", err)
	}
	if len(receivers) != 2 {
		t.Fatalf("len(receivers) = %d, want 2, got %+v", len(receivers), receivers)
	}

	byServer := make(map[string]domain.Receiver, len(receivers))
	for _, rcv := range receivers {
		byServer[rcv.ServerName] = rcv
	}
	if byServer["up"].State != domain.SyncStateSynced {
		t.Errorf("up server receiver state = %v, want Synced", byServer["up"].State)
	}
	if byServer["down"].State != domain.SyncStateError {
		t.Errorf("down server receiver state = %v, want Error", byServer["down"].State)
	}
}
