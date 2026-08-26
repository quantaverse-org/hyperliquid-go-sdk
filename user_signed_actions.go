package sdk

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

const UserSignedSignatureChainID = "0x66eee"

var (
	usdSendSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "destination", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "time", Type: "uint64"},
	}
	spotSendSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "destination", Type: "string"},
		{Name: "token", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "time", Type: "uint64"},
	}
	withdrawSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "destination", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "time", Type: "uint64"},
	}
	usdClassTransferSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "toPerp", Type: "bool"},
		{Name: "nonce", Type: "uint64"},
	}
	sendAssetSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "destination", Type: "string"},
		{Name: "sourceDex", Type: "string"},
		{Name: "destinationDex", Type: "string"},
		{Name: "token", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "fromSubAccount", Type: "string"},
		{Name: "nonce", Type: "uint64"},
	}
	tokenDelegateSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "validator", Type: "address"},
		{Name: "wei", Type: "uint64"},
		{Name: "isUndelegate", Type: "bool"},
		{Name: "nonce", Type: "uint64"},
	}
	approveAgentSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "agentAddress", Type: "address"},
		{Name: "agentName", Type: "string"},
		{Name: "nonce", Type: "uint64"},
	}
	approveBuilderFeeSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "maxFeeRate", Type: "string"},
		{Name: "builder", Type: "address"},
		{Name: "nonce", Type: "uint64"},
	}
	convertToMultiSigUserSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "signers", Type: "string"},
		{Name: "nonce", Type: "uint64"},
	}
	userDexAbstractionSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "user", Type: "address"},
		{Name: "enabled", Type: "bool"},
		{Name: "nonce", Type: "uint64"},
	}
	userSetAbstractionSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "user", Type: "address"},
		{Name: "abstraction", Type: "string"},
		{Name: "nonce", Type: "uint64"},
	}
	multiSigEnvelopeSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "multiSigActionHash", Type: "bytes32"},
		{Name: "nonce", Type: "uint64"},
	}
)

func (e *Exchange) hyperliquidChain() string {
	if e.client.baseURL == MainnetAPIURL {
		return "Mainnet"
	}
	return "Testnet"
}

func (e *Exchange) userSignedTypedData(primaryType string, payloadTypes []apitypes.Type, msg apitypes.TypedDataMessage) (*apitypes.TypedData, error) {
	action := make(apitypes.TypedDataMessage, len(msg)+2)
	for k, v := range msg {
		action[k] = v
	}
	action["signatureChainId"] = UserSignedSignatureChainID
	action["hyperliquidChain"] = e.hyperliquidChain()
	return UserSignedPayload(primaryType, payloadTypes, action)
}

func (e *Exchange) signUserAction(primaryType string, payloadTypes []apitypes.Type, msg apitypes.TypedDataMessage) (*Signature, error) {
	payload, err := e.userSignedTypedData(primaryType, payloadTypes, msg)
	if err != nil {
		return nil, err
	}
	return SignInner(e.signer, *payload)
}

