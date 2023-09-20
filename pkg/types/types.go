package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type TransferRecord struct {
	Sender      common.Address `csv:"sender_address"`
	Receiver    common.Address `csv:"receiver_address"`
	TxHash      common.Hash    `csv:"tx_hash"`
	TotalAmount *big.Int       `csv:"total_tokens_transferred"`
}

type TxFromTransferEvent struct {
	From   common.Address `csv:"sender_address"`
	To     common.Address `csv:"receiver_address"`
	TxHash common.Hash    `csv:"tx_hash"`
	Amount *big.Int       `csv:"total_tokens_transferred"`
}
