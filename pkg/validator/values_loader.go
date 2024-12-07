package validator

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ValuesLoader struct{}

func NewValuesLoader() *ValuesLoader {
	return &ValuesLoader{}
}

// LoadAndMergeValues loads and merges multiple values files
func (l *ValuesLoader) LoadAndMergeValues(valuesFiles []string) (map[string]any, error) {
	mergedValues := make(map[string]any)

	for _, path := range valuesFiles {
		values, err := l.loadValuesFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load values from %s: %v", path, err)
		}
		mergedValues = l.mergeValues(mergedValues, values)
	}

	return mergedValues, nil
}

// loadValuesFile loads a single values file
func (l *ValuesLoader) loadValuesFile(path string) (map[string]any, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read values file: %v", err)
	}

	var values map[string]any
	if err := yaml.Unmarshal(content, &values); err != nil {
		return nil, fmt.Errorf("failed to parse values file: %v", err)
	}

	return values, nil
}

// mergeValues deeply merges two value maps
func (l *ValuesLoader) mergeValues(base, overlay map[string]any) map[string]any {
	result := make(map[string]any)

	// Copy base values
	for k, v := range base {
		result[k] = v
	}

	// Merge overlay values
	for k, v := range overlay {
		// If both maps have the same key and both values are maps, merge recursively
		if baseVal, ok := result[k]; ok {
			if baseMap, isBaseMap := baseVal.(map[string]any); isBaseMap {
				if overlayMap, isOverlayMap := v.(map[string]any); isOverlayMap {
					result[k] = l.mergeValues(baseMap, overlayMap)
					continue
				}
			}
		}
		// Otherwise, overlay value takes precedence
		result[k] = v
	}

	return result
}
