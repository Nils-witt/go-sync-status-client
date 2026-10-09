package multi

import (
	"context"
	"go-sync-status-client/internal/usecase"
	"sync"
)

// Watcher is a ChangeWatcher that merges change notifications from several
// child watchers (typically one per configured server).
type Watcher struct {
	watchers []usecase.ChangeWatcher
}

// NewWatcher builds a Watcher that fans out to every watcher.
func NewWatcher(watchers ...usecase.ChangeWatcher) *Watcher {
	return &Watcher{watchers: watchers}
}

// Watch runs every child watcher concurrently, forwarding each one's
// changes to onChange, and returns once all of them have returned.
func (w *Watcher) Watch(ctx context.Context, onChange func()) {
	var wg sync.WaitGroup
	for _, child := range w.watchers {
		wg.Go(func() { child.Watch(ctx, onChange) })
	}
	wg.Wait()
}

var _ usecase.ChangeWatcher = (*Watcher)(nil)
