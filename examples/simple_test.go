package examples

import (
	"testing"

	sdk "github.com/funcblock-quant/hyperliquid-go-sdk"
)

func TestSimpleInfo(t *testing.T) {
	// 创建信息客户端
	info, err := sdk.NewInfo(sdk.TestnetAPIURL)
	if err != nil {
		t.Fatalf("Failed to create sdk.Info: %v", err)
	}

	// 获取所有交易对的中间价
	mids, err := info.AllMids()
	if err != nil {
		t.Fatalf("Failed to get all mids: %v", err)
	}

	// 输出一些价格信息
	t.Logf("Total trading pairs: %d", len(mids))

	// 检查是否有 BTC 价格
	if btcPrice, exists := mids["BTC"]; exists {
		t.Logf("BTC price: %s", btcPrice)
	} else {
		t.Log("BTC price not found")
	}

	// 获取永续合约代币列表
	perpCoins := info.PerpCoins()
	t.Logf("Perp coins count: %d", len(perpCoins))
	if len(perpCoins) > 0 {
		t.Logf("First perp coin: %s", perpCoins[0])
	}

	// 获取现货代币列表
	spotCoins := info.SpotCoins()
	t.Logf("Spot coins count: %d", len(spotCoins))
	if len(spotCoins) > 0 {
		t.Logf("First spot coin: %s", spotCoins[0])
	}
}

func TestMetaInfo(t *testing.T) {
	// 创建信息客户端
	info, err := sdk.NewInfo(sdk.TestnetAPIURL)
	if err != nil {
		t.Fatalf("Failed to create sdk.Info: %v", err)
	}

	// 获取元数据
	meta, err := info.Meta()
	if err != nil {
		t.Fatalf("Failed to get meta: %v", err)
	}

	t.Logf("Meta universe count: %d", len(meta.Universe))

	// 输出前几个交易对的信息
	count := 0
	for asset, info := range meta.Universe {
		if count >= 3 {
			break
		}
		t.Logf("Asset %d: %s (szDecimals: %d)", asset, info.Name, info.SzDecimals)
		count++
	}
}
