package sdk

import (
	"context" // Import the context package
	"encoding/json"
	"fmt"
)

type Info struct {
	client         *Client
	coinToAsset    map[string]int
	assetToDecimal map[int]int
	perpCoins      []string
	spotCoins      []string
}

func NewInfo(apiBaseURL string) (*Info, error) {
	return NewInfoWithPerpDexs(apiBaseURL, nil)
}

func NewInfoWithPerpDexs(apiBaseURL string, perpDexs []string) (*Info, error) {
	info := &Info{
		client:         NewClient(context.Background(), apiBaseURL),
		coinToAsset:    make(map[string]int),
		assetToDecimal: make(map[int]int),
	}

	// Always attempt to fetch meta and spotMeta as skipMeta is effectively false
	// and meta/spotMeta are effectively nil at this point.
	var meta *Meta
	var spotMeta *SpotMeta
	var err error

	spotMeta, err = info.SpotMeta()
	if err != nil {
		return nil, fmt.Errorf("error getting spot meta info: %w", err)
	}

	if len(perpDexs) == 0 {
		perpDexs = []string{""}
	}
	perpDexOffsets := map[string]int{"": 0}
	needsPerpDexOffsets := false
	for _, dex := range perpDexs {
		if dex != "" {
			needsPerpDexOffsets = true
			break
		}
	}
	if needsPerpDexOffsets {
		allDexs, err := info.PerpDexs()
		if err != nil {
			return nil, fmt.Errorf("error getting perp dex list: %w", err)
		}
		if len(allDexs) > 1 {
			for dexIndex, dex := range allDexs[1:] {
				perpDexOffsets[dex.Name] = 110000 + dexIndex*10000
			}
		}
	}

	for _, dex := range perpDexs {
		offset, ok := perpDexOffsets[dex]
		if !ok {
			return nil, fmt.Errorf("perp dex %s not found", dex)
		}
		meta, err = info.MetaForDex(dex)
		if err != nil {
			return nil, fmt.Errorf("error getting meta info for dex %s: %w", dex, err)
		}
		info.addPerpMeta(meta, offset)
	}

	// Map spot assets starting at 10000
	if err := info.addSpotMeta(spotMeta); err != nil {
		return nil, err
	}

	return info, nil
}

func (i *Info) addPerpMeta(meta *Meta, offset int) {
	if meta == nil {
		return
	}
	for asset, assetInfo := range meta.Universe {
		asset += offset
		infoName := assetInfo.Name
		i.coinToAsset[infoName] = asset
		i.assetToDecimal[asset] = assetInfo.SzDecimals
		i.perpCoins = append(i.perpCoins, infoName)
	}
}

func (i *Info) addSpotMeta(spotMeta *SpotMeta) error {
	if spotMeta == nil {
		return nil
	}

	tokenDecimals := make(map[int]int, len(spotMeta.Tokens))
	for _, tokenInfo := range spotMeta.Tokens {
		tokenDecimals[tokenInfo.Index] = tokenInfo.SzDecimals
		i.spotCoins = append(i.spotCoins, tokenInfo.Name)
	}

	for _, spotInfo := range spotMeta.Universe {
		if len(spotInfo.Tokens) == 0 {
			return fmt.Errorf("spot asset %s has no tokens", spotInfo.Name)
		}
		decimal, ok := tokenDecimals[spotInfo.Tokens[0]]
		if !ok {
			return fmt.Errorf("spot asset %s references unknown token index %d", spotInfo.Name, spotInfo.Tokens[0])
		}
		asset := spotInfo.Index + 10000
		i.coinToAsset[spotInfo.Name] = asset
		i.assetToDecimal[asset] = decimal
	}
	return nil
}

func (i *Info) ApiBaseUrl() string {
	return i.client.baseURL
}

func (i *Info) PerpCoins() []string {
	return i.perpCoins
}

func (i *Info) SpotCoins() []string {
	return i.spotCoins
}

func (i *Info) Meta() (*Meta, error) {
	return i.MetaForDex("")
}

func (i *Info) MetaForDex(dex string) (*Meta, error) {
	payload := map[string]any{"type": "meta"}
	if dex != "" {
		payload["dex"] = dex
	}
	resp, err := i.client.post("/info", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch meta: %w", err)
	}

	var meta Meta
	if err := json.Unmarshal(resp, &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal meta response: %w", err)
	}

	return &meta, nil
}

