Minimized inventory from the actual refused v0.3.4 Go SDK (SHA-256 `51010c81a072d85d9176b275b61736decea7a986a5cfcc5f1b0c957b44ee0c4c`). Only fields used for manifest relationships remain; this fixture is not the complete producer report or admission proof. `go.mod` preserves the exact delivered bytes.

`observed-inventory.json` retains the identities, metadata, hashes and dependency
edges from the subsequent actual same-byte Syft 1.51.0 scan. It includes eight
additional modules in the scanner's resolved graph. Irrelevant component fields
are omitted; this fixture is not the raw producer receipt or promotion proof.
