package integration

import (
	"encoding/json"
	"testing"

	"github.com/idsulik/helm-cel/pkg/models"
	"github.com/idsulik/helm-cel/pkg/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestValidationOutputJSON tests JSON format output for validation results
func TestValidationOutputJSON(t *testing.T) {
	tests := []struct {
		name           string
		values         string
		rules          string
		hasErrors      bool
		expectErrors   int
		expectWarnings int
	}{
		{
			name: "valid configuration",
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
`,
			hasErrors:      false,
			expectErrors:   0,
			expectWarnings: 0,
		},
		{
			name: "invalid configuration",
			values: `
service:
  type: ClusterIP
  port: 80000
`,
			rules: `
rules:
  - expr: "values.service.port >= 1 && values.service.port <= 65535"
    desc: "service port must be between 1 and 65535"
`,
			hasErrors:      true,
			expectErrors:   1,
			expectWarnings: 0,
		},
		{
			name: "warnings only",
			values: `
service:
  type: ClusterIP
  port: 80000
`,
			rules: `
rules:
  - expr: "values.service.port >= 1 && values.service.port <= 65535"
    desc: "service port must be between 1 and 65535"
    severity: "warning"
`,
			hasErrors:      false,
			expectErrors:   0,
			expectWarnings: 1,
		},
		{
			name: "errors and warnings",
			values: `
service:
  type: ClusterIP
  port: 80000
replicaCount: 0
`,
			rules: `
rules:
  - expr: "values.service.port >= 1 && values.service.port <= 65535"
    desc: "service port must be between 1 and 65535"
    severity: "warning"
  - expr: "values.replicaCount >= 1"
    desc: "replicaCount must be at least 1"
`,
			hasErrors:      true,
			expectErrors:   1,
			expectWarnings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			chartDir := setupTestChart(t, tt.values, tt.rules)
			v := validator.New()
			result, err := v.ValidateChart(chartDir, []string{"values.yaml"}, []string{"values.cel.yaml"})
			require.NoError(t, err)

			// Create output structure (same logic as in main.go)
			output := models.ValidationOutput{
				HasErrors:   result.HasErrors(),
				HasWarnings: len(result.Warnings) > 0,
				Result:      result,
			}

			// Convert to JSON
			jsonData, err := json.Marshal(output)
			require.NoError(t, err)

			// Parse back for verification
			var parsedOutput map[string]interface{}
			err = json.Unmarshal(jsonData, &parsedOutput)
			require.NoError(t, err)

			// Verify structure and values
			assert.Equal(t, tt.hasErrors, parsedOutput["has_errors"])
			assert.Equal(t, len(result.Warnings) > 0, parsedOutput["has_warnings"])

			// Check result structure
			resultMap, ok := parsedOutput["result"].(map[string]interface{})
			require.True(t, ok)

			// Check errors
			errors, ok := resultMap["errors"].([]interface{})
			require.True(t, ok)
			assert.Len(t, errors, tt.expectErrors)

			// Check warnings
			warnings, ok := resultMap["warnings"].([]interface{})
			require.True(t, ok)
			assert.Len(t, warnings, tt.expectWarnings)

			// Verify error structure if there are errors
			if tt.expectErrors > 0 {
				firstError, ok := errors[0].(map[string]interface{})
				require.True(t, ok)
				assert.Contains(t, firstError, "description")
				assert.Contains(t, firstError, "expression")
				assert.Contains(t, firstError, "value")
			}

			// Verify warning structure if there are warnings
			if tt.expectWarnings > 0 {
				firstWarning, ok := warnings[0].(map[string]interface{})
				require.True(t, ok)
				assert.Contains(t, firstWarning, "description")
				assert.Contains(t, firstWarning, "expression")
				assert.Contains(t, firstWarning, "value")
			}
		})
	}
}

// TestValidationOutputYAML tests YAML format output for validation results
func TestValidationOutputYAML(t *testing.T) {
	tests := []struct {
		name           string
		values         string
		rules          string
		hasErrors      bool
		expectErrors   int
		expectWarnings int
	}{
		{
			name: "valid configuration",
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
`,
			hasErrors:      false,
			expectErrors:   0,
			expectWarnings: 0,
		},
		{
			name: "invalid configuration",
			values: `
service:
  type: ClusterIP
  port: 80000
`,
			rules: `
rules:
  - expr: "values.service.port >= 1 && values.service.port <= 65535"
    desc: "service port must be between 1 and 65535"
`,
			hasErrors:      true,
			expectErrors:   1,
			expectWarnings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			chartDir := setupTestChart(t, tt.values, tt.rules)
			v := validator.New()
			result, err := v.ValidateChart(chartDir, []string{"values.yaml"}, []string{"values.cel.yaml"})
			require.NoError(t, err)

			// Create output structure (same logic as in main.go)
			output := models.ValidationOutput{
				HasErrors:   result.HasErrors(),
				HasWarnings: len(result.Warnings) > 0,
				Result:      result,
			}

			// Convert to YAML
			yamlData, err := yaml.Marshal(output)
			require.NoError(t, err)

			// Parse back for verification
			var parsedOutput models.ValidationOutput
			err = yaml.Unmarshal(yamlData, &parsedOutput)
			require.NoError(t, err)

			// Verify structure and values
			assert.Equal(t, tt.hasErrors, parsedOutput.HasErrors)
			assert.Equal(t, len(result.Warnings) > 0, parsedOutput.HasWarnings)
			assert.Len(t, parsedOutput.Result.Errors, tt.expectErrors)
			assert.Len(t, parsedOutput.Result.Warnings, tt.expectWarnings)

			// Verify error contents if there are errors
			if tt.expectErrors > 0 {
				assert.NotEmpty(t, parsedOutput.Result.Errors[0].Description)
				assert.NotEmpty(t, parsedOutput.Result.Errors[0].Expression)
			}
		})
	}
}

