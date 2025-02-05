package search

import (
	"context"
)

/* Restrict the possible values of the Type field to the following:
 * - "movie"
 * - "series"
 * - "episode"
 */
type ResultType string

const (
	Movie  ResultType = "movie"
	Series ResultType = "series"
)

// A SearchResult represents a single entity returned by the search service.
type SearchResult struct {
	Title     string
	Year      string
	ImdbID    string
	PosterURL string
	Type      ResultType
}

// A Searcher is a service that can search for movies, series, and episodes by title, and return zero or more matching results.
type Searcher interface {
	// Search performs a search operation based on the provided query string.
	// It returns a slice of SearchResult and an error, if any occurs during the search.
	//
	// Parameters:
	//   - ctx: The context for controlling cancellation and deadlines.
	//   - query: The search query string.
	//   - maxResults: The maximum number of search results to return.
	//
	// Returns:
	//   - []SearchResult: A slice containing the search results.
	//   - error: An error if the search operation fails.
	Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error)
}
