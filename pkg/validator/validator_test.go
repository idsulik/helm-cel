package validator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidator_InitCelEnv(t *testing.T) {
	v := New()
	env, err := v.initCelEnv()

	assert.NoError(t, err)
	assert.NotNil(t, env)
}

func TestValidator_ExtractPath(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected string
	}{
		{
			name:     "no such key error",
			errMsg:   "no such key: service.port",
			expected: "service.port",
		},
		{
			name:     "undefined field error",
			errMsg:   "undefined field 'replicas'",
			expected: "replicas",
		},
		{
			name:     "missing key error",
			errMsg:   "missing key resources.limits",
			expected: "resources.limits",
		},
		{
			name:     "unknown error format",
			errMsg:   "some random error",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				result := extractPath(tt.errMsg)
				assert.Equal(t, tt.expected, result)
			},
		)
	}
}

func TestValidator_ExtractValueFromValues(t *testing.T) {
	values := map[string]interface{}{
		"service": map[string]interface{}{
			"port":     8080,
			"type":     "ClusterIP",
			"nodePort": 30080,
		},
		"replicas": 3,
		"resources": map[string]interface{}{
			"limits": map[string]interface{}{
				"cpu":    "1",
				"memory": "1Gi",
			},
		},
	}

	tests := []struct {
		name          string
		expr          string
		expectedValue interface{}
		expectedPath  string
	}{
		{
			name:          "simple field",
			expr:          "values.replicas >= 1",
			expectedValue: 3,
			expectedPath:  "replicas",
		},
		{
			name:          "nested field",
			expr:          "values.service.port <= 65535",
			expectedValue: 8080,
			expectedPath:  "service.port",
		},
		{
			name:          "deeply nested field",
			expr:          "values.resources.limits.cpu == '1'",
			expectedValue: "1",
			expectedPath:  "resources.limits.cpu",
		},
		{
			name:          "expression without values prefix",
			expr:          "replicas >= 1",
			expectedValue: nil,
			expectedPath:  "",
		},
		{
			name:          "complex expression",
			expr:          "values.service.port >= 1 && values.service.port <= 65535",
			expectedValue: 8080,
			expectedPath:  "service.port",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				value, path := extractValueFromValues(values, tt.expr)
				assert.Equal(t, tt.expectedValue, value)
				assert.Equal(t, tt.expectedPath, path)
			},
		)
	}
}

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ValidationError
		expected string
	}{
		{
			name: "full error details",
			err: &ValidationError{
				Description: "port must be valid",
				Expression:  "values.service.port <= 65535",
				Value:       70000,
				Path:        "service.port",
			},
			expected: "❌ Validation failed: port must be valid\n   Rule: values.service.port <= 65535\n   Path: service.port\n   Current value: 70000",
		},
		{
			name: "error without path",
			err: &ValidationError{
				Description: "replicas must be positive",
				Expression:  "values.replicas > 0",
				Value:       0,
			},
			expected: "❌ Validation failed: replicas must be positive\n   Rule: values.replicas > 0\n   Current value: 0",
		},
		{
			name: "error without value",
			err: &ValidationError{
				Description: "service is required",
				Expression:  "has(values.service)",
				Path:        "service",
			},
			expected: "❌ Validation failed: service is required\n   Rule: has(values.service)\n   Path: service\n",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				result := tt.err.Error()
				assert.Equal(t, tt.expected, result)
			},
		)
	}
}

func TestValidationErrors_Error(t *testing.T) {
	tests := []struct {
		name     string
		errors   []*ValidationError
		expected string
	}{
		{
			name:     "no errors",
			errors:   []*ValidationError{},
			expected: "",
		},
		{
			name: "single error",
			errors: []*ValidationError{
				{
					Description: "port must be valid",
					Expression:  "values.service.port <= 65535",
					Value:       70000,
					Path:        "service.port",
				},
			},
			expected: "Found 1 validation error(s):\n\n❌ Validation failed: port must be valid\n   Rule: values.service.port <= 65535\n   Path: service.port\n   Current value: 70000",
		},
		{
			name: "multiple errors",
			errors: []*ValidationError{
				{
					Description: "port must be valid",
					Expression:  "values.service.port <= 65535",
					Value:       70000,
					Path:        "service.port",
				},
				{
					Description: "replicas must be positive",
					Expression:  "values.replicas > 0",
					Value:       0,
					Path:        "replicas",
				},
			},
			expected: "Found 2 validation error(s):\n\n❌ Validation failed: port must be valid\n   Rule: values.service.port <= 65535\n   Path: service.port\n   Current value: 70000\n\n❌ Validation failed: replicas must be positive\n   Rule: values.replicas > 0\n   Path: replicas\n   Current value: 0",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				ve := &ValidationErrors{Errors: tt.errors}
				result := ve.Error()
				assert.Equal(t, tt.expected, result)
			},
		)
	}
}

func TestValidator_LoadFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Test loading non-existent files
	t.Run(
		"non-existent values.yaml", func(t *testing.T) {
			v := New()
			_, err := v.loadValues(tempDir)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to read values.yaml")
		},
	)

	t.Run(
		"non-existent values.cel.yaml", func(t *testing.T) {
			v := New()
			_, err := v.loadRules(tempDir)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to read values.cel.yaml")
		},
	)

	// Test loading invalid YAML files
	t.Run(
		"invalid values.yaml", func(t *testing.T) {
			v := New()
			invalidYaml := "invalid: yaml: content: :"
			err := writeFile(t, tempDir, "values.yaml", invalidYaml)
			assert.NoError(t, err)

			_, err = v.loadValues(tempDir)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to parse values.yaml")
		},
	)

	t.Run(
		"invalid values.cel.yaml", func(t *testing.T) {
			v := New()
			invalidYaml := "rules: - expr: invalid: yaml:"
			err := writeFile(t, tempDir, "values.cel.yaml", invalidYaml)
			assert.NoError(t, err)

			_, err = v.loadRules(tempDir)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to parse values.cel.yaml")
		},
	)

	// Test loading valid files
	t.Run(
		"valid files", func(t *testing.T) {
			v := New()
			validValues := "service:\n  port: 80"
			validRules := "rules:\n- expr: values.service.port <= 65535\n  desc: port must be valid"

			err := writeFile(t, tempDir, "values.yaml", validValues)
			assert.NoError(t, err)
			err = writeFile(t, tempDir, "values.cel.yaml", validRules)
			assert.NoError(t, err)

			values, err := v.loadValues(tempDir)
			assert.NoError(t, err)
			assert.NotNil(t, values)
			assert.Equal(t, 80, values["service"].(map[string]interface{})["port"])

			rules, err := v.loadRules(tempDir)
			assert.NoError(t, err)
			assert.NotNil(t, rules)
			assert.Len(t, rules.Rules, 1)
			assert.Equal(t, "values.service.port <= 65535", rules.Rules[0].Expr)
			assert.Equal(t, "port must be valid", rules.Rules[0].Desc)
		},
	)
}

// Helper function to write test files
func writeFile(t *testing.T, dir, name, content string) error {
	t.Helper()
	return os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
}
