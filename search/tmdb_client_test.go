package search_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/h2non/gock"
	"github.com/jdahan/gogettitles/search"
)

func TestNewTmdbSearcher(t *testing.T) {
	searcher := search.NewTmdbSearcher(testAPIKey, http.DefaultClient)
	if searcher == nil {
		t.Fatal("expected non-nil TmdbSearcher")
	}
}

func TestTmdbSearcher_Search_InvalidMaxResults(t *testing.T) {
	searcher := search.NewTmdbSearcher(testAPIKey, http.DefaultClient)
	_, err := searcher.Search(context.Background(), "Matrix", 0)
	var mrErr *search.InvalidMaxResultsError
	if err == nil || !errors.As(err, &mrErr) {
		t.Fatalf("expected invalid max results error, got %v", err)
	}
}

func TestTmdbSearcher_Search_ResultParsingError(t *testing.T) {
	defer gock.Off() // Flush pending mocks after test execution

	query := "Matrix"
	invalidJSON := `{"invalid_json":`

	gock.New("https://api.themoviedb.org").
		Path("/3/search/multi").
		Get("/").
		MatchParam("query", query).
		Reply(200).
		BodyString(invalidJSON)

	searcher := search.NewTmdbSearcher(testAPIKey, http.DefaultClient)
	_, err := searcher.Search(context.Background(), query, 5)
	var rpErr *search.ResultParsingError
	if err == nil || !errors.As(err, &rpErr) {
		t.Fatalf("expected result parsing error, got %v", err)
	}
}

func TestTmdbSearcher_Search_ContextTimeout(t *testing.T) {
	defer gock.Off() // Flush pending mocks after test execution

	query := "Matrix"
	serverResponse := `{
        "results": [],
        "total_results": 0,
        "total_pages": 1
    }`

	gock.New("https://api.themoviedb.org").
		Path("/3/search/multi").
		Get("/").
		MatchParam("query", query).
		Reply(200).
		JSON(json.RawMessage(serverResponse))

	searcher := search.NewTmdbSearcher(testAPIKey, http.DefaultClient)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	_, err := searcher.Search(ctx, query, 5)
	if err == nil || !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("expected context deadline exceeded error, got %v", err)
	}
}

