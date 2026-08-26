package sdk

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWebsocketReconnectResubscribesAfterReadDisconnect(t *testing.T) {
	var upgrader websocket.Upgrader
	var connectionCount atomic.Int32
	var subscribeCount atomic.Int32
	var closeSecondSubscribe sync.Once
	secondSubscribe := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade websocket: %v", err)
			return
		}

		connectionCount.Add(1)
		go func() {
			defer conn.Close()
			for {
				var cmd WsCommand
				if err := conn.ReadJSON(&cmd); err != nil {
					return
				}
				if cmd.Method != "subscribe" {
					continue
				}
				switch subscribeCount.Add(1) {
				case 1:
					return
				case 2:
					closeSecondSubscribe.Do(func() {
						close(secondSubscribe)
					})
				}
			}
		}()
	}))
	defer server.Close()

	client := &WebsocketClient{
		url:           "ws" + strings.TrimPrefix(server.URL, "http"),
		subscriptions: make(map[subKey]map[int]*subscriptionCallback),
		done:          make(chan struct{}),
		reconnectWait: 10 * time.Millisecond,
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("connect: %v", err)
	}
	if _, err := client.Subscribe(Subscription{Type: SubTypeTrades, Coin: "BTC"}, func(WSMessage) {}); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	select {
	case <-secondSubscribe:
	case <-ctx.Done():
		t.Fatalf("timed out waiting for resubscribe; connections=%d subscribes=%d", connectionCount.Load(), subscribeCount.Load())
	}

	if got := connectionCount.Load(); got < 2 {
		t.Fatalf("expected reconnect to open a second connection, got %d", got)
	}
}
