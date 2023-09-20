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

type prestateTracerConfig struct {
	DiffMode bool `json:"diffMode"`
}

type prestateTracerResult struct {
	Post map[common.Address]struct {
		Balance *hexutil.Big                `json:"balance,omitempty"`
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
			return nil, fmt.Errorf("could not get block number: %w", err)
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
	success, err := c.ethClient.CallContract(
		context.Background(),
		ethereum.CallMsg{
			From: scenario.MsgSender,
			To:   &scenario.Token,
			Data: transferData,
		},
		new(big.Int).SetUint64(blockNumber),
	)
	if err != nil {
		return nil, fmt.Errorf("could not eth_call: %w", err)
	}
	if new(big.Int).SetBytes(success).Cmp(big.NewInt(1)) != 0 {
		return nil, fmt.Errorf("transfer not success")
	}

	transferTraceResult := new(prestateTracerResult)
	transferTracerConfig := prestateTracerConfig{
		DiffMode: true, // set diffMode to true to get the post state
	}
	transferTracerConfigEncoded, _ := json.Marshal(transferTracerConfig)
	// some tracing fails if we don't specify maxFeePerGas and maxPriorityFeePerGas
	var (
		gasPrice             string
		maxFeePerGas         string
		maxPriorityFeePerGas string
	)
	if scenario.GasFeeCap != nil && scenario.GasFeeCap.Cmp(big.NewInt(0)) != 0 {
		c := new(big.Int).Mul(scenario.GasFeeCap, big.NewInt(150))
		c = c.Div(c, big.NewInt(100))
		maxFeePerGas = hexutil.EncodeBig(c)

		if scenario.GasTipCap != nil {
			c := new(big.Int).Mul(scenario.GasTipCap, big.NewInt(150))
			c = c.Div(c, big.NewInt(100))
			maxPriorityFeePerGas = hexutil.EncodeBig(c)
		}
	} else {
		gasPrice = hexutil.EncodeBig(scenario.GasPrice)
	}
	err = jsonrpc.DebugTraceCall(
		c.client,
		&jsonrpc.DebugTraceCallCalldataParam{
			From:                 scenario.MsgSender.String(),
			GasPrice:             gasPrice,
			MaxFeePerGas:         maxFeePerGas,
			MaxPriorityFeePerGas: maxPriorityFeePerGas,
			To:                   scenario.Token.String(),
			Data:                 hexutil.Encode(transferData),
		},
		blockNumberHex,
		&jsonrpc.DebugTraceCallTracerConfigParam{
			// we are using the builtin prestateTracer in go-ethereum
			// https://github.com/ethereum/go-ethereum/blob/master/eth/tracers/native/prestate.go
			Tracer:       "prestateTracer",
			TracerConfig: transferTracerConfigEncoded,
			StateOverrides: map[common.Address]jsonrpc.OverrideAccount{
				scenario.MsgSender: {
					// very large balance
					Balance: (*hexutil.Big)(hexutil.MustDecodeBig("0xffffffffffffffffffffffffffffffff")),
				},
			},
		},
		transferTraceResult,
	)
	if err != nil {
		return nil, fmt.Errorf("could not debug_traceCall a transfer tx: %w", err)
	}

	/*
		Step 1.2: extract the statediff
	*/
	transferStateDiff := make(jsonrpc.StateOverride)
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
			storage[slot] = removeLeadingZerosFromHash(val)
		}
		transferStateDiff[addr] = jsonrpc.OverrideAccount{
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
		return nil, fmt.Errorf("could not eth_call balanceOf() before transfer: %w", err)
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
		return nil, fmt.Errorf("could not eth_call balanceOf() after transfer: %w", err)
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
		numScenarios int
		numEqual     int
		numLess      int
	)
	for _, s := range scenarios {
		if s.Token == token {
			numScenarios++
		}
	}
	fmt.Printf("numScenarios = %d\n", numScenarios)
	i := 0
	for _, s := range scenarios {
		if s.Token != token {
			// skip unrelated tokens
			continue
		}

		i++
		fmt.Printf("  checking scenario %d/%d\n", i, numScenarios)

		actualAmount, err := c.getActualBalanceReceivedAfterTransfer(s)
		if err != nil {
			fmt.Printf("    could not getActualBalanceReceivedAfterTransfer: %s\n", err)
			continue
		}

		fmt.Printf("    transfer amount = %s, actual received amount = %s\n", s.Amount, actualAmount)

		// if actual amount received is less than transfer amount, token is FOT
		switch actualAmount.Cmp(s.Amount) {
		case -1:
			numLess++
		case 0:
			numEqual++
		}
	}

	fmt.Printf("    numEqual = %d, numLess = %d\n", numEqual, numLess)

	if numEqual > 0 && numLess == 0 {
		return false, nil
	}
	if numLess > 0 {
		return true, nil
	}
	return false, fmt.Errorf("could not decide")
}
