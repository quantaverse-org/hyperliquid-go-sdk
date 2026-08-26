package examples

import (
	"context"
	"encoding/json"
	sdk "github.com/funcblock-quant/hyperliquid-go-sdk"
	"sync"
	"testing"
	"time"
)

func TestWsTrade(t *testing.T) {
	ws := sdk.NewWebsocketClient(sdk.TestnetAPIURL)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Log("Connecting to websocket")
	if err := ws.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	t.Log("Connected to websocket")

	// Test trades subscription
	t.Run("trades subscription", func(t *testing.T) {
		wg := new(sync.WaitGroup)
		wg.Add(1)
		received := make(chan struct{})

		tradesSub := sdk.Subscription{
			Type: "trades",
			Coin: "SOL",
		}

		// Create an info client to fetch user fills
		info, err := sdk.NewInfo(sdk.TestnetAPIURL)
		if err != nil {
			t.Fatalf("Failed to create sdk.Info for trade_test: %v", err)
		}

		_, err = ws.Subscribe(tradesSub, func(msg sdk.WSMessage) {
			trades := []sdk.Trade{}
			if err := json.Unmarshal(msg.Data, &trades); err != nil {
				t.Fatalf("Failed to unmarshal trades: %v", err)
			}

			// Process each trade individually
			for _, trade := range trades {
				// For each user in the trade, fetch their fills and find matching fill by trade ID
				for _, user := range trade.Users {
					// Fetch user fills
					userFills, err := info.UserFills(user)
					if err != nil {
						//t.Logf("Failed to fetch fills for user %s: %v", user, err)
						continue
					}

					// Find matching fill by trade ID
					var matchingFill *sdk.Fill
					for i, fill := range userFills {
						if fill.Tid == trade.Tid {
							matchingFill = &userFills[i]
							break
						}
					}

					if matchingFill != nil {
						t.Logf("Found matching fill for user %s: %+v, trade: %+v", user, matchingFill, trade)
					} else {
						//t.Logf("No matching fill found for user %s with trade ID %d", user, trade.Tid)
					}
				}
			}

			wg.Done()
		})

		if err != nil {
			t.Fatalf("Failed to subscribe to trades: %v", err)
		}

		go func() {
			wg.Wait()
			close(received)
		}()

		select {
		case <-received:
			// Test passed
		case <-ctx.Done():
			t.Fatal("Timeout waiting for trades messages")
		}
	})
}
