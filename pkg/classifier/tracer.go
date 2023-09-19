package classifier

import (
	"bytes"
	_ "embed"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/tdewolff/minify/v2/js"
)

//go:embed storeTracer.js
var storageTracer []byte

var storageTracerMinified []byte

type StorageTracingResult struct {
	Op      vm.OpCode `json:"op"`
	Address string    `json:"addr"`
	Slot    string    `json:"slot"`
	Value   string    `json:"value"`
}

type tracingResult struct {
	ops    []StorageTracingResult `json:"ops"`
	Output string                 `json:"output"`
}

func init() {
	// we need to minify the tracer script because we can not put multipleline string in JSON value
	minified := new(bytes.Buffer)
	err := js.Minify(nil, minified, bytes.NewReader(storageTracer), nil)
	if err != nil {
		panic(err)
	}
	storageTracerMinified = bytes.TrimPrefix(minified.Bytes(), []byte("var tracer="))
}
