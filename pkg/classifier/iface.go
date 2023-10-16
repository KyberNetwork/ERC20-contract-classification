package classifier

import "github.com/ethereum/go-ethereum/common"

// Classifier define required functionalities for a classifier.
type Classifier interface {
	IsFeeOnTransfer(ercContract common.Address) (bool, error)
}
