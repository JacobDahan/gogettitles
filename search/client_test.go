package search_test

import "os"

const (
	testAPIKey = "testkey"
)

func loadMockResponse(file string) ([]byte, error) {
	return os.ReadFile("testdata/" + file)
}
