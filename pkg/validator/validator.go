package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/idsulik/helm-cel/pkg/models"
	"gopkg.in/yaml.v3"
)

const (
	// WarningSeverity represents a validation warning
	WarningSeverity = "warning"
)

// Validator handles the validation of Helm values using CEL
type Validator struct {
	env *cel.Env
}

// New creates a new Validator instance
func New() *Validator {
	return &Validator{}
}

// ValidateChart validates the values.yaml file against CEL rules.
func (v *Validator) ValidateChart(chartPath, valuesFile, rulesFile string) (*models.ValidationResult, error) {
	// Read and parse values file
	valuesPath := filepath.Join(chartPath, valuesFile)
	valuesFileContent, err := os.ReadFile(valuesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read values file %s: %v", valuesFile, err)
	}

	var valuesData map[string]interface{}
	if err := yaml.Unmarshal(valuesFileContent, &valuesData); err != nil {
		return nil, fmt.Errorf("failed to parse values file %s: %v", valuesFile, err)
	}

	// Read and parse rules file
	rulesPath := filepath.Join(chartPath, rulesFile)
	rulesContent, err := os.ReadFile(rulesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules file %s: %v", rulesFile, err)
	}

	var rules models.ValidationRules
	if err := yaml.Unmarshal(rulesContent, &rules); err != nil {
		return nil, fmt.Errorf("failed to parse rules file %s: %v", rulesFile, err)
	}

	env, err := v.initCelEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize CEL environment: %v", err)
	}

	v.env = env

	// Prepare named expressions
	if err := v.prepareNamedExpressions(&rules); err != nil {
		return nil, err
	}

	return v.validateRules(valuesData, &rules), nil
}

// prepareNamedExpressions expands all named expressions in validation rules
func (v *Validator) prepareNamedExpressions(rules *models.ValidationRules) error {
	if rules.Expressions == nil {
		rules.Expressions = make(map[string]string)
	}

	for i, rule := range rules.Rules {
		expandedExpr, err := v.expandExpression(rule.Expr, rules.Expressions)
		if err != nil {
			return fmt.Errorf("failed to expand rule '%s': %v", rule.Desc, err)
		}
		rules.Rules[i].Expr = expandedExpr
	}

	return nil
}

// expandExpression expands a single expression by replacing named expression references
func (v *Validator) expandExpression(expr string, expressions map[string]string) (string, error) {
	if expressions == nil {
		expressions = make(map[string]string)
	}

	result := expr
	processedRefs := make(map[string]bool)

	// Try maximum number of iterations based on number of expressions
	maxIterations := len(expressions) + 1
	iteration := 0

	for strings.Contains(result, "${") && iteration < maxIterations {
		iteration++
		foundReplacement := false

		for name, namedExpr := range expressions {
			placeholder := "${" + name + "}"

			// Skip if we've already processed this reference in a previous iteration
			if strings.Contains(result, placeholder) {
				foundReplacement = true
				if processedRefs[name] {
					return "", fmt.Errorf("circular reference detected in expression: %s", expr)
				}
				processedRefs[name] = true
				result = strings.ReplaceAll(result, placeholder, "("+namedExpr+")")
			}
		}

		// If no replacements were made in this iteration, we have unresolvable references
		if !foundReplacement {
			return "", fmt.Errorf("undefined reference in expression: %s", expr)
		}
	}

	// If we still have references after max iterations, we have a circular reference
	if strings.Contains(result, "${") {
		return "", fmt.Errorf("circular reference detected in expression: %s", expr)
	}

	return result, nil
}

// initCelEnv initializes the CEL environment with required variables and functions
func (v *Validator) initCelEnv() (*cel.Env, error) {
	return cel.NewEnv(
		cel.Variable("values", cel.DynType),
	)
}

// loadValues reads and parses the values.yaml file from the chart path
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

// loadRules reads and parses the values.cel.yaml file containing validation rules
func (v *Validator) loadRules(chartPath string) (*models.ValidationRules, error) {
	rulesPath := filepath.Join(chartPath, "values.cel.yaml")
	rulesContent, err := os.ReadFile(rulesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read values.cel.yaml: %v", err)
	}

	var rules models.ValidationRules
	if err := yaml.Unmarshal(rulesContent, &rules); err != nil {
		return nil, fmt.Errorf("failed to parse values.cel.yaml: %v", err)
	}

	return &rules, nil
}

// validateRules validates values against all rules and returns the validation result
func (v *Validator) validateRules(
	values map[string]interface{},
	rules *models.ValidationRules,
) *models.ValidationResult {
	result := &models.ValidationResult{
		Errors:   make([]*models.ValidationError, 0),
		Warnings: make([]*models.ValidationError, 0),
	}

	for _, rule := range rules.Rules {
		ast, issues := v.env.Compile(rule.Expr)
		if issues != nil && issues.Err() != nil {
			result.Errors = append(
				result.Errors, &models.ValidationError{
					Description: fmt.Sprintf("Invalid rule syntax in '%s': %v", rule.Desc, issues.Err()),
					Expression:  rule.Expr,
				},
			)
			continue
		}

		program, err := v.env.Program(ast)
		if err != nil {
			result.Errors = append(
				result.Errors, &models.ValidationError{
					Description: fmt.Sprintf("Failed to process rule '%s': %v", rule.Desc, err),
					Expression:  rule.Expr,
				},
			)
			continue
		}

		out, _, err := program.Eval(
			map[string]interface{}{
				"values": values,
			},
		)

		validationError := &models.ValidationError{
			Description: rule.Desc,
			Expression:  rule.Expr,
		}

		if err != nil {
			validationError.Path = extractPath(err.Error())
			if rule.Severity == WarningSeverity {
				result.Warnings = append(result.Warnings, validationError)
			} else {
				result.Errors = append(result.Errors, validationError)
			}
			continue
		}

		if out.Value() != true {
			value, path := extractValueFromValues(values, rule.Expr)
			validationError.Value = value
			validationError.Path = path
			if rule.Severity == WarningSeverity {
				result.Warnings = append(result.Warnings, validationError)
			} else {
				result.Errors = append(result.Errors, validationError)
			}
		}
	}

	return result
}

// extractPath extracts the path from a CEL error message
func extractPath(errMsg string) string {
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

// extractValueFromValues extracts the relevant value from the values map based on the CEL expression
func extractValueFromValues(values map[string]interface{}, expr string) (interface{}, string) {
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
