package classifier

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
)

func TestClassifier_TraceCallAndGetBalance(t *testing.T) {
	type fields struct {
		probe     *Probe
		client    *rpc.Client
		ethClient *ethclient.Client
	}
	type args struct {
		contractAddress common.Address
		txs             []*TxFromTransferEvent
		balanceSlotMap  map[common.Address]common.Hash
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[common.Hash]*StateChanges
		wantErr assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Classifier{
				probe:     tt.fields.probe,
				client:    tt.fields.client,
				ethClient: tt.fields.ethClient,
			}
			got, err := c.TraceCallAndGetBalance(tt.args.contractAddress, tt.args.txs, tt.args.balanceSlotMap)
			if !tt.wantErr(t, err, fmt.Sprintf("TraceCallAndGetBalance(%v, %v, %v)", tt.args.contractAddress, tt.args.txs, tt.args.balanceSlotMap)) {
				return
			}
			assert.Equalf(t, tt.want, got, "TraceCallAndGetBalance(%v, %v, %v)", tt.args.contractAddress, tt.args.txs, tt.args.balanceSlotMap)
		})
	}
}
