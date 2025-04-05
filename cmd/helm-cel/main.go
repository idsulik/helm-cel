package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/idsulik/helm-cel/pkg/generator"
	"github.com/idsulik/helm-cel/pkg/models"
	"github.com/idsulik/helm-cel/pkg/validator"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	// Flags for generate command
	forceOverwrite bool
	genValuesFile  string
	outputFile     string

	// Flags for validate command
	valuesFiles  []string
	rulesFiles   []string
	outputFormat string
)

const (
	validateShort = "Validate Helm values using CEL expressions"
	validateLong  = `A Helm plugin to validate values.yaml using CEL expressions defined in .cel.yaml files.
Example using defaults: helm cel validate ./mychart
Example with specific values: helm cel validate ./mychart -v values1.yaml -v values2.yaml
Example with multiple files: helm cel validate ./mychart -v prod.yaml,staging.yaml -r rules1.cel.yaml,rules2.cel.yaml
Example with JSON output: helm cel validate ./mychart -o json
Example with YAML output: helm cel validate ./mychart -o yaml`

	generateShort = "Generate CEL validation rules from values.yaml"
	generateLong  = `Generate values.cel.yaml file with validation rules based on the structure of values.yaml.
Example: helm cel generate ./mychart
Example with custom values file: helm cel generate ./mychart --values-file prod.values.yaml
Example with force overwrite: helm cel generate ./mychart --force`
)

var rootCmd = &cobra.Command{}

var validateCmd = &cobra.Command{
	Use:           "validate [flags] CHART",
	Short:         validateShort,
	Long:          validateLong,
	RunE:          runValidator,
	SilenceErrors: true,
	SilenceUsage:  true,
}

var generateCmd = &cobra.Command{
	Use:           "generate [flags] CHART",
	Short:         generateShort,
	Long:          generateLong,
	RunE:          runGenerator,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(generateCmd)

	validateCmd.Flags().StringSliceVarP(
		&valuesFiles,
		"values-file",
		"v",
		[]string{"values.yaml"},
		"Values files to validate (comma-separated or multiple -v flags)",
	)
	validateCmd.Flags().StringSliceVarP(
		&rulesFiles,
		"rules-file",
		"r",
		[]string{"values.cel.yaml"},
		"Rules files to validate against (comma-separated or multiple -r flags)",
	)
	validateCmd.Flags().StringVarP(
		&outputFormat,
		"output",
		"o",
		"text",
		"Output format: text, json, or yaml",
	)

	generateCmd.Flags().BoolVarP(&forceOverwrite, "force", "f", false, "Force overwrite existing values.cel.yaml")
	generateCmd.Flags().StringVarP(
		&genValuesFile,
		"values-file",
		"v",
		"values.yaml",
		"Values file to generate rules from",
	)
	generateCmd.Flags().StringVarP(
		&outputFile,
		"output-file",
		"o",
		"values.cel.yaml",
		"Output file for generated rules",
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runValidator(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("chart path is required")
	}

	chartPath := args[0]
	absPath, err := filepath.Abs(chartPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	v := validator.New()
	result, err := v.ValidateChart(absPath, valuesFiles, rulesFiles)

	if err != nil {
		return err
	}

	output := models.ValidationOutput{
		HasErrors:   result.HasErrors(),
		HasWarnings: len(result.Warnings) > 0,
		Result:      result,
	}

	switch outputFormat {
	case "json":
		if err := outputJson(output); err != nil {
			return err
		}
	case "yaml":
		if err := outputYaml(output); err != nil {
			return err
		}
	default:
		if result.HasErrors() {
			return result
		}

		if len(result.Warnings) > 0 {
			fmt.Println(result.Error())
			fmt.Println("-------------------------------------------------")
			fmt.Println("⚠️✅ Values validation successful with warnings!")
		} else {
			fmt.Println("✅ Values validation successful!")
		}
	}

	return nil
}

func outputJson(output models.ValidationOutput) error {
	json, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal output to JSON: %v", err)
	}
	fmt.Println(string(json))
	return nil
}

func outputYaml(output models.ValidationOutput) error {
	yaml, err := yaml.Marshal(output)
	if err != nil {
		return fmt.Errorf("failed to marshal output to YAML: %v", err)
	}
	fmt.Println(string(yaml))
	return nil
}

func runGenerator(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("chart path is required")
	}

	chartPath := args[0]
	absPath, err := filepath.Abs(chartPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Construct paths
	valuesPath := filepath.Join(absPath, genValuesFile)
	celPath := filepath.Join(absPath, outputFile)

	// Check if values file exists
	if _, err := os.Stat(valuesPath); os.IsNotExist(err) {
		return fmt.Errorf("values file not found: %s", valuesPath)
	}

	// Check if output file exists and handle force flag
	if !forceOverwrite {
		if _, err := os.Stat(celPath); err == nil {
			return fmt.Errorf("output file already exists: %s (use --force to overwrite)", celPath)
		}
	}

	g := generator.New()

	// Generate rules using the specified values file
	rules, err := g.GenerateRules(absPath, genValuesFile)
	if err != nil {
		return fmt.Errorf("failed to generate rules: %v", err)
	}

	// Write to output file
	if err := g.WriteRules(celPath, rules); err != nil {
		return fmt.Errorf("failed to write rules: %v", err)
	}

	fmt.Printf("✅ Successfully generated %s\n", celPath)
	return nil
}
