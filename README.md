# ERC20-contract-classification

The ERC20 Contract Classification library is a Go package that provides a simple interface for classifying Ethereum ERC20 contracts. It allows users to determine whether a given contract is a fee-on-transfer contract and obtain information about the fee-on-transfer formula. Additionally, it provides a method to check if a contract is a standard ERC20 contract.

## Installation
To use this library in your Go project, you can simply install it using the following go get command:

```bash
go get github.com/KyberNetwork/erc20-contract-classification
```

## Usage
Import the library in your Go code:

```go
import "github.com/your-username/ERC20-contract-classification"
```

Initialize the classifier: for now the classifer we're going to use is EventFilter (https://www.notion.so/kybernetwork/Transfer-event-based-heuristic-approach-40bcf7e9fa55404884984395267d5263). 

```go
	// numTx is the minimum number of Tx to get before classification. Higher numTx will take longer to run but will result in a finer result 
	// regressR2 is the threshold R2 in regression of which the contract is recognized as fot. Default is 0.9
	c := NewEventFiterClassifier(rpcClient, numTx, regressR2)
```

Example: Check if a Contract is Fee-On-Transfer

```go
ercContract := common.HexToAddress("0x123456789abcdef123456789abcdef123456789a")

//if logs == nil, the program will automatically fetch required log
result, err := classifier.IsFeeOnTransfer(ercContract, logs)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Is Fee-On-Transfer: %t\n", result.IsFeeOnTransfer)
if result.IsFeeOnTransfer {
    fmt.Printf("Fee Formula: %s\n", result.FeeFormula)
}
```

Example: Check if a Contract is ERC20
```go
ercContract := common.HexToAddress("0x123456789abcdef123456789abcdef123456789a")


//if codes == nil, the program will automatically fetch required codes
isERC20 := classifier.IsErc20(ercContract, codes)
fmt.Printf("Is ERC20: %t\n", isERC20)
```
