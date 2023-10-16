package jsonrpc

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	"erc20-contract-classification/pkg/classifier/abis"
	"erc20-contract-classification/pkg/utils"
)

type TransferScenario struct {
	// transfer() tx sender, might be wallet or contract
	MsgSender common.Address `json:"msgSender"`
	// ERC20 token contract address
	Token common.Address `json:"token"`
	// If true, call transferFrom(), otherwise, call transfer()
	IsTransferFrom bool `json:"isTransferFrom"`
	// If IsTransferFrom, From is transferFrom() from address
	From common.Address `json:"from"`
	// transferFrom() or transfer() to address
	To common.Address `json:"to"`
	// transferFrom() or transfer() transfer amount
	Amount *big.Int `json:"amount"`
	// block number to call/trace on
	BlockNumber string `json:"blockNumber"`
	// Gas for tracing call
	GasPrice  *big.Int `json:"gasPrice"`
	GasFeeCap *big.Int `json:"gasFeeCap"`
	GasTipCap *big.Int `json:"gasTipCap"`
}

type PrestateTracerConfig struct {
	DiffMode bool `json:"diffMode"`
}

var (
	TransferTracerConfig = PrestateTracerConfig{
		DiffMode: true, // set diffMode to true to get the post state
	}
	TransferTracerConfigEncoded, _ = json.Marshal(
		TransferTracerConfig,
	)
)

type PrestateTracerResult struct {
	Post map[common.Address]struct {
		Balance *hexutil.Big                `json:"balance,omitempty"`
		Code    []byte                      `json:"code,omitempty"`
		Nonce   uint64                      `json:"nonce,omitempty"`
		Storage map[common.Hash]common.Hash `json:"storage,omitempty"`
	} `json:"post"`
}

func ExtractStateDiff(scenario *TransferScenario, transferTraceResult *PrestateTracerResult, blockNumberHex string, client *rpc.Client) (*big.Int, error) {
	transferStateDiff := make(StateOverride)
	/*
		Step 1.2: extract the stateAfter
	*/

	for addr, override := range transferTraceResult.Post {
		var (
			balance *hexutil.Big
			nonce   = new(hexutil.Uint64)
			storage = make(map[common.Hash]string)
		)
		if override.Balance != nil {
			balance = override.Balance
		}
		*nonce = hexutil.Uint64(override.Nonce)
		for slot, val := range override.Storage {
			storage[slot] = utils.RemoveLeadingZerosFromHash(val)
		}
		transferStateDiff[addr] = OverrideAccount{
			Balance:   balance,
			Code:      override.Code,
			Nonce:     nonce,
			StateDiff: storage,
		}
	}

	/*
		Step 2: Make 2 balanceOf(to) calls: 1 without statediff overrides and 1 with statediff overrides
		to get the balance before and after transfering.
	*/

	balanceOfData, err := abis.ERC20.Pack("balanceOf", scenario.To)
	if err != nil {
		return nil, err
	}

	var (
		balanceBeforeTransfer *big.Int
		balanceAfterTransfer  *big.Int
	)

	balanceOfBeforeResult, err := EthCall(
		client,
		&EthCallCalldataParam{
			From: scenario.MsgSender.String(),
			To:   scenario.Token.String(),
			Data: hexutil.Encode(balanceOfData),
		},
		blockNumberHex,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("could not eth_call balanceOf() before transfer: %w", err)
	}
	decoded, err := hexutil.Decode(*balanceOfBeforeResult)
	if err != nil {
		return nil, err
	}
	balanceBeforeTransfer = new(big.Int).SetBytes(decoded)

	balanceOfAfterResult, err := EthCall(
		client,
		&EthCallCalldataParam{
			From: scenario.MsgSender.String(),
			To:   scenario.Token.String(),
			Data: hexutil.Encode(balanceOfData),
		},
		blockNumberHex,
		transferStateDiff,
	)
	if err != nil {
		return nil, fmt.Errorf("could not eth_call balanceOf() after transfer: %w", err)
	}
	decoded, err = hexutil.Decode(*balanceOfAfterResult)
	if err != nil {
		return nil, err
	}
	balanceAfterTransfer = new(big.Int).SetBytes(decoded)

	if balanceAfterTransfer.Cmp(balanceBeforeTransfer) <= 0 {
		return nil, fmt.Errorf("balance after transfer is <= balance before transfer")
	}

	// the actual amount received is the different between balance after and balance before
	return new(big.Int).Sub(balanceAfterTransfer, balanceBeforeTransfer), nil

}
