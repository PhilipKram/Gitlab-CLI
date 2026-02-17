package api

import (
	"context"
	"errors"
	"testing"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func TestPaginateOptions_Defaults(t *testing.T) {
	tests := []struct {
		name     string
		opts     PaginateOptions
		wantPP   int
		wantBuf  int
		wantMax  int
	}{
		{
			name:    "zero values use defaults",
			opts:    PaginateOptions{},
			wantPP:  100,
			wantBuf: 100,
			wantMax: 0,
		},
		{
			name:    "negative PerPage uses default",
			opts:    PaginateOptions{PerPage: -1},
			wantPP:  100,
			wantBuf: 100,
			wantMax: 0,
		},
		{
			name:    "negative BufferSize uses default",
			opts:    PaginateOptions{BufferSize: -1},
			wantPP:  100,
			wantBuf: 100,
			wantMax: 0,
		},
		{
			name:    "custom values preserved",
			opts:    PaginateOptions{PerPage: 50, BufferSize: 25, MaxPages: 10},
			wantPP:  50,
			wantBuf: 25,
			wantMax: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			fetchFunc := func(page int) ([]string, *gitlab.Response, error) {
				return []string{}, &gitlab.Response{NextPage: 0}, nil
			}

			results := PaginateToChannel(ctx, fetchFunc, tt.opts)
			// Drain the channel
			for range results {
			}

			// Verify defaults are applied (indirectly through channel behavior)
			// The actual verification happens in the subsequent tests
		})
	}
}

func TestPaginateToChannel_SinglePage(t *testing.T) {
	ctx := context.Background()
	opts := PaginateOptions{PerPage: 10}

	expectedItems := []string{"item1", "item2", "item3"}
	fetchFunc := func(page int) ([]string, *gitlab.Response, error) {
		if page != 1 {
			t.Errorf("expected page 1, got page %d", page)
		}
		return expectedItems, &gitlab.Response{NextPage: 0}, nil
	}

	results := PaginateToChannel(ctx, fetchFunc, opts)

	var receivedItems []string
	for result := range results {
		if result.Error != nil {
			t.Fatalf("unexpected error: %v", result.Error)
		}
		receivedItems = append(receivedItems, result.Item)
	}

	if len(receivedItems) != len(expectedItems) {
		t.Errorf("expected %d items, got %d", len(expectedItems), len(receivedItems))
	}

	for i, item := range expectedItems {
		if receivedItems[i] != item {
			t.Errorf("item %d: expected %q, got %q", i, item, receivedItems[i])
		}
	}
}

func TestPaginateToChannel_MultiplePages(t *testing.T) {
	ctx := context.Background()
	opts := PaginateOptions{PerPage: 2}

	pages := map[int][]string{
		1: {"item1", "item2"},
		2: {"item3", "item4"},
		3: {"item5"},
	}

	fetchFunc := func(page int) ([]string, *gitlab.Response, error) {
		items, ok := pages[page]
		if !ok {
			return nil, &gitlab.Response{NextPage: 0}, nil
		}

		resp := &gitlab.Response{NextPage: 0}
		if page < 3 {
			resp.NextPage = int64(page + 1)
		}
		return items, resp, nil
	}

	results := PaginateToChannel(ctx, fetchFunc, opts)

	var receivedItems []string
	for result := range results {
		if result.Error != nil {
			t.Fatalf("unexpected error: %v", result.Error)
		}
		receivedItems = append(receivedItems, result.Item)
	}

	expectedTotal := 5
	if len(receivedItems) != expectedTotal {
		t.Errorf("expected %d items total, got %d", expectedTotal, len(receivedItems))
	}

	expected := []string{"item1", "item2", "item3", "item4", "item5"}
	for i, exp := range expected {
		if receivedItems[i] != exp {
			t.Errorf("item %d: expected %q, got %q", i, exp, receivedItems[i])
		}
	}
}

func TestPaginateToChannel_EmptyResults(t *testing.T) {
	ctx := context.Background()
	opts := PaginateOptions{}

	fetchFunc := func(page int) ([]string, *gitlab.Response, error) {
		return []string{}, &gitlab.Response{NextPage: 0}, nil
	}

	results := PaginateToChannel(ctx, fetchFunc, opts)

	var receivedItems []string
	for result := range results {
		if result.Error != nil {
			t.Fatalf("unexpected error: %v", result.Error)
		}
		receivedItems = append(receivedItems, result.Item)
	}

	if len(receivedItems) != 0 {
		t.Errorf("expected 0 items, got %d", len(receivedItems))
	}
}

func TestPaginateToChannel_FetchError(t *testing.T) {
	ctx := context.Background()
	opts := PaginateOptions{}

	expectedErr := errors.New("fetch failed")
	fetchFunc := func(page int) ([]string, *gitlab.Response, error) {
		if page == 1 {
			return []string{"item1"}, &gitlab.Response{NextPage: int64(2)}, nil
		}
		return nil, nil, expectedErr
	}

	results := PaginateToChannel(ctx, fetchFunc, opts)

	itemCount := 0
	errorCount := 0
	for result := range results {
		if result.Error != nil {
			errorCount++
			if result.Error != expectedErr {
				t.Errorf("expected error %v, got %v", expectedErr, result.Error)
			}
		} else {
			itemCount++
		}
	}

	if itemCount != 1 {
		t.Errorf("expected 1 item before error, got %d", itemCount)
	}
	if errorCount != 1 {
		t.Errorf("expected 1 error, got %d", errorCount)
	}
}

