package test

import (
	"fmt"
	"net/http"
	"testing"
)

func TestGetAllUsers(t *testing.T) {
	CreateServer()
	// Test without token
	resp, err := http.Get(fmt.Sprintf("%s/api/users", GetServer().URL))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if resp.StatusCode != 401 {
		t.Fatalf("Expected status code 401, got %v", resp.StatusCode)
	}

}