func TestTmdbSearcher_Search_Success(t *testing.T) {
	defer gock.Off() // Flush pending mocks after test execution

	query := "Star Wars"
	mockData, err := loadMockResponse("tmdb_response.json")
	if err != nil {
		t.Fatalf("unexpected error reading test data: %v", err)
	}

	gock.New("https://api.themoviedb.org").
		Path("/3/search/multi").
		Get("/").
		MatchParam("page", "1").
		MatchParam("query", query).
		MatchHeader("accept", "application/json").
		MatchHeader("Authorization", "Bearer "+testAPIKey).
		Reply(200).
		JSON(json.RawMessage(mockData))

	searcher := search.NewTmdbSearcher(testAPIKey, http.DefaultClient)
	results, err := searcher.Search(context.Background(), query, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 5 {
		t.Errorf("expected 5 results, got %d", len(results))
	}
}

func TestTmdbSearcher_Search_Success_Max_Results_Greater_Than_Total(t *testing.T) {
	defer gock.Off() // Flush pending mocks after test execution

	query := "Star Wars"
	mockData, err := loadMockResponse("tmdb_response.json")
	if err != nil {
		t.Fatalf("unexpected error reading test data: %v", err)
	}

	gock.New("https://api.themoviedb.org").
		Path("/3/search/multi").
		Get("/").
		MatchParam("page", "1").
		MatchParam("query", query).
		MatchHeader("accept", "application/json").
		MatchHeader("Authorization", "Bearer "+testAPIKey).
		Reply(200).
		JSON(json.RawMessage(mockData))

	searcher := search.NewTmdbSearcher(testAPIKey, http.DefaultClient)
	results, err := searcher.Search(context.Background(), query, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 5 {
		t.Errorf("expected 5 results, got %d", len(results))
	}
}

func TestTmdbSearcher_Search_Success_Max_Results_Lower_Than_Total(t *testing.T) {
	defer gock.Off() // Flush pending mocks after test execution

	query := "Star Wars"
	mockData, err := loadMockResponse("tmdb_response.json")
	if err != nil {
		t.Fatalf("unexpected error reading test data: %v", err)
	}

	gock.New("https://api.themoviedb.org").
		Path("/3/search/multi").
		Get("/").
		MatchParam("page", "1").
		MatchParam("query", query).
		MatchHeader("accept", "application/json").
		MatchHeader("Authorization", "Bearer "+testAPIKey).
		Reply(200).
		JSON(json.RawMessage(mockData))

	searcher := search.NewTmdbSearcher(testAPIKey, http.DefaultClient)
	results, err := searcher.Search(context.Background(), query, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestTmdbSearcher_Search_Success_Pagination(t *testing.T) {
	defer gock.Off() // Flush pending mocks after test execution

	query := "Star Wars"

	// Mock the first page of results (complete)
	mockData, err := loadMockResponse("tmdb_paginated_response_1.json")
	if err != nil {
		t.Fatalf("unexpected error reading test data: %v", err)
	}

	gock.New("https://api.themoviedb.org").
		Path("/3/search/multi").
		Get("/").
		MatchParam("page", "1").
		MatchParam("query", query).
		MatchHeader("accept", "application/json").
		MatchHeader("Authorization", "Bearer "+testAPIKey).
		Reply(200).
		JSON(json.RawMessage(mockData))

	// Mock the second page of results (incomplete)
	mockData, err = loadMockResponse("tmdb_paginated_response_2.json")
	if err != nil {
		t.Fatalf("unexpected error reading test data: %v", err)
	}

	gock.New("https://api.themoviedb.org").
		Path("/3/search/multi").
		Get("/").
		MatchParam("page", "2").
		MatchParam("query", query).
		MatchHeader("accept", "application/json").
		MatchHeader("Authorization", "Bearer "+testAPIKey).
		Reply(200).
		JSON(json.RawMessage(mockData))

	searcher := search.NewTmdbSearcher(testAPIKey, http.DefaultClient)
	results, err := searcher.Search(context.Background(), query, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 5 {
		t.Errorf("expected 5 results, got %d", len(results))
	}
}

func TestTmdbSearcher_Search_NotAuthorized(t *testing.T) {
	defer gock.Off() // Flush pending mocks after test execution

	query := "Star Wars"
	serverResponse := `{
		"status_code":7,
		"status_message":"Invalid API key: You must be granted a valid key.",
		"success":false
	}`

	gock.New("https://api.themoviedb.org").
		Path("/3/search/multi").
		Get("/").
		MatchParam("page", "1").
		MatchParam("query", query).
		MatchHeader("accept", "application/json").
		MatchHeader("Authorization", "Bearer "+testAPIKey).
		Reply(401).
		JSON(json.RawMessage(serverResponse))

	searcher := search.NewTmdbSearcher(testAPIKey, http.DefaultClient)

	_, err := searcher.Search(context.Background(), query, 5)
	if err == nil || !strings.Contains(err.Error(), "Invalid API key") {
		t.Errorf("expected error containing 'Invalid API key', got %v", err)
	}
}

func TestTmdbSearcher_Search_SearchProviderError(t *testing.T) {
	defer gock.Off() // Flush pending mocks after test execution

	query := "Matrix"

	gock.New("https://api.themoviedb.org").
		Path("/3/search/multi").
		Get("/").
		MatchParam("query", query).
		ReplyError(&http.ProtocolError{ErrorString: "mock protocol error"})

	searcher := search.NewTmdbSearcher(testAPIKey, http.DefaultClient)
	_, err := searcher.Search(context.Background(), query, 5)
	if err == nil || !strings.Contains(err.Error(), "mock protocol error") {
		t.Fatalf("expected search provider error, got %v", err)
	}
}