func (e *Exchange) postUserSignedActionAndParseResponse(action Action, signature *Signature, nonce uint64) (string, []any, error) {
	payload := ExchangeRequest{
		Action:    action,
		Nonce:     nonce,
		Signature: signature,
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
	if respInner == nil {
		return "", nil, errors.New("missing exchange response")
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

func firstStatusOrType(respType string, statuses []any) any {
	if len(statuses) > 0 {
		return statuses[0]
	}
	return respType
}

func (e *Exchange) postSignedUserAction(action Action, primaryType string, payloadTypes []apitypes.Type, msg apitypes.TypedDataMessage, nonce uint64) (any, error) {
	sig, err := e.signUserAction(primaryType, payloadTypes, msg)
	if err != nil {
		return nil, err
	}
	respType, statuses, err := e.postUserSignedActionAndParseResponse(action, sig, nonce)
	if err != nil {
		return nil, err
	}
	return firstStatusOrType(respType, statuses), nil
}

func (e *Exchange) Withdraw3(destination string, amount string) (any, error) {
	nonce := e.NextNonce()
	destination = strings.ToLower(destination)
	action := &Withdraw3Action{
		Type:             "withdraw3",
		HyperliquidChain: e.hyperliquidChain(),
		SignatureChainID: UserSignedSignatureChainID,
		Destination:      destination,
		Amount:           amount,
		Time:             nonce,
	}
	msg := apitypes.TypedDataMessage{
		"destination": destination,
		"amount":      amount,
		"time":        new(big.Int).SetUint64(nonce),
	}
	return e.postSignedUserAction(action, "HyperliquidTransaction:Withdraw", withdrawSignTypes, msg, nonce)
}

func (e *Exchange) USDSend(destination string, amount string) (any, error) {
	nonce := e.NextNonce()
	destination = strings.ToLower(destination)
	action := &USDSendAction{
		Type:             "usdSend",
		HyperliquidChain: e.hyperliquidChain(),
		SignatureChainID: UserSignedSignatureChainID,
		Destination:      destination,
		Amount:           amount,
		Time:             nonce,
	}
	msg := apitypes.TypedDataMessage{
		"destination": destination,
		"amount":      amount,
		"time":        new(big.Int).SetUint64(nonce),
	}
	return e.postSignedUserAction(action, "HyperliquidTransaction:UsdSend", usdSendSignTypes, msg, nonce)
}

func (e *Exchange) USDTransfer(destination string, amount string) (any, error) {
	return e.USDSend(destination, amount)
}

func (e *Exchange) SpotSend(destination string, token string, amount string) (any, error) {
	nonce := e.NextNonce()
	destination = strings.ToLower(destination)
	action := &SpotSendAction{
		Type:             "spotSend",
		HyperliquidChain: e.hyperliquidChain(),
		SignatureChainID: UserSignedSignatureChainID,
		Destination:      destination,
		Token:            token,
		Amount:           amount,
		Time:             nonce,
	}
	msg := apitypes.TypedDataMessage{
		"destination": destination,
		"token":       token,
		"amount":      amount,
		"time":        new(big.Int).SetUint64(nonce),
	}
	return e.postSignedUserAction(action, "HyperliquidTransaction:SpotSend", spotSendSignTypes, msg, nonce)
}

func (e *Exchange) USDClassTransfer(amount string, toPerp bool) (any, error) {
	nonce := e.NextNonce()
	action := &USDClassTransferExchangeAction{
		Type:             "usdClassTransfer",
		HyperliquidChain: e.hyperliquidChain(),
		SignatureChainID: UserSignedSignatureChainID,
		Amount:           amount,
		ToPerp:           toPerp,
		Nonce:            nonce,
	}
	msg := apitypes.TypedDataMessage{
		"amount": amount,
		"toPerp": toPerp,
		"nonce":  new(big.Int).SetUint64(nonce),
	}
	return e.postSignedUserAction(action, "HyperliquidTransaction:UsdClassTransfer", usdClassTransferSignTypes, msg, nonce)
}

func (e *Exchange) SendAsset(destination string, sourceDex string, destinationDex string, token string, amount string, fromSubAccount string) (any, error) {
	nonce := e.NextNonce()
	destination = strings.ToLower(destination)
	fromSubAccount = strings.ToLower(fromSubAccount)
	action := &SendAssetAction{
		Type:             "sendAsset",
		HyperliquidChain: e.hyperliquidChain(),
		SignatureChainID: UserSignedSignatureChainID,
		Destination:      destination,
		SourceDex:        sourceDex,
		DestinationDex:   destinationDex,
		Token:            token,
		Amount:           amount,
		FromSubAccount:   fromSubAccount,
		Nonce:            nonce,
	}
	msg := apitypes.TypedDataMessage{
		"destination":    destination,
		"sourceDex":      sourceDex,
		"destinationDex": destinationDex,
		"token":          token,
		"amount":         amount,
		"fromSubAccount": fromSubAccount,
		"nonce":          new(big.Int).SetUint64(nonce),
	}
	return e.postSignedUserAction(action, "HyperliquidTransaction:SendAsset", sendAssetSignTypes, msg, nonce)
}

func (e *Exchange) TokenDelegate(validator common.Address, wei uint64, isUndelegate bool) (any, error) {
	nonce := e.NextNonce()
	action := &TokenDelegateAction{
		Type:             "tokenDelegate",
		HyperliquidChain: e.hyperliquidChain(),
		SignatureChainID: UserSignedSignatureChainID,
		Validator:        validator,
		Wei:              wei,
		IsUndelegate:     isUndelegate,
		Nonce:            nonce,
	}
	msg := apitypes.TypedDataMessage{
		"validator":    validator.Hex(),
		"wei":          new(big.Int).SetUint64(wei),
		"isUndelegate": isUndelegate,
		"nonce":        new(big.Int).SetUint64(nonce),
	}
	return e.postSignedUserAction(action, "HyperliquidTransaction:TokenDelegate", tokenDelegateSignTypes, msg, nonce)
}

func (e *Exchange) ApproveAgent(agentAddress common.Address, agentName string) (any, error) {
	nonce := e.NextNonce()
	action := &ApproveAgentAction{
		Type:             "approveAgent",
		HyperliquidChain: e.hyperliquidChain(),
		SignatureChainID: UserSignedSignatureChainID,
		AgentAddress:     agentAddress,
		AgentName:        agentName,
		Nonce:            nonce,
	}
	msg := apitypes.TypedDataMessage{
		"agentAddress": agentAddress.Hex(),
		"agentName":    agentName,
		"nonce":        new(big.Int).SetUint64(nonce),
	}
	return e.postSignedUserAction(action, "HyperliquidTransaction:ApproveAgent", approveAgentSignTypes, msg, nonce)
}

func (e *Exchange) ApproveBuilderFee(builder common.Address, maxFeeRate string) (any, error) {
	nonce := e.NextNonce()
	action := &ApproveBuilderFeeAction{
		Type:             "approveBuilderFee",
		HyperliquidChain: e.hyperliquidChain(),
		SignatureChainID: UserSignedSignatureChainID,
		Builder:          builder,
		MaxFeeRate:       maxFeeRate,
		Nonce:            nonce,
	}
	msg := apitypes.TypedDataMessage{
		"maxFeeRate": maxFeeRate,
		"builder":    builder.Hex(),
		"nonce":      new(big.Int).SetUint64(nonce),
	}
	return e.postSignedUserAction(action, "HyperliquidTransaction:ApproveBuilderFee", approveBuilderFeeSignTypes, msg, nonce)
}

func (e *Exchange) ConvertToMultiSigUser(signers string) (any, error) {
	nonce := e.NextNonce()
	action := &ConvertToMultiSigUserAction{
		Type:             "convertToMultiSigUser",
		HyperliquidChain: e.hyperliquidChain(),
		SignatureChainID: UserSignedSignatureChainID,
		Signers:          signers,
		Nonce:            nonce,
	}
	msg := apitypes.TypedDataMessage{
		"signers": signers,
		"nonce":   new(big.Int).SetUint64(nonce),
	}
	return e.postSignedUserAction(action, "HyperliquidTransaction:ConvertToMultiSigUser", convertToMultiSigUserSignTypes, msg, nonce)
}

func (e *Exchange) ConvertToMultiSigUserWithSigners(signers []MultiSigSigner) (any, error) {
	signersJSON, err := CanonicalMultiSigSigners(signers)
	if err != nil {
		return nil, err
	}
	return e.ConvertToMultiSigUser(signersJSON)
}

func CanonicalMultiSigSigners(signers []MultiSigSigner) (string, error) {
	canonical := make([]MultiSigSigner, len(signers))
	copy(canonical, signers)
	for i := range canonical {
		canonical[i].User = strings.ToLower(canonical[i].User)
	}
	sort.Slice(canonical, func(i, j int) bool {
		return canonical[i].User < canonical[j].User
	})
	data, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("failed to marshal multi-sig signers: %w", err)
	}
	return string(data), nil
}

func (e *Exchange) UserDexAbstraction(user common.Address, enabled bool) (any, error) {
	nonce := e.NextNonce()
	action := &UserDexAbstractionAction{
		Type:             "userDexAbstraction",
		HyperliquidChain: e.hyperliquidChain(),
		SignatureChainID: UserSignedSignatureChainID,
		User:             user.Hex(),
		Enabled:          enabled,
		Nonce:            nonce,
	}
	msg := apitypes.TypedDataMessage{
		"user":    user.Hex(),
		"enabled": enabled,
		"nonce":   new(big.Int).SetUint64(nonce),
	}
	return e.postSignedUserAction(action, "HyperliquidTransaction:UserDexAbstraction", userDexAbstractionSignTypes, msg, nonce)
}

func (e *Exchange) UserSetAbstraction(user common.Address, abstraction string) (any, error) {
	nonce := e.NextNonce()
	action := &UserSetAbstractionAction{
		Type:             "userSetAbstraction",
		HyperliquidChain: e.hyperliquidChain(),
		SignatureChainID: UserSignedSignatureChainID,
		User:             user.Hex(),
		Abstraction:      abstraction,
		Nonce:            nonce,
	}
	msg := apitypes.TypedDataMessage{
		"user":        user.Hex(),
		"abstraction": abstraction,
		"nonce":       new(big.Int).SetUint64(nonce),
	}
	return e.postSignedUserAction(action, "HyperliquidTransaction:UserSetAbstraction", userSetAbstractionSignTypes, msg, nonce)
}

func (e *Exchange) SignMultiSigAction(multiSigUser common.Address, action any, nonce uint64) (*Signature, error) {
	hash, err := actionHash(action, e.vault, nonce, e.expiresAfter)
	if err != nil {
		return nil, err
	}
	msg := apitypes.TypedDataMessage{
		"multiSigActionHash": hash.Bytes(),
		"nonce":              new(big.Int).SetUint64(nonce),
	}
	return e.signUserAction("HyperliquidTransaction:SendMultiSig", multiSigEnvelopeSignTypes, msg)
}

func (e *Exchange) MultiSig(multiSigUser common.Address, outerSigner common.Address, action any, signatures []Signature) (any, error) {
	nonce := e.NextNonce()
	multiSigAction := &MultiSigAction{
		Type:             "multiSig",
		SignatureChainID: UserSignedSignatureChainID,
		Signatures:       signatures,
		Payload: MultiSigPayload{
			MultiSigUser: strings.ToLower(multiSigUser.Hex()),
			OuterSigner:  strings.ToLower(outerSigner.Hex()),
			Action:       action,
		},
	}

	sig, err := e.signL1Action(multiSigAction, nonce)
	if err != nil {
		return nil, err
	}
	return e.postActionAndParseRaw(multiSigAction, sig, nonce)
}

func (e *Exchange) SubAccountTransfer(subAccountUser string, isDeposit bool, usd int) (any, error) {
	nonce := e.NextNonce()
	action := &SubAccountTransferAction{
		Type:           "subAccountTransfer",
		SubAccountUser: strings.ToLower(subAccountUser),
		IsDeposit:      isDeposit,
		Usd:            usd,
	}
	sig, err := e.signL1Action(action, nonce)
	if err != nil {
		return nil, err
	}
	respType, statuses, err := e.PostActionAndParseResponse(action, sig, nonce)
	if err != nil {
		return nil, fmt.Errorf("subAccountTransfer request failed: %w", err)
	}
	return firstStatusOrType(respType, statuses), nil
}

func (e *Exchange) SubAccountSpotTransfer(subAccountUser string, isDeposit bool, token string, amount string) (any, error) {
	nonce := e.NextNonce()
	action := &SubAccountSpotTransferAction{
		Type:           "subAccountSpotTransfer",
		SubAccountUser: strings.ToLower(subAccountUser),
		IsDeposit:      isDeposit,
		Token:          token,
		Amount:         amount,
	}
	sig, err := e.signL1Action(action, nonce)
	if err != nil {
		return nil, err
	}
	respType, statuses, err := e.PostActionAndParseResponse(action, sig, nonce)
	if err != nil {
		return nil, fmt.Errorf("subAccountSpotTransfer request failed: %w", err)
	}
	return firstStatusOrType(respType, statuses), nil
}
