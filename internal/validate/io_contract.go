// io_contract.go holds ContractStore, the slice's consumed-contract filesystem pipe
// (module-tree.md §6). io: none — a pure pipe, no validation, not unit-tested (its failure
// branch is component scenario 3). Validation lives in NewConsumedContract (logic.go, ticket
// 09), which alone turns RawContract into a valid-by-construction ConsumedContract.
package validate

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ContractStore reads and decodes the consumed-contract artifact from the local filesystem.
// Deps = — (OS filesystem encapsulated in the zero-value object; no fields).
type ContractStore struct{}

// Load is the pipe (contracts.md §ContractStore.Load): reads path's bytes and decodes them
// (YAML/JSON — yaml.v3 accepts both, JSON being a YAML subset) into RawContract. No validation
// here (that is NewConsumedContract).
//
// Antecedent: —.
// Consequent: success -> RawContract, nil; failure (missing/unreadable) -> RawContract{},
// ErrFileNotFound (FILE_NOT_FOUND).
func (s ContractStore) Load(path string) (RawContract, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RawContract{}, fmt.Errorf("%w: read %s: %v", ErrFileNotFound, path, err)
	}

	var raw RawContract
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return RawContract{}, fmt.Errorf("%w: decode %s: %v", ErrFileNotFound, path, err)
	}

	return raw, nil
}
