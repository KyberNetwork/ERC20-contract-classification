package classifier

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"erc20-contract-classification/pkg/classifier/jsonrpc"
	"erc20-contract-classification/pkg/types"
)

type Classifier struct {
	probe     *Probe
	client    *rpc.Client
	ethClient *ethclient.Client
}

func NewClassifier(rpcClient *rpc.Client, erc20balanceSlotProbe *Probe) *Classifier {
	return &Classifier{
		probe:     erc20balanceSlotProbe,
		client:    rpcClient,
		ethClient: ethclient.NewClient(rpcClient),
	}
}

type BalanceDiff struct {
	Address common.Address
	Before  *big.Int
	After   *big.Int
}

type StateChanges struct {
	Contract *BalanceDiff
	From     *BalanceDiff
	To       *BalanceDiff
}

// getTxsWithTransferLog will return a list of tx with transfer log from a contract
// TODO: wait for data team to provide this
func getTxsWithTransferLog(contract common.Address) {

}

func (c *Classifier) ReadSlotStorage(txs []*types.TxFromTransferEvent, contractAddr common.Address) (balanceSlot map[common.Address]common.Hash) {
	var (
		balanceSlotMap map[common.Address]common.Hash
	)
	balanceSlotMap[contractAddr] = common.Hash{}

	for _, t := range txs {
		balanceSlotMap[t.From] = common.Hash{}
		balanceSlotMap[t.To] = common.Hash{}
	}

	for address, _ := range balanceSlotMap {
		slot, err := c.probe.ProbeBalanceSlot(address)
		if err != nil {
			logger.Warnw("failed to probe balance slot", "address", address, "error", err)
			continue
		}
		balanceSlotMap[address] = slot
	}
	return balanceSlotMap
}

func (c *Classifier) TraceCallAndGetBalance(contractAddress common.Address, txs []*types.TxFromTransferEvent, balanceSlotMap map[common.Address]common.Hash) (map[common.Hash]*StateChanges, error) {
	var (
		results = make(map[common.Hash]*StateChanges, len(txs))
	)
	contract, avail := balanceSlotMap[contractAddress]
	if !avail {
		return nil, errors.New("no contract storage slot available")
	}
	for _, tx := range txs {
		from, avail := balanceSlotMap[tx.From]
		if !avail {
			continue
		}
		to, avail := balanceSlotMap[tx.To]
		if !avail {
			continue
		}

		var opsResult tracingResult
		err := jsonrpc.DebugTraceTransaction(
			c.client,
			tx.TxHash,
			nil,
			opsResult,
		)
		if err != nil {
			logger.Warnw("failed to to call json rpc ")
		}
		sd := &StateChanges{
			Contract: &BalanceDiff{
				Address: contractAddress,
				Before:  nil,
				After:   nil,
			},
			From: &BalanceDiff{
				Address: tx.From,
				Before:  nil,
				After:   nil,
			},
			To: &BalanceDiff{
				Address: tx.To,
				Before:  nil,
				After:   nil,
			},
		}
		if eErr := extractBalance(opsResult, sd, from, to, contract); eErr != nil {
			logger.Warnw("cannot extract balance", "error", eErr)
			continue
		}

		results[tx.TxHash] = sd
	}
	return results, nil
}

func extractBalance(opsResult tracingResult, sd *StateChanges, from, to, contract common.Hash) error {
	for _, op := range opsResult.ops {
		decoded, err := hexutil.Decode(op.Value)
		if err != nil {
			return err
		}
		if op.Op == vm.SLOAD {
			switch op.Address {
			case from.String():
				if sd.From.Before != nil {
					logger.Warnw("sd.From.Before is already set", "current value", sd.From.Before.String())
				}
				sd.From.Before = big.NewInt(0).SetBytes(decoded)
			case to.String():
				if sd.To.Before != nil {
					logger.Warnw("sd.To.Before is already set", "current value", sd.To.Before.String())
				}
				sd.To.Before = big.NewInt(0).SetBytes(decoded)
			case contract.String():
				if sd.Contract.Before != nil {
					logger.Warnw("sd.Contract.Before is already set", "current value", sd.Contract.Before.String())
				}
				sd.Contract.Before = big.NewInt(0).SetBytes(decoded)
			}
		}
		if op.Op == vm.SLOAD {
			switch op.Address {
			case from.String():
				if sd.From.After != nil {
					logger.Warnw("sd.From.After is already set", "current value", sd.From.After.String())
				}
				sd.From.After = big.NewInt(0).SetBytes(decoded)
			case to.String():
				if sd.To.After != nil {
					logger.Warnw("sd.To.After is already set", "current value", sd.To.After.String())
				}
				sd.To.After = big.NewInt(0).SetBytes(decoded)
			case contract.String():
				if sd.Contract.After != nil {
					logger.Warnw("sd.Contract.After is already set", "current value", sd.Contract.After.String())
				}
				sd.Contract.After = big.NewInt(0).SetBytes(decoded)
			}
		}
	}

	return nil
}
func (c *Classifier) IsFeeOnTransfer(ercContract common.Address) {

	// filter for transfer tx

	// find slotstorage from & ti

	// trace call get balance before after

}
