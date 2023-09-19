package classifier

import (
	"context"
	"encoding/json"
	"erc20-contract-classification/pkg/classifier/abis"
	"erc20-contract-classification/pkg/classifier/jsonrpc"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type TransferScenario struct {
	// transfer() tx sender, might be wallet or contract
	MsgSender common.Address
	// ERC20 token contract address
	Token common.Address
	// If true, call transferFrom(), otherwise, call transfer()
	IsTransferFrom bool
	// If IsTransferFrom, From is transferFrom() from address
	From common.Address
	// transferFrom() or transfer() to address
	To common.Address
	// transferFrom() or transfer() transfer amount
	Amount *big.Int
	// block number to call/trace on
	BlockNumber string
}

type prestateTracerConfig struct {
	DiffMode bool `json:"diffMode"`
}

type prestateTracerResult struct {
	Post map[common.Address]struct {
		Balance *big.Int                    `json:"balance,omitempty"`
		Code    []byte                      `json:"code,omitempty"`
		Nonce   uint64                      `json:"nonce,omitempty"`
		Storage map[common.Hash]common.Hash `json:"storage,omitempty"`
	} `json:"post"`
}

func (c *Classifier) getActualBalanceReceivedAfterTransfer(scenario *TransferScenario) (*big.Int, error) {
	/*
		Step 0: If not specific block number, get the latest block number to make the following step consistent.
	*/
	var (
		blockNumber    uint64
		blockNumberHex string
		err            error
	)
	if scenario.BlockNumber != "" {
		blockNumberHex = scenario.BlockNumber
		blockNumber, err = hexutil.DecodeUint64(blockNumberHex)
		if err != nil {
			return nil, err
		}
	} else {
		blockNumber, err := c.ethClient.BlockNumber(context.Background())
		if err != nil {
			return nil, err
		}
		blockNumberHex = hexutil.EncodeUint64(blockNumber)
	}

	/*
		Step 1: Trace a transfer(to, amount) (or transferFrom(from, to, amount)) tx and extract the statediff.
	*/
	var transferData []byte
	if scenario.IsTransferFrom {
		transferData, err = abis.ERC20.Pack("transferFrom", scenario.From, scenario.To, scenario.Amount)
	} else {
		transferData, err = abis.ERC20.Pack("transfer", scenario.To, scenario.Amount)
	}
	if err != nil {
		return nil, err
	}

	// make sure the tranfer tx is success
	_, err = c.ethClient.CallContract(
		context.Background(),
		ethereum.CallMsg{
			From: scenario.MsgSender,
			To:   &scenario.Token,
			Data: transferData,
		},
		new(big.Int).SetUint64(blockNumber),
	)
	if err != nil {
		return nil, err
	}

	transferTraceResult := new(prestateTracerResult)
	transferTracerConfig := prestateTracerConfig{
		DiffMode: true, // set diffMode to true to get the post state
	}
	transferTracerConfigEncoded, _ := json.Marshal(transferTracerConfig)
	err = jsonrpc.DebugTraceCall(
		c.client,
		&jsonrpc.DebugTraceCallCalldataParam{
			From: scenario.MsgSender.String(),
			To:   scenario.Token.String(),
			Data: hexutil.Encode(transferData),
		},
		blockNumberHex,
		&jsonrpc.DebugTraceCallTracerConfigParam{
			// we are using the builtin prestateTracer in go-ethereum
			// https://github.com/ethereum/go-ethereum/blob/master/eth/tracers/native/prestate.go
			Tracer:       "prestateTracer",
			TracerConfig: transferTracerConfigEncoded,
		},
		transferTraceResult,
	)
	if err != nil {
		return nil, err
	}

	/*
		Step 1.2: extract the statediff
	*/
	transferStateDiff := make(jsonrpc.StateOverride)
	for addr, override := range transferTraceResult.Post {
		var (
			balance *hexutil.Big
			nonce   = new(hexutil.Uint64)
		)
		if override.Balance != nil {
			balance = (*hexutil.Big)(new(big.Int).Set(override.Balance))
		}
		*nonce = hexutil.Uint64(override.Nonce)
		transferStateDiff[addr] = jsonrpc.OverrideAccount{
			Balance:   balance,
			Code:      override.Code,
			Nonce:     nonce,
			StateDiff: override.Storage,
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

	balanceOfBeforeResult, err := jsonrpc.EthCall(
		c.client,
		&jsonrpc.EthCallCalldataParam{
			From: scenario.MsgSender.String(),
			To:   scenario.Token.String(),
			Data: hexutil.Encode(balanceOfData),
		},
		blockNumberHex,
		nil,
	)
	if err != nil {
		return nil, err
	}
	decoded, err := hexutil.Decode(*balanceOfBeforeResult)
	if err != nil {
		return nil, err
	}
	balanceBeforeTransfer = new(big.Int).SetBytes(decoded)

	balanceOfAfterResult, err := jsonrpc.EthCall(
		c.client,
		&jsonrpc.EthCallCalldataParam{
			From: scenario.MsgSender.String(),
			To:   scenario.Token.String(),
			Data: hexutil.Encode(balanceOfData),
		},
		blockNumberHex,
		transferStateDiff,
	)
	if err != nil {
		return nil, err
	}
	decoded, err = hexutil.Decode(*balanceOfAfterResult)
	if err != nil {
		return nil, err
	}
	balanceAfterTransfer = new(big.Int).SetBytes(decoded)

	// the actual amount received is the different between balance after and balance before
	actualAmount := new(big.Int).Sub(balanceAfterTransfer, balanceBeforeTransfer)
	return actualAmount, nil
}

func (c *Classifier) IsFeeOnTransferNewToken(token common.Address, scenarios []*TransferScenario) (bool, error) {
	fmt.Printf("checking token %s\n", token)
	var (
		numEqual int
		numLess  int
	)
	for _, s := range scenarios {
		if s.Token != token {
			// skip unrelated tokens
			continue
		}

		actualAmount, err := c.getActualBalanceReceivedAfterTransfer(s)
		if err != nil {
			fmt.Printf("could not getActualBalanceReceivedAfterTransfer, err = %s\n", err)
			continue
		}

		// if actual amount received is less than transfer amount, token is FOT
		switch actualAmount.Cmp(s.Amount) {
		case -1:
			fmt.Printf("actualAmount = %s is less than transfer amount %s => token is FOT\n", actualAmount, s.Amount)
			numLess++
		case 0:
			numEqual++
		case 1:
			fmt.Printf("abnormal result, actualAmount = %s is larger than transfer amount %s", actualAmount, s.Amount)
		}
	}

	if numEqual > 0 && numLess == 0 {
		return false, nil
	}
	if numLess > 0 {
		return true, nil
	}
	return false, fmt.Errorf("could not decide")
}
