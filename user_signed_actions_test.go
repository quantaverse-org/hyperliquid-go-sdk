package sdk

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestWithdraw3UserSignedActionMatchesOfficialVector(t *testing.T) {
	signer, err := NewLocalSignerFromHex("0123456789012345678901234567890123456789012345678901234567890123")
	if err != nil {
		t.Fatal(err)
	}
	exchange := NewExchange(TestnetAPIURL, nil, nil, signer)
	nonce := uint64(1687816341423)
	msg := map[string]interface{}{
		"destination": "0x5e9ee1089755c3435139848e47e6635505d5a13a",
		"amount":      "1",
		"time":        new(big.Int).SetUint64(nonce),
	}

	sig, err := exchange.signUserAction("HyperliquidTransaction:Withdraw", withdrawSignTypes, msg)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := hexutil.Encode(sig.R), "0x8363524c799e90ce9bc41022f7c39b4e9bdba786e5f9c72b20e43e1462c37cf9"; got != want {
		t.Fatalf("r mismatch: got %s want %s", got, want)
	}
	if got, want := hexutil.Encode(sig.S), "0x58b1411a775938b83e29182e8ef74975f9054c8e97ebf5ec2dc8d51bfc893881"; got != want {
		t.Fatalf("s mismatch: got %s want %s", got, want)
	}
	if got, want := sig.V, uint8(28); got != want {
		t.Fatalf("v mismatch: got %d want %d", got, want)
	}
}

func TestSubAccountTransferL1ActionMatchesOfficialVector(t *testing.T) {
	signer, err := NewLocalSignerFromHex("0123456789012345678901234567890123456789012345678901234567890123")
	if err != nil {
		t.Fatal(err)
	}
	action := &SubAccountTransferAction{
		Type:           "subAccountTransfer",
		SubAccountUser: "0x1d9470d4b963f552e6f671a81619d395877bf409",
		IsDeposit:      true,
		Usd:            10,
	}

	sig, err := SignL1Action(signer, action, nil, 0, true)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := hexutil.Encode(sig.R), "0x43592d7c6c7d816ece2e206f174be61249d651944932b13343f4d13f306ae602"; got != want {
		t.Fatalf("r mismatch: got %s want %s", got, want)
	}
	if got, want := hexutil.Encode(sig.S), "0x71a926cb5c9a7c01c3359ec4c4c34c16ff8107d610994d4de0e6430e5cc0f4c9"; got != want {
		t.Fatalf("s mismatch: got %s want %s", got, want)
	}
	if got, want := sig.V, uint8(28); got != want {
		t.Fatalf("v mismatch: got %d want %d", got, want)
	}
}
