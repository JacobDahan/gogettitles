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

func TestNewOmdbSearcher(t *testing.T) {
	searcher := search.NewOmdbSearcher(testAPIKey, http.DefaultClient)
	if searcher == nil {
		t.Fatal("expected non-nil OmdbSearcher")
	}
}

func TestOmdbSearcher_Search_InvalidMaxResults(t *testing.T) {
	searcher := search.NewOmdbSearcher(testAPIKey, http.DefaultClient)
	_, err := searcher.Search(context.Background(), "Test", 0)
	var mrErr *search.InvalidMaxResultsError
	if err == nil || !errors.As(err, &mrErr) {
		t.Fatalf("expected invalid max results error, got %v", err)
	}
}

func TestOmdbSearcher_Search_ResultParsingError(t *testing.T) {
	defer gock.Off() // Flush pending mocks after test execution

	query := "Test"
	invalidJSON := `{"invalid_json":`

	gock.New("https://www.omdbapi.com").
		Get("/").
		MatchParam("apiKey", testAPIKey).
		MatchParam("s", query).
		Reply(200).
		BodyString(invalidJSON)

	searcher := search.NewOmdbSearcher(testAPIKey, http.DefaultClient)
	_, err := searcher.Search(context.Background(), query, 5)
	var rpErr *search.ResultParsingError
	if err == nil || !errors.As(err, &rpErr) {
		t.Fatalf("expected result parsing error, got %v", err)
	}
}

func TestOmdbSearcher_Search_ContextTimeout(t *testing.T) {
	defer gock.Off() // Flush pending mocks after test execution

	query := "Test"
	serverResponse := `{
        "Response":"True",
        "Search":[]
    }`

	gock.New("https://www.omdbapi.com").
		Get("/").
		MatchParam("apiKey", testAPIKey).
		MatchParam("s", query).
		Reply(200).
		JSON(json.RawMessage(serverResponse))

	searcher := search.NewOmdbSearcher(testAPIKey, http.DefaultClient)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	_, err := searcher.Search(ctx, query, 5)
	if err == nil || !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("expected context deadline exceeded error, got %v", err)
	}
}

