package firestore

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

type fakeObjectReader struct {
	data  map[string][]byte
	errs  map[string]error
	delay time.Duration

	mu          sync.Mutex
	inFlight    int
	maxInFlight int
}

func (r *fakeObjectReader) ReadObject(ctx context.Context, path string) ([]byte, error) {
	r.mu.Lock()
	r.inFlight++
	if r.inFlight > r.maxInFlight {
		r.maxInFlight = r.inFlight
	}
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		r.inFlight--
		r.mu.Unlock()
	}()

	if r.delay > 0 {
		select {
		case <-time.After(r.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if err, ok := r.errs[path]; ok {
		return nil, err
	}
	data, ok := r.data[path]
	if !ok {
		return nil, fmt.Errorf("unexpected path %q", path)
	}
	return data, nil
}

func (r *fakeObjectReader) maxConcurrent() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.maxInFlight
}

func TestReadObjectsInParallel_BoundsConcurrencyAndPreservesOrder(t *testing.T) {
	const (
		pathCount   = 12
		concurrency = 3
	)

	paths := make([]string, pathCount)
	data := make(map[string][]byte, pathCount)
	for i := range paths {
		paths[i] = fmt.Sprintf("object-%d", i)
		data[paths[i]] = fmt.Appendf(nil, "content-%d", i)
	}

	reader := &fakeObjectReader{data: data, delay: 20 * time.Millisecond}
	results, errs := readObjectsInParallel(context.Background(), reader, paths, concurrency)

	for i := range paths {
		if errs[i] != nil {
			t.Fatalf("errs[%d] = %v, want nil", i, errs[i])
		}
		want := fmt.Sprintf("content-%d", i)
		if string(results[i]) != want {
			t.Errorf("results[%d] = %q, want %q", i, results[i], want)
		}
	}

	if got := reader.maxConcurrent(); got > concurrency {
		t.Errorf("max concurrent reads = %d, want at most %d", got, concurrency)
	} else if got < 2 {
		t.Errorf("max concurrent reads = %d, want reads to overlap", got)
	}
}

func TestReadObjectsInParallel_ReportsPerObjectErrors(t *testing.T) {
	paths := []string{"ok-1", "bad", "ok-2"}
	reader := &fakeObjectReader{
		data: map[string][]byte{
			"ok-1": []byte("one"),
			"ok-2": []byte("two"),
		},
		errs: map[string]error{
			"bad": errors.New("boom"),
		},
	}

	results, errs := readObjectsInParallel(context.Background(), reader, paths, 4)

	if errs[0] != nil || errs[2] != nil {
		t.Errorf("unexpected errors for healthy objects: %v, %v", errs[0], errs[2])
	}
	if errs[1] == nil {
		t.Error("errs[1] = nil, want error for failed object")
	}
	if string(results[0]) != "one" || string(results[2]) != "two" {
		t.Errorf("healthy results = %q, %q, want %q, %q", results[0], results[2], "one", "two")
	}
	if results[1] != nil {
		t.Errorf("results[1] = %q, want nil", results[1])
	}
}

func TestReadObjectsInParallel_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	paths := []string{"a", "b", "c", "d"}
	reader := &fakeObjectReader{
		data:  map[string][]byte{"a": []byte("a"), "b": []byte("b"), "c": []byte("c"), "d": []byte("d")},
		delay: time.Hour,
	}

	_, errs := readObjectsInParallel(ctx, reader, paths, 2)

	for i, err := range errs {
		if !errors.Is(err, context.Canceled) {
			t.Errorf("errs[%d] = %v, want context.Canceled", i, err)
		}
	}
}

func TestNormalizeListConcurrency(t *testing.T) {
	tests := []struct {
		in   int
		want int
	}{
		{0, defaultListConcurrency},
		{-1, defaultListConcurrency},
		{7, 7},
	}

	for _, tt := range tests {
		if got := normalizeListConcurrency(tt.in); got != tt.want {
			t.Errorf("normalizeListConcurrency(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}
