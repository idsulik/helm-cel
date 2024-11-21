package validator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidator_InitCelEnv(t *testing.T) {
	v := New()
	env, err := v.initCelEnv()

	assert.NoError(t, err)
	assert.NotNil(t, env)
}

func TestValidator_ValidateChart(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name           string
		valuesContent  string
		rulesContent   string
		expectedError  string
		shouldValidate bool
	}{
		{
			name: "valid rules and values",
			valuesContent: `
service:
  port: 8080
replicas: 3`,
			rulesContent: `
rules:
  - expr: "values.service.port <= 65535"
    desc: "port must be valid"
  - expr: "values.replicas > 0"
    desc: "replicas must be positive"`,
			shouldValidate: true,
		},
		{
			name: "complex validation with multiple rules",
			valuesContent: `
service:
  port: 8080
  type: LoadBalancer
resources:
  limits:
    cpu: "2"
    memory: "2Gi"`,
			rulesContent: `
rules:
  - expr: 'values.service.port <= 65535'
    desc: "port must be valid"
  - expr: 'values.service.type in ["ClusterIP", "NodePort", "LoadBalancer"]'
    desc: "valid service type"
  - expr: 'double(values.resources.limits.cpu) <= 4.0'
    desc: "CPU limit must be <= 4.0"`,
			shouldValidate: true,
		},
		{
			name: "invalid rule syntax",
			valuesContent: `
service:
  port: 8080`,
			rulesContent: `
rules:
  - expr: "invalid syntax >>>"
    desc: "invalid rule"`,
			expectedError: "❌ Invalid rule syntax in 'invalid rule': ERROR: <input>:1:9: Syntax error: mismatched input 'syntax' expecting <EOF>\n | invalid syntax >>>\n | ........^",
		},
		{
			name: "validation failure",
			valuesContent: `
service:
  port: 70000`,
			rulesContent: `
rules:
  - expr: "values.service.port <= 65535"
    desc: "port must be valid"`,
			expectedError: "Found 1 validation error(s):\n\n❌ Validation failed: port must be valid\n   Rule: values.service.port <= 65535\n   Path: service.port\n   Current value: 70000",
		},
		{
			name: "missing required field",
			valuesContent: `
service: {}`,
			rulesContent: `
rules:
  - expr: "has(values.service.port)"
    desc: "port is required"`,
			expectedError: "Found 1 validation error(s):\n\n❌ Validation failed: port is required\n   Rule: has(values.service.port)\n   Path: service.port\n   Current value: <nil>",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				// Write test files
				require.NoError(t, writeFile(t, tempDir, "values.yaml", tt.valuesContent))
				require.NoError(t, writeFile(t, tempDir, "values.cel.yaml", tt.rulesContent))

				v := New()
				err := v.ValidateChart(tempDir)

				if tt.shouldValidate {
					assert.NoError(t, err)
				} else {
					assert.Error(t, err)
					assert.Equal(t, tt.expectedError, err.Error())
				}
			},
		)
	}
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
		{
			name:     "complex error message",
			errMsg:   "no such key: nested.deep.field in map",
			expected: "nested.deep.field",
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
			"nested": map[string]interface{}{
				"field": "value",
			},
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
		{
			name:          "deep nested with non-map value",
			expr:          "values.service.nested.field == 'value'",
			expectedValue: "value",
			expectedPath:  "service.nested.field",
		},
		{
			name:          "non-existent path",
			expr:          "values.nonexistent.field == true",
			expectedValue: nil,
			expectedPath:  "nonexistent",
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
			expected: "❌ Validation failed: service is required\n   Rule: has(values.service)\n   Path: service\n   Current value: <nil>",
		},
		{
			name: "error with nil value",
			err: &ValidationError{
				Description: "invalid configuration",
				Expression:  "has(values.config)",
				Value:       nil,
				Path:        "config",
			},
			expected: "❌ Validation failed: invalid configuration\n   Rule: has(values.config)\n   Path: config\n   Current value: <nil>",
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

	t.Run(
		"non-existent files", func(t *testing.T) {
			v := New()
			_, err := v.loadValues(tempDir)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to read values.yaml")

			_, err = v.loadRules(tempDir)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to read values.cel.yaml")
		},
	)

	t.Run(
		"invalid YAML syntax", func(t *testing.T) {
			v := New()
			invalidYaml := "invalid: yaml: content: :"
			require.NoError(t, writeFile(t, tempDir, "values.yaml", invalidYaml))
			require.NoError(t, writeFile(t, tempDir, "values.cel.yaml", invalidYaml))

			_, err := v.loadValues(tempDir)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to parse values.yaml")

			_, err = v.loadRules(tempDir)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to parse values.cel.yaml")
		},
	)

	t.Run(
		"empty files", func(t *testing.T) {
			v := New()
			require.NoError(t, writeFile(t, tempDir, "values.yaml", ""))
			require.NoError(t, writeFile(t, tempDir, "values.cel.yaml", ""))

			values, err := v.loadValues(tempDir)
			assert.NoError(t, err)
			assert.NotNil(t, values)

			rules, err := v.loadRules(tempDir)
			assert.NoError(t, err)
			assert.NotNil(t, rules)
		},
	)

	t.Run(
		"valid files with content", func(t *testing.T) {
			v := New()
			validValues := `
service:
  port: 80
  type: ClusterIP
replicas: 3`
			validRules := `
rules:
  - expr: values.service.port <= 65535
    desc: port must be valid
  - expr: values.replicas > 0
    desc: replicas must be positive`

			require.NoError(t, writeFile(t, tempDir, "values.yaml", validValues))
			require.NoError(t, writeFile(t, tempDir, "values.cel.yaml", validRules))

			values, err := v.loadValues(tempDir)
			assert.NoError(t, err)
			assert.NotNil(t, values)
			assert.Equal(t, 80, values["service"].(map[string]interface{})["port"])

			rules, err := v.loadRules(tempDir)
			assert.NoError(t, err)
			assert.NotNil(t, rules)
			assert.Len(t, rules.Rules, 2)
		},
	)
}

// Helper function to write test files
func writeFile(t *testing.T, dir, name, content string) error {
	t.Helper()
	return os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
}
