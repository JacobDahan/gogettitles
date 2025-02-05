package search

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

type OmdbConstants struct {
	baseURL         string
	apiKeyParameter string
	searchParameter string
	pageParameter   string
}

var omdbConstants = OmdbConstants{
	baseURL:         "https://www.omdbapi.com",
	apiKeyParameter: "apiKey",
	searchParameter: "s",
	pageParameter:   "page",
}

// An OMDB-based Searcher implementation.
type OmdbSearcher struct {
	// The OMDB API key to use for searching.
	apiKey string
	// The HTTP client to use for making requests.
	client *http.Client
}

// NewOmdbSearcher creates a new instance of OmdbSearcher with the specified API key and client.
func NewOmdbSearcher(apiKey string, httpClient *http.Client) *OmdbSearcher {
	return &OmdbSearcher{
		apiKey: apiKey,
		client: httpClient,
	}
}

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
func (os *OmdbSearcher) Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	if maxResults <= 0 {
		return nil, fmt.Errorf("invalid value for maxResults: %d", maxResults)
	}

	results := make([]SearchResult, 0, maxResults)

	// Paginate the search results until we've accumulated maxResults or there are no more results
	pageNumber := 1

	for len(results) < maxResults {
		nextPageExists, err := os.searchPage(ctx, query, maxResults-len(results), pageNumber, &results)
		if err != nil {
			return nil, err
		}

		if !nextPageExists {
			break
		}

		pageNumber++
	}

	return results, nil
}

// searchPage performs a paginated search request to the OMDB API and processes the results.
//
// Parameters:
//   - ctx: The context for the request, allowing for cancellation and timeouts.
//   - query: The search query string.
//   - maxResults: The maximum number of results to return. Must be greater than 0.
//   - pageNumber: The page number to retrieve from the OMDB API.
//   - results: A pointer to a slice of SearchResult where the results will be appended.
//
// Returns:
//   - bool: A boolean indicating whether there are more pages to retrieve.
//   - error: An error if the search request failed or the response could not be processed.
func (os *OmdbSearcher) searchPage(ctx context.Context, query string, maxResults int, pageNumber int, results *[]SearchResult) (bool, error) {
	if maxResults <= 0 {
		return false, fmt.Errorf("invalid value for maxResults: %d", maxResults)
	}

	// Build the URL for the search request
	endpoint, err := url.Parse(omdbConstants.baseURL)
	if err != nil {
		return false, err
	}

	params := url.Values{}
	params.Add(omdbConstants.apiKeyParameter, os.apiKey)
	params.Add(omdbConstants.searchParameter, query)
	params.Add(omdbConstants.pageParameter, fmt.Sprintf("%d", pageNumber))
	endpoint.RawQuery = params.Encode()

	// Create the request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return false, err
	}

	// Perform the request
	resp, err := os.client.Do(req)
	if err != nil {
		return false, err
	}

	defer resp.Body.Close()

	// Define the response structure
	var omdbResponse struct {
		Result []struct {
			Title     string     `json:"Title"`
			Year      string     `json:"Year"`
			ImdbID    string     `json:"imdbID"`
			PosterURL string     `json:"Poster"`
			Type      ResultType `json:"Type"`
		} `json:"Search"`
		TotalResults string `json:"totalResults"`
		Error        string `json:"Error"`
	}

	// Decode the JSON response
	if err := json.NewDecoder(resp.Body).Decode(&omdbResponse); err != nil {
		return false, err
	}

	log.Printf("Found %d results for query \"%s\" on page %d\n", len(omdbResponse.Result), query, pageNumber)

	// Check for TitleNotFound exceptions
	if omdbResponse.Error == "Movie not found!" {
		return false, nil
	}

	// Check for other errors in the response (OMDB API returns an error field if the request fails)
	if omdbResponse.Error != "" {
		return false, fmt.Errorf("OMDB API request failed with error: %s", omdbResponse.Error)
	}

	// Convert the response to the SearchResult format
	for _, result := range omdbResponse.Result {
		maxResults--

		if maxResults < 0 {
			break
		}

		*results = append(*results, SearchResult{
			Title:     result.Title,
			Year:      result.Year,
			ImdbID:    result.ImdbID,
			PosterURL: result.PosterURL,
			Type:      result.Type,
		})
	}

	// Check if there are more pages to retrieve
	totalResults, err := strconv.Atoi(omdbResponse.TotalResults)
	if err != nil {
		return false, fmt.Errorf("failed to convert totalResults to int: %v", err)
	}

	if len(*results) < totalResults && maxResults > 0 {
		return true, nil
	}

	// Otherwise, stop paginating
	return false, nil
}
