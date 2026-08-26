package sdk

import (
	"context"
	"testing"
	"time"
)

func TestNewClientDefaultsToMainnetAndHasPracticalTimeout(t *testing.T) {
	client := NewClient(context.Background(), "")

	if client.baseURL != MainnetAPIURL {
		t.Fatalf("base URL mismatch: got %q want %q", client.baseURL, MainnetAPIURL)
	}
	if client.httpClient.Timeout != 10*time.Second {
		t.Fatalf("timeout mismatch: got %s want %s", client.httpClient.Timeout, 10*time.Second)
	}
}

func TestNewClientUsesExplicitBaseURL(t *testing.T) {
	client := NewClient(context.Background(), TestnetAPIURL)

	if client.baseURL != TestnetAPIURL {
		t.Fatalf("base URL mismatch: got %q want %q", client.baseURL, TestnetAPIURL)
	}
}
