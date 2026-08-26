package sdk

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/ethereum/go-ethereum/crypto"
)

func actionHash(action any, vault *common.Address, nonce uint64, expiresAfter *uint64) (common.Hash, error) {
	data, err := msgpack.Marshal(action)
	if err != nil {
		return common.Hash{}, fmt.Errorf("error while marshaling action: %s", err)
	}
	nonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBytes, nonce)
	data = append(data, nonceBytes...)

	if vault == nil {
		data = append(data, '\x00')
	} else {
		data = append(data, '\x01')
		data = append(data, vault.Bytes()...)
	}
	if expiresAfter != nil {
		expiresAfterBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(expiresAfterBytes, *expiresAfter)
		data = append(data, '\x00')
		data = append(data, expiresAfterBytes...)
	}

	return crypto.Keccak256Hash(data), nil
}

func constructPhantomAgent(hash common.Hash, isMainnet bool) apitypes.TypedDataMessage {
	if isMainnet {
		return apitypes.TypedDataMessage{
			"source":       "a",
			"connectionId": hash.Bytes(),
		}
	} else {
		return apitypes.TypedDataMessage{
			"source":       "b",
			"connectionId": hash.Bytes(),
		}
	}
}

func l1Payload(phantomAgent apitypes.TypedDataMessage) apitypes.TypedData {
	return apitypes.TypedData{
		Domain: apitypes.TypedDataDomain{
			Name:              "Exchange",
			Version:           "1",
			ChainId:           math.NewHexOrDecimal256(1337),
			VerifyingContract: "0x0000000000000000000000000000000000000000",
		},
		Types: apitypes.Types{
			"Agent": []apitypes.Type{
				{Name: "source", Type: "string"},
				{Name: "connectionId", Type: "bytes32"},
			},
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
		},
		Message:     phantomAgent,
		PrimaryType: "Agent",
	}
}

func SignL1Action(
	signer Signer,
	action any,
	vault *common.Address,
	nonce uint64,
	isMainnet bool,
) (*Signature, error) {
	return SignL1ActionWithExpiresAfter(signer, action, vault, nonce, nil, isMainnet)
}

func SignL1ActionWithExpiresAfter(
	signer Signer,
	action any,
	vault *common.Address,
	nonce uint64,
	expiresAfter *uint64,
	isMainnet bool,
) (*Signature, error) {
	hash, err := actionHash(action, vault, nonce, expiresAfter)
	if err != nil {
		return nil, err
	}
	phantomAgent := constructPhantomAgent(hash, isMainnet)
	payload := l1Payload(phantomAgent)
	return SignInner(signer, payload)
}

func SignL1ActionRaw(
	signer Signer,
	actionData []byte,
	vault *common.Address,
	nonce uint64,
	isMainnet bool,
) (*Signature, error) {
	hash, err := actionHash(actionData, vault, nonce, nil)
	if err != nil {
		return nil, err
	}
	phantomAgent := constructPhantomAgent(hash, isMainnet)
	payload := l1Payload(phantomAgent)
	return SignInner(signer, payload)
}

func SignUserSignedAction(
	signer Signer,
	action apitypes.TypedDataMessage,
	payloadTypes []apitypes.Type,
	primaryType string,
) (*Signature, error) {
	payload, err := UserSignedPayload(primaryType, payloadTypes, action)
	if err != nil {
		return nil, err
	}
	return SignInner(signer, *payload)
}

// / Generate Final EIP-712 TypedData Package for signing on Hyper
func UserSignedPayload(primaryType string, payloadTypes []apitypes.Type, action map[string]interface{}) (*apitypes.TypedData, error) {
	chainIdHex, ok := action["signatureChainId"]
	if !ok {
		return nil, fmt.Errorf("no signatureChainId found in action %#v", action)
	}
	chainIdHexString, ok := chainIdHex.(string)
	if !ok {
		return nil, fmt.Errorf("signatureChainId is not valid in action %#v", action)
	}

	var chainID *big.Int
	// set 0 let func detect.
	chainID, success := new(big.Int).SetString(chainIdHexString, 0)
	if !success {
		return nil, fmt.Errorf("signatureChainId not a value hex %v", chainIdHex)
	}
	delete(action, "signatureChainId")

	// types field spec on EIP-712
	types := map[string][]apitypes.Type{
		primaryType: payloadTypes,
		"EIP712Domain": {
			{Name: "name", Type: "string"},
			{Name: "version", Type: "string"},
			{Name: "chainId", Type: "uint256"},
			{Name: "verifyingContract", Type: "address"},
		},
	}

	domain := apitypes.TypedDataDomain{
		Name:              "HyperliquidSignTransaction",
		Version:           "1",
		ChainId:           math.NewHexOrDecimal256(chainID.Int64()),
		VerifyingContract: "0x0000000000000000000000000000000000000000",
	}

	return &apitypes.TypedData{
		Types:       types,
		PrimaryType: primaryType,
		Domain:      domain,
		Message:     action,
	}, nil
}
