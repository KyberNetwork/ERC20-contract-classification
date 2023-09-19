package classifier

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/asm"
	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"

	"erc20-contract-classification/pkg/classifier/abis"
)

var logger *zap.SugaredLogger

func init() {
	l, err := zap.NewDevelopment()
	defer l.Sync() // flushes buffer, if any
	logger = l.Sugar()

	if err != nil {
		panic(err)
	}
}

func getMethodHash(methodSignature string) string {
	fullhash := crypto.Keccak256Hash([]byte(methodSignature))
	return fullhash.String()[0:10]
}

func hasSelector(instructions []string, contractABI abi.ABI) bool {
	var (
		hashes           = make(map[string]string, len(contractABI.Methods))
		availableMethods = make(map[string]bool, len(contractABI.Methods))
		count            = 0
	)
	for i, method := range contractABI.Methods {
		hashes[i] = getMethodHash(method.Sig)
		logger.Infow("method", "method", method.Sig, "hash", hashes[i])
	}
	for _, ins := range instructions {
		for _, methodHash := range hashes {
			if strings.Contains(ins, methodHash) {
				availableMethods[methodHash] = true
			}

		}
	}

	for _, methodHash := range hashes {
		if availableMethods[methodHash] {
			count++
		}
	}
	logger.Infow("finish looking for method hash..", "availableMethodCount", count)
	return count == len(hashes)
}

// DisassembleWithTolerance returns all disassembled EVM instructions in human-readable format.
func DisassembleWithTolerance(script []byte) []string {
	instrs := make([]string, 0)

	it := asm.NewInstructionIterator(script)
	for it.Next() {
		if it.Arg() != nil && 0 < len(it.Arg()) {
			instrs = append(instrs, fmt.Sprintf("%05x: %v %#x\n", it.PC(), it.Op(), it.Arg()))
		} else {
			instrs = append(instrs, fmt.Sprintf("%05x: %v\n", it.PC(), it.Op()))
		}
	}
	return instrs
}

func IsErc20(bytecode []byte) (bool) {

	data := DisassembleWithTolerance(bytecode)

	for i, oc := range data {
		logger.Infow("opcode", "pc", i, "oc", oc)
	}
	if hasSelector(data, abis.ERC20) {
		return true
	}
	return false
}
