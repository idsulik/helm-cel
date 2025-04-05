package models

import (
	"fmt"
	"strings"
)

// Rule represents a single CEL validation rule with severity and name
type Rule struct {
	Expr     string `yaml:"expr"`
	Desc     string `yaml:"desc"`
	Severity string `yaml:"severity,omitempty"` // "error" or "warning", defaults to "error"
}

// ValidationRules contains all CEL validation rules and named expressions
type ValidationRules struct {
	Rules       []Rule            `yaml:"rules"`
	Expressions map[string]string `yaml:"expressions,omitempty"`
}

// ValidationResult represents the outcome of validation
type ValidationResult struct {
	Errors   []*ValidationError `json:"errors" yaml:"errors"`
	Warnings []*ValidationError `json:"warnings" yaml:"warnings"`
}

// ValidationError represents a validation failure
type ValidationError struct {
	Description string `json:"description" yaml:"description"`
	Expression  string `json:"expression" yaml:"expression"`
	Value       any    `json:"value" yaml:"value"`
	Path        string `json:"path,omitempty" yaml:"path,omitempty"`
}

// ValidationOutput is used for structured output in JSON/YAML format
type ValidationOutput struct {
	HasErrors   bool              `json:"has_errors" yaml:"has_errors"`
	HasWarnings bool              `json:"has_warnings" yaml:"has_warnings"`
	Result      *ValidationResult `json:"result" yaml:"result"`
}

func (vr *ValidationResult) HasErrors() bool {
	return len(vr.Errors) > 0
}

func (vr *ValidationResult) Error() string {
	var msg strings.Builder

	if len(vr.Errors) > 0 {
		msg.WriteString(fmt.Sprintf("Found %d error(s):\n\n", len(vr.Errors)))
		for i, err := range vr.Errors {
			msg.WriteString(err.Error())
			if i < len(vr.Errors)-1 {
				msg.WriteString("\n\n")
			}
		}
	}

	if len(vr.Warnings) > 0 {
		if len(vr.Errors) > 0 {
			msg.WriteString("\n\n")
		}
		msg.WriteString(fmt.Sprintf("Found %d warning(s):\n\n", len(vr.Warnings)))
		for i, warn := range vr.Warnings {
			msg.WriteString(warn.Warning())
			if i < len(vr.Warnings)-1 {
				msg.WriteString("\n\n")
			}
		}
	}

	return msg.String()
}

func (e *ValidationError) Error() string {
	return e.format("❌")
}

func (e *ValidationError) Warning() string {
	return e.format("⚠️")
}

func (e *ValidationError) format(symbol string) string {
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("%s %s\n", symbol, e.Description))
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
