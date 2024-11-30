package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/idsulik/helm-cel/pkg/models"
	"gopkg.in/yaml.v3"
)

// Generator handles the generation of CEL validation rules
type Generator struct{}

// New creates a new Generator instance
func New() *Generator {
	return &Generator{}
}

// GenerateRules generates validation rules for a chart
func (g *Generator) GenerateRules(chartPath, valuesFile string) (*models.ValidationRules, error) {
	valuesPath := filepath.Join(chartPath, valuesFile)
	content, err := os.ReadFile(valuesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read values file %s: %v", valuesFile, err)
	}

	var values map[string]interface{}
	if err := yaml.Unmarshal(content, &values); err != nil {
		return nil, fmt.Errorf("failed to parse values file %s: %v", valuesFile, err)
	}

	rules := &models.ValidationRules{
		Rules:       make([]models.Rule, 0),
		Expressions: make(map[string]string),
	}

	g.generateRulesForMap("values", values, rules)

	return rules, nil
}

func (g *Generator) generateRulesForMap(prefix string, m map[string]interface{}, rules *models.ValidationRules) {
	for k, v := range m {
		path := fmt.Sprintf("%s.%s", prefix, k)

		switch val := v.(type) {
		case map[string]interface{}:
			rules.Rules = append(
				rules.Rules, models.Rule{
					Expr: fmt.Sprintf("has(%s)", path),
					Desc: fmt.Sprintf("%s must be defined", k),
				},
			)
			g.generateRulesForMap(path, val, rules)

		case []interface{}:
			rules.Rules = append(
				rules.Rules, models.Rule{
					Expr: fmt.Sprintf("size(%s) >= 0", path),
					Desc: fmt.Sprintf("%s must be an array", k),
				},
			)

			// If array is not empty, check first element type for additional rules
			if len(val) > 0 {
				if firstElement, ok := val[0].(map[string]interface{}); ok {
					// For each field in the first element, generate rules as if it were a map
					// but use array index notation
					for fieldKey, fieldValue := range firstElement {
						elementPath := fmt.Sprintf("%s[0].%s", path, fieldKey)
						g.generateRulesForValue(elementPath, fieldValue, fieldKey, rules)
					}
				} else {
					// Handle scalar array elements
					g.generateRulesForValue(path+"[0]", val[0], k, rules)
				}
			}

		default:
			g.generateRulesForValue(path, v, k, rules)
		}
	}
}
func (g *Generator) generateRulesForValue(path string, v interface{}, key string, rules *models.ValidationRules) {
	switch v.(type) {
	case string:
		rules.Rules = append(
			rules.Rules, models.Rule{
				Expr: fmt.Sprintf("type(%s) == string", path),
				Desc: fmt.Sprintf("%s must be a string", key),
			},
		)
		if g.isResource(key) {
			rules.Rules = append(rules.Rules, g.generateResourceRule(path, key))
		}

	case float64:
		rules.Rules = append(
			rules.Rules, models.Rule{
				Expr: fmt.Sprintf("type(%s) == int || type(%s) == double", path, path),
				Desc: fmt.Sprintf("%s must be a number", key),
			},
		)

	case bool:
		rules.Rules = append(
			rules.Rules, models.Rule{
				Expr: fmt.Sprintf("type(%s) == bool", path),
				Desc: fmt.Sprintf("%s must be a boolean", key),
			},
		)

	case int:
		rules.Rules = append(
			rules.Rules, models.Rule{
				Expr: fmt.Sprintf("type(%s) == int", path),
				Desc: fmt.Sprintf("%s must be an integer", key),
			},
		)
		if g.isPort(key, v.(int)) {
			rules.Rules = append(rules.Rules, g.generatePortRule(path, key))
		}
	}
}

func (g *Generator) isPort(key string, value int) bool {
	if value < 1 || value > 65535 {
		return false
	}

	return strings.Contains(strings.ToLower(key), "port")
}

func (g *Generator) isResource(key string) bool {
	key = strings.ToLower(key)
	return key == "cpu" || key == "memory" || strings.HasSuffix(key, ".cpu") || strings.HasSuffix(key, ".memory")
}

func (g *Generator) generatePortRule(path, key string) models.Rule {
	return models.Rule{
		Expr: fmt.Sprintf("%s >= 1 && %s <= 65535", path, path),
		Desc: fmt.Sprintf("%s must be a valid port number (1-65535)", key),
	}
}

func (g *Generator) generateResourceRule(path, key string) models.Rule {
	return models.Rule{
		Expr: fmt.Sprintf("%s.matches('^[0-9]+(.[0-9]+)?(m|Mi|Gi|Ti|Pi|Ei|n|u|m|k|M|G|T|P|E)?$')", path),
		Desc: fmt.Sprintf("%s must be a valid resource quantity", key),
	}
}

// WriteRules writes the generated rules to a file
func (g *Generator) WriteRules(path string, rules *models.ValidationRules) error {
	content, err := yaml.Marshal(rules)
	if err != nil {
		return fmt.Errorf("failed to marshal rules: %v", err)
	}

	return os.WriteFile(path, content, 0644)
}
