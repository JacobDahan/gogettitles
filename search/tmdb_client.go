package search

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
)

type TmdbConstants struct {
	baseURL         string
	apiVersion      string
	searchEndpoint  string
	searchType      string
	searchParameter string
	pageParameter   string
}

var tmdbConstants = TmdbConstants{
	baseURL:         "https://api.themoviedb.org",
	apiVersion:      "3",
	searchEndpoint:  "search",
	searchType:      "multi",
	searchParameter: "query",
	pageParameter:   "page",
}

// An TMDB-based Searcher implementation.
type TmdbSearcher struct {
	// The TMDB API key to use for searching.
	apiKey string
	// The HTTP client to use for making requests.
	client *http.Client
}

// NewTmdbSearcher creates a new instance of TmdbSearcher with the specified API key and client.
func NewTmdbSearcher(apiKey string, httpClient *http.Client) *TmdbSearcher {
	return &TmdbSearcher{
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
func (os *TmdbSearcher) Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
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

// searchPage performs a paginated search request to the TMDB API and processes the results.
//
// Parameters:
//   - ctx: The context for the request, allowing for cancellation and timeouts.
//   - query: The search query string.
//   - maxResults: The maximum number of results to return. Must be greater than 0.
//   - pageNumber: The page number to retrieve from the TMDB API.
//   - results: A pointer to a slice of SearchResult where the results will be appended.
//
// Returns:
//   - bool: A boolean indicating whether there are more pages to retrieve.
//   - error: An error if the search request failed or the response could not be processed.
func (os *TmdbSearcher) searchPage(ctx context.Context, query string, maxResults int, pageNumber int, results *[]SearchResult) (bool, error) {
	if maxResults <= 0 {
		return false, fmt.Errorf("invalid value for maxResults: %d", maxResults)
	}

	// Build the URL for the search request
	u, err := url.JoinPath(tmdbConstants.baseURL, tmdbConstants.apiVersion, tmdbConstants.searchEndpoint, tmdbConstants.searchType)
	if err != nil {
		return false, err
	}

	endpoint, err := url.Parse(u)
	if err != nil {
		return false, err
	}

	params := url.Values{}
	params.Add(tmdbConstants.searchParameter, query)
	params.Add(tmdbConstants.pageParameter, fmt.Sprintf("%d", pageNumber))
	endpoint.RawQuery = params.Encode()

	// Create the request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return false, err
	}

	// Add authentication header
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", os.apiKey))

	// Add the accept header
	req.Header.Add("accept", "application/json")

	// Perform the request
	resp, err := os.client.Do(req)
	if err != nil {
		return false, err
	}

	defer resp.Body.Close()

	// Define the response structure
	var tmdbResponse struct {
		Result []struct {
			Title     string `json:"title"`
			Name      string `json:"name"`
			Year      string `json:"first_air_date"`
			ImdbID    string `json:"imdb_id"`
			PosterURL string `json:"poster_path"`
			Type      string `json:"media_type"`
			TmdbId    int    `json:"id"`
		} `json:"results"`
		TotalResults  int    `json:"total_results"`
		TotalPages    int    `json:"total_pages"`
		Success       bool   `json:"success"`
		StatusMessage string `json:"status_message"`
	}

	// Decode the JSON response
	if err := json.NewDecoder(resp.Body).Decode(&tmdbResponse); err != nil {
		return false, err
	}

	// Check for a successful response
	if resp.StatusCode != http.StatusOK || (!tmdbResponse.Success && tmdbResponse.StatusMessage != "") {
		return false, fmt.Errorf("search request failed: %s", tmdbResponse.StatusMessage)
	}

	log.Printf("Found %d results for query \"%s\" on page %d\n", len(tmdbResponse.Result), query, pageNumber)

	// Converts the response to the SearchResult format
	for _, result := range tmdbResponse.Result {
		var resultTitle string
		if result.Title != "" {
			resultTitle = result.Title
		} else {
			resultTitle = result.Name
		}

		var resultType ResultType
		switch result.Type {
		case "movie":
			resultType = Movie
		case "tv":
			resultType = Series
		default:
			// TMDB returns other types like "person"
			continue
		}

		maxResults--
		if maxResults < 0 {
			break
		}

		*results = append(*results, SearchResult{
			Title:      resultTitle,
			Year:       result.Year,
			ImdbID:     result.ImdbID,
			PosterURL:  result.PosterURL,
			Type:       resultType,
			ProviderId: fmt.Sprintf("%d", result.TmdbId),
		})
	}

	if pageNumber < tmdbResponse.TotalPages && maxResults > 0 {
		return true, nil
	}

	// Otherwise, stop paginating
	return false, nil
}
