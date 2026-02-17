package api

import (
	"context"
	"sync"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// PaginateOptions configures pagination behavior.
type PaginateOptions struct {
	PerPage    int // Items per page (default: 100)
	MaxPages   int // Maximum pages to fetch (0 = unlimited)
	BufferSize int // Channel buffer size for prefetching (default: 100)
}

// Result wraps an item with a potential error.
type Result[T any] struct {
	Item  T
	Error error
}

// FetchPageFunc is a function that fetches a single page of items.
// It receives the page number and should return the items, response metadata, and any error.
type FetchPageFunc[T any] func(page int) ([]T, *gitlab.Response, error)

// PaginateToChannel fetches items progressively using the provided fetch function
// and sends them to a channel. It automatically handles pagination and prefetching
// to ensure smooth streaming of large result sets.
//
// The function returns a read-only channel that will be closed when all items
// have been fetched or an error occurs. Errors are sent through the channel
// as Result items with a non-nil Error field.
//
// Example usage:
//
//	opts := api.PaginateOptions{PerPage: 100, BufferSize: 50}
//	fetchFunc := func(page int) ([]*gitlab.MergeRequest, *gitlab.Response, error) {
//	    listOpts := &gitlab.ListProjectMergeRequestsOptions{
//	        ListOptions: gitlab.ListOptions{Page: page, PerPage: opts.PerPage},
//	    }
//	    return client.MergeRequests.ListProjectMergeRequests(projectID, listOpts)
//	}
//	results := api.PaginateToChannel(ctx, fetchFunc, opts)
//	for result := range results {
//	    if result.Error != nil {
//	        // handle error
//	    }
//	    // process result.Item
//	}
func PaginateToChannel[T any](ctx context.Context, fetchFunc FetchPageFunc[T], opts PaginateOptions) <-chan Result[T] {
	// Set defaults
	if opts.PerPage <= 0 {
		opts.PerPage = 100
	}
	if opts.BufferSize <= 0 {
		opts.BufferSize = 100
	}

	// Create buffered channel for results
	results := make(chan Result[T], opts.BufferSize)

	// Start goroutine to fetch pages and send items
	go func() {
		defer close(results)

		page := 1
		var wg sync.WaitGroup
		prefetchChan := make(chan fetchResult[T], 1)
		prefetchPage := 0

		for {
			// Check context cancellation
			select {
			case <-ctx.Done():
				results <- Result[T]{Error: ctx.Err()}
				return
			default:
			}

			// Fetch current page (either from prefetch or new fetch)
			var items []T
			var resp *gitlab.Response
			var err error

			if prefetchPage == page {
				// Use prefetched result, also check for context cancellation
				select {
				case <-ctx.Done():
					results <- Result[T]{Error: ctx.Err()}
					return
				case fetched := <-prefetchChan:
					items, resp, err = fetched.items, fetched.resp, fetched.err
				}
				prefetchPage = 0
			} else {
				// Fetch current page
				items, resp, err = fetchFunc(page)
			}

			if err != nil {
				// Send error and stop pagination
				results <- Result[T]{Error: err}
				return
			}

			// Send items to channel
			for _, item := range items {
				select {
				case <-ctx.Done():
					results <- Result[T]{Error: ctx.Err()}
					return
				case results <- Result[T]{Item: item}:
				}
			}

			// Check if we've reached the last page
			if resp == nil || resp.NextPage == 0 || len(items) == 0 {
				break
			}

			// Check if we've hit max pages limit
			if opts.MaxPages > 0 && page >= opts.MaxPages {
				break
			}

			// Prefetch next page in background
			nextPage := page + 1
			if prefetchPage == 0 {
				prefetchPage = nextPage
				wg.Add(1)
				go func(p int) {
					defer wg.Done()
					items, resp, err := fetchFunc(p)
					select {
					case <-ctx.Done():
						return
					case prefetchChan <- fetchResult[T]{items: items, resp: resp, err: err}:
					}
				}(nextPage)
			}

			page = nextPage
		}

		// Wait for any pending prefetch to complete
		wg.Wait()
	}()

	return results
}

// fetchResult holds the result of a page fetch operation.
type fetchResult[T any] struct {
	items []T
	resp  *gitlab.Response
	err   error
}
