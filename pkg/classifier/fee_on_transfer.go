package classifier

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"erc20-contract-classification/pkg/classifier/jsonrpc"
)

type TxFromTransferEvent struct {
	From         common.Address
	To           common.Address
	ContractAddr common.Address
	TxHash       common.Hash
	Amount       *big.Int
}

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

func (c *Classifier) ReadSlotStorage(txs []*TxFromTransferEvent) (balanceSlot map[common.Address]common.Hash) {
	var (
		balanceSlotMap map[common.Address]common.Hash
	)

	for _, t := range txs {
		balanceSlotMap[t.ContractAddr] = common.Hash{}
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

func (c *Classifier) TraceCallAndGetBalance(txs []*TxFromTransferEvent, balanceSlotMap map[common.Address]common.Hash) {
	var (
		results = make(map[common.Hash]*StateChanges, len(txs))
	)
	for _, tx := range txs {
		var result interface{}
		err := jsonrpc.DebugTraceTransaction(
			c.client,
			tx.TxHash,
			nil,
			result,
		)
		if err != nil {
			logger.Warnw("failed to to call json rpc ")
		}
		// TODO: print out all state diffs based on balanceSlotMap in the form BalanceDiff
		results[tx.TxHash] = &StateChanges{
			Contract: nil,
			From:     nil,
			To:       nil,
		}
	}
}

func (c *Classifier) IsFeeOnTransfer(ercContract common.Address) {

	// filter for transfer tx

	// find slotstorage from & ti

	// trace call get balance before after

}