func (i *Info) PerpDexs() ([]PerpDexInfo, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "perpDexs",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch perp dexs: %w", err)
	}

	var result []PerpDexInfo
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal perp dexs: %w", err)
	}
	return result, nil
}

func (i *Info) SpotMeta() (*SpotMeta, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "spotMeta",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch spot meta: %w", err)
	}

	var spotMeta SpotMeta
	if err := json.Unmarshal(resp, &spotMeta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal spot meta response: %w", err)
	}

	return &spotMeta, nil
}

func (i *Info) CoinToAsset(coin string) (int, error) {
	asset, exist := i.coinToAsset[coin]
	if !exist {
		return 0, fmt.Errorf("coin %s not found", coin)
	}
	return asset, nil
}

func (i *Info) AssetToDecimal(asset int) (int, error) {
	decimal, exist := i.assetToDecimal[asset]
	if !exist {
		return 0, fmt.Errorf("asset %d not found", asset)
	}
	return decimal, nil
}

func (i *Info) UserState(address string) (*UserState, error) {
	return i.UserStateForDex(address, "")
}

func (i *Info) UserStateForDex(address string, dex string) (*UserState, error) {
	payload := map[string]any{
		"type": "clearinghouseState",
		"user": address,
	}
	if dex != "" {
		payload["dex"] = dex
	}
	resp, err := i.client.post("/info", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user state: %w", err)
	}

	var result UserState
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user state: %w", err)
	}
	return &result, nil
}

func (i *Info) SpotUserState(address string) (*SpotState, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "spotClearinghouseState",
		"user": address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch spot user state: %w", err)
	}

	var result SpotState
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal spot user state: %w", err)
	}
	return &result, nil
}

func (i *Info) OpenOrders(address string) ([]OpenOrder, error) {
	return i.OpenOrdersForDex(address, "")
}

func (i *Info) OpenOrdersForDex(address string, dex string) ([]OpenOrder, error) {
	payload := map[string]any{
		"type": "openOrders",
		"user": address,
	}
	if dex != "" {
		payload["dex"] = dex
	}
	resp, err := i.client.post("/info", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch open orders: %w", err)
	}

	var result []OpenOrder
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal open orders: %w", err)
	}
	return result, nil
}

func (i *Info) FrontendOpenOrders(address string) ([]FrontendOpenOrder, error) {
	return i.FrontendOpenOrdersForDex(address, "")
}

func (i *Info) FrontendOpenOrdersForDex(address string, dex string) ([]FrontendOpenOrder, error) {
	payload := map[string]any{
		"type": "frontendOpenOrders",
		"user": address,
	}
	if dex != "" {
		payload["dex"] = dex
	}
	resp, err := i.client.post("/info", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch frontend open orders: %w", err)
	}

	var frontendOrders []FrontendOpenOrder
	if err := json.Unmarshal(resp, &frontendOrders); err != nil {
		return nil, fmt.Errorf("failed to unmarshal frontend open orders: %w", err)
	}

	// Convert frontend orders to standard open orders
	return frontendOrders, nil
}

func (i *Info) UserDepositWithdrawTxs(address string, startTime, endTime *int64) ([]DepositWithdrawTx, error) {
	payload := map[string]any{
		"type":      "userNonFundingLedgerUpdates",
		"user":      address,
		"startTime": 0,
	}
	if endTime != nil {
		payload["endTime"] = *endTime
	}
	if startTime != nil {
		payload["startTime"] = *startTime
	}

	resp, err := i.client.post("/info", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user deposit & withdraw txs: %w", err)
	}

	var result []DepositWithdrawTx
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user deposit & withdraw txs: %w", err)
	}
	return result, nil
}

func (i *Info) UserPortfolio(address string) ([]PortFolioTimeRangeItem, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "portfolio",
		"user": address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user portfolio: %w", err)
	}

	var result []PortFolioTimeRangeItem
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user portfolio: %w", err)
	}
	return result, nil
}

func (i *Info) AllMids() (map[string]string, error) {
	return i.AllMidsForDex("")
}

func (i *Info) AllMidsForDex(dex string) (map[string]string, error) {
	payload := map[string]any{"type": "allMids"}
	if dex != "" {
		payload["dex"] = dex
	}
	resp, err := i.client.post("/info", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch all mids: %w", err)
	}

	var result map[string]string
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal all mids: %w", err)
	}
	return result, nil
}

func (i *Info) UserFills(address string) ([]Fill, error) {
	return i.UserFillsForDex(address, "")
}

