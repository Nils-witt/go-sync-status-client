package multi

import "sync"

// fanOut runs fetch concurrently for each entry and concatenates the
// results. Result order across entries is not preserved, which is fine
// here: callers only key results by ID, never by position.
func fanOut[E, T any](entries []E, fetch func(E) []T) []T {
	results := make([][]T, len(entries))

	var wg sync.WaitGroup
	for i, entry := range entries {
		wg.Add(1)
		go func(i int, entry E) {
			defer wg.Done()
			results[i] = fetch(entry)
		}(i, entry)
	}
	wg.Wait()

	var out []T
	for _, res := range results {
		out = append(out, res...)
	}
	return out
}
