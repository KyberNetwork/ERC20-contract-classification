package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gocarina/gocsv"

	"erc20-contract-classification/pkg/classifier/jsonrpc"
	"erc20-contract-classification/pkg/types"
)

const (
	rpcURL = "http://localhost:8545" // CHANGE ME
)

func main() {
	in, err := os.Open("erc20_transfer_tx.csv")
	if err != nil {
		panic(err)
	}
	defer in.Close()

	rpcClient, err := rpc.Dial(rpcURL)
	if err != nil {
		panic(err)
	}
	// ethClient := ethclient.NewClient(rpcClient)

	var (
		transfers []types.TransferRecord
		txHashes  = make(map[common.Hash]struct{})
		// txs        = make(map[common.Hash]*types.Transaction)
		// receipts   = make(map[common.Hash]*types.Receipt)
		callFrames = make(map[common.Hash]*jsonrpc.CallFrame)
	)

	if err := gocsv.UnmarshalFile(in, &transfers); err != nil {
		panic(err)
	}
	for _, t := range transfers {
		txHashes[t.TxHash] = struct{}{}
	}

	fmt.Printf("len(txs) = %d\n", len(txHashes))

	// if _, err := os.Stat("erc20_transfer_txs.json"); os.IsNotExist(err) {
	// 	var i int
	// 	for txHash := range txHashes {
	// 		tx, _, err := ethClient.TransactionByHash(context.Background(), txHash)
	// 		if err != nil {
	// 			panic(err)
	// 		}
	// 		txs[txHash] = tx
	// 		i++
	// 		fmt.Printf("fetched tx %d/%d\n", i, len(txHashes))
	// 	}

	// 	txsEncoded, err := json.MarshalIndent(txs, "", "  ")
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	if err := os.WriteFile("erc20_transfer_txs.json", txsEncoded, 0666); err != nil {
	// 		panic(err)
	// 	}
	// }

	// if _, err := os.Stat("erc20_transfer_tx_receipts.json"); os.IsNotExist(err) {
	// 	var i int
	// 	for txHash := range txHashes {
	// 		receipt, err := ethClient.TransactionReceipt(context.Background(), txHash)
	// 		if err != nil {
	// 			panic(err)
	// 		}
	// 		receipts[txHash] = receipt
	// 		i++
	// 		fmt.Printf("fetched tx receipts %d/%d\n", i, len(txHashes))
	// 	}
	// 	receiptsEncoded, err := json.MarshalIndent(receipts, "", "  ")
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	if err := os.WriteFile("erc20_transfer_tx_receipts.json", receiptsEncoded, 0666); err != nil {
	// 		panic(err)
	// 	}
	// }

	if _, err := os.Stat("erc20_transfer_tx_callframes.json"); os.IsNotExist(err) {
		var i int
		for txHash := range txHashes {
			result := new(jsonrpc.CallFrame)
			err := jsonrpc.DebugTraceTransaction(
				rpcClient,
				txHash,
				&jsonrpc.DebugTraceCallTracerConfigParam{
					Tracer: "callTracer",
				},
				result,
			)
			if err != nil {
				fmt.Printf("%s\n", err)
				panic(err)
			}
			callFrames[txHash] = result
			i++
			fmt.Printf("trace tx call frames %d/%d\n", i, len(txHashes))
		}

		callFramesEncoded, err := json.MarshalIndent(callFrames, "", "  ")
		if err != nil {
			panic(err)
		}
		if err := os.WriteFile("erc20_transfer_tx_callframes.json", callFramesEncoded, 0666); err != nil {
			panic(err)
		}
	}
}
