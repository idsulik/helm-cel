package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/idsulik/helm-cel/pkg/generator"
	"github.com/idsulik/helm-cel/pkg/validator"
	"github.com/spf13/cobra"
)

var (
	// Flags for generate command
	forceOverwrite bool
	genValuesFile  string
	outputFile     string

	// Flags for validate command
	valuesFile string
	rulesFile  string
)

const (
	validateShort = "Validate Helm values using CEL expressions"
	validateLong  = `A Helm plugin to validate values.yaml using CEL expressions defined in values.cel.yaml.
Example short: helm cel validate ./mychart
Example long: helm cel validate ./mychart --values-file prod.values.yaml --rules-file prod.values.cel.yaml`

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

	validateCmd.Flags().StringVarP(&valuesFile, "values-file", "v", "values.yaml", "Values file to validate")
	validateCmd.Flags().StringVarP(&rulesFile, "rules-file", "r", "values.cel.yaml", "Rules file to validate against")

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
		fmt.Fprintln(os.Stderr, err)
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

	// Construct full paths
	valuesPath := filepath.Join(absPath, valuesFile)
	rulesPath := filepath.Join(absPath, rulesFile)

	// Check if files exist
	if _, err := os.Stat(valuesPath); os.IsNotExist(err) {
		return fmt.Errorf("values file not found: %s", valuesPath)
	}
	if _, err := os.Stat(rulesPath); os.IsNotExist(err) {
		return fmt.Errorf("rules file not found: %s", rulesPath)
	}

	v := validator.New()
	result, err := v.ValidateChart(absPath, valuesFile, rulesFile)

	if err != nil {
		return err
	}

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
