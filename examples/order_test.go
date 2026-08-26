package examples

import (
	"testing"

	sdk "github.com/funcblock-quant/hyperliquid-go-sdk"
)

func TestOrders(t *testing.T) {
	exchange := getTestExchange(t)

	t.Run("place limit order and cancel then", func(t *testing.T) {
		// place a limit buy order
		coin := "SOL"
		marketPrice := testMarketPrice(t, coin)
		result, err := exchange.Order(
			sdk.OrderRequest{
				Coin:    coin,
				IsBuy:   true,
				Size:    1, // Smaller size for testing
				LimitPx: marketPrice * 0.5,
				OrderType: sdk.OrderType{
					Limit: &sdk.LimitOrderType{
						Tif: sdk.TifGtc,
					},
				},
			},
			nil,
		)
		if err != nil {
			t.Fatalf("Order failed: %v", err)
		}
		//lint:ignore S1034 no reason
		switch result.(type) {
		case error:
			t.Fatalf("Order response failed: %v", result)
		case *sdk.ExchangeRestingOrder:
			order := result.(*sdk.ExchangeRestingOrder)
			t.Logf("Order is placed: %+v", order)

			// cancel the limit order
			result, err = exchange.Cancel(
				sdk.CancelRequest{
					Coin: coin,
					Oid:  order.Oid,
				},
			)
			if err != nil {
				t.Fatalf("Cancel failed: %v", err)
			}
			switch result.(type) {
			case error:
				t.Fatalf("Cancel response failed: %v", result)
			case string:
				t.Logf("Cancel result: %s", result)
			}
		case *sdk.ExchangeFilledOrder: // lint:ignore S1034 no reason
			order := result.(*sdk.ExchangeFilledOrder)
			t.Logf("Order is filled: %+v", order)
		}
	})

	t.Run("open position and close then", func(t *testing.T) {
		// open a long position
		coin := "kPEPE"
		marketPrice := testMarketPrice(t, coin)
		result, err := exchange.MarketOrder(
			sdk.MarketRequest{
				Coin:        coin,
				IsBuy:       true,
				ReduceOnly:  false,
				Size:        14078,
				MarketPrice: marketPrice,
				Slippage:    0.05,
				Cloid:       nil,
			},
			nil,
		)
		if err != nil {
			t.Fatalf("Market order failed: %v", err)
		}
		switch result.(type) {
		case error:
			t.Fatalf("Market order response failed: %v", result)
		case *sdk.ExchangeFilledOrder:
			order := result.(*sdk.ExchangeFilledOrder)
			t.Logf("Market order is filled: %+v", order)

			// close the position
			result, err = exchange.MarketOrder(
				sdk.MarketRequest{
					Coin:        coin,
					IsBuy:       false,
					ReduceOnly:  true,
					Size:        14078,
					MarketPrice: testMarketPrice(t, coin),
					Slippage:    0.20,
					Cloid:       nil,
				},
				nil,
			)
			if err != nil {
				t.Fatalf("Close position failed: %v", err)
			}
			//lint:ignore S1034 no reason
			switch result.(type) {
			case error:
				t.Fatalf("Close position response failed: %v", result)
			case *sdk.ExchangeFilledOrder:
				//lint:ignore no reason
				closeOrder := result.(*sdk.ExchangeFilledOrder)
				t.Logf("Close position is filled: %+v", closeOrder)
			}
		}
	})
}