// TestValidationOutputFunctions tests the main.go output functions
func TestValidationOutputFunctions(t *testing.T) {
	// Create a mock ValidationResult
	result := &models.ValidationResult{
		Errors: []*models.ValidationError{
			{
				Description: "Error description",
				Expression:  "has(values.required)",
				Path:        "required",
				Value:       nil,
			},
		},
		Warnings: []*models.ValidationError{
			{
				Description: "Warning description",
				Expression:  "has(values.optional)",
				Path:        "optional",
				Value:       nil,
			},
		},
	}

	// Create output structure
	output := models.ValidationOutput{
		HasErrors:   result.HasErrors(),
		HasWarnings: len(result.Warnings) > 0,
		Result:      result,
	}

	// Test JSON marshaling and unmarshaling
	t.Run("JSON Marshal/Unmarshal", func(t *testing.T) {
		jsonData, err := json.Marshal(output)
		require.NoError(t, err)
		assert.NotEmpty(t, jsonData)

		var parsedOutput models.ValidationOutput
		err = json.Unmarshal(jsonData, &parsedOutput)
		require.NoError(t, err)

		// Verify structure was preserved
		assert.Equal(t, output.HasErrors, parsedOutput.HasErrors)
		assert.Equal(t, output.HasWarnings, parsedOutput.HasWarnings)
		assert.Len(t, parsedOutput.Result.Errors, len(output.Result.Errors))
		assert.Len(t, parsedOutput.Result.Warnings, len(output.Result.Warnings))
		assert.Equal(t, output.Result.Errors[0].Description, parsedOutput.Result.Errors[0].Description)
	})

	// Test YAML marshaling and unmarshaling
	t.Run("YAML Marshal/Unmarshal", func(t *testing.T) {
		yamlData, err := yaml.Marshal(output)
		require.NoError(t, err)
		assert.NotEmpty(t, yamlData)

		var parsedOutput models.ValidationOutput
		err = yaml.Unmarshal(yamlData, &parsedOutput)
		require.NoError(t, err)

		// Verify structure was preserved
		assert.Equal(t, output.HasErrors, parsedOutput.HasErrors)
		assert.Equal(t, output.HasWarnings, parsedOutput.HasWarnings)
		assert.Len(t, parsedOutput.Result.Errors, len(output.Result.Errors))
		assert.Len(t, parsedOutput.Result.Warnings, len(output.Result.Warnings))
		assert.Equal(t, output.Result.Errors[0].Description, parsedOutput.Result.Errors[0].Description)
	})
}