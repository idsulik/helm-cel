package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/idsulik/helm-cel/pkg/validator"
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
		name            string
		values          string
		rules           string
		expectedWarning string
		expectedError   string
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
			expectedError: "Found 1 error(s):\n\n❌ service port must be between 1 and 65535\n   Rule: values.service.port >= 1 && values.service.port <= 65535\n   Path: service.port\n   Current value: 70000",
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
			expectedError: "Found 1 error(s):\n\n❌ service port is required\n   Rule: has(values.service) && has(values.service.port)\n   Path: service\n   Current value: map[type:ClusterIP]",
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
			expectedError: "Found 1 error(s):\n\n❌ if replicaCount is set, it must be at least 1\n   Rule: !(has(values.replicaCount)) || values.replicaCount >= 1\n   Path: replicaCount\n   Current value: 0",
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
			expectedError: "Found 1 error(s):\n\n❌ ports must be a list when specified\n   Rule: !(has(values.ports)) || type(values.ports) == list\n   Path: ports\n   Current value: not-a-list",
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
			expectedError: "Found 1 error(s):\n\n❌ if image is specified, both repository and tag are required\n   Rule: !(has(values.image)) || (has(values.image.repository) && has(values.image.tag))\n   Path: image\n   Current value: map[repository:nginx]",
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
			expectedError: "Found 2 error(s):\n\n❌ service port must be between 1 and 65535\n   Rule: values.service.port >= 1 && values.service.port <= 65535\n   Path: service.port\n   Current value: 70000\n\n❌ replicaCount must be at least 1\n   Rule: values.replicaCount >= 1\n   Path: replicaCount\n   Current value: 0",
		},
		{
			name: "warnings only",
			values: `
service:
  port: 70000
replicaCount: 0
`,
			rules: `
rules:
  - expr: "values.service.port >= 1 && values.service.port <= 65535"
    desc: "service port must be between 1 and 65535"
    severity: "warning"
  - expr: "values.replicaCount >= 1"
    desc: "replicaCount must be at least 1"
    severity: "warning"
`,
			expectedWarning: "Found 2 warning(s):\n\n⚠️ service port must be between 1 and 65535\n   Rule: values.service.port >= 1 && values.service.port <= 65535\n   Path: service.port\n   Current value: 70000\n\n⚠️ replicaCount must be at least 1\n   Rule: values.replicaCount >= 1\n   Path: replicaCount\n   Current value: 0",
		},
		{
			name: "errors and warnings",
			values: `
service:
  port: 70000
replicaCount: 0
`,
			rules: `
rules:
  - expr: "values.service.port >= 1 && values.service.port <= 65535"
    desc: "service port must be between 1 and 65535"
    severity: "warning"
  - expr: "values.replicaCount >= 1"
    desc: "replicaCount must be at least 1"
    severity: "error"
`,
			expectedError: "Found 1 error(s):\n\n❌ replicaCount must be at least 1\n   Rule: values.replicaCount >= 1\n   Path: replicaCount\n   Current value: 0\n\nFound 1 warning(s):\n\n⚠️ service port must be between 1 and 65535\n   Rule: values.service.port >= 1 && values.service.port <= 65535\n   Path: service.port\n   Current value: 70000",
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
				res, _ := v.ValidateChart(chartDir)

				// Check results
				if tt.expectedError == "" {
					assert.False(t, res.HasErrors())
				} else {
					assert.True(t, res.HasErrors())
					assert.Equal(t, tt.expectedError, res.Error())
				}

				if tt.expectedWarning != "" {
					assert.Equal(t, tt.expectedWarning, res.Error())
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
	res, _ := v.ValidateChart(chartDir)

	assert.True(t, res.HasErrors())
	assert.Contains(t, res.Error(), "Invalid rule syntax")
}

// TestFileNotFound tests handling of missing files
func TestFileNotFound(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "helm-cel-missing-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	v := validator.New()
	_, err = v.ValidateChart(tempDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read")
}