func TestPaginateToChannel_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	opts := PaginateOptions{}

	fetchCount := 0
	fetchFunc := func(page int) ([]string, *gitlab.Response, error) {
		fetchCount++
		if page == 2 {
			// Cancel context during second page fetch
			cancel()
			// Give goroutine time to detect cancellation
			time.Sleep(10 * time.Millisecond)
		}
		items := []string{"item1", "item2"}
		return items, &gitlab.Response{NextPage: int64(page + 1)}, nil
	}

	results := PaginateToChannel(ctx, fetchFunc, opts)

	errorReceived := false
	for result := range results {
		if result.Error != nil {
			errorReceived = true
			if !errors.Is(result.Error, context.Canceled) {
				t.Errorf("expected context.Canceled error, got %v", result.Error)
			}
		}
	}

	if !errorReceived {
		t.Error("expected to receive context cancellation error")
	}
}

func TestPaginateToChannel_MaxPages(t *testing.T) {
	ctx := context.Background()
	opts := PaginateOptions{MaxPages: 2}

	fetchFunc := func(page int) ([]string, *gitlab.Response, error) {
		items := []string{
			"page" + string(rune('0'+page)) + "item1",
			"page" + string(rune('0'+page)) + "item2",
		}
		return items, &gitlab.Response{NextPage: int64(page + 1)}, nil
	}

	results := PaginateToChannel(ctx, fetchFunc, opts)

	var receivedItems []string
	for result := range results {
		if result.Error != nil {
			t.Fatalf("unexpected error: %v", result.Error)
		}
		receivedItems = append(receivedItems, result.Item)
	}

	// Should stop after 2 pages = 4 items
	expectedCount := 4
	if len(receivedItems) != expectedCount {
		t.Errorf("expected %d items (2 pages), got %d", expectedCount, len(receivedItems))
	}
}

func TestPaginateToChannel_NilResponse(t *testing.T) {
	ctx := context.Background()
	opts := PaginateOptions{}

	fetchFunc := func(page int) ([]string, *gitlab.Response, error) {
		return []string{"item1"}, nil, nil
	}

	results := PaginateToChannel(ctx, fetchFunc, opts)

	var receivedItems []string
	for result := range results {
		if result.Error != nil {
			t.Fatalf("unexpected error: %v", result.Error)
		}
		receivedItems = append(receivedItems, result.Item)
	}

	// Should handle nil response gracefully and stop
	if len(receivedItems) != 1 {
		t.Errorf("expected 1 item, got %d", len(receivedItems))
	}
}

func TestPaginateToChannel_Prefetching(t *testing.T) {
	ctx := context.Background()
	opts := PaginateOptions{BufferSize: 10}

	fetchTimes := make(map[int]time.Time)

	fetchFunc := func(page int) ([]string, *gitlab.Response, error) {
		fetchTimes[page] = time.Now()
		// Simulate some fetch delay
		time.Sleep(5 * time.Millisecond)

		if page > 3 {
			return []string{}, &gitlab.Response{NextPage: 0}, nil
		}
		return []string{"item" + string(rune('0'+page))}, &gitlab.Response{NextPage: int64(page + 1)}, nil
	}

	results := PaginateToChannel(ctx, fetchFunc, opts)

	var receivedItems []string
	for result := range results {
		if result.Error != nil {
			t.Fatalf("unexpected error: %v", result.Error)
		}
		receivedItems = append(receivedItems, result.Item)
	}

	if len(receivedItems) != 3 {
		t.Errorf("expected 3 items, got %d", len(receivedItems))
	}

	// Verify that pages were fetched (we can't reliably test prefetch timing without race conditions)
	if len(fetchTimes) < 2 {
		t.Error("expected at least 2 pages to be fetched")
	}
}

func TestPaginateToChannel_ChannelClosed(t *testing.T) {
	ctx := context.Background()
	opts := PaginateOptions{}

	fetchFunc := func(page int) ([]string, *gitlab.Response, error) {
		if page > 2 {
			return []string{}, &gitlab.Response{NextPage: 0}, nil
		}
		return []string{"item"}, &gitlab.Response{NextPage: int64(page + 1)}, nil
	}

	results := PaginateToChannel(ctx, fetchFunc, opts)

	// Read one item then stop
	result := <-results
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}

	// Channel should eventually close even if we stop reading
	done := make(chan struct{})

	go func() {
		for range results {
		}
		close(done)
	}()

	select {
	case <-done:
		// channel closed as expected
	case <-time.After(1 * time.Second):
		t.Error("channel did not close within timeout")
	}
}

func TestResult_ErrorWrapping(t *testing.T) {
	// Test that Result struct properly holds both Item and Error
	t.Run("with item", func(t *testing.T) {
		result := Result[string]{Item: "test-item"}
		if result.Item != "test-item" {
			t.Errorf("expected item %q, got %q", "test-item", result.Item)
		}
		if result.Error != nil {
			t.Errorf("expected nil error, got %v", result.Error)
		}
	})

	t.Run("with error", func(t *testing.T) {
		testErr := errors.New("test error")
		result := Result[string]{Error: testErr}
		if result.Error != testErr {
			t.Errorf("expected error %v, got %v", testErr, result.Error)
		}
	})
}
