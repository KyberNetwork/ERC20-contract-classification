package classifier

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"

	"erc20-contract-classification/pkg/classifier/data"
	"erc20-contract-classification/pkg/types"
)

const (
	rpcURL   = "" // CHANGE ME
	csv_file = "../../erc20_transfer_tx.csv"
)

var (
	contractAddress = common.HexToAddress("0x36e6309aa7a923fb111ae50b56bfb3cfb2256f89")
)

func TestClassifier_TraceCallAndGetBalance(t *testing.T) {
	//we won't be able to test this unless we got a custom node.
	t.Skip()
	type fields struct {
		probe     *Probe
		client    *rpc.Client
		ethClient *ethclient.Client
	}
	type args struct {
		contractAddress common.Address
		txs             []*types.TxFromTransferEvent
		balanceSlotMap  map[common.Address]common.Hash
	}

	rpcClient, err := rpc.Dial(rpcURL)
	erc20BalanceSlotProbe := NewProbe(rpcClient)
	c := NewClassifier(rpcClient, erc20BalanceSlotProbe)

	input, err := data.ReadDataFromCSV(csv_file, contractAddress)
	assert.NoError(t, err)
	balanceSlotsMap := c.ReadSlotStorage(input, contractAddress)
	stateChanges, err := c.TraceCallAndGetBalance(contractAddress, input, balanceSlotsMap)
	assert.NoError(t, err)

	for txHash, sc := range stateChanges {
		logger.Info("State changes", "tx", txHash, "state change", sc)
		break
	}
}

func TestEventFilterClassifier_FetchTxAndEvents(t *testing.T) {

	type args struct {
		contractAddress common.Address
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "emoticon address",
			args: args{contractAddress: common.HexToAddress("0x9b0e1c344141fb361b842d397df07174e1cdb988")},
		},
	}

	rpcClient, err := rpc.Dial(rpcURL)
	assert.NoError(t, err)
	var counter int
	c := NewEventFiterClassifier(rpcClient, 1000, 0.9)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := c.FetchTxAndEvents(tt.args.contractAddress)
			for txhash, txs := range res {
				if len(txs) == 1 {
					continue
				}
				counter++
				fmt.Println("tx hash", txhash)
				for _, tx := range txs {
					fmt.Printf("-- From: %s, To: %s, Value: %s\n", tx.From.Hex(), tx.To.Hex(), tx.Amount.String())
				}
			}
			fmt.Printf("Got %d tx, %d with multiple transfer \n", len(res), counter)

		})
	}
}

func TestEventFilterClassifier_IsFeeOnTransfer(t *testing.T) {

	type args struct {
		ercContract common.Address
	}
	const (
		numTx     = 10000
		regressR2 = 0.9
	)
	rpcClient, err := rpc.Dial(rpcURL)
	assert.NoError(t, err)
	c := NewEventFiterClassifier(rpcClient, numTx, regressR2)
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "emoticon address",
			args:    args{ercContract: common.HexToAddress("0x9b0e1c344141fb361b842d397df07174e1cdb988")},
			want:    true,
			wantErr: assert.NoError,
		},
		{
			name:    "XRP20Token",
			args:    args{ercContract: common.HexToAddress("0xe4ab0be415e277d82c38625b72bd7dea232c2e7d")},
			want:    true,
			wantErr: assert.NoError,
		},
		{
			name:    "doglord address",
			args:    args{ercContract: common.HexToAddress("0x6580685617a8721df77ca42a08e7b1d58da79cf9")},
			want:    true,
			wantErr: assert.NoError,
		}, {
			name:    "normal address",
			args:    args{ercContract: common.HexToAddress("0x04c17b9d3b29a78f7bd062a57cf44fc633e71f85")},
			want:    false,
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.IsFeeOnTransfer(tt.args.ercContract)
			if !tt.wantErr(t, err, fmt.Sprintf("IsFeeOnTransfer(%v)", tt.args.ercContract)) {
				return
			}
			assert.Equalf(t, tt.want, got, "IsFeeOnTransfer(%v)", tt.args.ercContract)
		})
	}
}