func (i *Info) UserFillsForDex(address string, dex string) ([]Fill, error) {
	payload := map[string]any{
		"type": "userFills",
		"user": address,
	}
	if dex != "" {
		payload["dex"] = dex
	}
	resp, err := i.client.post("/info", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user fills: %w", err)
	}

	var result []Fill
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user fills: %w", err)
	}
	return result, nil
}

func (i *Info) UserFillsByTime(address string, startTime int64, endTime *int64) ([]Fill, error) {
	return i.UserFillsByTimeForDex(address, startTime, endTime, "", true)
}

func (i *Info) UserFillsByTimeForDex(address string, startTime int64, endTime *int64, dex string, aggregateByTime bool) ([]Fill, error) {
	payload := map[string]any{
		"type":            "userFillsByTime",
		"user":            address,
		"startTime":       startTime,
		"aggregateByTime": aggregateByTime,
	}
	if endTime != nil {
		payload["endTime"] = *endTime
	}
	if dex != "" {
		payload["dex"] = dex
	}

	resp, err := i.client.post("/info", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user fills by time: %w", err)
	}

	var result []Fill
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user fills by time: %w", err)
	}
	return result, nil
}

func (i *Info) MetaAndAssetCtxs() (map[string]any, error) {
	return i.MetaAndAssetCtxsForDex("")
}

func (i *Info) MetaAndAssetCtxsForDex(dex string) (map[string]any, error) {
	payload := map[string]any{"type": "metaAndAssetCtxs"}
	if dex != "" {
		payload["dex"] = dex
	}
	resp, err := i.client.post("/info", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch meta and asset contexts: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal meta and asset contexts: %w", err)
	}
	return result, nil
}

func (i *Info) SpotMetaAndAssetCtxs() (map[string]any, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "spotMetaAndAssetCtxs",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch spot meta and asset contexts: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal spot meta and asset contexts: %w", err)
	}
	return result, nil
}

func (i *Info) FundingHistory(
	coin string,
	startTime int64,
	endTime *int64,
) ([]FundingHistory, error) {

	payload := map[string]any{
		"type":      "fundingHistory",
		"coin":      coin,
		"startTime": startTime,
	}
	if endTime != nil {
		payload["endTime"] = *endTime
	}

	resp, err := i.client.post("/info", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch funding history: %w", err)
	}

	var result []FundingHistory
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal funding history: %w", err)
	}
	return result, nil
}

func (i *Info) UserFundingHistory(
	user string,
	startTime int64,
	endTime *int64,
) ([]UserFundingHistory, error) {
	payload := map[string]any{
		"type":      "userFunding",
		"user":      user,
		"startTime": startTime,
	}
	if endTime != nil {
		payload["endTime"] = *endTime
	}

	resp, err := i.client.post("/info", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user funding history: %w", err)
	}

	var result []UserFundingHistory
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user funding history: %w", err)
	}
	return result, nil
}

func (i *Info) L2Snapshot(coin string) (*L2Book, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "l2Book",
		"coin": coin,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch L2 snapshot: %w", err)
	}

	var result L2Book
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal L2 snapshot: %w", err)
	}
	return &result, nil
}

func (i *Info) CandlesSnapshot(coin, interval string, startTime, endTime int64) ([]Candle, error) {
	req := map[string]any{
		"coin":      coin,
		"interval":  interval,
		"startTime": startTime,
		"endTime":   endTime,
	}

	resp, err := i.client.post("/info", map[string]any{
		"type": "candleSnapshot",
		"req":  req,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch candles snapshot: %w", err)
	}

	var result []Candle
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal candles snapshot: %w", err)
	}
	return result, nil
}

func (i *Info) UserFees(address string) (*UserFees, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "userFees",
		"user": address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user fees: %w", err)
	}

	var result UserFees
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user fees: %w", err)
	}
	return &result, nil
}

func (i *Info) UserStakingSummary(address string) (*StakingSummary, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "delegatorSummary",
		"user": address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch staking summary: %w", err)
	}

	var result StakingSummary
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal staking summary: %w", err)
	}
	return &result, nil
}

func (i *Info) UserStakingDelegations(address string) ([]StakingDelegation, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "delegations",
		"user": address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch staking delegations: %w", err)
	}

	var result []StakingDelegation
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal staking delegations: %w", err)
	}
	return result, nil
}

