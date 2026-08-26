package examples

import (
	"context"
	"testing"
	"time"

	sdk "github.com/funcblock-quant/hyperliquid-go-sdk"
)

func TestCandlesSnapshot(t *testing.T) {
	info, err := sdk.NewInfo(sdk.TestnetAPIURL)
	if err != nil {
		t.Fatalf("Failed to create sdk.Info for candles_test: %v", err)
	}

	now := time.Now()
	startTime := now.Add(-1 * time.Hour).UnixMilli()
	endTime := now.UnixMilli()

	tests := []struct {
		name     string
		coin     string
		interval string
	}{
		{name: "BTC 1m", coin: "BTC", interval: "1m"},
		{name: "ETH 5m", coin: "ETH", interval: "5m"},
		{name: "BTC 15m", coin: "BTC", interval: "15m"},
		{name: "ETH 1h", coin: "ETH", interval: "1h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candles, err := info.CandlesSnapshot(tt.coin, tt.interval, startTime, endTime)
			if err != nil {
				t.Fatalf("Failed to fetch candles: %v", err)
			}

			if len(candles) == 0 {
				t.Error("Expected non-empty candles response")
			}

			// Print first candle for inspection
			first := candles[0]
			t.Logf("First candle: %+v", first)
		})
	}
}

func TestCandleWebSocket(t *testing.T) {
	ws := sdk.NewWebsocketClient(sdk.TestnetAPIURL)

	if err := ws.Connect(context.Background()); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	done := make(chan bool)
	sub := sdk.Subscription{
		Type:     "candle",
		Coin:     "BTC",
		Interval: "1m",
	}

	_, err := ws.Subscribe(sub, func(msg sdk.WSMessage) {
		if msg.Channel != "candle" {
			t.Errorf("Expected channel 'candle', got %s", msg.Channel)
		}

		// Validate candle data exists
		if msg.Data == nil {
			t.Error("Expected non-nil candle data")
		}

		done <- true
	})

	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	select {
	case <-done:
		// Test passed
	case <-time.After(10 * time.Second):
		t.Error("Timeout waiting for candle update")
	}
}
