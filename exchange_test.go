package sdk

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

type capturedExchangeRequest struct {
	Action       json.RawMessage `json:"action"`
	Nonce        uint64          `json:"nonce"`
	Signature    *Signature      `json:"signature"`
	VaultAddress *common.Address `json:"vaultAddress,omitempty"`
}

type captureTransport struct {
	t        *testing.T
	requests []capturedExchangeRequest
}

func (c *captureTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c.t.Helper()
	if r.URL.Path != "/exchange" {
		c.t.Fatalf("unexpected path: %s", r.URL.Path)
	}
	var req capturedExchangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.t.Fatalf("decode exchange request: %v", err)
	}
	c.requests = append(c.requests, req)
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewBufferString(`{"status":"ok","response":{"type":"default","data":{"statuses":["success"]}}}`)),
	}, nil
}

func newTestExchangeWithRecorder(t *testing.T) (*Exchange, *[]capturedExchangeRequest, func()) {
	t.Helper()

	signer, err := NewLocalSignerFromHex("0123456789012345678901234567890123456789012345678901234567890123")
	if err != nil {
		t.Fatal(err)
	}
	meta := &Meta{Universe: []AssetInfo{
		{Name: "BTC", SzDecimals: 5},
		{Name: "ETH", SzDecimals: 4},
	}}
	exchange := NewExchange(TestnetAPIURL, nil, meta, signer)
	transport := &captureTransport{t: t}
	exchange.client.httpClient = &http.Client{Transport: transport}
	return exchange, &transport.requests, func() {}
}

func decodeCapturedAction(t *testing.T, req capturedExchangeRequest) map[string]any {
	t.Helper()
	var action map[string]any
	if err := json.Unmarshal(req.Action, &action); err != nil {
		t.Fatalf("decode action: %v", err)
	}
	return action
}

func assertLastActionType(t *testing.T, requests *[]capturedExchangeRequest, want string) map[string]any {
	t.Helper()
	if len(*requests) == 0 {
		t.Fatal("expected at least one exchange request")
	}
	req := (*requests)[len(*requests)-1]
	if req.Signature == nil {
		t.Fatal("expected request signature")
	}
	action := decodeCapturedAction(t, req)
	if got := action["type"]; got != want {
		t.Fatalf("action type mismatch: got %v want %s", got, want)
	}
	return action
}

func TestExchangePostingActionsUseCapturedHTTPPayloads(t *testing.T) {
	exchange, requests, cleanup := newTestExchangeWithRecorder(t)
	defer cleanup()

	if _, err := exchange.Order(OrderRequest{
		Coin:       "BTC",
		IsBuy:      true,
		Size:       0.00123456,
		LimitPx:    42123.456,
		OrderType:  OrderType{Limit: &LimitOrderType{Tif: TifGtc}},
		ReduceOnly: false,
	}, nil); err != nil {
		t.Fatalf("Order failed: %v", err)
	}
	action := assertLastActionType(t, requests, "order")
	orders := action["orders"].([]any)
	firstOrder := orders[0].(map[string]any)
	if got := firstOrder["a"]; got != float64(0) {
		t.Fatalf("asset mismatch: got %v want 0", got)
	}

	if _, err := exchange.Cancel(CancelRequest{Coin: "ETH", Oid: 123}); err != nil {
		t.Fatalf("Cancel failed: %v", err)
	}
	assertLastActionType(t, requests, "cancel")

	if err := exchange.UpdateLeverage("BTC", true, 5); err != nil {
		t.Fatalf("UpdateLeverage failed: %v", err)
	}
	action = assertLastActionType(t, requests, "updateLeverage")
	if got := action["isCross"]; got != true {
		t.Fatalf("isCross mismatch: got %v want true", got)
	}

	if _, err := exchange.TwapOrder(TwapRequest{
		Coin:    "ETH",
		IsBuy:   false,
		Size:    0.25,
		Minutes: 10,
	}); err != nil {
		t.Fatalf("TwapOrder failed: %v", err)
	}
	assertLastActionType(t, requests, "twapOrder")
}

func TestUserSignedActionsAreBuiltForTestnet(t *testing.T) {
	exchange, requests, cleanup := newTestExchangeWithRecorder(t)
	defer cleanup()

	if _, err := exchange.USDSend("0xABCDEFabcdefABCDEFabcdefABCDEFabcdefABCD", "1.23"); err != nil {
		t.Fatalf("USDSend failed: %v", err)
	}
	action := assertLastActionType(t, requests, "usdSend")
	if got := action["hyperliquidChain"]; got != "Testnet" {
		t.Fatalf("hyperliquidChain mismatch: got %v want Testnet", got)
	}
	if got := action["signatureChainId"]; got != UserSignedSignatureChainID {
		t.Fatalf("signatureChainId mismatch: got %v want %s", got, UserSignedSignatureChainID)
	}
	if got := action["destination"]; got != "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd" {
		t.Fatalf("destination was not lower-cased: %v", got)
	}

	if _, err := exchange.USDClassTransfer("2.5", true); err != nil {
		t.Fatalf("USDClassTransfer failed: %v", err)
	}
	action = assertLastActionType(t, requests, "usdClassTransfer")
	if got := action["toPerp"]; got != true {
		t.Fatalf("toPerp mismatch: got %v want true", got)
	}
	if (*requests)[len(*requests)-1].VaultAddress != nil {
		t.Fatal("usdClassTransfer should not include vaultAddress")
	}

	if _, err := exchange.SendAsset(
		"0xABCDEFabcdefABCDEFabcdefABCDEFabcdefABCD",
		"",
		"testDex",
		"USDC",
		"4.5",
		"0x1111111111111111111111111111111111111111",
	); err != nil {
		t.Fatalf("SendAsset failed: %v", err)
	}
	action = assertLastActionType(t, requests, "sendAsset")
	if got := action["hyperliquidChain"]; got != "Testnet" {
		t.Fatalf("hyperliquidChain mismatch: got %v want Testnet", got)
	}
	if (*requests)[len(*requests)-1].VaultAddress != nil {
		t.Fatal("sendAsset should not include vaultAddress")
	}
}

func TestCanonicalMultiSigSignersLowercasesAndSorts(t *testing.T) {
	got, err := CanonicalMultiSigSigners([]MultiSigSigner{
		{User: "0xBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB", Threshold: 2},
		{User: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Threshold: 1},
	})
	if err != nil {
		t.Fatal(err)
	}

	want := `[{"user":"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","threshold":1},{"user":"0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","threshold":2}]`
	if got != want {
		t.Fatalf("canonical signers mismatch:\ngot  %s\nwant %s", got, want)
	}
}
