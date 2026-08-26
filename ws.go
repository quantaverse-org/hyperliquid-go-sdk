package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	SubTypeTrades                      = "trades"
	SubTypeL2Book                      = "l2Book"
	SubTypeUserFills                   = "userFills"
	SubTypeOrderUpdates                = "orderUpdates"
	SubTypeCandle                      = "candle"
	SubTypeAllMids                     = "allMids"
	SubTypeNotification                = "notification"
	SubTypeWebData2                    = "webData2"
	SubTypeWebData3                    = "webData3"
	SubTypeActiveAssetCtx              = "activeAssetCtx"
	SubTypeActiveAssetData             = "activeAssetData"
	SubTypeUserEvents                  = "userEvents"
	SubTypeUserFundings                = "userFundings"
	SubTypeUserNonFundingLedgerUpdates = "userNonFundingLedgerUpdates"
	SubTypeBBO                         = "bbo"
	SubTypeTwapHistory                 = "twapHistory"
	SubTypeTwapSliceFills              = "twapSliceFills"
	SubTypeTwapStates                  = "twapStates"
	SubTypeSpotState                   = "spotState"
)

type WebsocketClient struct {
	url           string
	conn          *websocket.Conn
	mu            sync.RWMutex
	writeMu       sync.Mutex
	reconnectMu   sync.Mutex
	subscriptions map[subKey]map[int]*subscriptionCallback
	nextSubID     atomic.Int32
	done          chan struct{}
	reconnectWait time.Duration
}

func NewWebsocketClient(baseURL string) *WebsocketClient {
	if baseURL == "" {
		baseURL = MainnetAPIURL
	}
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		log.Fatalf("invalid URL: %v", err)
	}
	parsedURL.Scheme = "wss"
	parsedURL.Path = "/ws"
	wsURL := parsedURL.String()

	return &WebsocketClient{
		url:           wsURL,
		subscriptions: make(map[subKey]map[int]*subscriptionCallback),
		done:          make(chan struct{}),
		reconnectWait: time.Second,
	}
}

func (w *WebsocketClient) Connect(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.conn != nil {
		return nil
	}

	dialer := websocket.Dialer{}

	conn, _, err := dialer.DialContext(ctx, w.url, nil)
	if err != nil {
		return fmt.Errorf("websocket dial: %w", err)
	}

	w.conn = conn

	go w.readPump(ctx, conn)
	go w.pingPump(ctx, conn)

	return w.resubscribeAll()
}

func (w *WebsocketClient) Subscribe(sub Subscription, callback func(WSMessage)) (int, error) {
	if callback == nil {
		return 0, fmt.Errorf("callback cannot be nil")
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	key := sub.key()
	id := int(w.nextSubID.Add(1))

	if w.subscriptions[key] == nil {
		w.subscriptions[key] = make(map[int]*subscriptionCallback)
	}

	w.subscriptions[key][id] = &subscriptionCallback{
		id:       id,
		callback: callback,
	}

	if err := w.sendSubscribe(sub); err != nil {
		delete(w.subscriptions[key], id)
		return 0, fmt.Errorf("subscribe: %w", err)
	}

	return id, nil
}

func (w *WebsocketClient) Unsubscribe(sub Subscription, id int) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	key := sub.key()
	subs, ok := w.subscriptions[key]
	if !ok {
		return fmt.Errorf("subscription not found")
	}

	if _, ok := subs[id]; !ok {
		return fmt.Errorf("subscription ID not found")
	}

	delete(subs, id)

	if len(subs) == 0 {
		delete(w.subscriptions, key)
		if err := w.sendUnsubscribe(sub); err != nil {
			return fmt.Errorf("unsubscribe: %w", err)
		}
	}

	return nil
}

func (w *WebsocketClient) Close() error {
	close(w.done)

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.conn != nil {
		return w.conn.Close()
	}
	return nil
}

// Private methods

func (w *WebsocketClient) readPump(ctx context.Context, conn *websocket.Conn) {
	defer func() {
		w.clearConn(conn)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		default:
			_, msg, err := conn.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					log.Printf("websocket read error: %v", err)
					go w.reconnect(conn)
				}
				return
			}

			if string(msg) == "Websocket connection established." {
				continue
			}

			var wsMsg WSMessage
			if err := json.Unmarshal(msg, &wsMsg); err != nil {
				log.Printf("websocket message parse error: %v", err)
				continue
			}

			w.dispatch(wsMsg)
		}
	}
}

func (w *WebsocketClient) pingPump(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(50 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.done:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.writeJSONTo(conn, WsCommand{Method: "ping"}); err != nil {
				log.Printf("ping error: %v", err)
				go w.reconnect(conn)
				return
			}
		}
	}
}

func (w *WebsocketClient) dispatch(msg WSMessage) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	for key, subs := range w.subscriptions {
		if matchSubscription(key, msg) {
			for _, sub := range subs {
				sub.callback(msg)
			}
		}
	}
}

func (w *WebsocketClient) reconnect(failedConn *websocket.Conn) {
	w.reconnectMu.Lock()
	defer w.reconnectMu.Unlock()

	w.clearConn(failedConn)

	for {
		select {
		case <-w.done:
			return
		default:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			err := w.Connect(ctx)
			cancel()
			if err == nil {
				w.reconnectWait = time.Second
				return
			}
			time.Sleep(w.reconnectWait)
			w.reconnectWait *= 2
			if w.reconnectWait > time.Minute {
				w.reconnectWait = time.Minute
			}
		}
	}
}

