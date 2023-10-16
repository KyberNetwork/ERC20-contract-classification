package classifier

import (
	"context"
	"fmt"
	"math/big"

	"erc20-contract-classification/pkg/classifier/abis"
	"erc20-contract-classification/pkg/types"

	"github.com/sajari/regression"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type EventFilterClassifier struct {
	ethClient    *ethclient.Client
	TxsThreshold int //TxsThreshold is the limitation of tx we should get for historical txs
	RegressR2    float64
}

func NewEventFiterClassifier(rpcClient *rpc.Client, txsThreshold int, regressR2 float64) *EventFilterClassifier {
	return &EventFilterClassifier{
		ethClient:    ethclient.NewClient(rpcClient),
		TxsThreshold: txsThreshold,
		RegressR2:    regressR2,
	}
}

func (c *EventFilterClassifier) FetchTxAndEvents(contractAddress common.Address) map[common.Hash][]types.TxFromTransferEvent {
	eventSignature := abis.ERC20.Events["Transfer"].ID
	blockNumber, err := c.ethClient.BlockNumber(context.Background())
	if err != nil {
		logger.Errorw("could not get block number", "error", err)
		return nil
	}
	var (
		topics = [][]common.Hash{{eventSignature}}
		logs   []ethtypes.Log
	)
	for {
		var from uint64
		if blockNumber > 10000 {
			from = blockNumber - 10000
		}

		query := ethereum.FilterQuery{
			Addresses: []common.Address{contractAddress},
			Topics:    topics,
			FromBlock: big.NewInt(int64(from)),
			ToBlock:   big.NewInt(int64(blockNumber)),
		}
		newLogs, lerr := c.ethClient.FilterLogs(context.Background(), query)
		if err != nil {
			logger.Errorw("could not get event log", "error", lerr)
			break
		}
		if len(newLogs) == 0 {
			code, cerr := c.ethClient.CodeAt(context.Background(), contractAddress, big.NewInt(int64(from)))
			if cerr != nil {
				logger.Warn("cannot get code", "error", err, "block", from)
				break
			}
			if len(code) == 0 {
				logger.Warnw("no more tx to get")
				break
			}
		}
		logs = append(logs, newLogs...)
		if len(logs) >= c.TxsThreshold {
			break
		}
		blockNumber = from
	}

	var (
		txToEventMap = make(map[common.Hash][]types.TxFromTransferEvent, len(logs))
	)

	for _, vLog := range logs {
		//logger.Infow("data", "data", vLog.Data, "tx", vLog.TxHash, "topic", vLog.Topics)
		event, eerr := abis.ERC20.Unpack("Transfer", vLog.Data)
		if eerr != nil {
			logger.Errorw("could not unpack event log", "error", eerr, "tx", vLog.TxHash)
		}
		//logger.Infow("event", "event", event)
		tx := types.TxFromTransferEvent{
			From:   common.HexToAddress(vLog.Topics[1].Hex()),
			To:     common.HexToAddress(vLog.Topics[2].Hex()),
			TxHash: vLog.TxHash,
			Amount: event[0].(*big.Int),
		}
		if txToEventMap[vLog.TxHash] == nil {
			txToEventMap[vLog.TxHash] = []types.TxFromTransferEvent{
				tx,
			}
		} else {
			txToEventMap[vLog.TxHash] = append(txToEventMap[vLog.TxHash], tx)
		}
	}

	return txToEventMap
}

type Receiver struct {
	From   common.Address
	Amount *big.Int
	Tx     common.Hash
}

type Sender struct {
	To     common.Address
	Amount *big.Int
	Tx     common.Hash
}

// IsFeeOnTransfer implement token classifier for EventFilterClassifier
// on the rational that the patterns of multiple transfer event transmitted from a certain address
// in the same tx indicate that fee is being transfered somewhere.
func (c *EventFilterClassifier) IsFeeOnTransfer(ercContract common.Address) (bool, error) {
	txsFromEvents := c.FetchTxAndEvents(ercContract)
	type counter struct {
		address common.Address
		count   int
	}
	var (
		numTxsWithMultipleTransfer = 0
		multipleTransferReceivers  = make(map[common.Address]int, len(txsFromEvents)*3)
		multipleTransferSender     = make(map[common.Address]int, len(txsFromEvents)*3)
		mostReceved, mostSent      counter
	)

	for _, events := range txsFromEvents {
		if len(events) == 1 {
			continue
		}
		numTxsWithMultipleTransfer++
		prev := events[0]
		alreadyIncrease := false
		//logger.Infow("tx Hash", "hash", txHash)
		for i := 1; i < len(events); i++ {
			//look for 2 transfer consecutively with the same sender
			if events[i].From == prev.From {

				if !alreadyIncrease {
					multipleTransferReceivers[prev.To]++
					multipleTransferSender[prev.From]++
					alreadyIncrease = true
				}
				multipleTransferReceivers[events[i].To]++

				multipleTransferSender[events[i].From]++
			}
			prev = events[i]
			if mostReceved.count < multipleTransferReceivers[prev.To] {
				mostReceved = counter{
					address: prev.To,
					count:   multipleTransferReceivers[prev.To],
				}
			}
			if mostSent.count < multipleTransferSender[prev.From] {
				mostSent = counter{
					address: prev.From,
					count:   multipleTransferSender[prev.From],
				}
			}

			if mostReceved.count < multipleTransferReceivers[events[i].To] {
				mostReceved = counter{
					address: events[i].To,
					count:   multipleTransferReceivers[events[i].From],
				}
			}
			if mostSent.count < multipleTransferSender[events[i].From] {
				mostSent = counter{
					address: events[i].From,
					count:   multipleTransferSender[events[i].From],
				}
			}
		}

		//concern the most received and most sender from
	}

	logger.Infow("finished counting", "len(txs)", len(txsFromEvents), "txs with multiple transfers", numTxsWithMultipleTransfer, "most sent from", mostSent.address, "most received", mostReceved.address)

	var (
		supposedFeeReceived     = make([]float64, numTxsWithMultipleTransfer)
		supposedRealBenefactory = make([]float64, numTxsWithMultipleTransfer)
		index                   = 0
	)

	//prepare data for regression
	for _, events := range txsFromEvents {
		if len(events) == 1 {
			continue
		}
		prev := events[0]

		for i := 1; i < len(events); i++ {
			//look for 2 transfer consecutively with the same sender
			if events[i].From == prev.From {
				if prev.To == mostReceved.address {
					supposedFeeReceived[index], _ = prev.Amount.Float64()
					supposedRealBenefactory[index], _ = events[i].Amount.Float64()
				} else if events[i].To == mostReceved.address {
					supposedFeeReceived[index], _ = events[i].Amount.Float64()
					supposedRealBenefactory[index], _ = prev.Amount.Float64()
				}
			}
		}
		index++
	}
	r, err := regress(supposedFeeReceived, supposedRealBenefactory)
	if err != nil {
		logger.Infow("cannot regress from data", "error", err)
		return false, nil
	}
	if (numTxsWithMultipleTransfer/2 > mostReceved.count) && (numTxsWithMultipleTransfer < 100) {
		logger.Warnw("Prediction might not be corrected since there is not enough tx")
	}
	if r.R2 >= c.RegressR2 {
		return true, nil
	}

	return false, nil
}

func regress(supposedFee []float64, supposedReceived []float64) (*regression.Regression, error) {
	r := new(regression.Regression)
	r.SetObserved("correlation fee on transfer")
	r.SetVar(0, "real received")
	var data = make(regression.DataPoints, len(supposedFee))
	for i := 0; i < len(supposedFee); i++ {
		data[i] = regression.DataPoint(supposedFee[i], []float64{supposedReceived[i]})
	}
	r.Train(data...)
	if err := r.Run(); err != nil {
		return nil, err
	}
	fmt.Printf("Regression formula:\n%v\n", r.Formula)
	fmt.Printf("Regression R2:\n%f\n", r.R2)
	return r, nil
}
