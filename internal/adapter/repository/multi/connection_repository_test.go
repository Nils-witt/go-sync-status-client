package multi

import (
	"go-sync-status-client/internal/domain"
	"testing"
)

type stubConnectionRepo []domain.Connection

func (s stubConnectionRepo) Connections() []domain.Connection { return s }

func TestConnections_TagsServerNames(t *testing.T) {
	repo := NewConnectionRepository(
		ConnectionEntry{Name: testServerOne, Repo: stubConnectionRepo{{Live: true}}},
		ConnectionEntry{Name: testServerTwo, Repo: stubConnectionRepo{{Live: false}}},
	)

	got := repo.Connections()
	if len(got) != 2 {
		t.Fatalf("len(Connections()) = %d, want 2", len(got))
	}
	if got[0].ServerName != testServerOne || !got[0].Live {
		t.Errorf("Connections()[0] = %+v, want live %q", got[0], testServerOne)
	}
	if got[1].ServerName != testServerTwo || got[1].Live {
		t.Errorf("Connections()[1] = %+v, want not-live %q", got[1], testServerTwo)
	}
}
