package sdk

import "testing"

func TestAddSpotMetaUsesTokenIndexNotSlicePosition(t *testing.T) {
	info := &Info{
		coinToAsset:    make(map[string]int),
		assetToDecimal: make(map[int]int),
	}
	spotMeta := &SpotMeta{
		Universe: []SpotAssetInfo{
			{Name: "@1", Tokens: []int{1657, 0}, Index: 12},
		},
		Tokens: []SpotTokenInfo{
			{Name: "USDC", Index: 0, SzDecimals: 6},
			{Name: "TEST", Index: 1657, SzDecimals: 4},
		},
	}

	if err := info.addSpotMeta(spotMeta); err != nil {
		t.Fatal(err)
	}
	if got, want := info.coinToAsset["@1"], 10012; got != want {
		t.Fatalf("asset mismatch: got %d want %d", got, want)
	}
	if got, want := info.assetToDecimal[10012], 4; got != want {
		t.Fatalf("decimal mismatch: got %d want %d", got, want)
	}
}
