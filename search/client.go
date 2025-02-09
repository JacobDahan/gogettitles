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
	Movie   ResultType = "movie"
	Series  ResultType = "series"
	Episode ResultType = "episode"
)

// A SearchResult represents a single entity returned by the search service.
type SearchResult struct {
	Title      string
	Year       string
	ImdbID     string
	ProviderId string
	PosterURL  string
	Type       ResultType
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

// InvalidMaxResultsError is an error type that is returned when the maxResults parameter is invalid.
type InvalidMaxResultsError struct{}

// NewInvalidMaxResultsError creates a new InvalidMaxResultsError.
func NewInvalidMaxResultsError() *InvalidMaxResultsError {
	return &InvalidMaxResultsError{}
}

// Error returns the default error message associated with the InvalidMaxResultsError.
func (e *InvalidMaxResultsError) Error() string {
	return "maxResults must be greater than zero"
}

// SearchProviderError is an error type that is returned when the search provider encounters an error.
type SearchProviderError struct {
	errorMessage string
}

// NewSearchProviderError creates a new SearchProviderError with the specified error message.
func NewSearchProviderError(errorMessage string) *SearchProviderError {
	return &SearchProviderError{errorMessage: errorMessage}
}

// Error returns the error message associated with the SearchProviderError.
func (e *SearchProviderError) Error() string {
	return e.errorMessage
}

// ResultParsingError is an error type that is returned when there is an error parsing the search results.
type ResultParsingError struct {
	reason string
}

// NewResultParsingError creates a new ResultParsingError with the specified reason.
func NewResultParsingError(reason string) *ResultParsingError {
	return &ResultParsingError{reason: reason}
}

// Error returns the reason associated with the ResultParsingError.
func (e *ResultParsingError) Error() string {
	return e.reason
}
