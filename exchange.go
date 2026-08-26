package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type Exchange struct {
	client         *Client
	vault          *common.Address
	coinToAsset    map[string]int
	assetToDecimal map[int]int
	signer         Signer
	nonce          atomic.Uint64
	expiresAfter   *uint64
	accountAddress *common.Address
}

func NewExchange(baseApiURL string, vaultAddr *common.Address, meta *Meta, signer Signer) *Exchange {
	coinToAsset := make(map[string]int)
	assetToDecimal := make(map[int]int)
	if meta != nil {
		for asset, info := range meta.Universe {
			coinToAsset[info.Name] = asset
			assetToDecimal[asset] = info.SzDecimals
		}
	}
	exchange := Exchange{
		client:         NewClient(context.Background(), baseApiURL),
		vault:          vaultAddr,
		coinToAsset:    coinToAsset,
		assetToDecimal: assetToDecimal,
		signer:         signer,
		nonce:          atomic.Uint64{},
	}
	exchange.nonce.Store(nowTimestamp())
	return &exchange
}

func (e *Exchange) Signer() Signer {
	return e.signer
}

func (e *Exchange) VaultAddress() *common.Address {
	return e.vault
}

func (e *Exchange) SetExpiresAfter(expiresAfter *uint64) {
	e.expiresAfter = expiresAfter
}

func (e *Exchange) SetAccountAddress(accountAddress *common.Address) {
	e.accountAddress = accountAddress
}

func (e *Exchange) Order(req OrderRequest, builder *BuilderInfo) (any, error) {
	orders, err := e.BulkOrders([]OrderRequest{req}, builder)
	if err != nil {
		return nil, err
	}
	if len(orders) != 1 {
		return nil, fmt.Errorf("expected 1 order, got %d", len(orders))
	}
	return orders[0], nil
}

func (e *Exchange) MarketOrder(req MarketRequest, builder *BuilderInfo) (any, error) {
	orders, err := e.BulkMarketOrders([]MarketRequest{req}, builder)
	if err != nil {
		return nil, err
	}
	if len(orders) != 1 {
		return nil, fmt.Errorf("expected 1 order, got %d", len(orders))
	}
	return orders[0], nil
}

func (e *Exchange) BulkMarketOrders(req []MarketRequest, builder *BuilderInfo) ([]any, error) {
	orderReqs := make([]OrderRequest, len(req))
	for i, r := range req {
		// Get slippage price
		price := e.slippagePrice(r.IsBuy, r.Slippage, r.MarketPrice)
		orderReqs[i] = OrderRequest{
			Coin:    r.Coin,
			IsBuy:   r.IsBuy,
			Size:    r.Size,
			LimitPx: price,
			OrderType: OrderType{
				Limit: &LimitOrderType{Tif: TifIoc},
			},
			ReduceOnly: r.ReduceOnly,
			Cloid:      r.Cloid,
		}
	}
	return e.BulkOrders(orderReqs, builder)
}

func (e *Exchange) MarketOpen(coin string, isBuy bool, size float64, marketPrice float64, slippage float64, cloid *string, builder *BuilderInfo) (any, error) {
	return e.MarketOrder(MarketRequest{
		Coin:        coin,
		IsBuy:       isBuy,
		ReduceOnly:  false,
		Size:        size,
		MarketPrice: marketPrice,
		Slippage:    slippage,
		Cloid:       cloid,
	}, builder)
}

func (e *Exchange) MarketClose(coin string, size *float64, marketPrice float64, slippage float64, cloid *string, builder *BuilderInfo) (any, error) {
	address := e.signer.Address()
	if e.accountAddress != nil {
		address = *e.accountAddress
	} else if e.vault != nil {
		address = *e.vault
	}
	userState, err := e.userState(address.Hex())
	if err != nil {
		return nil, err
	}

	closeSize := 0.0
	isBuy := false
	for _, assetPosition := range userState.AssetPositions {
		if assetPosition.Position.Coin != coin {
			continue
		}
		szi, err := strconv.ParseFloat(assetPosition.Position.Szi, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse position size: %w", err)
		}
		if szi == 0 {
			return nil, fmt.Errorf("no open position for %s", coin)
		}
		if szi < 0 {
			isBuy = true
			closeSize = -szi
		} else {
			closeSize = szi
		}
		break
	}
	if closeSize == 0 {
		return nil, fmt.Errorf("no open position for %s", coin)
	}
	if size != nil {
		closeSize = *size
	}

	return e.MarketOrder(MarketRequest{
		Coin:        coin,
		IsBuy:       isBuy,
		ReduceOnly:  true,
		Size:        closeSize,
		MarketPrice: marketPrice,
		Slippage:    slippage,
		Cloid:       cloid,
	}, builder)
}

