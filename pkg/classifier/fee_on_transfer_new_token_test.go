package classifier

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"erc20-contract-classification/pkg/classifier/jsonrpc"
)

func bigIntMustFromString(s string) *big.Int {
	bn, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic("big.Int must from string")
	}
	return bn
}

type testScenario struct {
	transfers       []*jsonrpc.TransferScenario
	isFeeOnTransfer bool
}

var (
	testScenarios = map[common.Address]testScenario{
		// GROWTH token which is FOT
		common.HexToAddress("0x0c7361B70e8F8530B7c0CcB17EeA89278E670C93"): {
			transfers: []*jsonrpc.TransferScenario{
				{
					MsgSender: common.HexToAddress("0x2FD45E9c69D50cD08a03792253daC3CA37a81cBf"), // a holder
					Token:     common.HexToAddress("0x0c7361B70e8F8530B7c0CcB17EeA89278E670C93"),
					To:        common.HexToAddress("0x49003cc3b1d8835c3b4aa5a581a6be0b0843e91d"),
					Amount:    bigIntMustFromString("30000000000000000000000"), // 30K GROWTH
				},
			},
			isFeeOnTransfer: true,
		},
		// USDT which is not FOT (yet)
		common.HexToAddress("0xdac17f958d2ee523a2206206994597c13d831ec7"): {
			transfers: []*jsonrpc.TransferScenario{
				{
					MsgSender: common.HexToAddress("0xBDa23B750dD04F792ad365B5F2a6F1d8593796f2"), // a holder
					Token:     common.HexToAddress("0xdac17f958d2ee523a2206206994597c13d831ec7"),
					To:        common.HexToAddress("0x49003cc3b1d8835c3b4aa5a581a6be0b0843e91d"),
					Amount:    bigIntMustFromString("30000000"), // 30K USDT
				},
			},
			isFeeOnTransfer: false,
		},
	}
)

func TestIsFeeOnTransferNewToken(t *testing.T) {
	rpcClient, err := rpc.Dial(rpcURL)
	require.NoError(t, err)

	c := NewClassifier(rpcClient, nil)
	for token, s := range testScenarios {
		fot, err := c.IsFeeOnTransferNewToken(token, s.transfers)
		require.NoError(t, err)
		require.Equal(t, s.isFeeOnTransfer, fot)
	}
}
