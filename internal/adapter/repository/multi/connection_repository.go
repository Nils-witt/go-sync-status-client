package multi

import (
	"go-sync-status-client/internal/domain"
	"go-sync-status-client/internal/usecase"
)

// ConnectionEntry is one named child repository to fan out to.
type ConnectionEntry struct {
	// Name identifies the server this repository talks to.
	Name string
	Repo usecase.ConnectionRepository
}

// ConnectionRepository is a ConnectionRepository that merges results from
// several named child repositories.
type ConnectionRepository struct {
	entries []ConnectionEntry
}

// NewConnectionRepository builds a ConnectionRepository that fans out to
// every entry.
func NewConnectionRepository(entries ...ConnectionEntry) *ConnectionRepository {
	return &ConnectionRepository{entries: entries}
}

// Connections returns every child's connections, in entry order, tagged
// with their server name.
func (r *ConnectionRepository) Connections() []domain.Connection {
	var connections []domain.Connection
	for _, entry := range r.entries {
		for _, c := range entry.Repo.Connections() {
			c.ServerName = entry.Name
			connections = append(connections, c)
		}
	}
	return connections
}

var _ usecase.ConnectionRepository = (*ConnectionRepository)(nil)