func (e *Exchange) Cancel(req CancelRequest) (any, error) {
	statuses, err := e.BulkCancel([]CancelRequest{req})
	if err != nil {
		return nil, err
	}
	if len(statuses) != 1 {
		return nil, fmt.Errorf("expected 1 status, got %d", len(statuses))
	}
	return statuses[0], nil
}

func (e *Exchange) CancelByCloid(req CancelByCloidRequest) (any, error) {
	statuses, err := e.BulkCancelByCloid([]CancelByCloidRequest{req})
	if err != nil {
		return nil, err
	}
	if len(statuses) != 1 {
		return nil, fmt.Errorf("expected 1 status, got %d", len(statuses))
	}
	return statuses[0], nil
}

func (e *Exchange) ModifyOrder(request ModifyRequest) (any, error) {
	statuses, err := e.BulkModifyOrders([]ModifyRequest{request})
	if err != nil {
		return nil, err
	}
	if len(statuses) != 1 {
		return nil, fmt.Errorf("expected 1 status, got %d", len(statuses))
	}
	return statuses[0], nil
}

func (e *Exchange) BulkOrders(orders []OrderRequest, builder *BuilderInfo) ([]any, error) {
	return e.BulkOrdersWithGrouping(orders, builder, GroupingNa)
}

func (e *Exchange) BulkOrdersWithGrouping(orders []OrderRequest, builder *BuilderInfo, grouping string) ([]any, error) {
	nonce := e.NextNonce()

	orderWires := make([]OrderWire, len(orders))
	for i, order := range orders {
		asset, exist := e.coinToAsset[order.Coin]
		if !exist {
			return nil, fmt.Errorf("coin %s does not exist", order.Coin)
		}
		wire := order.ToWire(asset, e.assetToDecimal[asset])
		orderWires[i] = wire
	}

	action := &OrderAction{
		Type:     "order",
		Orders:   orderWires,
		Grouping: grouping,
		Builder:  builder,
	}

	sig, err := e.signL1Action(action, nonce)
	if err != nil {
		return nil, err
	}

	_, statuses, err := e.PostActionAndParseResponse(action, sig, nonce)
	return statuses, err
}

func (e *Exchange) BulkModifyOrders(request []ModifyRequest) ([]any, error) {
	nonce := e.NextNonce()

	modifyWires := make([]ModifyWire, len(request))
	for i, req := range request {
		asset, exist := e.coinToAsset[req.OrderRequest.Coin]
		if !exist {
			return nil, fmt.Errorf("coin %s does not exist", req.OrderRequest.Coin)
		}
		// to wire
		modifyWires[i] = req.ToWire(asset, e.assetToDecimal[asset])
	}

	action := &ModifyAction{
		Type:     "batchModify",
		Modifies: modifyWires,
	}

	sig, err := e.signL1Action(action, nonce)
	if err != nil {
		return nil, err
	}

	_, statuses, err := e.PostActionAndParseResponse(action, sig, nonce)
	return statuses, err
}

func (e *Exchange) BulkCancel(request []CancelRequest) ([]any, error) {
	nonce := e.NextNonce()

	cancelWires := make([]CancelWire, len(request))
	for i, req := range request {
		asset, exist := e.coinToAsset[req.Coin]
		if !exist {
			return nil, fmt.Errorf("coin %s does not exist", req.Coin)
		}
		cancelWires[i] = req.ToWire(asset)
	}

	action := &CancelAction{
		Type:    "cancel",
		Cancels: cancelWires,
	}

	sig, err := e.signL1Action(action, nonce)
	if err != nil {
		return nil, err
	}

	_, statuses, err := e.PostActionAndParseResponse(action, sig, nonce)
	return statuses, err
}

func (e *Exchange) BulkCancelByCloid(request []CancelByCloidRequest) ([]any, error) {
	nonce := e.NextNonce()

	cancelWires := make([]CancelByCloidWire, len(request))
	for i, req := range request {
		asset, exist := e.coinToAsset[req.Coin]
		if !exist {
			return nil, fmt.Errorf("coin %s does not exist", req.Coin)
		}
		cancelWires[i] = req.ToWire(asset)
	}

	action := &CancelByCloidAction{
		Type:    "cancelByCloid",
		Cancels: cancelWires,
	}

	sig, err := e.signL1Action(action, nonce)
	if err != nil {
		return nil, err
	}

	_, statuses, err := e.PostActionAndParseResponse(action, sig, nonce)
	return statuses, err
}