func TestOmdbSearcher_Search_Success(t *testing.T) {
	defer gock.Off() // Flush pending mocks after test execution

	query := "Test"
	mockData, err := loadMockResponse("omdb_response.json")
	if err != nil {
		t.Fatalf("unexpected error reading test data: %v", err)
	}

	gock.New("https://www.omdbapi.com").
		Get("/").
		MatchParam("apiKey", testAPIKey).
		MatchParam("page", "1").
		MatchParam("s", query).
		Reply(200).
		JSON(json.RawMessage(mockData))

	searcher := search.NewOmdbSearcher(testAPIKey, http.DefaultClient)
	results, err := searcher.Search(context.Background(), query, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 5 {
		t.Errorf("expected 5 results, got %d", len(results))
	}
}

func TestOmdbSearcher_Search_Success_Max_Results_Greater_Than_Total(t *testing.T) {
	defer gock.Off() // Flush pending mocks after test execution

	query := "Test"
	mockData, err := loadMockResponse("omdb_response.json")
	if err != nil {
		t.Fatalf("unexpected error reading test data: %v", err)
	}

	gock.New("https://www.omdbapi.com").
		Get("/").
		MatchParam("apiKey", testAPIKey).
		MatchParam("page", "1").
		MatchParam("s", query).
		Reply(200).
		JSON(json.RawMessage(mockData))

	searcher := search.NewOmdbSearcher(testAPIKey, http.DefaultClient)
	results, err := searcher.Search(context.Background(), query, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 5 {
		t.Errorf("expected 5 results, got %d", len(results))
	}
}

func TestOmdbSearcher_Search_Success_Max_Results_Lower_Than_Total(t *testing.T) {
	defer gock.Off() // Flush pending mocks after test execution

	query := "Test"
	mockData, err := loadMockResponse("omdb_response.json")
	if err != nil {
		t.Fatalf("unexpected error reading test data: %v", err)
	}

	gock.New("https://www.omdbapi.com").
		Get("/").
		MatchParam("apiKey", testAPIKey).
		MatchParam("page", "1").
		MatchParam("s", query).
		Reply(200).
		JSON(json.RawMessage(mockData))

	searcher := search.NewOmdbSearcher(testAPIKey, http.DefaultClient)
	results, err := searcher.Search(context.Background(), query, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestOmdbSearcher_Search_Success_Pagination(t *testing.T) {
	defer gock.Off() // Flush pending mocks after test execution

	query := "Test"

	// Mock the first page of results (complete)
	mockData, err := loadMockResponse("omdb_paginated_response_1.json")
	if err != nil {
		t.Fatalf("unexpected error reading test data: %v", err)
	}

	gock.New("https://www.omdbapi.com").
		Get("/").
		MatchParam("apiKey", testAPIKey).
		MatchParam("page", "1").
		MatchParam("s", query).
		Reply(200).
		JSON(json.RawMessage(mockData))

	// Mock the second page of results (incomplete)
	mockData, err = loadMockResponse("omdb_paginated_response_2.json")
	if err != nil {
		t.Fatalf("unexpected error reading test data: %v", err)
	}

	gock.New("https://www.omdbapi.com").
		Get("/").
		MatchParam("apiKey", testAPIKey).
		MatchParam("page", "2").
		MatchParam("s", query).
		Reply(200).
		JSON(json.RawMessage(mockData))

	searcher := search.NewOmdbSearcher(testAPIKey, http.DefaultClient)
	results, err := searcher.Search(context.Background(), query, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 5 {
		t.Errorf("expected 5 results, got %d", len(results))
	}
}

func TestOmdbSearcher_Search_NotFound(t *testing.T) {
	defer gock.Off() // Flush pending mocks after test execution

	query := "Unknown"
	serverResponse := `{
		"Response":"False",
		"Error":"Movie not found!"
	}`

	gock.New("https://www.omdbapi.com").
		Get("/").
		MatchParam("apiKey", testAPIKey).
		MatchParam("page", "1").
		MatchParam("s", query).
		Reply(200).
		JSON(json.RawMessage(serverResponse))

	searcher := search.NewOmdbSearcher(testAPIKey, http.DefaultClient)

	results, err := searcher.Search(context.Background(), query, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for not found, got %d", len(results))
	}
}

func TestOmdbSearcher_Search_NotAuthorized(t *testing.T) {
	defer gock.Off() // Flush pending mocks after test execution

	query := "Test"
	serverResponse := `{
		"Response":"False",
		"Error":"Invalid API key!"
	}`

	gock.New("https://www.omdbapi.com").
		Get("/").
		MatchParam("apiKey", testAPIKey).
		MatchParam("page", "1").
		MatchParam("s", query).
		Reply(401).
		JSON(json.RawMessage(serverResponse))

	searcher := search.NewOmdbSearcher(testAPIKey, http.DefaultClient)

	_, err := searcher.Search(context.Background(), query, 5)
	if err == nil || !strings.Contains(err.Error(), "Invalid API key!") {
		t.Errorf("expected error containing 'Invalid API key!', got %v", err)
	}
}

func TestOmdbSearcher_Search_ErrorResponse(t *testing.T) {
	defer gock.Off() // Flush pending mocks after test execution

	query := "Test"
	serverResponse := `{
		"Response":"False",
		"Error":"Server error"
	}`

	gock.New("https://www.omdbapi.com").
		Get("/").
		MatchParam("apiKey", testAPIKey).
		MatchParam("page", "1").
		MatchParam("s", query).
		Reply(500).
		JSON(json.RawMessage(serverResponse))

	searcher := search.NewOmdbSearcher(testAPIKey, http.DefaultClient)

	_, err := searcher.Search(context.Background(), query, 5)
	if err == nil || !strings.Contains(err.Error(), "Server error") {
		t.Errorf("expected error containing 'Server error', got %v", err)
	}
}
