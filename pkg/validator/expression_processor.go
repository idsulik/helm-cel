package validator

import (
	"fmt"
	"strings"

	"github.com/idsulik/helm-cel/pkg/models"
)

type ExpressionProcessor struct{}

func NewExpressionProcessor() *ExpressionProcessor {
	return &ExpressionProcessor{}
}

// PrepareNamedExpressions expands all named expressions in validation rules
func (p *ExpressionProcessor) PrepareNamedExpressions(rules *models.ValidationRules) error {
	if rules.Expressions == nil {
		rules.Expressions = make(map[string]string)
	}

	for i, rule := range rules.Rules {
		expandedExpr, err := p.expandExpression(rule.Expr, rules.Expressions)
		if err != nil {
			return fmt.Errorf("failed to expand rule '%s': %v", rule.Desc, err)
		}
		rules.Rules[i].Expr = expandedExpr
	}

	return nil
}

// expandExpression expands a single expression by replacing named expression references
func (p *ExpressionProcessor) expandExpression(expr string, expressions map[string]string) (string, error) {
	if expressions == nil {
		expressions = make(map[string]string)
	}

	result := expr
	processedRefs := make(map[string]bool)
	maxIterations := len(expressions) + 1
	iteration := 0

	for strings.Contains(result, "${") && iteration < maxIterations {
		iteration++
		foundReplacement := false

		for name, namedExpr := range expressions {
			placeholder := "${" + name + "}"

			if strings.Contains(result, placeholder) {
				foundReplacement = true
				if processedRefs[name] {
					return "", fmt.Errorf("circular reference detected in expression: %s", expr)
				}
				processedRefs[name] = true
				result = strings.ReplaceAll(result, placeholder, "("+namedExpr+")")
			}
		}

		if !foundReplacement {
			return "", fmt.Errorf("undefined reference in expression: %s", expr)
		}
	}

	if strings.Contains(result, "${") {
		return "", fmt.Errorf("circular reference detected in expression: %s", expr)
	}

	return result, nil
}
