// Package updates resolves value sources and applies replacements to XML files.
package updates

import (
	"fmt"
	"maps"
	"os"
	"slices"

	"configarr/internal/xmlconfig"
)

// Inputs contains the dynamic fields validated by the command-line library.
type Inputs struct {
	// Paths maps update names to required XML selectors.
	Paths map[string]string
	// Values contains explicitly supplied literal values, including empty strings.
	Values map[string]string
	// ValueFromEnv maps update names to environment variable names.
	ValueFromEnv map[string]string
}

// Resolve converts validated fields into replacements in ascending update-name order.
func Resolve(inputs Inputs) ([]xmlconfig.Update, error) {
	keys := maps.Keys(inputs.Paths)
	ids := slices.Collect(keys)
	slices.Sort(ids)

	var result []xmlconfig.Update
	for _, id := range ids {
		value, err := inputs.resolveValue(id)
		if err != nil {
			return nil, err
		}

		result = append(result, xmlconfig.Update{ID: id, Path: inputs.Paths[id], Value: value})
	}

	return result, nil
}

// resolveValue reads an environment-backed value or returns the validated literal source.
func (inputs Inputs) resolveValue(id string) (string, error) {
	envName, fromEnv := inputs.ValueFromEnv[id]
	if fromEnv {
		return environmentValue(id, envName)
	}

	return inputs.Values[id], nil
}

// environmentValue reads a named variable while distinguishing empty from unset values.
func environmentValue(id, name string) (string, error) {
	value, exists := os.LookupEnv(name)
	if !exists {
		return "", fmt.Errorf("update %s references an unset environment variable", id)
	}

	return value, nil
}
