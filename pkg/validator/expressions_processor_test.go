package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpressionProcessor_ExpandExpression(t *testing.T) {
	tests := []struct {
		name        string
		expr        string
		expressions map[string]string
		want        string
		wantErr     string
	}{
		{
			name: "simple expression without references",
			expr: "values.service.port <= 65535",
			expressions: map[string]string{
				"validPort": "port > 0",
			},
			want: "values.service.port <= 65535",
		},
		{
			name: "single reference",
			expr: "${validPort}",
			expressions: map[string]string{
				"validPort": "values.service.port <= 65535",
			},
			want: "(values.service.port <= 65535)",
		},
		{
			name: "multiple references",
			expr: "${validPort} && ${validType}",
			expressions: map[string]string{
				"validPort": "values.service.port <= 65535",
				"validType": "values.service.type in ['ClusterIP', 'NodePort']",
			},
			want: "(values.service.port <= 65535) && (values.service.type in ['ClusterIP', 'NodePort'])",
		},
		{
			name: "nested references",
			expr: "${validateService}",
			expressions: map[string]string{
				"validPort":       "values.service.port <= 65535",
				"validType":       "values.service.type in ['ClusterIP', 'NodePort']",
				"validateService": "${validPort} && ${validType}",
			},
			want: "((values.service.port <= 65535) && (values.service.type in ['ClusterIP', 'NodePort']))",
		},
		{
			name: "multiple nested references",
			expr: "${validateAll}",
			expressions: map[string]string{
				"validPort":       "values.service.port <= 65535",
				"validType":       "values.service.type in ['ClusterIP', 'NodePort']",
				"validateService": "${validPort} && ${validType}",
				"validateAll":     "${validateService} && has(values.replicas)",
			},
			want: "(((values.service.port <= 65535) && (values.service.type in ['ClusterIP', 'NodePort'])) && has(values.replicas))",
		},
		{
			name: "reference in middle of expression",
			expr: "has(values.service) && ${validPort} && has(values.replicas)",
			expressions: map[string]string{
				"validPort": "values.service.port <= 65535",
			},
			want: "has(values.service) && (values.service.port <= 65535) && has(values.replicas)",
		},
		{
			name: "circular reference - direct",
			expr: "${a}",
			expressions: map[string]string{
				"a": "${a}",
			},
			wantErr: "circular reference detected in expression: ${a}",
		},
		{
			name: "circular reference - indirect",
			expr: "${a}",
			expressions: map[string]string{
				"a": "${b}",
				"b": "${c}",
				"c": "${a}",
			},
			wantErr: "circular reference detected in expression: ${a}",
		},
		{
			name: "undefined reference",
			expr: "${undefinedRef}",
			expressions: map[string]string{
				"validPort": "values.service.port <= 65535",
			},
			wantErr: "undefined reference in expression: ${undefinedRef}",
		},
		{
			name: "multiple identical references",
			expr: "${validPort} && ${validPort}",
			expressions: map[string]string{
				"validPort": "values.service.port <= 65535",
			},
			want: "(values.service.port <= 65535) && (values.service.port <= 65535)",
		},
		{
			name:        "empty expressions map",
			expr:        "${validPort}",
			expressions: map[string]string{},
			wantErr:     "undefined reference in expression: ${validPort}",
		},
		{
			name:        "nil expressions map",
			expr:        "${validPort}",
			expressions: nil,
			wantErr:     "undefined reference in expression: ${validPort}",
		},
		{
			name: "empty expression",
			expr: "",
			expressions: map[string]string{
				"validPort": "values.service.port <= 65535",
			},
			want: "",
		},
		{
			name: "malformed reference - unclosed",
			expr: "${unclosed",
			expressions: map[string]string{
				"unclosed": "values.service.port <= 65535",
			},
			wantErr: "undefined reference in expression: ${unclosed",
		},
		{
			name: "complex nested expression",
			expr: "${validateResources}",
			expressions: map[string]string{
				"memoryPattern":     "matches(string(value), r\"^[0-9]+(Mi|Gi)$\")",
				"cpuPattern":        "matches(string(value), r\"^[0-9]+m$\")",
				"validateResources": "has(values.resources.requests) && has(values.resources.limits) && ${memoryPattern} && ${cpuPattern}",
			},
			want: "(has(values.resources.requests) && has(values.resources.limits) && (matches(string(value), r\"^[0-9]+(Mi|Gi)$\")) && (matches(string(value), r\"^[0-9]+m$\")))",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				expressionProcessor := &ExpressionProcessor{}
				got, err := expressionProcessor.expandExpression(tt.expr, tt.expressions)

				if tt.wantErr != "" {
					assert.Error(t, err)
					assert.Equal(t, tt.wantErr, err.Error())
					return
				}

				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			},
		)
	}
}
