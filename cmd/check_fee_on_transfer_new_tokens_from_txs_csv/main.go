package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/KyberNetwork/erc20-contract-classification/pkg/classifier"
	"github.com/KyberNetwork/erc20-contract-classification/pkg/classifier/abis"
	"github.com/KyberNetwork/erc20-contract-classification/pkg/classifier/jsonrpc"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gocarina/gocsv"
)

const (
	rpcURL = "http://localhost:8545" // CHANGE ME
)

type TransferRecord struct {
	Sender      common.Address `csv:"sender_address"`
	Receiver    common.Address `csv:"receiver_address"`
	TxHash      common.Hash    `csv:"tx_hash"`
	TotalAmount *big.Int       `csv:"total_tokens_transferred"`
}

var (
	transferMethodABI     = abis.ERC20.Methods["transfer"]
	transferFromMethodABI = abis.ERC20.Methods["transferFrom"]
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
	ethClient := ethclient.NewClient(rpcClient)

	var transfers []*TransferRecord
	if err := gocsv.UnmarshalFile(in, &transfers); err != nil {
		panic(err)
	}

	txHashes := make(map[common.Hash]struct{})
	for _, t := range transfers {
		txHashes[t.TxHash] = struct{}{}
	}

	fmt.Printf("len(txs) = %d\n", len(txHashes))

	txs, err := fetchOrGetCachedTransactions(ethClient, txHashes)
	if err != nil {
		panic(err)
	}
	_ = txs

	receipts, err := fetchOrGetCachedTransactionReceipts(ethClient, txHashes)
	if err != nil {
		panic(err)
	}

	callFrames, err := traceOrGetCachedTransactionCallFrames(rpcClient, txHashes)
	if err != nil {
		panic(err)
	}

	scenarios, err := extractOrGetCachedTransferScenarios(txs, receipts, callFrames)
	if err != nil {
		panic(err)
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

	// tokens := map[common.Address]struct{}{
	//	common.HexToAddress("0x9b0E1C344141fB361B842d397dF07174E1CDb988"): {},
	// 	common.HexToAddress("0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"): {},
	// }

	fmt.Printf("len(tokens) = %d\n", len(tokens))

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

func fetchOrGetCachedTransactions(ethClient *ethclient.Client, txHashes map[common.Hash]struct{}) (map[common.Hash]*types.Transaction, error) {
	var txs = make(map[common.Hash]*types.Transaction)
	if _, err := os.Stat("erc20_transfer_txs.json"); os.IsNotExist(err) {
		var i int
		for txHash := range txHashes {
			tx, _, err := ethClient.TransactionByHash(context.Background(), txHash)
			if err != nil {
				return nil, err
			}
			txs[txHash] = tx
			i++
			fmt.Printf("fetched tx %d/%d\n", i, len(txHashes))
		}

		txsEncoded, err := json.MarshalIndent(txs, "", "  ")
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile("erc20_transfer_txs.json", txsEncoded, 0666); err != nil {
			return nil, err
		}
	} else {
		f, err := os.Open("erc20_transfer_txs.json")
		if err != nil {
			return nil, err
		}
		defer f.Close()
		if err := json.NewDecoder(f).Decode(&txs); err != nil {
			return nil, err
		}
	}

	return txs, nil
}

func fetchOrGetCachedTransactionReceipts(ethClient *ethclient.Client, txHashes map[common.Hash]struct{}) (map[common.Hash]*types.Receipt, error) {
	var receipts = make(map[common.Hash]*types.Receipt)
	if _, err := os.Stat("erc20_transfer_tx_receipts.json"); os.IsNotExist(err) {
		var i int
		for txHash := range txHashes {
			receipt, err := ethClient.TransactionReceipt(context.Background(), txHash)
			if err != nil {
				return nil, err
			}
			receipts[txHash] = receipt
			i++
			fmt.Printf("fetched tx receipts %d/%d\n", i, len(txHashes))
		}
		receiptsEncoded, err := json.MarshalIndent(receipts, "", "  ")
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile("erc20_transfer_tx_receipts.json", receiptsEncoded, 0666); err != nil {
			return nil, err
		}
	} else {
		f, err := os.Open("erc20_transfer_tx_receipts.json")
		if err != nil {
			return nil, err
		}
		defer f.Close()
		if err := json.NewDecoder(f).Decode(&receipts); err != nil {
			return nil, err
		}
	}
	return receipts, nil
}

func traceOrGetCachedTransactionCallFrames(rpcClient *rpc.Client, txHashes map[common.Hash]struct{}) (map[common.Hash]*jsonrpc.CallFrame, error) {
	var callFrames = make(map[common.Hash]*jsonrpc.CallFrame)
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
				fmt.Printf("could not debug_traceTransaction %s\n", err)
				return nil, err
			}
			callFrames[txHash] = result
			i++
			fmt.Printf("trace tx call frames %d/%d\n", i, len(txHashes))
		}

		callFramesEncoded, err := json.MarshalIndent(callFrames, "", "  ")
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile("erc20_transfer_tx_callframes.json", callFramesEncoded, 0666); err != nil {
			return nil, err
		}
	} else {
		f, err := os.Open("erc20_transfer_tx_callframes.json")
		if err != nil {
			return nil, err
		}
		defer f.Close()
		if err := json.NewDecoder(f).Decode(&callFrames); err != nil {
			return nil, err
		}
	}
	return callFrames, nil
}

