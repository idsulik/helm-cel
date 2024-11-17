package integration

import (
	"os"
	"path/filepath"
	"testing"

	validator "github.com/idsulik/helm-cel/pkg/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestChart creates a temporary chart directory with the given values and rules
func setupTestChart(t *testing.T, values, rules string) string {
	t.Helper()

	// Create a temporary directory for the test chart
	chartDir, err := os.MkdirTemp("", "helm-cel-test-*")
	require.NoError(t, err)

	// Clean up after the test
	t.Cleanup(
		func() {
			os.RemoveAll(chartDir)
		},
	)

	// Write values.yaml
	err = os.WriteFile(filepath.Join(chartDir, "values.yaml"), []byte(values), 0644)
	require.NoError(t, err)

	// Write values.cel.yaml
	err = os.WriteFile(filepath.Join(chartDir, "values.cel.yaml"), []byte(rules), 0644)
	require.NoError(t, err)

	return chartDir
}

func TestValidation(t *testing.T) {
	tests := []struct {
		name          string
		values        string
		rules         string
		expectedError string
	}{
		{
			name: "valid service configuration",
			values: `
service:
  type: ClusterIP
  port: 80
replicaCount: 1
`,
			rules: `
rules:
  - expr: "has(values.service) && has(values.service.port)"
    desc: "service port is required"
  - expr: "values.service.port >= 1 && values.service.port <= 65535"
    desc: "service port must be between 1 and 65535"
  - expr: has(values.service.type) && values.service.type in ['ClusterIP', 'NodePort', 'LoadBalancer']
    desc: "service type must be one of ClusterIP, NodePort, LoadBalancer"
`,
			expectedError: "",
		},
		{
			name: "invalid service port",
			values: `
service:
  type: ClusterIP
  port: 70000
`,
			rules: `
rules:
  - expr: "values.service.port >= 1 && values.service.port <= 65535"
    desc: "service port must be between 1 and 65535"
`,
			expectedError: "service port must be between 1 and 65535",
		},
		{
			name: "missing required field",
			values: `
service:
  type: ClusterIP
`,
			rules: `
rules:
  - expr: "has(values.service) && has(values.service.port)"
    desc: "service port is required"
`,
			expectedError: "service port is required",
		},
		{
			name: "conditional validation",
			values: `
replicaCount: 0
`,
			rules: `
rules:
  - expr: "!(has(values.replicaCount)) || values.replicaCount >= 1"
    desc: "if replicaCount is set, it must be at least 1"
`,
			expectedError: "if replicaCount is set, it must be at least 1",
		},
		{
			name: "type validation",
			values: `
ports: "not-a-list"
`,
			rules: `
rules:
  - expr: "!(has(values.ports)) || type(values.ports) == list"
    desc: "ports must be a list when specified"
`,
			expectedError: "ports must be a list when specified",
		},
		{
			name: "complex object validation",
			values: `
image:
  repository: nginx
`,
			rules: `
rules:
  - expr: "!(has(values.image)) || (has(values.image.repository) && has(values.image.tag))"
    desc: "if image is specified, both repository and tag are required"
`,
			expectedError: "if image is specified, both repository and tag are required",
		},
		{
			name: "multiple validation rules",
			values: `
service:
  port: 70000
replicaCount: 0
`,
			rules: `
rules:
  - expr: "values.service.port >= 1 && values.service.port <= 65535"
    desc: "service port must be between 1 and 65535"
  - expr: "values.replicaCount >= 1"
    desc: "replicaCount must be at least 1"
`,
			expectedError: "Found 2 validation error(s)",
		},
		{
			name: "valid nested structure",
			values: `
resources:
  limits:
    cpu: "1"
    memory: "1Gi"
  requests:
    cpu: "500m"
    memory: "512Mi"
`,
			rules: `
rules:
  - expr: |
      has(values.resources) &&
      has(values.resources.limits) &&
      has(values.resources.requests) &&
      has(values.resources.limits.cpu) &&
      has(values.resources.limits.memory) &&
      has(values.resources.requests.cpu) &&
      has(values.resources.requests.memory)
    desc: "complete resource configuration is required"
`,
			expectedError: "",
		},
		{
			name:   "empty values with optional fields",
			values: `{}`,
			rules: `
rules:
  - expr: "!(has(values.optional)) || values.optional >= 0"
    desc: "if optional is set, it must be non-negative"
`,
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				// Setup test chart
				chartDir := setupTestChart(t, tt.values, tt.rules)

				// Run validation
				v := validator.New()
				err := v.ValidateChart(chartDir)

				// Check results
				if tt.expectedError == "" {
					assert.NoError(t, err)
				} else {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			},
		)
	}
}

// TestInvalidRuleSyntax tests handling of invalid CEL expressions
func TestInvalidRuleSyntax(t *testing.T) {
	values := "service:\n  port: 80\n"
	rules := `
rules:
  - expr: "this is not a valid cel expression )))))"
    desc: "invalid syntax"
`

	chartDir := setupTestChart(t, values, rules)
	v := validator.New()
	err := v.ValidateChart(chartDir)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid rule syntax")
}

// TestFileNotFound tests handling of missing files
func TestFileNotFound(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "helm-cel-missing-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	v := validator.New()
	err = v.ValidateChart(tempDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read")
}
