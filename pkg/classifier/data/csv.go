package data

import (
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gocarina/gocsv"

	"github.com/KyberNetwork/erc20-contract-classification/pkg/types"
)

func ReadDataFromCSV(csv_file string, contractAddress common.Address) ([]*types.TxFromTransferEvent, error) {
	f, err := os.Open(csv_file)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	var (
		transfers []*types.TxFromTransferEvent
	)

	if err := gocsv.UnmarshalFile(f, &transfers); err != nil {
		return nil, err
	}
	return transfers, nil
}