func (e *Exchange) UpdateLeverage(coin string, isCross bool, leverage int) error {
	nonce := e.NextNonce()

	asset, exist := e.coinToAsset[coin]
	if !exist {
		return fmt.Errorf("coin %s does not exist", coin)
	}
	action := &UpdateLeverageAction{
		Type:     "updateLeverage",
		Asset:    asset,
		IsCross:  isCross,
		Leverage: leverage,
	}

	sig, err := e.signL1Action(action, nonce)
	if err != nil {
		return err
	}

	_, _, err = e.PostActionAndParseResponse(action, sig, nonce)
	return err
}

func (e *Exchange) UpdateIsolatedMargin(coin string, amount float64) error {
	nonce := e.NextNonce()

	amountInt := FloatToUsdInt(amount)
	asset, exist := e.coinToAsset[coin]
	if !exist {
		return fmt.Errorf("coin %s does not exist", coin)
	}
	action := &UpdateIsolatedMarginAction{
		Type:   "updateIsolatedMargin",
		Asset:  asset,
		IsBuy:  true,
		Amount: amountInt,
	}

	sig, err := e.signL1Action(action, nonce)
	if err != nil {
		return err
	}

	_, _, err = e.PostActionAndParseResponse(action, sig, nonce)
	return err
}

func (e *Exchange) TopUpIsolatedOnlyMargin(coin string, leverage string) error {
	nonce := e.NextNonce()

	asset, exist := e.coinToAsset[coin]
	if !exist {
		return fmt.Errorf("coin %s does not exist", coin)
	}
	action := &TopUpIsolatedOnlyMarginAction{
		Type:     "topUpIsolatedOnlyMargin",
		Asset:    asset,
		Leverage: leverage,
	}

	sig, err := e.signL1Action(action, nonce)
	if err != nil {
		return err
	}

	_, _, err = e.PostActionAndParseResponse(action, sig, nonce)
	return err
}

func (e *Exchange) ScheduleCancel(cancelTime *uint64) error {
	nonce := e.NextNonce()

	action := &ScheduleCancelAction{
		Type: "scheduleCancel",
		Time: cancelTime,
	}

	sig, err := e.signL1Action(action, nonce)
	if err != nil {
		return err
	}

	_, _, err = e.PostActionAndParseResponse(action, sig, nonce)
	return err
}

func (e *Exchange) TwapOrder(req TwapRequest) (any, error) {
	nonce := e.NextNonce()

	asset, exist := e.coinToAsset[req.Coin]
	if !exist {
		return nil, fmt.Errorf("coin %s does not exist", req.Coin)
	}
	action := &TwapOrderAction{
		Type: "twapOrder",
		Twap: TwapWire{
			Asset:      asset,
			IsBuy:      req.IsBuy,
			Size:       FloatToString(RoundToDecimal(req.Size, e.assetToDecimal[asset])),
			ReduceOnly: req.ReduceOnly,
			Minutes:    req.Minutes,
			Randomize:  req.Randomize,
		},
	}

	sig, err := e.signL1Action(action, nonce)
	if err != nil {
		return nil, err
	}
	return e.postActionAndParseRaw(action, sig, nonce)
}

func (e *Exchange) TwapCancel(coin string, twapID int64) (any, error) {
	nonce := e.NextNonce()

	asset, exist := e.coinToAsset[coin]
	if !exist {
		return nil, fmt.Errorf("coin %s does not exist", coin)
	}
	action := &TwapCancelAction{
		Type:   "twapCancel",
		Asset:  asset,
		TwapID: twapID,
	}

	sig, err := e.signL1Action(action, nonce)
	if err != nil {
		return nil, err
	}
	return e.postActionAndParseRaw(action, sig, nonce)
}

func (e *Exchange) SetReferrer(code string) (any, error) {
	nonce := e.NextNonce()
	action := &SetReferrerAction{
		Type: "setReferrer",
		Code: code,
	}

	sig, err := e.signL1Action(action, nonce)
	if err != nil {
		return nil, err
	}
	return e.postActionAndParseRaw(action, sig, nonce)
}

func (e *Exchange) CreateSubAccount(name string) (any, error) {
	nonce := e.NextNonce()
	action := &CreateSubAccountAction{
		Type: "createSubAccount",
		Name: name,
	}

	sig, err := e.signL1Action(action, nonce)
	if err != nil {
		return nil, err
	}
	return e.postActionAndParseRaw(action, sig, nonce)
}

