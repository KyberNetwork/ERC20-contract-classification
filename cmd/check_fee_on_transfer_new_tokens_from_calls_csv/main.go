package main

import (
	"fmt"
	"math/big"
	"os"

	"erc20-contract-classification/pkg/classifier"
	"erc20-contract-classification/pkg/classifier/jsonrpc"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gocarina/gocsv"
)

const (
	rpcURL = "http://localhost:8545" // CHANGE ME
)

// TransferCall result from this query https://dune.com/queries/3038453
// msg_sender,token,is_transfer_from,sender,receiver,amount,block_number,gas_price,max_fee_per_gas,max_priority_fee_per_gas,tx_hash,tx_index
type TransferCall struct {
	MsgSender            common.Address `csv:"msg_sender"`
	Token                common.Address `csv:"token"`
	IsTransferFrom       bool           `csv:"is_transfer_from"`
	Sender               common.Address `csv:"sender"`
	Receiver             common.Address `csv:"receiver"`
	Amount               *big.Int       `csv:"amount"`
	BlockNumber          string         `csv:"block_number"`
	GasPrice             string         `csv:"gas_price"`
	MaxFeePerGas         string         `csv:"max_fee_per_gas"`
	MaxPriorityFeePerGas string         `csv:"max_priority_fee_per_gas"`
	TxHash               common.Hash    `csv:"tx_hash"`
	TxIndex              uint64         `csv:"tx_index"`
}

func main() {
	in, err := os.Open("erc20_transfer_calls.csv")
	if err != nil {
		panic(err)
	}
	defer in.Close()

	var transferCalls []*TransferCall
	if err := gocsv.UnmarshalFile(in, &transferCalls); err != nil {
		panic(err)
	}

	var scenarios []*jsonrpc.TransferScenario
	for _, call := range transferCalls {
		blockNumber, _ := new(big.Int).SetString(call.BlockNumber, 0)
		gasPrice, _ := new(big.Int).SetString(call.GasPrice, 0)
		gasFeeCap, _ := new(big.Int).SetString(call.MaxFeePerGas, 0)
		gasTipCap, _ := new(big.Int).SetString(call.MaxPriorityFeePerGas, 0)
		scenarios = append(scenarios, &jsonrpc.TransferScenario{
			MsgSender:      call.MsgSender,
			Token:          call.Token,
			IsTransferFrom: call.IsTransferFrom,
			From:           call.Sender,
			To:             call.Receiver,
			Amount:         call.Amount,
			BlockNumber:    hexutil.EncodeBig(blockNumber),
			GasPrice:       gasPrice,
			GasFeeCap:      gasFeeCap,
			GasTipCap:      gasTipCap,
		})
	}

	fmt.Printf("len(scenarios) = %d\n", len(scenarios))

	for _, s := range scenarios {
		blockNumber := hexutil.MustDecodeBig(s.BlockNumber)
		s.BlockNumber = hexutil.EncodeBig(new(big.Int).Sub(blockNumber, big.NewInt(1)))
	}
	fmt.Println("set simulation block to the previous block of the block where the tx is executed to maximize the probability that sender has enough amount")

	tokens := make(map[common.Address]struct{})
	for _, s := range scenarios {
		tokens[s.Token] = struct{}{}
	}

	fmt.Printf("len(tokens) = %d\n", len(tokens))

	rpcClient, err := rpc.Dial(rpcURL)
	if err != nil {
		panic(err)
	}

	// don't need erc20BalanceSlotProbe
	clz := classifier.NewClassifier(rpcClient, nil)

	outputFile, err := os.Create("output.csv")
	if err != nil {
		panic(err)
	}
	defer outputFile.Close()
	writer := gocsv.DefaultCSVWriter(outputFile)

	for token := range tokens {
		fot, err := clz.IsFeeOnTransferNewToken(token, scenarios)
		if err != nil {
			fmt.Printf("%s\n", err)
		}
		if err != nil {
			writer.Write([]string{token.String(), "could not decide"})
			continue
		}
		if fot {
			fmt.Printf("    token %s is fee-on-transfer\n", token)
			writer.Write([]string{token.String(), "fot"})
		} else {
			fmt.Printf("    token %s is NOT fee-on-transfer\n", token)
			writer.Write([]string{token.String(), "not fot"})
		}
	}
	writer.Flush()
}