func (i *Info) UserStakingRewards(address string) ([]StakingReward, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "delegatorRewards",
		"user": address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch staking rewards: %w", err)
	}

	var result []StakingReward
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal staking rewards: %w", err)
	}
	return result, nil
}

func (i *Info) QueryOrderByOid(user string, oid int64) (*OpenOrder, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "orderStatus",
		"user": user,
		"oid":  oid,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order status: %w", err)
	}

	var result OpenOrder
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal order status: %w", err)
	}
	return &result, nil
}

func (i *Info) QueryOrderByCloid(user string, cloid string) (*OpenOrder, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "orderStatus",
		"user": user,
		"oid":  cloid,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order status by cloid: %w", err)
	}

	var result OpenOrder
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal order status: %w", err)
	}
	return &result, nil
}

func (i *Info) QueryReferralState(user string) (*ReferralState, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "referral",
		"user": user,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch referral state: %w", err)
	}

	var result ReferralState
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal referral state: %w", err)
	}
	return &result, nil
}

func (i *Info) QuerySubAccounts(user string) ([]SubAccount, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "subAccounts",
		"user": user,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sub accounts: %w", err)
	}

	var result []SubAccount
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sub accounts: %w", err)
	}
	return result, nil
}

func (i *Info) QueryUserToMultiSigSigners(multiSigUser string) ([]MultiSigSigner, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "userToMultiSigSigners",
		"user": multiSigUser,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch multi-sig signers: %w", err)
	}

	var result []MultiSigSigner
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal multi-sig signers: %w", err)
	}
	return result, nil
}

func (i *Info) ExtraAgents(user string) ([]ExtraAgent, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "extraAgents",
		"user": user,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch extra agents: %w", err)
	}

	var result []ExtraAgent
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal extra agents: %w", err)
	}
	return result, nil
}

func (i *Info) UserRateLimit(user string) (map[string]any, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "userRateLimit",
		"user": user,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user rate limit: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user rate limit: %w", err)
	}
	return result, nil
}

func (i *Info) UserTwapSliceFills(user string) ([]json.RawMessage, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "userTwapSliceFills",
		"user": user,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user twap slice fills: %w", err)
	}

	var result []json.RawMessage
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user twap slice fills: %w", err)
	}
	return result, nil
}

func (i *Info) UserTwapHistory(user string) ([]json.RawMessage, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "userTwapHistory",
		"user": user,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user twap history: %w", err)
	}

	var result []json.RawMessage
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user twap history: %w", err)
	}
	return result, nil
}

func (i *Info) PortfolioMarginUserState(user string) (map[string]any, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "portfolioMarginUserState",
		"user": user,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch portfolio margin user state: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal portfolio margin user state: %w", err)
	}
	return result, nil
}

func (i *Info) PerpsAtOpenInterestCap() ([]string, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "perpsAtOpenInterestCap",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch perps at open interest cap: %w", err)
	}

	var result []string
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal perps at open interest cap: %w", err)
	}
	return result, nil
}

func (i *Info) PredictedFundings() ([]json.RawMessage, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "predictedFundings",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch predicted fundings: %w", err)
	}

	var result []json.RawMessage
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal predicted fundings: %w", err)
	}
	return result, nil
}

func (i *Info) HistoricalOrders(user string) ([]json.RawMessage, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "historicalOrders",
		"user": user,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch historical orders: %w", err)
	}

	var result []json.RawMessage
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal historical orders: %w", err)
	}
	return result, nil
}

func (i *Info) VaultDetails(vaultAddress string, user *string) (map[string]any, error) {
	payload := map[string]any{
		"type":         "vaultDetails",
		"vaultAddress": vaultAddress,
	}
	if user != nil {
		payload["user"] = *user
	}
	resp, err := i.client.post("/info", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch vault details: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal vault details: %w", err)
	}
	return result, nil
}

func (i *Info) UserVaultEquities(user string) ([]json.RawMessage, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "userVaultEquities",
		"user": user,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user vault equities: %w", err)
	}

	var result []json.RawMessage
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user vault equities: %w", err)
	}
	return result, nil
}

func (i *Info) UserRole(user string) (map[string]any, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "userRole",
		"user": user,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user role: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user role: %w", err)
	}
	return result, nil
}

func (i *Info) MaxBuilderFee(user string, builder string) (map[string]any, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type":    "maxBuilderFee",
		"user":    user,
		"builder": builder,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch max builder fee: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal max builder fee: %w", err)
	}
	return result, nil
}
