package validator

import (
	"fmt"
	"regexp"
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
	maxIterations := len(expressions) + 1
	iteration := 0

	for strings.Contains(result, "${") && iteration < maxIterations {
		iteration++
		foundReplacement := false
		previousResult := result

		// Find all ${...} patterns manually to handle nested parentheses
		matches := p.findExpressionReferences(result)

		for _, match := range matches {
			fullMatch := match.fullMatch // e.g., "${foo(arg1, arg2)}"
			exprName := match.name       // e.g., "foo"
			argsWithParens := match.args // e.g., "(arg1, arg2)" or empty string

			namedExpr, exists := expressions[exprName]
			if !exists {
				return "", fmt.Errorf("undefined reference in expression: %s", fullMatch)
			}

			foundReplacement = true

			var expandedExpr string
			if argsWithParens != "" {
				// Parse arguments from "(arg1, arg2)" format
				args, err := p.parseArguments(argsWithParens)
				if err != nil {
					return "", fmt.Errorf("failed to parse arguments in %s: %v", fullMatch, err)
				}

				// Replace parameters in the named expression
				expandedExpr, err = p.replaceParameters(namedExpr, args)
				if err != nil {
					return "", fmt.Errorf("failed to replace parameters in %s: %v", exprName, err)
				}
			} else {
				// No arguments, use the expression as-is
				expandedExpr = namedExpr
			}

			// Replace the match with the expanded expression wrapped in parentheses
			result = strings.Replace(result, fullMatch, "("+expandedExpr+")", 1)
		}

		if !foundReplacement {
			return "", fmt.Errorf("undefined reference in expression: %s", expr)
		}

		// Check for circular reference by seeing if we're stuck in a loop
		if result == previousResult {
			return "", fmt.Errorf("circular reference detected in expression: %s", expr)
		}
	}

	if strings.Contains(result, "${") {
		return "", fmt.Errorf("circular reference detected in expression: %s", expr)
	}

	return result, nil
}

type expressionMatch struct {
	fullMatch string
	name      string
	args      string
}

// findExpressionReferences finds all ${name} or ${name(...)} patterns in the expression
func (p *ExpressionProcessor) findExpressionReferences(expr string) []expressionMatch {
	matches := make([]expressionMatch, 0)
	i := 0

	for i < len(expr) {
		// Look for ${
		if i < len(expr)-1 && expr[i] == '$' && expr[i+1] == '{' {
			start := i
			i += 2

			// Extract the name
			nameStart := i
			for i < len(expr) && (expr[i] == '_' || (expr[i] >= 'a' && expr[i] <= 'z') || (expr[i] >= 'A' && expr[i] <= 'Z') || (expr[i] >= '0' && expr[i] <= '9')) {
				i++
			}
			name := expr[nameStart:i]

			// Check if there are arguments
			args := ""
			if i < len(expr) && expr[i] == '(' {
				// Find matching closing parenthesis
				argsStart := i
				depth := 0
				for i < len(expr) {
					if expr[i] == '(' {
						depth++
					} else if expr[i] == ')' {
						depth--
						if depth == 0 {
							i++
							args = expr[argsStart:i]
							break
						}
					}
					i++
				}
			}

			// Check for closing }
			if i < len(expr) && expr[i] == '}' {
				i++
				fullMatch := expr[start:i]
				matches = append(matches, expressionMatch{
					fullMatch: fullMatch,
					name:      name,
					args:      args,
				})
			} else {
				// Malformed reference, skip it
				i++
			}
		} else {
			i++
		}
	}

	return matches
}

// parseArguments extracts arguments from a string like "(arg1, arg2, arg3)"
func (p *ExpressionProcessor) parseArguments(argsWithParens string) ([]string, error) {
	// Remove the outer parentheses
	argsStr := strings.TrimPrefix(argsWithParens, "(")
	argsStr = strings.TrimSuffix(argsStr, ")")
	argsStr = strings.TrimSpace(argsStr)

	if argsStr == "" {
		return []string{}, nil
	}

	// Split by comma, but we need to handle nested structures carefully
	args := make([]string, 0)
	current := strings.Builder{}
	parenDepth := 0
	bracketDepth := 0
	inSingleQuote := false
	inDoubleQuote := false

	for i := 0; i < len(argsStr); i++ {
		ch := rune(argsStr[i])
		switch ch {
		case '(':
			if !inSingleQuote && !inDoubleQuote {
				parenDepth++
			}
			current.WriteRune(ch)
		case ')':
			if !inSingleQuote && !inDoubleQuote {
				parenDepth--
			}
			current.WriteRune(ch)
		case '[':
			if !inSingleQuote && !inDoubleQuote {
				bracketDepth++
			}
			current.WriteRune(ch)
		case ']':
			if !inSingleQuote && !inDoubleQuote {
				bracketDepth--
			}
			current.WriteRune(ch)
		case '\'':
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			}
			current.WriteRune(ch)
		case '"':
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			}
			current.WriteRune(ch)
		case ',':
			if parenDepth == 0 && bracketDepth == 0 && !inSingleQuote && !inDoubleQuote {
				args = append(args, strings.TrimSpace(current.String()))
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		case '\\':
			// Handle escape sequences
			current.WriteRune(ch)
			if i+1 < len(argsStr) {
				i++
				current.WriteRune(rune(argsStr[i]))
			}
		default:
			current.WriteRune(ch)
		}
	}

	// Add the last argument
	if current.Len() > 0 {
		args = append(args, strings.TrimSpace(current.String()))
	}

	return args, nil
}

// replaceParameters replaces parameter placeholders like $0, $1, $2, etc. with actual arguments
func (p *ExpressionProcessor) replaceParameters(expr string, args []string) (string, error) {
	result := expr

	// Pattern to match positional parameter placeholders: $0, $1, $2, etc.
	paramPattern := regexp.MustCompile(`\$(\d+)`)

	// Find all parameter placeholders
	matches := paramPattern.FindAllStringSubmatch(expr, -1)

	// Check if parameters are used but no arguments provided
	if len(matches) > 0 && len(args) == 0 {
		return "", fmt.Errorf("expression requires parameters but none were provided")
	}

	// Replace positional parameters ($0, $1, $2, etc.)
	for i, arg := range args {
		placeholder := fmt.Sprintf("$%d", i)
		result = strings.ReplaceAll(result, placeholder, arg)
	}

	// Check if there are any unresolved parameters
	if paramPattern.MatchString(result) {
		return "", fmt.Errorf("not enough arguments provided for parameters")
	}

	return result, nil
}
