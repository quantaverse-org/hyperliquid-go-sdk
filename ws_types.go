package sdk

import (
	"encoding/json"
)

type WSMessage struct {
	Channel string          `json:"channel"`
	Data    json.RawMessage `json:"data"`
}

type Subscription struct {
	Type              string `json:"type"`
	Coin              string `json:"coin,omitempty"`
	User              string `json:"user,omitempty"`
	Interval          string `json:"interval,omitempty"`
	Dex               string `json:"dex,omitempty"`
	NSigFigs          *int   `json:"nSigFigs,omitempty"`
	Mantissa          *int   `json:"mantissa,omitempty"`
	AggregateByTime   *bool  `json:"aggregateByTime,omitempty"`
	IsPortfolioMargin *bool  `json:"isPortfolioMargin,omitempty"`
}

type subKey struct {
	typ               string
	coin              string
	user              string
	interval          string
	dex               string
	nSigFigs          int
	mantissa          int
	aggregateByTime   bool
	isPortfolioMargin bool
	hasNSigFigs       bool
	hasMantissa       bool
	hasAggregate      bool
	hasPortfolio      bool
}

func (s Subscription) key() subKey {
	key := subKey{
		typ:      s.Type,
		coin:     s.Coin,
		user:     s.User,
		interval: s.Interval,
		dex:      s.Dex,
	}
	if s.NSigFigs != nil {
		key.nSigFigs = *s.NSigFigs
		key.hasNSigFigs = true
	}
	if s.Mantissa != nil {
		key.mantissa = *s.Mantissa
		key.hasMantissa = true
	}
	if s.AggregateByTime != nil {
		key.aggregateByTime = *s.AggregateByTime
		key.hasAggregate = true
	}
	if s.IsPortfolioMargin != nil {
		key.isPortfolioMargin = *s.IsPortfolioMargin
		key.hasPortfolio = true
	}
	return key
}

type WsCommand struct {
	Method       string        `json:"method"`
	Subscription *Subscription `json:"subscription,omitempty"`
}

type subscriptionCallback struct {
	id       int
	callback func(WSMessage)
}

// WsUserFills represents user fills data from WebSocket
type WsUserFills struct {
	IsSnapshot *bool    `json:"isSnapshot,omitempty"`
	User       string   `json:"user"`
	Fills      []WsFill `json:"fills"`
}

// WsFill represents a single fill from WebSocket
type WsFill struct {
	Coin          string           `json:"coin"`
	Px            string           `json:"px"` // price
	Sz            string           `json:"sz"` // size
	Side          string           `json:"side"`
	Time          int64            `json:"time"`
	StartPosition string           `json:"startPosition"`
	Dir           string           `json:"dir"` // used for frontend display
	ClosedPnl     string           `json:"closedPnl"`
	Hash          string           `json:"hash"`    // L1 transaction hash
	Oid           int64            `json:"oid"`     // order id
	Crossed       bool             `json:"crossed"` // whether order crossed the spread (was taker)
	Fee           string           `json:"fee"`     // negative means rebate
	Tid           int64            `json:"tid"`     // unique trade id
	Liquidation   *FillLiquidation `json:"liquidation,omitempty"`
	FeeToken      string           `json:"feeToken"`             // the token the fee was paid in
	BuilderFee    *string          `json:"builderFee,omitempty"` // amount paid to builder, also included in fee
}

// FillLiquidation represents liquidation information for a fill
type FillLiquidation struct {
	LiquidatedUser *string `json:"liquidatedUser,omitempty"`
	MarkPx         float64 `json:"markPx"`
	Method         string  `json:"method"` // Possible values: "market" or "backstop"
}

// WsOrder represents an order from WebSocket
type WsOrder struct {
	Order           WsBasicOrder `json:"order"`
	Status          string       `json:"status"` // Possible values: open, filled, canceled, triggered, rejected, marginCanceled
	StatusTimestamp int64        `json:"statusTimestamp"`
}

// WsBasicOrder represents basic order information
type WsBasicOrder struct {
	Coin      string  `json:"coin"`
	Side      string  `json:"side"`
	LimitPx   string  `json:"limitPx"`
	Sz        string  `json:"sz"`
	Oid       int64   `json:"oid"`
	Timestamp int64   `json:"timestamp"`
	OrigSz    string  `json:"origSz"`
	Cloid     *string `json:"cloid,omitempty"`
}
