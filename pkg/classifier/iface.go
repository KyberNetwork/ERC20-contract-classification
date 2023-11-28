package classifier

import (
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

// FeeOnTransferResult store the fee on transfer classification result
type FeeOnTransferResult struct {
	//IsFeeOnTransfer set to true if the contract induce fee on transfer.
	IsFeeOnTransfer bool
	//FeeReceiver set to the address of the one to receive fee. Zero address meaning the fee is burnt
	FeeReceiver common.Address
	//Coefficients is the list of coefficient in fee formular
	// For now we're using linear re-geression hence the coefficient is at the form
	// fee= Coefficients[0]*amountIn + Coefficients[1]
	Coefficients []float64
	//Formular is the string representation of  the fee formular
	Formular string
}

// Classifier define required functionalities for a classifier.
type Classifier interface {
	// IsFeeOnTransfer returns if the contract is fee on transfer and its fomular
	// if codes[] is nil, the classifier will have to go fetch it
	// if []logs is nil, the classifier will have to go fetch it based on configuraion
	IsFeeOnTransfer(ercContract common.Address, logs []ethtypes.Log) (FeeOnTransferResult, error)
	// IsErc20 returns if the contract is fee on transfer and its fomular
	// if codes[] is nil, the classifier will have to go fetch it
	IsErc20(ercContract common.Address, codes []byte) bool
}
