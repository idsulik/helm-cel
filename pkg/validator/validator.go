package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/idsulik/helm-cel/pkg/models"
	"github.com/idsulik/helm-cel/pkg/utils"
	"gopkg.in/yaml.v3"
)

const (
	// WarningSeverity represents a validation warning
	WarningSeverity = "warning"
)

// Validator handles the validation of Helm values using CEL
type Validator struct {
	env           *cel.Env
	valuesLoader  *ValuesLoader
	rulesLoader   *RulesLoader
	exprProcessor *ExpressionProcessor
}

func New() *Validator {
	return &Validator{
		valuesLoader:  NewValuesLoader(),
		rulesLoader:   NewRulesLoader(),
		exprProcessor: NewExpressionProcessor(),
	}
}

// ValidateChart validates the values.yaml file against CEL rules.
func (v *Validator) ValidateChart(
	chartPath string,
	valuesFiles []string,
	rulesFiles []string,
) (*models.ValidationResult, error) {
	// Process values files
	valuesFiles, err := utils.GetAbsolutePaths(chartPath, valuesFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to get values absolute paths: %v", err)
	}

	// Process rules files
	rulesFiles, err = utils.GetAbsolutePaths(chartPath, rulesFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules absolute paths: %v", err)
	}

	mergedValues, err := v.valuesLoader.LoadAndMergeValues(valuesFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to load values: %v", err)
	}

	mergedRules, err := v.rulesLoader.LoadAndMergeRules(rulesFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to load rules: %v", err)
	}

	if len(mergedRules.Rules) == 0 {
		return &models.ValidationResult{}, nil
	}

	env, err := v.initCelEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize CEL environment: %v", err)
	}
	v.env = env

	if err := v.exprProcessor.PrepareNamedExpressions(mergedRules); err != nil {
		return nil, err
	}

	return v.validateRules(mergedValues, mergedRules), nil
}

// initCelEnv initializes the CEL environment with required variables and functions
func (v *Validator) initCelEnv() (*cel.Env, error) {
	return cel.NewEnv(
		cel.Variable("values", cel.DynType),
	)
}

// loadValues reads and parses the values.yaml file from the chart path
func (v *Validator) loadValues(chartPath string) (map[string]any, error) {
	valuesPath := filepath.Join(chartPath, "values.yaml")
	valuesContent, err := os.ReadFile(valuesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read values.yaml: %v", err)
	}

	values := make(map[string]any)
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
	values map[string]any,
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
			map[string]any{
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
func extractValueFromValues(values map[string]any, expr string) (any, string) {
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
		if v, ok := current[part].(map[string]any); ok {
			current = v
		} else {
			// If we can't navigate further, return the last valid value and path
			return current[part], strings.Join(pathParts[:i+1], ".")
		}
	}

	lastPart := pathParts[len(pathParts)-1]
	return current[lastPart], path
}
