// io_config.go holds ConfigStore, the slice's config.yaml filesystem pipe (module-tree.md §6).
// io: none — a pure pipe, no validation, not unit-tested (its failure branch is component
// scenario 2). Validation lives in NewConfig (logic.go, ticket 07), which alone turns RawConfig
// into a valid-by-construction Config.
package validate

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ConfigStore reads and decodes config.yaml from the local filesystem. Deps = — (OS filesystem
// encapsulated in the zero-value object; no fields).
type ConfigStore struct{}

// Load is the pipe (contracts.md §ConfigStore.Load): reads path's bytes and decodes them
// (YAML/JSON — yaml.v3 accepts both, JSON being a YAML subset) into RawConfig.
//
// Antecedent: —.
// Consequent: success -> RawConfig, nil; failure (missing/unreadable/undecodable) ->
// RawConfig{}, ErrConfigInvalid (CONFIG_ERROR).
func (s ConfigStore) Load(path string) (RawConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RawConfig{}, fmt.Errorf("%w: read %s: %v", ErrConfigInvalid, path, err)
	}

	var raw RawConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return RawConfig{}, fmt.Errorf("%w: decode %s: %v", ErrConfigInvalid, path, err)
	}

	return raw, nil
}
