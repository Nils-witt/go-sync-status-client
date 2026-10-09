package usecase

import "context"

// ChangeWatcher is the port an adapter can satisfy to push notice that the
// data served by its StatusRepository/ReceiverRepository may have changed,
// so presentation adapters can re-read it right away instead of waiting for
// their next poll. Defined here, on the consumer side, so this package
// stays in control of the contract.
type ChangeWatcher interface {
	// Watch blocks until ctx is done, calling onChange (possibly from
	// another goroutine) whenever the repository's data may have changed.
	Watch(ctx context.Context, onChange func())
}
