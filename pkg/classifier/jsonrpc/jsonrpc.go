package jsonrpc

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

// EthCallCalldataParam eth_call's calldata param
type EthCallCalldataParam struct {
	From string `json:"from"`
	To   string `json:"to"`
	Gas  string `json:"gas,omitempty"`
	Data string `json:"data"`
}

// EthCall eth_call wrapper
func EthCall(client *rpc.Client, calldata *EthCallCalldataParam, blockNumber string, override StateOverride) (*string, error) {
	resultHex := new(string)
	args := []interface{}{calldata, blockNumber}
	if override != nil {
		args = append(args, override)
	}
	err := client.Call(resultHex, "eth_call", args...)
	if err != nil {
		return nil, err
	}
	return resultHex, nil
}

// DebugTraceCallCalldataParam debug_traceCall's calldata param
type DebugTraceCallCalldataParam struct {
	From string `json:"from"`
	To   string `json:"to"`
	Gas  string `json:"gas,omitempty"`
	Data string `json:"data"`
}

// DebugTraceCallTracerConfigParam debug_traceCall's tracer config param
type DebugTraceCallTracerConfigParam struct {
	Tracer       string          `json:"tracer"`
	TracerConfig json.RawMessage `json:"tracerConfig"`
}

// DebugTraceCall debug_traceCall wrapper
func DebugTraceCall(
	client *rpc.Client,
	calldata *DebugTraceCallCalldataParam,
	blockNumber string,
	tracer *DebugTraceCallTracerConfigParam,
	result interface{},
) error {
	err := client.Call(result, "debug_traceCall", calldata, blockNumber, tracer)
	return err
}

//type DebugTraceTransactionConfigParam struct {
//	disableStorage   bool   `json:"disableStorage"`
//	disableStack     bool   `json:"disableStack"`
//	enableMemory     bool   `json:"enableMemory"`
//	enableReturnData bool   `json:"enableReturnData"`
//	Tracer           string `json:"tracer"`
//}

func DebugTraceTransaction(
	client *rpc.Client,
	txHash common.Hash,
	tracer *DebugTraceCallTracerConfigParam,
	result interface{},
) error {
	return client.Call(result, "debug_traceTransaction", txHash, tracer)
}

// OverrideAccount similar to ethapi.OverrideAccount
type OverrideAccount struct {
	Nonce     *hexutil.Uint64             `json:"nonce,omitempty"`
	Code      hexutil.Bytes               `json:"code,omitempty"`
	Balance   *hexutil.Big                `json:"balance,omitempty"`
	State     map[common.Hash]common.Hash `json:"state,omitempty"`
	StateDiff map[common.Hash]common.Hash `json:"stateDiff,omitempty"`
}

// StateOverride similar to ethapi.StateOverride
type StateOverride = map[common.Address]OverrideAccount

// CallLog similar to eth/tracers/native.callLog
type CallLog struct {
	Address common.Address `json:"address"`
	Topics  []common.Hash  `json:"topics"`
	Data    hexutil.Bytes  `json:"data"`
}

// CallFrame similar to eth/tracers/native.callFrame
type CallFrame struct {
	Type         string          `json:"type"`
	From         common.Address  `json:"from"`
	Gas          hexutil.Uint64  `json:"gas"`
	GasUsed      hexutil.Uint64  `json:"gasUsed"`
	To           *common.Address `json:"to,omitempty"`
	Input        hexutil.Bytes   `json:"input"`
	Output       hexutil.Bytes   `json:"output,omitempty"`
	Error        string          `json:"error,omitempty"`
	RevertReason string          `json:"revertReason,omitempty"`
	Calls        []CallFrame     `json:"calls,omitempty"`
	Logs         []CallLog       `json:"logs,omitempty"`
	Value        *hexutil.Big    `json:"value,omitempty"`
}
