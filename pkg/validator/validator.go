package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/cel-go/cel"
	"gopkg.in/yaml.v3"
)

// Rule represents a single CEL validation rule
type Rule struct {
	Expr string `yaml:"expr"`
	Desc string `yaml:"desc"`
}

// ValidationRules contains all CEL validation rules
type ValidationRules struct {
	Rules []Rule `yaml:"rules"`
}

// Validator handles the validation of Helm values using CEL
type Validator struct {
	env *cel.Env
}

// New creates a new Validator instance
func New() *Validator {
	return &Validator{}
}

// ValidateChart validates the values.yaml file against CEL rules in values.cel.yaml
func (v *Validator) ValidateChart(chartPath string) error {
	// Initialize CEL environment
	env, err := v.initCelEnv()
	if err != nil {
		return fmt.Errorf("failed to initialize CEL environment: %v", err)
	}

	v.env = env
	values, err := v.loadValues(chartPath)
	if err != nil {
		return err
	}

	rules, err := v.loadRules(chartPath)
	if err != nil {
		return err
	}

	return v.validateRules(values, rules)
}

func (v *Validator) initCelEnv() (*cel.Env, error) {
	return cel.NewEnv(
		cel.Variable("values", cel.DynType),
	)
}

func (v *Validator) loadValues(chartPath string) (map[string]interface{}, error) {
	valuesPath := filepath.Join(chartPath, "values.yaml")
	valuesContent, err := os.ReadFile(valuesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read values.yaml: %v", err)
	}

	values := make(map[string]interface{})
	if err := yaml.Unmarshal(valuesContent, &values); err != nil {
		return nil, fmt.Errorf("failed to parse values.yaml: %v", err)
	}

	return values, nil
}

func (v *Validator) loadRules(chartPath string) (*ValidationRules, error) {
	rulesPath := filepath.Join(chartPath, "values.cel.yaml")
	rulesContent, err := os.ReadFile(rulesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read values.cel.yaml: %v", err)
	}

	var rules ValidationRules
	if err := yaml.Unmarshal(rulesContent, &rules); err != nil {
		return nil, fmt.Errorf("failed to parse values.cel.yaml: %v", err)
	}

	return &rules, nil
}

// ValidationError represents a validation failure
type ValidationError struct {
	Description string
	Expression  string
	Value       interface{}
	Path        string
}

func (e *ValidationError) Error() string {
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("❌ Validation failed: %s\n", e.Description))
	msg.WriteString(fmt.Sprintf("   Rule: %s\n", e.Expression))
	if e.Path != "" {
		msg.WriteString(fmt.Sprintf("   Path: %s\n", e.Path))
	}
	if e.Value != nil {
		msg.WriteString(fmt.Sprintf("   Current value: %v", e.Value))
	} else {
		msg.WriteString("   Current value: <nil>")
	}
	return msg.String()
}

// ValidationErrors holds multiple validation errors
type ValidationErrors struct {
	Errors []*ValidationError
}

func (ve *ValidationErrors) Error() string {
	if len(ve.Errors) == 0 {
		return ""
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("Found %d validation error(s):\n\n", len(ve.Errors)))
	for i, err := range ve.Errors {
		msg.WriteString(err.Error())
		if i < len(ve.Errors)-1 {
			msg.WriteString("\n\n")
		}
	}
	return msg.String()
}

func (v *Validator) validateRules(values map[string]interface{}, rules *ValidationRules) error {
	var validationErrors ValidationErrors

	for _, rule := range rules.Rules {
		ast, issues := v.env.Compile(rule.Expr)
		if issues != nil && issues.Err() != nil {
			return fmt.Errorf("❌ Invalid rule syntax in '%s': %v", rule.Desc, issues.Err())
		}

		program, err := v.env.Program(ast)
		if err != nil {
			return fmt.Errorf("❌ Failed to process rule '%s': %v", rule.Desc, err)
		}

		out, _, err := program.Eval(
			map[string]interface{}{
				"values": values,
			},
		)
		if err != nil {
			// Handle evaluation errors more gracefully
			validationErrors.Errors = append(
				validationErrors.Errors, &ValidationError{
					Description: rule.Desc,
					Expression:  rule.Expr,
					Path:        extractPath(err.Error()),
				},
			)
			continue
		}

		if out.Value() != true {
			// Try to extract relevant value based on the expression
			value, path := extractValueFromValues(values, rule.Expr)
			validationErrors.Errors = append(
				validationErrors.Errors, &ValidationError{
					Description: rule.Desc,
					Expression:  rule.Expr,
					Value:       value,
					Path:        path,
				},
			)
		}
	}

	if len(validationErrors.Errors) > 0 {
		return &validationErrors
	}
	return nil
}

// extractPath tries to extract the path from a CEL error message
func extractPath(errMsg string) string {
	// Common patterns in CEL error messages
	patterns := []string{
		"no such key: ",
		"undefined field '",
		"missing key ",
	}

	for _, pattern := range patterns {
		if idx := strings.Index(errMsg, pattern); idx != -1 {
			// Extract the path after the pattern
			path := errMsg[idx+len(pattern):]
			// Clean up the path
			path = strings.Trim(path, "'\"")
			path = strings.Split(path, " ")[0] // Take first word only
			return path
		}
	}
	return ""
}

// extractValueFromValues tries to extract the relevant value based on the CEL expression
func extractValueFromValues(values map[string]interface{}, expr string) (interface{}, string) {
	// Extract path from expression (this is a simple example, might need to be more sophisticated)
	parts := strings.Split(expr, "values.")
	if len(parts) < 2 {
		return nil, ""
	}

	path := strings.Split(parts[1], " ")[0] // Take first part before any operators
	path = strings.Trim(path, "()")

	// Navigate the values map based on the path
	current := values
	pathParts := strings.Split(path, ".")

	for i, part := range pathParts[:len(pathParts)-1] {
		if v, ok := current[part].(map[string]interface{}); ok {
			current = v
		} else {
			// If we can't navigate further, return the last valid value and path
			return current[part], strings.Join(pathParts[:i+1], ".")
		}
	}

	lastPart := pathParts[len(pathParts)-1]
	return current[lastPart], path
}
