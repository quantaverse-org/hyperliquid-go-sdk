package examples

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	sdk "github.com/funcblock-quant/hyperliquid-go-sdk"
)

func TestWebsocket(t *testing.T) {
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

		_, err := ws.Subscribe(tradesSub, func(msg sdk.WSMessage) {
			trades := []sdk.Trade{}
			if err := json.Unmarshal(msg.Data, &trades); err != nil {
				t.Fatalf("Failed to unmarshal trades: %v", err)
			}

			t.Logf("Received trade: %+v", trades)
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

	// Test L2 book subscription
	t.Run("l2book subscription", func(t *testing.T) {
		wg := new(sync.WaitGroup)
		wg.Add(1)
		received := make(chan struct{})

		l2Sub := sdk.Subscription{
			Type: "l2Book",
			Coin: "BTC",
		}

		_, err := ws.Subscribe(l2Sub, func(msg sdk.WSMessage) {
			l2Update := sdk.L2Book{}
			if err := json.Unmarshal(msg.Data, &l2Update); err != nil {
				t.Fatalf("Failed to unmarshal L2 update: %v", err)
			}
			t.Logf("Received L2 update: %+v", l2Update)
			wg.Done()
		})

		if err != nil {
			t.Fatalf("Failed to subscribe to L2 book: %v", err)
		}

		go func() {
			wg.Wait()
			close(received)
		}()

		select {
		case <-received:
			// Test passed
		case <-ctx.Done():
			t.Fatal("Timeout waiting for L2 book messages")
		}
	})

	// Test user fills subscription
	t.Run("user fills subscription", func(t *testing.T) {
		wg := new(sync.WaitGroup)
		wg.Add(1)

		received := make(chan struct{})

		userFillsSub := sdk.Subscription{
			Type: "userFills",
			User: getTestAddress(t),
		}

		_, err := ws.Subscribe(userFillsSub, func(msg sdk.WSMessage) {
			userFills := sdk.WsUserFills{}
			if err := json.Unmarshal(msg.Data, &userFills); err != nil {
				t.Fatalf("Failed to unmarshal user fills: %v", err)
			}
			t.Logf("Received user fills: %+v", userFills)
			wg.Done()
		})

		if err != nil {
			t.Fatalf("Failed to subscribe to user fills: %v", err)
		}

		go func() {
			wg.Wait()
			close(received)
		}()

		select {
		case <-received:
			// Test passed
		case <-ctx.Done():
			t.Fatal("Timeout waiting for user fills messages")
		}
	})

	// Test user order updates subscription
	t.Run("user order updates subscription", func(t *testing.T) {
		wg := new(sync.WaitGroup)
		wg.Add(1)

		received := make(chan struct{})

		orderUpdatesSub := sdk.Subscription{
			Type: sdk.SubTypeOrderUpdates,
			User: getTestAddress(t),
		}

		_, err := ws.Subscribe(orderUpdatesSub, func(msg sdk.WSMessage) {
			orders := []sdk.WsOrder{}
			if err := json.Unmarshal(msg.Data, &orders); err != nil {
				t.Fatalf("Failed to unmarshal order updates: %v", err)
			}
			t.Logf("Received order updates: %+v", orders)
			wg.Done()
		})

		if err != nil {
			t.Fatalf("Failed to subscribe to order updates: %v", err)
		}

		go func() {
			wg.Wait()
			close(received)
		}()

		select {
		case <-received:
			// Test passed
		case <-ctx.Done():
			t.Fatal("Timeout waiting for order updates messages")
		}
	})
}
