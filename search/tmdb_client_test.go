package search_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

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
	if err == nil {
		t.Fatal("expected error for invalid maxResults")
	}
}

func TestTmdbSearcher_Search_ContextExpired(t *testing.T) {
	searcher := search.NewTmdbSearcher(testAPIKey, http.DefaultClient)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := searcher.Search(ctx, "Matrix", 5)
	if err == nil {
		t.Fatal("expected error for expired context")
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