func extractOrGetCachedTransferScenarios(txs map[common.Hash]*types.Transaction, receipts map[common.Hash]*types.Receipt, callFrames map[common.Hash]*jsonrpc.CallFrame) ([]*jsonrpc.TransferScenario, error) {
	var scenarios []*jsonrpc.TransferScenario
	if _, err := os.Stat("erc20_transfer_scenarios.json"); os.IsNotExist(err) {
		for txHash, call := range callFrames {
			blockNumber := receipts[txHash].BlockNumber
			blockNumberHex := hexutil.EncodeBig(blockNumber)
			gasFeeCap := txs[txHash].GasFeeCap()
			gasTipCap := txs[txHash].GasTipCap()
			iterateCallFrames(call, func(c *jsonrpc.CallFrame) {
				if bytes.HasPrefix(c.Input, transferMethodABI.ID) {
					if new(big.Int).SetBytes(c.Output).Cmp(big.NewInt(1)) != 0 {
						fmt.Printf("skipped transfer() returns false\n")
						return
					}
					params, err := transferMethodABI.Inputs.Unpack(c.Input[4:])
					if err != nil {
						fmt.Printf("could not unpack transfer() method params %s\n", err)
						return
					}
					toAddr, ok := params[0].(common.Address)
					if !ok {
						fmt.Printf("params[0] must be common.Address")
						return
					}
					amount, ok := params[1].(*big.Int)
					if !ok {
						fmt.Printf("params[1] must be *big.Int")
						return
					}
					scenarios = append(scenarios, &jsonrpc.TransferScenario{
						MsgSender:      c.From,
						Token:          *c.To,
						IsTransferFrom: false,
						To:             toAddr,
						Amount:         amount,
						BlockNumber:    blockNumberHex,
						GasFeeCap:      gasFeeCap,
						GasTipCap:      gasTipCap,
					})
				} else if bytes.HasPrefix(c.Input, transferFromMethodABI.ID) {
					if new(big.Int).SetBytes(c.Output).Cmp(big.NewInt(1)) != 0 {
						fmt.Printf("skipped transferFrom() returns false\n")
						return
					}
					params, err := transferFromMethodABI.Inputs.Unpack(c.Input[4:])
					if err != nil {
						fmt.Printf("could not unpack transferFrom() method params %s\n", err)
						return
					}
					fromAddr, ok := params[0].(common.Address)
					if !ok {
						fmt.Printf("params[0] must be common.Address")
						return
					}
					toAddr, ok := params[1].(common.Address)
					if !ok {
						fmt.Printf("params[1] must be common.Address")
						return
					}
					amount, ok := params[2].(*big.Int)
					if !ok {
						fmt.Printf("params[2] must be *big.Int")
						return
					}
					scenarios = append(scenarios, &jsonrpc.TransferScenario{
						MsgSender:      c.From,
						Token:          *c.To,
						IsTransferFrom: true,
						From:           fromAddr,
						To:             toAddr,
						Amount:         amount,
						BlockNumber:    blockNumberHex,
						GasFeeCap:      gasFeeCap,
						GasTipCap:      gasTipCap,
					})
				}
			})
		}

		scenariosEncoded, err := json.MarshalIndent(scenarios, "", "  ")
		if err != nil {
			panic(err)
		}
		if err := os.WriteFile("erc20_transfer_scenarios.json", scenariosEncoded, 0666); err != nil {
			panic(err)
		}
	} else {
		f, err := os.Open("erc20_transfer_scenarios.json")
		if err != nil {
			panic(err)
		}
		defer f.Close()
		if err := json.NewDecoder(f).Decode(&scenarios); err != nil {
			panic(err)
		}
	}
	return scenarios, nil
}

func iterateCallFrames(call *jsonrpc.CallFrame, callback func(*jsonrpc.CallFrame)) {
	callback(call)
	for _, c := range call.Calls {
		iterateCallFrames(&c, callback)
	}
}
