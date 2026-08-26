package examples

import (
	"strings"

	ex "github.com/funcblock-quant/hyperliquid-go-sdk/exchange_api"
	"testing"
)

func TestTransferUSD(t *testing.T) {
	exchange := getTestExchange(t)

	req := ex.TransferUSDRequest{
		Destination:      getTestAddress(t),
		Amount:           "2",
		HyperliquidChain: "Testnet",
		SignatureChainId: "0x66eee",
	}

	res, err := ex.TansferUSD(exchange, req)
	if err != nil {
		if strings.Contains(err.Error(), "Action disabled when unified account is active") {
			t.Logf("Transfer USDC reached testnet exchange and is disabled for unified account mode: %v", err)
			return
		}
		t.Fatalf("Transfer USDC failed: %v", err)
	}

	t.Logf("Transfer USDC completed successfully: %+v", res)
}
