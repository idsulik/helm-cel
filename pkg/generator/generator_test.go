package generator

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/idsulik/helm-cel/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Helper function to sort rules
func sortRules(rules *models.ValidationRules) {
	sort.Slice(
		rules.Rules, func(i, j int) bool {
			return rules.Rules[i].Expr < rules.Rules[j].Expr
		},
	)
}

func TestGenerator_GenerateRules(t *testing.T) {
	tests := []struct {
		name          string
		values        string
		expectedRules *models.ValidationRules
		wantErr       bool
		errMsg        string
	}{
		{
			name: "basic types",
			values: `
stringVal: hello
intVal: 42
boolVal: true
floatVal: 3.14
`,
			expectedRules: &models.ValidationRules{
				Rules: []models.Rule{
					{
						Expr: "type(values.stringVal) == string",
						Desc: "stringVal must be a string",
					},
					{
						Expr: "type(values.intVal) == int",
						Desc: "intVal must be an integer",
					},
					{
						Expr: "type(values.boolVal) == bool",
						Desc: "boolVal must be a boolean",
					},
					{
						Expr: "type(values.floatVal) == int || type(values.floatVal) == double",
						Desc: "floatVal must be a number",
					},
				},
				Expressions: make(map[string]string),
			},
		},
		{
			name: "nested objects",
			values: `
service:
  port: 8080
  name: myapp
`,
			expectedRules: &models.ValidationRules{
				Rules: []models.Rule{
					{
						Expr: "has(values.service)",
						Desc: "service must be defined",
					},
					{
						Expr: "type(values.service.port) == int",
						Desc: "port must be an integer",
					},
					{
						Expr: "values.service.port >= 1 && values.service.port <= 65535",
						Desc: "port must be a valid port number (1-65535)",
					},
					{
						Expr: "type(values.service.name) == string",
						Desc: "name must be a string",
					},
				},
				Expressions: make(map[string]string),
			},
		},
		{
			name: "array validation",
			values: `
containers:
  - name: app
    port: 8080
`,
			expectedRules: &models.ValidationRules{
				Rules: []models.Rule{
					{
						Expr: "size(values.containers) >= 0",
						Desc: "containers must be an array",
					},
					{
						Expr: "type(values.containers[0].name) == string",
						Desc: "name must be a string",
					},
					{
						Expr: "type(values.containers[0].port) == int",
						Desc: "port must be an integer",
					},
					{
						Expr: "values.containers[0].port >= 1 && values.containers[0].port <= 65535",
						Desc: "port must be a valid port number (1-65535)",
					},
				},
				Expressions: make(map[string]string),
			},
		},
		{
			name: "resource requirements",
			values: `
resources:
  requests:
    cpu: 100m
    memory: 128Mi
`,
			expectedRules: &models.ValidationRules{
				Rules: []models.Rule{
					{
						Expr: "has(values.resources)",
						Desc: "resources must be defined",
					},
					{
						Expr: "has(values.resources.requests)",
						Desc: "requests must be defined",
					},
					{
						Expr: "type(values.resources.requests.cpu) == string",
						Desc: "cpu must be a string",
					},
					{
						Expr: `values.resources.requests.cpu.matches('^[0-9]+(.[0-9]+)?(m|Mi|Gi|Ti|Pi|Ei|n|u|m|k|M|G|T|P|E)?$')`,
						Desc: "cpu must be a valid resource quantity",
					},
					{
						Expr: "type(values.resources.requests.memory) == string",
						Desc: "memory must be a string",
					},
					{
						Expr: `values.resources.requests.memory.matches('^[0-9]+(.[0-9]+)?(m|Mi|Gi|Ti|Pi|Ei|n|u|m|k|M|G|T|P|E)?$')`,
						Desc: "memory must be a valid resource quantity",
					},
				},
				Expressions: make(map[string]string),
			},
		},
		{
			name: "empty values file",
			values: `
`,
			expectedRules: &models.ValidationRules{
				Rules:       []models.Rule{},
				Expressions: make(map[string]string),
			},
		},
		{
			name:    "invalid yaml",
			values:  "invalid: [yaml: content",
			wantErr: true,
			errMsg:  "failed to parse values file",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				// Create temporary directory
				tmpDir, err := os.MkdirTemp("", "generator-test-*")
				require.NoError(t, err)
				defer os.RemoveAll(tmpDir)

				// Create values.yaml
				valuesPath := filepath.Join(tmpDir, "values.yaml")
				err = os.WriteFile(valuesPath, []byte(tt.values), 0644)
				require.NoError(t, err)

				g := New()
				rules, err := g.GenerateRules(tmpDir, "values.yaml")

				if tt.wantErr {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tt.errMsg)
					return
				}

				require.NoError(t, err)

				// Sort both expected and actual rules before comparison
				if tt.expectedRules != nil {
					sortRules(tt.expectedRules)
				}
				if rules != nil {
					sortRules(rules)
				}

				assert.Equal(t, tt.expectedRules, rules)
			},
		)
	}
}

func TestGenerator_WriteRules(t *testing.T) {
	tests := []struct {
		name  string
		rules *models.ValidationRules
	}{
		{
			name: "write basic rules",
			rules: &models.ValidationRules{
				Rules: []models.Rule{
					{
						Expr: "type(values.test) == string",
						Desc: "test must be a string",
					},
				},
				Expressions: map[string]string{
					"test": "values.test == 'value'",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				// Create temporary directory
				tmpDir, err := os.MkdirTemp("", "generator-test-*")
				require.NoError(t, err)
				defer os.RemoveAll(tmpDir)

				rulesPath := filepath.Join(tmpDir, "values.cel.yaml")
				g := New()

				// Write rules
				err = g.WriteRules(rulesPath, tt.rules)
				require.NoError(t, err)

				// Read back and verify
				content, err := os.ReadFile(rulesPath)
				require.NoError(t, err)

				var readRules models.ValidationRules
				err = yaml.Unmarshal(content, &readRules)
				require.NoError(t, err)

				assert.Equal(t, tt.rules, &readRules)
			},
		)
	}
}
