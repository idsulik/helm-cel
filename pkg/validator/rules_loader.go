package validator

import (
	"fmt"
	"os"

	"github.com/idsulik/helm-cel/pkg/models"
	"gopkg.in/yaml.v3"
)

type RulesLoader struct{}

func NewRulesLoader() *RulesLoader {
	return &RulesLoader{}
}

// LoadAndMergeRules loads and merges rules from multiple files
func (l *RulesLoader) LoadAndMergeRules(rulesFiles []string) (*models.ValidationRules, error) {
	mergedRules := &models.ValidationRules{
		Rules:       make([]models.Rule, 0),
		Expressions: make(map[string]string),
	}

	for _, path := range rulesFiles {
		rules, err := l.loadRulesFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load rules from %s: %v", path, err)
		}

		// Merge rules
		mergedRules.Rules = append(mergedRules.Rules, rules.Rules...)

		// Merge expressions, checking for duplicates
		for k, v := range rules.Expressions {
			if existing, ok := mergedRules.Expressions[k]; ok {
				return nil, fmt.Errorf(
					"duplicate named expression '%s' found in %s (already defined as '%s')",
					k,
					path,
					existing,
				)
			}
			mergedRules.Expressions[k] = v
		}
	}

	return mergedRules, nil
}

// loadRulesFile loads validation rules from a specific file
func (l *RulesLoader) loadRulesFile(path string) (*models.ValidationRules, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules file: %v", err)
	}

	var rules models.ValidationRules
	if err := yaml.Unmarshal(content, &rules); err != nil {
		return nil, fmt.Errorf("failed to parse rules file: %v", err)
	}

	return &rules, nil
}
