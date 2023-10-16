package utils

import (
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// RemoveLeadingZerosFromHash remove the leading 0 in hash
// fix "invalid argument 2: hex number with leading zero digits" error
func RemoveLeadingZerosFromHash(h common.Hash) string {
	return "0x" + strings.TrimLeft(strings.TrimPrefix(h.Hex(), "0x"), "0")
}
