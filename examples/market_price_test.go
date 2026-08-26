package examples

import (
	"strconv"
	"testing"

	sdk "github.com/funcblock-quant/hyperliquid-go-sdk"
)

func testMarketPrice(t *testing.T, coin string) float64 {
	t.Helper()
	info, err := sdk.NewInfo(sdk.TestnetAPIURL)
	if err != nil {
		t.Fatalf("Failed to create sdk.Info for market price: %v", err)
	}
	mids, err := info.AllMids()
	if err != nil {
		t.Fatalf("Failed to fetch all mids: %v", err)
	}
	mid, ok := mids[coin]
	if !ok {
		t.Fatalf("Missing mid price for %s", coin)
	}
	price, err := strconv.ParseFloat(mid, 64)
	if err != nil {
		t.Fatalf("Failed to parse mid price for %s: %v", coin, err)
	}
	return price
}

func testCoinWithoutOpenPosition(t *testing.T, candidates ...string) string {
	t.Helper()
	info, err := sdk.NewInfo(sdk.TestnetAPIURL)
	if err != nil {
		t.Fatalf("Failed to create sdk.Info for position lookup: %v", err)
	}
	userState, err := info.UserState(getTestAddress(t))
	if err != nil {
		t.Fatalf("Failed to fetch user state: %v", err)
	}

	openPositions := make(map[string]bool)
	for _, assetPosition := range userState.AssetPositions {
		szi, err := strconv.ParseFloat(assetPosition.Position.Szi, 64)
		if err != nil {
			t.Fatalf("Failed to parse position size for %s: %v", assetPosition.Position.Coin, err)
		}
		if szi != 0 {
			openPositions[assetPosition.Position.Coin] = true
		}
	}

	for _, coin := range candidates {
		if !openPositions[coin] {
			return coin
		}
	}
	t.Fatalf("No candidate coin without open position: %v", candidates)
	return ""
}
