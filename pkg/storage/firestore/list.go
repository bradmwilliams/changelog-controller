package firestore

import (
	"context"
	"sync"
)

// defaultListConcurrency is the default number of GCS objects fetched in
// parallel when warming caches from the Firestore index.
const defaultListConcurrency = 32

func normalizeListConcurrency(concurrency int) int {
	if concurrency <= 0 {
		return defaultListConcurrency
	}
	return concurrency
}

// readObjectsInParallel reads GCS objects with at most concurrency reads in
// flight, preserving input order. Per-object failures are returned in the
// matching errors slot so callers can skip bad objects and keep warming.
func readObjectsInParallel(ctx context.Context, reader gcsObjectReader, paths []string, concurrency int) ([][]byte, []error) {
	results := make([][]byte, len(paths))
	errs := make([]error, len(paths))

	sem := make(chan struct{}, normalizeListConcurrency(concurrency))
	var wg sync.WaitGroup

	for i := range paths {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			errs[i] = ctx.Err()
			continue
		}

		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			data, err := reader.ReadObject(ctx, paths[i])
			if err != nil {
				errs[i] = err
				return
			}
			results[i] = data
		}()
	}

	wg.Wait()
	return results, errs
}
