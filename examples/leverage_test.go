package examples

import (
	"testing"

	sdk "github.com/funcblock-quant/hyperliquid-go-sdk"
)

func TestUpdateLeverage(t *testing.T) {
	exchange := getTestExchange(t)
	coin := testCoinWithoutOpenPosition(t, "kPEPE", "BTC", "SOL", "ETH")
	leverage := 10

	t.Run("isolated margin", func(t *testing.T) {
		err := exchange.UpdateLeverage(coin, false, leverage)
		if err != nil {
			t.Fatalf("Failed to update leverage: %v", err)
		}
		t.Log("Update leverage successfully")
	})

	t.Run("cross margin", func(t *testing.T) {
		err := exchange.UpdateLeverage(coin, true, leverage)
		if err != nil {
			t.Fatalf("Failed to update leverage: %v", err)
		}
		t.Log("Update leverage successfully")
	})
}

func TestUpdateIsolatedMargin(t *testing.T) {
	exchange := getTestExchange(t)

	amount := 1.0
	coin := "kPEPE"
	size := 14078.0

	if err := exchange.UpdateLeverage(coin, false, 10); err != nil {
		t.Fatalf("Failed to set isolated leverage: %v", err)
	}

	openResult, err := exchange.MarketOrder(
		sdk.MarketRequest{
			Coin:        coin,
			IsBuy:       true,
			ReduceOnly:  false,
			Size:        size,
			MarketPrice: testMarketPrice(t, coin),
			Slippage:    0.20,
		},
		nil,
	)
	if err != nil {
		t.Fatalf("Failed to open isolated test position: %v", err)
	}
	if errResult, ok := openResult.(error); ok {
		t.Fatalf("Open isolated test position response failed: %v", errResult)
	}

	err = exchange.UpdateIsolatedMargin(coin, amount)
	if err != nil {
		t.Fatalf("Failed to update isolated margin: %v", err)
	}

	closeResult, err := exchange.MarketOrder(
		sdk.MarketRequest{
			Coin:        coin,
			IsBuy:       false,
			ReduceOnly:  true,
			Size:        size,
			MarketPrice: testMarketPrice(t, coin),
			Slippage:    0.20,
		},
		nil,
	)
	if err != nil {
		t.Fatalf("Failed to close isolated test position: %v", err)
	}
	if errResult, ok := closeResult.(error); ok {
		t.Fatalf("Close isolated test position response failed: %v", errResult)
	}
	t.Log("Update isolated margin successfully")
}