func (w *WebsocketClient) clearConn(conn *websocket.Conn) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.conn != conn {
		return
	}
	w.conn = nil
	conn.Close()
}

func (w *WebsocketClient) resubscribeAll() error {
	for key, subs := range w.subscriptions {
		if len(subs) > 0 {
			sub := Subscription{
				Type:     key.typ,
				Coin:     key.coin,
				User:     key.user,
				Interval: key.interval,
				Dex:      key.dex,
			}
			if key.hasNSigFigs {
				sub.NSigFigs = &key.nSigFigs
			}
			if key.hasMantissa {
				sub.Mantissa = &key.mantissa
			}
			if key.hasAggregate {
				sub.AggregateByTime = &key.aggregateByTime
			}
			if key.hasPortfolio {
				sub.IsPortfolioMargin = &key.isPortfolioMargin
			}
			if err := w.sendSubscribe(sub); err != nil {
				return fmt.Errorf("resubscribe: %w", err)
			}
		}
	}
	return nil
}

func (w *WebsocketClient) sendSubscribe(sub Subscription) error {
	return w.writeJSON(WsCommand{
		Method:       "subscribe",
		Subscription: &sub,
	})
}

func (w *WebsocketClient) sendUnsubscribe(sub Subscription) error {
	return w.writeJSON(WsCommand{
		Method:       "unsubscribe",
		Subscription: &sub,
	})
}

func (w *WebsocketClient) sendPing() error {
	w.mu.RLock()
	conn := w.conn
	w.mu.RUnlock()

	return w.writeJSONTo(conn, WsCommand{Method: "ping"})
}

func (w *WebsocketClient) writeJSON(v any) error {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()

	if w.conn == nil {
		return fmt.Errorf("connection closed")
	}

	return w.conn.WriteJSON(v)
}

func (w *WebsocketClient) writeJSONTo(conn *websocket.Conn, v any) error {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()

	if conn == nil {
		return fmt.Errorf("connection closed")
	}

	return conn.WriteJSON(v)
}

func (w *WebsocketClient) SubscribeToTrades(coin string, callback func(WSMessage)) (int, error) {
	sub := Subscription{Type: SubTypeTrades, Coin: coin}
	return w.subscribe(sub, callback)
}

func (w *WebsocketClient) SubscribeToOrderbook(coin string, callback func(WSMessage)) (int, error) {
	sub := Subscription{Type: SubTypeL2Book, Coin: coin}
	return w.subscribe(sub, callback)
}

func (w *WebsocketClient) SubscribeToUserFills(user string, callback func(WSMessage)) (int, error) {
	sub := Subscription{Type: SubTypeUserFills, User: user}
	return w.subscribe(sub, callback)
}

func (w *WebsocketClient) SubscribeToOrderUpdates(user string, callback func(WSMessage)) (int, error) {
	sub := Subscription{Type: SubTypeOrderUpdates, User: user}
	return w.subscribe(sub, callback)
}

func (w *WebsocketClient) subscribe(sub Subscription, callback func(WSMessage)) (int, error) {
	if callback == nil {
		return 0, fmt.Errorf("callback cannot be nil")
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	key := sub.key()
	id := int(w.nextSubID.Add(1))

	if w.subscriptions[key] == nil {
		w.subscriptions[key] = make(map[int]*subscriptionCallback)
	}

	w.subscriptions[key][id] = &subscriptionCallback{
		id:       id,
		callback: callback,
	}

	if err := w.sendSubscribe(sub); err != nil {
		delete(w.subscriptions[key], id)
		return 0, fmt.Errorf("subscribe: %w", err)
	}

	return id, nil
}

func matchSubscription(key subKey, msg WSMessage) bool {
	switch key.typ {
	case SubTypeL2Book:
		return msg.Channel == SubTypeL2Book
	case SubTypeTrades:
		return msg.Channel == SubTypeTrades
	case SubTypeUserFills:
		return msg.Channel == SubTypeUserFills
	case SubTypeOrderUpdates:
		return msg.Channel == SubTypeOrderUpdates
	case SubTypeCandle:
		return msg.Channel == SubTypeCandle
	case SubTypeAllMids:
		return msg.Channel == SubTypeAllMids
	case SubTypeNotification:
		return msg.Channel == SubTypeNotification
	case SubTypeWebData2:
		return msg.Channel == SubTypeWebData2
	case SubTypeWebData3:
		return msg.Channel == SubTypeWebData3
	case SubTypeActiveAssetCtx:
		return msg.Channel == SubTypeActiveAssetCtx
	case SubTypeActiveAssetData:
		return msg.Channel == SubTypeActiveAssetData
	case SubTypeUserEvents:
		return msg.Channel == SubTypeUserEvents
	case SubTypeUserFundings:
		return msg.Channel == SubTypeUserFundings
	case SubTypeUserNonFundingLedgerUpdates:
		return msg.Channel == SubTypeUserNonFundingLedgerUpdates
	case SubTypeBBO:
		return msg.Channel == SubTypeBBO
	case SubTypeTwapHistory:
		return msg.Channel == SubTypeTwapHistory
	case SubTypeTwapSliceFills:
		return msg.Channel == SubTypeTwapSliceFills
	case SubTypeTwapStates:
		return msg.Channel == SubTypeTwapStates
	case SubTypeSpotState:
		return msg.Channel == SubTypeSpotState
	default:
		return false
	}
}