func (e *Exchange) AgentEnableDexAbstraction() (any, error) {
	nonce := e.NextNonce()
	action := &AgentEnableDexAbstractionAction{Type: "agentEnableDexAbstraction"}
	sig, err := e.signL1Action(action, nonce)
	if err != nil {
		return nil, err
	}
	return e.postActionAndParseRaw(action, sig, nonce)
}

func (e *Exchange) AgentSetAbstraction(abstraction string) (any, error) {
	nonce := e.NextNonce()
	action := &AgentSetAbstractionAction{
		Type:        "agentSetAbstraction",
		Abstraction: abstraction,
	}
	sig, err := e.signL1Action(action, nonce)
	if err != nil {
		return nil, err
	}
	return e.postActionAndParseRaw(action, sig, nonce)
}

func (e *Exchange) UseBigBlocks(enable bool) (any, error) {
	nonce := e.NextNonce()
	action := &EvmUserModifyAction{
		Type:           "evmUserModify",
		UsingBigBlocks: enable,
	}
	sig, err := e.signL1Action(action, nonce)
	if err != nil {
		return nil, err
	}
	return e.postActionAndParseRaw(action, sig, nonce)
}

func (e *Exchange) VaultUsdTransfer(isDeposit bool, vaultAddress string, amount int) (*ExchangeRequest, error) {
	nonce := e.NextNonce()

	action := &VaultUsdTransferAction{
		Type:         "vaultTransfer",
		VaultAddress: vaultAddress,
		IsDeposit:    isDeposit,
		Usd:          amount,
	}

	sig, err := e.signL1Action(action, nonce)
	if err != nil {
		return nil, err
	}

	return &ExchangeRequest{
		Action:    action,
		Nonce:     nonce,
		Signature: sig,
	}, nil
}

func (e *Exchange) signL1Action(action any, nonce uint64) (*Signature, error) {
	return SignL1ActionWithExpiresAfter(
		e.signer,
		action,
		e.vault,
		nonce,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
}

func (e *Exchange) slippagePrice(isBuy bool, slippage float64, price float64) float64 {
	if isBuy {
		price *= 1 + slippage
	} else {
		price *= 1 - slippage
	}
	return price
}

func (e *Exchange) PostActionAndParseResponse(action Action, signature *Signature, nonce uint64) (string, []any, error) {
	payload := ExchangeRequest{
		Action:       action,
		Nonce:        nonce,
		Signature:    signature,
		ExpiresAfter: e.expiresAfter,
	}
	if action.Tp() != "usdClassTransfer" && action.Tp() != "sendAsset" {
		payload.VaultAddress = e.vault
	}
	response, err := e.client.post("/exchange", payload)
	if err != nil {
		return "", nil, err
	}
	respStatus := new(ExchangeResponsesStatus)
	if err = json.Unmarshal(response, respStatus); err != nil {
		return "", nil, err
	}
	respInner, err := respStatus.Parse()
	if err != nil {
		return "", nil, err
	}
	if respInner.Data == nil {
		return respInner.Type, nil, nil
	}
	statuses := make([]any, len(respInner.Data.Statuses))
	for i, status := range respInner.Data.Statuses {
		statuses[i] = status.Parse()
	}
	return respInner.Type, statuses, nil
}

func (e *Exchange) postActionAndParseRaw(action Action, signature *Signature, nonce uint64) (any, error) {
	payload := ExchangeRequest{
		Action:       action,
		Nonce:        nonce,
		Signature:    signature,
		ExpiresAfter: e.expiresAfter,
	}
	if action.Tp() != "usdClassTransfer" && action.Tp() != "sendAsset" {
		payload.VaultAddress = e.vault
	}
	response, err := e.client.post("/exchange", payload)
	if err != nil {
		return nil, err
	}
	var result any
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (e *Exchange) userState(address string) (*UserState, error) {
	resp, err := e.client.post("/info", map[string]any{
		"type": "clearinghouseState",
		"user": address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user state: %w", err)
	}

	var result UserState
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user state: %w", err)
	}
	return &result, nil
}

func (e *Exchange) NextNonce() uint64 {
	nonce := e.nonce.Add(1)
	now := nowTimestamp()
	// more than 300 seconds behind
	if nonce+300_000 < now {
		e.nonce.Swap(now)
	}
	return nonce
}

func nowTimestamp() uint64 {
	return uint64(time.Now().UnixMilli())
}
