package api

import (
	"os"
	"sync"

	"github.com/CosmWasm/wasmd/wasmvm/v2/types"
)

// Cache defines the interface for caching Wasm code
type Cache interface {
	GetCode(checksum []byte) ([]byte, error)
	StoreCode(code []byte) ([]byte, error)
	StoreCodeUnchecked(code []byte) ([]byte, error)
	Pin(checksum []byte) error
	Unpin(checksum []byte) error
	GetMetrics() (*types.Metrics, error)
	GetPinnedMetrics() (*types.PinnedMetrics, error)
}

// WasmCache represents a wazero runtime and module cache
type WasmCache struct {
	dir      string
	modules  sync.Map // map[string][]byte
	wasmCode sync.Map // map[string][]byte
	lockfile *os.File
}
