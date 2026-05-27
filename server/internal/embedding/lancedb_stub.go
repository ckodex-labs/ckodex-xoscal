//go:build !lancedb

package embedding

import (
	"fmt"

	"github.com/mchorfa/xoscal/server/internal/config"
)

// NewLanceDBVectorStore returns an error when LanceDB support is not compiled in.
// Build with -tags lancedb to enable LanceDB (requires CGO + Rust toolchain).
func NewLanceDBVectorStore(cfg config.Vector) (VectorStore, error) {
	return nil, fmt.Errorf("lancedb support not compiled in; build with -tags lancedb (requires CGO + Rust toolchain)")
}
